// Package wasm provides browser-side search and lookup logic.
package wasm

import (
	"encoding/json"
	"strings"
)

// SearchCtx holds the deserialized index + pre-computed data structures.
// Allocated in WASM linear memory; JS glue reads/writes via memory.buffer.
type SearchCtx struct {
	Index        *Index
	byPackageID  map[string]*Package
	byFileID     map[string]*File
	bySymbolID   map[string]*Symbol
	SearchIndex  map[string][]string    // lowercase name → symbol IDs
	XrefIndex    map[string]*XrefData   // symbol ID → pre-computed xref
	Snippets     map[string]string      // symID:kind → text
	DocHTML      map[string]string      // slug → pre-rendered HTML
	DocManifest  []PageMeta             // pages list
}

// PageMeta mirrors docs.PageMeta.
type PageMeta struct {
	Slug  string `json:"slug"`
	Title string `json:"title"`
	Path  string `json:"path"`
}

// XrefData is pre-computed cross-reference data per symbol.
type XrefData struct {
	UsedBy []RefSummary `json:"usedBy"`
	Uses   []UseTarget  `json:"uses"`
}

// RefSummary is a lightweight ref for "usedBy" (who calls this symbol).
type RefSummary struct {
	FromSymbolID string `json:"fromSymbolId"`
	Kind         string `json:"kind"`
	StartLine    int    `json:"startLine"`
	EndLine      int    `json:"endLine"`
}

// UseTarget is a deduplicated "uses" entry (what this symbol calls).
type UseTarget struct {
	ToSymbolID string `json:"toSymbolId"`
	Kind       string `json:"kind"`
	Count      int    `json:"count"`
	Occurrences []RefOccurrence `json:"occurrences"`
}

// RefOccurrence is a single occurrence of a ref.
type RefOccurrence struct {
	StartLine int `json:"startLine"`
	EndLine   int `json:"endLine"`
}

// Init loads all index data from JSON byte slices.
// Called once from JS after WASM loads.
func Init(jsonIndex, jsonSearchIdx, jsonXrefIdx, jsonSnippets, jsonDocManifest, jsonDocHTML []byte) (*SearchCtx, error) {
	var idx Index
	if err := json.Unmarshal(jsonIndex, &idx); err != nil {
		return nil, err
	}

	var searchIdx map[string][]string
	if err := json.Unmarshal(jsonSearchIdx, &searchIdx); err != nil {
		return nil, err
	}

	var xrefIdx map[string]*XrefData
	if err := json.Unmarshal(jsonXrefIdx, &xrefIdx); err != nil {
		return nil, err
	}

	var snippets map[string]string
	if err := json.Unmarshal(jsonSnippets, &snippets); err != nil {
		return nil, err
	}

	var docManifest []PageMeta
	if err := json.Unmarshal(jsonDocManifest, &docManifest); err != nil {
		return nil, err
	}

	var docHTML map[string]string
	if err := json.Unmarshal(jsonDocHTML, &docHTML); err != nil {
		return nil, err
	}

	ctx := &SearchCtx{
		Index:       &idx,
		byPackageID: make(map[string]*Package, len(idx.Packages)),
		byFileID:    make(map[string]*File, len(idx.Files)),
		bySymbolID:  make(map[string]*Symbol, len(idx.Symbols)),
		SearchIndex: searchIdx,
		XrefIndex:   xrefIdx,
		Snippets:    snippets,
		DocHTML:     docHTML,
		DocManifest: docManifest,
	}

	for i := range idx.Packages {
		ctx.byPackageID[idx.Packages[i].ID] = &idx.Packages[i]
	}
	for i := range idx.Files {
		ctx.byFileID[idx.Files[i].ID] = &idx.Files[i]
	}
	for i := range idx.Symbols {
		ctx.bySymbolID[idx.Symbols[i].ID] = &idx.Symbols[i]
	}

	return ctx, nil
}

// FindSymbols performs substring-match search over symbol names.
// nameQuery is lowercased before matching. If empty, matches everything.
// kind filters by symbol kind (e.g. "func", "type"); empty means all.
// Returns JSON-encoded []*Symbol, capped at 200.
func (s *SearchCtx) FindSymbols(nameQuery, kind string) []byte {
	nameQuery = strings.ToLower(nameQuery)
	out := []*Symbol{}
	for i := range s.Index.Symbols {
		sym := &s.Index.Symbols[i]
		if kind != "" && sym.Kind != kind {
			continue
		}
		if nameQuery == "" || strings.Contains(strings.ToLower(sym.Name), nameQuery) {
			out = append(out, sym)
		}
		if len(out) >= 200 {
			break
		}
	}
	data, _ := json.Marshal(out)
	return data
}

// GetSymbol returns the symbol with the given ID, or null JSON if not found.
func (s *SearchCtx) GetSymbol(id string) []byte {
	sym, ok := s.bySymbolID[id]
	if !ok {
		return []byte("null")
	}
	data, _ := json.Marshal(sym)
	return data
}

// GetXref returns pre-computed cross-reference data for a symbol.
func (s *SearchCtx) GetXref(id string) []byte {
	data, _ := json.Marshal(s.XrefIndex[id])
	return data
}

// GetSnippet returns the pre-extracted snippet text for a symbol.
func (s *SearchCtx) GetSnippet(id, kind string) []byte {
	key := id + ":" + kind
	text, ok := s.Snippets[key]
	if !ok {
		// Fallback to declaration
		text, ok = s.Snippets[id+":declaration"]
	}
	if !ok {
		// Last resort: bare id
		text = s.Snippets[id]
	}
	data, _ := json.Marshal(map[string]string{"text": text})
	return data
}

// GetPackages returns a lightweight package summary.
func (s *SearchCtx) GetPackages() []byte {
	type lite struct {
		ID         string `json:"id"`
		ImportPath string `json:"importPath"`
		Name       string `json:"name"`
		Files      int    `json:"files"`
		Symbols    int    `json:"symbols"`
	}
	out := make([]lite, 0, len(s.Index.Packages))
	for i := range s.Index.Packages {
		p := &s.Index.Packages[i]
		out = append(out, lite{
			ID:         p.ID,
			ImportPath: p.ImportPath,
			Name:       p.Name,
			Files:      len(p.FileIDs),
			Symbols:    len(p.SymbolIDs),
		})
	}
	data, _ := json.Marshal(out)
	return data
}

// GetIndexSummary returns a lightweight summary of the index.
func (s *SearchCtx) GetIndexSummary() []byte {
	type summary struct {
		Module   string `json:"module"`
		Packages int    `json:"packages"`
		Symbols  int    `json:"symbols"`
		Files    int    `json:"files"`
	}
	data, _ := json.Marshal(summary{
		Module:   s.Index.Module,
		Packages: len(s.Index.Packages),
		Symbols:  len(s.Index.Symbols),
		Files:    len(s.Index.Files),
	})
	return data
}

// GetDocPages returns the doc page manifest.
func (s *SearchCtx) GetDocPages() []byte {
	data, _ := json.Marshal(s.DocManifest)
	return data
}

// GetDocPage returns the pre-rendered HTML for a doc page slug.
func (s *SearchCtx) GetDocPage(slug string) []byte {
	html, ok := s.DocHTML[slug]
	if !ok {
		return []byte("null")
	}
	data, _ := json.Marshal(map[string]string{
		"slug":  slug,
		"title": s.docPageTitle(slug),
		"html":  html,
	})
	return data
}

func (s *SearchCtx) docPageTitle(slug string) string {
	for _, p := range s.DocManifest {
		if p.Slug == slug {
			return p.Title
		}
	}
	return slug
}
