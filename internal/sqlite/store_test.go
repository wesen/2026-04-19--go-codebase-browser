package sqlite

import (
	"context"
	"database/sql"
	"testing"

	"github.com/wesen/codebase-browser/internal/indexer"
)

func TestLoadFromIndex(t *testing.T) {
	idx := &indexer.Index{
		Version:     "test",
		GeneratedAt: "2026-04-23T00:00:00Z",
		Module:      "example.com/project",
		GoVersion:   "go1.25.5",
		Packages: []indexer.Package{{
			ID:         "pkg:example.com/project",
			ImportPath: "example.com/project",
			Name:       "project",
			Language:   "go",
		}},
		Files: []indexer.File{{
			ID:        "file:main.go",
			Path:      "main.go",
			PackageID: "pkg:example.com/project",
			LineCount: 10,
			SHA256:    "abc",
			Language:  "go",
		}},
		Symbols: []indexer.Symbol{
			{
				ID:        "sym:example.com/project.func.Main",
				Kind:      "func",
				Name:      "Main",
				PackageID: "pkg:example.com/project",
				FileID:    "file:main.go",
				Range:     indexer.Range{StartLine: 3, EndLine: 5},
				Exported:  true,
				Language:  "go",
			},
			{
				ID:        "sym:example.com/project.func.helper",
				Kind:      "func",
				Name:      "helper",
				PackageID: "pkg:example.com/project",
				FileID:    "file:main.go",
				Range:     indexer.Range{StartLine: 7, EndLine: 9},
				Language:  "go",
			},
		},
		Refs: []indexer.Ref{{
			FromSymbolID: "sym:example.com/project.func.Main",
			ToSymbolID:   "sym:example.com/project.func.helper",
			Kind:         "call",
			FileID:       "file:main.go",
			Range:        indexer.Range{StartLine: 4, EndLine: 4},
		}},
	}

	store, err := Create(t.TempDir() + "/codebase.db")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	defer store.Close()

	if err := store.LoadFromIndex(context.Background(), idx); err != nil {
		t.Fatalf("LoadFromIndex() error = %v", err)
	}

	assertCount(t, store.DB(), "packages", len(idx.Packages))
	assertCount(t, store.DB(), "files", len(idx.Files))
	assertCount(t, store.DB(), "symbols", len(idx.Symbols))
	assertCount(t, store.DB(), "refs", len(idx.Refs))

	rows, err := store.FindSymbols(context.Background(), ByKind("func"), NameLike("main"), IsExported(), Limit(10))
	if err != nil {
		t.Fatalf("FindSymbols() error = %v", err)
	}
	if len(rows) != 1 || rows[0].Name != "Main" {
		t.Fatalf("FindSymbols() = %#v, want exported Main", rows)
	}
}

func assertCount(t *testing.T, db *sql.DB, table string, want int) {
	t.Helper()
	var got int
	if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&got); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	if got != want {
		t.Fatalf("count %s = %d, want %d", table, got, want)
	}
}
