// Package static performs build-time pre-computation for the static WASM build.
package static

import (
	"github.com/wesen/codebase-browser/internal/browser"
)

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
	ToSymbolID   string          `json:"toSymbolId"`
	Kind         string          `json:"kind"`
	Count        int             `json:"count"`
	Occurrences  []RefOccurrence `json:"occurrences"`
}

// RefOccurrence is a single occurrence of a ref.
type RefOccurrence struct {
	StartLine int `json:"startLine"`
	EndLine   int `json:"endLine"`
}

// BuildXrefIndex pre-computes cross-reference data for every symbol.
// It walks the Refs slice once and groups by ToSymbolID (usedBy) and
// FromSymbolID (uses, deduplicated by target).
func BuildXrefIndex(loaded *browser.Loaded) map[string]*XrefData {
	out := make(map[string]*XrefData)

	// First pass: group refs by target (usedBy) and source (uses)
	usedByMap := make(map[string][]RefSummary)
	usesMap := make(map[string]map[string]*UseTarget)

	for i := range loaded.Index.Refs {
		ref := &loaded.Index.Refs[i]

		// usedBy: refs TO this symbol
		usedByMap[ref.ToSymbolID] = append(usedByMap[ref.ToSymbolID], RefSummary{
			FromSymbolID: ref.FromSymbolID,
			Kind:         ref.Kind,
			StartLine:    ref.Range.StartLine,
			EndLine:      ref.Range.EndLine,
		})

		// uses: refs FROM this symbol
		if usesMap[ref.FromSymbolID] == nil {
			usesMap[ref.FromSymbolID] = make(map[string]*UseTarget)
		}
		target, ok := usesMap[ref.FromSymbolID][ref.ToSymbolID]
		if !ok {
			target = &UseTarget{
				ToSymbolID:  ref.ToSymbolID,
				Kind:        ref.Kind,
				Occurrences: []RefOccurrence{},
			}
			usesMap[ref.FromSymbolID][ref.ToSymbolID] = target
		}
		target.Count++
		if len(target.Occurrences) < 5 {
			target.Occurrences = append(target.Occurrences, RefOccurrence{
				StartLine: ref.Range.StartLine,
				EndLine:   ref.Range.EndLine,
			})
		}
	}

	// Build output, capping usedBy at 200 entries
	for i := range loaded.Index.Symbols {
		symID := loaded.Index.Symbols[i].ID
		data := &XrefData{UsedBy: []RefSummary{}, Uses: []UseTarget{}}

		if refs, ok := usedByMap[symID]; ok {
			if len(refs) > 200 {
				refs = refs[:200]
			}
			data.UsedBy = refs
		}

		if targets, ok := usesMap[symID]; ok {
			for _, t := range targets {
				data.Uses = append(data.Uses, *t)
			}
		}

		out[symID] = data
	}

	return out
}

// BuildFileXrefIndex pre-computes cross-reference data for every file.
func BuildFileXrefIndex(loaded *browser.Loaded) map[string]*FileXrefData {
	out := make(map[string]*FileXrefData)

	// Collect symbol IDs per file
	inFile := make(map[string]map[string]bool)
	for i := range loaded.Index.Symbols {
		sym := &loaded.Index.Symbols[i]
		if inFile[sym.FileID] == nil {
			inFile[sym.FileID] = make(map[string]bool)
		}
		inFile[sym.FileID][sym.ID] = true
	}

	// Walk refs
	for i := range loaded.Index.Refs {
		ref := &loaded.Index.Refs[i]
		fileID := ref.FileID
		path := ""
		if f, ok := loaded.File(fileID); ok {
			path = f.Path
		}
		if path == "" {
			continue
		}

		if out[path] == nil {
			out[path] = &FileXrefData{Path: path, UsedBy: []FileRef{}, Uses: []FileUseTarget{}}
		}

		toInFile := inFile[fileID][ref.ToSymbolID]
		fromInFile := inFile[fileID][ref.FromSymbolID]

		if toInFile && !fromInFile {
			// used by: ref from outside into one of our symbols
			out[path].UsedBy = append(out[path].UsedBy, FileRef{
				FromSymbolID: ref.FromSymbolID,
				ToSymbolID:   ref.ToSymbolID,
				Kind:         ref.Kind,
				StartLine:    ref.Range.StartLine,
				EndLine:      ref.Range.EndLine,
			})
		} else if fromInFile && !toInFile {
			// uses: ref from one of our symbols out to a target elsewhere
			if _, known := loaded.Symbol(ref.ToSymbolID); !known {
				continue
			}
			out[path].addUse(ref.ToSymbolID, ref.Kind, ref.Range.StartLine, ref.Range.EndLine)
		}
	}

	// Cap usedBy at 200
	for _, data := range out {
		if len(data.UsedBy) > 200 {
			data.UsedBy = data.UsedBy[:200]
		}
	}

	return out
}

// FileXrefData is pre-computed xref data for a file.
type FileXrefData struct {
	Path   string           `json:"path"`
	UsedBy []FileRef        `json:"usedBy"`
	Uses   []FileUseTarget  `json:"uses"`
}

// FileRef is a ref in the context of a file.
type FileRef struct {
	FromSymbolID string `json:"fromSymbolId"`
	ToSymbolID   string `json:"toSymbolId"`
	Kind         string `json:"kind"`
	StartLine    int    `json:"startLine"`
	EndLine      int    `json:"endLine"`
}

// FileUseTarget is a deduplicated use target in a file.
type FileUseTarget struct {
	ToSymbolID  string          `json:"toSymbolId"`
	Kind        string          `json:"kind"`
	Count       int             `json:"count"`
	Occurrences []RefOccurrence `json:"occurrences"`
}

func (f *FileXrefData) addUse(toSymID, kind string, startLine, endLine int) {
	for i := range f.Uses {
		if f.Uses[i].ToSymbolID == toSymID && f.Uses[i].Kind == kind {
			f.Uses[i].Count++
			if len(f.Uses[i].Occurrences) < 5 {
				f.Uses[i].Occurrences = append(f.Uses[i].Occurrences, RefOccurrence{
					StartLine: startLine,
					EndLine:   endLine,
				})
			}
			return
		}
	}
	f.Uses = append(f.Uses, FileUseTarget{
		ToSymbolID:  toSymID,
		Kind:        kind,
		Count:       1,
		Occurrences: []RefOccurrence{{StartLine: startLine, EndLine: endLine}},
	})
}
