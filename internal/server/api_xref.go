package server

import (
	"net/http"
	"strings"

	"github.com/wesen/codebase-browser/internal/indexer"
)

type xrefResponse struct {
	ID     string          `json:"id"`
	UsedBy []indexer.Ref   `json:"usedBy"`
	Uses   []xrefUseTarget `json:"uses"`
}

type xrefUseTarget struct {
	ToSymbolID  string        `json:"toSymbolId"`
	Kind        string        `json:"kind"`
	Count       int           `json:"count"`
	Occurrences []indexer.Ref `json:"occurrences"`
}

// handleXref reports who uses the symbol (usedBy) and what the symbol itself
// references inside its body (uses, deduplicated by target). Both slices are
// bounded to keep response size predictable for heavily-used helpers.
func (s *Server) handleXref(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/xref/")
	if _, ok := s.Loaded.Symbol(id); !ok {
		http.Error(w, "symbol not found", http.StatusNotFound)
		return
	}
	resp := xrefResponse{ID: id}
	byTarget := map[string]*xrefUseTarget{}
	for i := range s.Loaded.Index.Refs {
		ref := &s.Loaded.Index.Refs[i]
		if ref.ToSymbolID == id {
			resp.UsedBy = append(resp.UsedBy, *ref)
			continue
		}
		if ref.FromSymbolID == id {
			target, ok := byTarget[ref.ToSymbolID]
			if !ok {
				target = &xrefUseTarget{
					ToSymbolID: ref.ToSymbolID,
					Kind:       ref.Kind,
				}
				byTarget[ref.ToSymbolID] = target
			}
			target.Count++
			if len(target.Occurrences) < 5 {
				target.Occurrences = append(target.Occurrences, *ref)
			}
		}
	}
	for _, t := range byTarget {
		resp.Uses = append(resp.Uses, *t)
	}
	if len(resp.UsedBy) > 200 {
		resp.UsedBy = resp.UsedBy[:200]
	}
	writeJSON(w, resp)
}

// SnippetRefView is the payload returned by /api/snippet-refs — one entry per
// identifier use inside a symbol's declaration, with offsets relative to the
// snippet bytes (not the whole file). This lets the frontend linkify tokens
// in a <Code> block without re-deriving byte offsets.
type SnippetRefView struct {
	ToSymbolID   string `json:"toSymbolId"`
	Kind         string `json:"kind"`
	OffsetInSnip int    `json:"offsetInSnippet"`
	Length       int    `json:"length"`
}

func (s *Server) handleSnippetRefs(w http.ResponseWriter, r *http.Request) {
	symID := r.URL.Query().Get("sym")
	sym, ok := s.Loaded.Symbol(symID)
	if !ok {
		http.Error(w, "symbol not found", http.StatusNotFound)
		return
	}
	base := sym.Range.StartOffset
	end := sym.Range.EndOffset
	out := []SnippetRefView{}
	for i := range s.Loaded.Index.Refs {
		ref := &s.Loaded.Index.Refs[i]
		if ref.FileID != sym.FileID {
			continue
		}
		if ref.Range.StartOffset < base || ref.Range.EndOffset > end {
			continue
		}
		// Only report refs where the target is indexed — the frontend filters
		// unresolvable ones anyway, and this keeps external-package noise down
		// unless the frontend explicitly asks.
		if _, known := s.Loaded.Symbol(ref.ToSymbolID); !known {
			continue
		}
		out = append(out, SnippetRefView{
			ToSymbolID:   ref.ToSymbolID,
			Kind:         ref.Kind,
			OffsetInSnip: ref.Range.StartOffset - base,
			Length:       ref.Range.EndOffset - ref.Range.StartOffset,
		})
	}
	writeJSON(w, out)
}
