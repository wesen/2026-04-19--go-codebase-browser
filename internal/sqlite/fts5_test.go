//go:build sqlite_fts5

package sqlite

import (
	"context"
	"testing"

	"github.com/wesen/codebase-browser/internal/indexer"
)

func TestEnableFTS5(t *testing.T) {
	idx := &indexer.Index{
		Version:     "test",
		GeneratedAt: "2026-04-23T00:00:00Z",
		Module:      "example.com/project",
		GoVersion:   "go1.25.5",
		Packages: []indexer.Package{{
			ID:         "pkg:example.com/project",
			ImportPath: "example.com/project",
			Name:       "project",
		}},
		Files: []indexer.File{{
			ID:        "file:main.go",
			Path:      "main.go",
			PackageID: "pkg:example.com/project",
		}},
		Symbols: []indexer.Symbol{{
			ID:        "sym:example.com/project.func.SearchIndex",
			Kind:      "func",
			Name:      "SearchIndex",
			PackageID: "pkg:example.com/project",
			FileID:    "file:main.go",
			Doc:       "SearchIndex builds a searchable symbol table.",
		}},
	}

	store, err := Create(t.TempDir() + "/codebase.db")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	if err := store.LoadFromIndex(ctx, idx); err != nil {
		t.Fatalf("LoadFromIndex() error = %v", err)
	}
	if err := store.EnableFTS5(ctx); err != nil {
		t.Fatalf("EnableFTS5() error = %v", err)
	}

	var name string
	if err := store.DB().QueryRowContext(ctx, `
SELECT name
FROM symbol_fts
WHERE symbol_fts MATCH ?
LIMIT 1`, "searchable").Scan(&name); err != nil {
		t.Fatalf("FTS query error = %v", err)
	}
	if name != "SearchIndex" {
		t.Fatalf("FTS query name = %q, want SearchIndex", name)
	}
}
