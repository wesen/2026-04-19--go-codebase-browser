//go:build ignore

// generate_build.go is invoked by `go generate ./internal/static`.
// It loads the built index.json, pre-computes search data, xref data,
// snippets, and doc pages, and writes precomputed.json.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/wesen/codebase-browser/internal/browser"
	"github.com/wesen/codebase-browser/internal/docs"
	"github.com/wesen/codebase-browser/internal/static"
)

func main() {
	root, err := findRepoRoot()
	if err != nil {
		log.Fatal(err)
	}

	// 1. Load index
	idxPath := filepath.Join(root, "internal", "indexfs", "embed", "index.json")
	loaded, err := browser.LoadFromFile(idxPath)
	if err != nil {
		log.Fatal("load index:", err)
	}
	fmt.Printf("Loaded index: %d packages, %d files, %d symbols, %d refs\n",
		len(loaded.Index.Packages), len(loaded.Index.Files),
		len(loaded.Index.Symbols), len(loaded.Index.Refs))

	// 2. Build search index (use fast variant for reasonable size)
	searchIdx := static.BuildSearchIndexFast(loaded)
	fmt.Printf("Built search index: %d keys\n", len(searchIdx))

	// 3. Build xref index
	xrefIdx := static.BuildXrefIndex(loaded)
	fmt.Printf("Built xref index: %d symbols\n", len(xrefIdx))

	// 4. Extract snippets
	sourceFS := os.DirFS(filepath.Join(root, "internal", "sourcefs", "embed", "source"))
	snippets, err := static.ExtractSnippets(loaded, sourceFS)
	if err != nil {
		log.Fatal("extract snippets:", err)
	}
	fmt.Printf("Extracted snippets: %d entries\n", len(snippets))

	// 5. Extract snippet refs
	snippetRefs := static.ExtractSnippetRefs(loaded)
	fmt.Printf("Extracted snippet refs: %d symbols\n", len(snippetRefs))

	// 6. Extract source refs
	sourceRefs := static.ExtractSourceRefs(loaded)
	fmt.Printf("Extracted source refs: %d files\n", len(sourceRefs))

	// 7. Render doc pages
	pagesFS := docs.PagesFS()
	renderer := &static.DocRenderer{
		Loaded:   loaded,
		SourceFS: sourceFS,
		PagesFS:  pagesFS,
	}
	manifest, docHTML, renderErrors, err := renderer.RenderAll()
	if err != nil {
		log.Fatal("render docs:", err)
	}
	fmt.Printf("Rendered docs: %d pages, %d errors\n", len(manifest), len(renderErrors))
	for _, e := range renderErrors {
		fmt.Println("  doc error:", e)
	}

	// 8. Build file xref index
	fileXrefIdx := static.BuildFileXrefIndex(loaded)
	fmt.Printf("Built file xref index: %d files\n", len(fileXrefIdx))

	// 9. Assemble precomputed data
	// Include raw index JSON so the WASM can serve /api/index responses
	rawIndex, err := os.ReadFile(idxPath)
	if err != nil {
		log.Fatal("read raw index:", err)
	}

	precomputed := map[string]interface{}{
		"version":       "1",
		"module":        loaded.Index.Module,
		"generatedAt":   loaded.Index.GeneratedAt,
		"indexJSON":     json.RawMessage(rawIndex),
		"searchIndex":   searchIdx,
		"xrefIndex":     xrefIdx,
		"fileXrefIndex": fileXrefIdx,
		"snippets":      snippets,
		"snippetRefs":   snippetRefs,
		"sourceRefs":    sourceRefs,
		"docManifest":   manifest,
		"docHTML":       docHTML,
	}

	// 10. Write precomputed.json
	outDir := filepath.Join(root, "internal", "static", "embed")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		log.Fatal(err)
	}
	outPath := filepath.Join(outDir, "precomputed.json")
	data, err := json.MarshalIndent(precomputed, "", "  ")
	if err != nil {
		log.Fatal("marshal:", err)
	}
	if err := os.WriteFile(outPath, data, 0644); err != nil {
		log.Fatal("write:", err)
	}

	info, _ := os.Stat(outPath)
	fmt.Printf("Wrote %s (%.1f KB)\n", outPath, float64(info.Size())/1024)
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("go.mod not found (started from %s)", dir)
}
