// Package static performs build-time pre-computation for the static WASM build.
package static

import (
	"fmt"
	"io/fs"
	"strings"

	"github.com/wesen/codebase-browser/internal/browser"
)

// ExtractSnippets reads source files and pre-extracts declaration, body, and
// signature text for every symbol. The result is a map of symID:kind → text.
func ExtractSnippets(loaded *browser.Loaded, sourceFS fs.FS) (map[string]string, error) {
	out := make(map[string]string)

	for i := range loaded.Index.Symbols {
		sym := &loaded.Index.Symbols[i]
		file, ok := loaded.File(sym.FileID)
		if !ok {
			continue
		}

		data, err := fs.ReadFile(sourceFS, file.Path)
		if err != nil {
			// Source file may not be available (e.g., external dependency)
			continue
		}

		start, end := sym.Range.StartOffset, sym.Range.EndOffset
		if start < 0 || end > len(data) || start > end {
			continue
		}

		snippet := string(data[start:end])

		// Store declaration (full text)
		out[sym.ID+":declaration"] = snippet

		// Store signature (first line)
		if nl := strings.IndexByte(snippet, '\n'); nl > 0 {
			out[sym.ID+":signature"] = strings.TrimSpace(snippet[:nl])
		} else {
			out[sym.ID+":signature"] = strings.TrimSpace(snippet)
		}

		// Store body (from first '{' to matching '}')
		if open := strings.IndexByte(snippet, '{'); open >= 0 {
			body := strings.TrimSpace(snippet[open:])
			out[sym.ID+":body"] = body
		}

		// Also store bare ID as declaration fallback
		out[sym.ID] = snippet
	}

	return out, nil
}

// ExtractSnippetRefs returns all refs whose byte ranges fall inside a symbol's
// declaration. These are used by the frontend to linkify tokens in code blocks.
func ExtractSnippetRefs(loaded *browser.Loaded) map[string][]SnippetRef {
	out := make(map[string][]SnippetRef)

	for i := range loaded.Index.Symbols {
		sym := &loaded.Index.Symbols[i]
		base := sym.Range.StartOffset
		end := sym.Range.EndOffset

		var refs []SnippetRef
		for j := range loaded.Index.Refs {
			ref := &loaded.Index.Refs[j]
			if ref.FileID != sym.FileID {
				continue
			}
			if ref.Range.StartOffset < base || ref.Range.EndOffset > end {
				continue
			}
			if _, known := loaded.Symbol(ref.ToSymbolID); !known {
				continue
			}
			refs = append(refs, SnippetRef{
				ToSymbolID:   ref.ToSymbolID,
				Kind:         ref.Kind,
				OffsetInSnip: ref.Range.StartOffset - base,
				Length:       ref.Range.EndOffset - ref.Range.StartOffset,
			})
		}

		out[sym.ID] = refs
	}

	return out
}

// ExtractSourceRefs returns all refs in a file with absolute offsets.
func ExtractSourceRefs(loaded *browser.Loaded) map[string][]SourceRef {
	out := make(map[string][]SourceRef)

	for j := range loaded.Index.Refs {
		ref := &loaded.Index.Refs[j]
		fileID := ref.FileID
		if _, ok := loaded.File(fileID); !ok {
			continue
		}
		if _, known := loaded.Symbol(ref.ToSymbolID); !known {
			continue
		}

		path := ""
		if f, ok := loaded.File(fileID); ok {
			path = f.Path
		}

		out[path] = append(out[path], SourceRef{
			ToSymbolID: ref.ToSymbolID,
			Kind:       ref.Kind,
			Offset:     ref.Range.StartOffset,
			Length:     ref.Range.EndOffset - ref.Range.StartOffset,
		})
	}

	return out
}

// SnippetRef is a ref inside a symbol's declaration.
type SnippetRef struct {
	ToSymbolID   string `json:"toSymbolId"`
	Kind         string `json:"kind"`
	OffsetInSnip int    `json:"offsetInSnippet"`
	Length       int    `json:"length"`
}

// SourceRef is a ref in a source file.
type SourceRef struct {
	ToSymbolID string `json:"toSymbolId"`
	Kind       string `json:"kind"`
	Offset     int    `json:"offset"`
	Length     int    `json:"length"`
}

// ResolveDocSnippets resolves codebase-* directives in a markdown string
// and returns the extracted text for each directive.
func ResolveDocSnippets(loaded *browser.Loaded, sourceFS fs.FS, mdSource []byte) (map[string]string, error) {
	// This is a simplified version that extracts snippets referenced in markdown.
	// The full implementation is in docs.Render().
	// For the static build, we use docs.Render() directly in the doc renderer.
	return nil, fmt.Errorf("use docs.Render() directly")
}
