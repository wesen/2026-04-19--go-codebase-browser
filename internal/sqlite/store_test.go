package sqlite

import (
	"context"
	"database/sql"
	"testing"

	"github.com/go-go-golems/codebase-browser/internal/indexer"
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

	ctx := context.Background()
	rows, err := store.FindSymbols(ctx, ByKind("func"), NameLike("main"), IsExported(), Limit(10))
	if err != nil {
		t.Fatalf("FindSymbols() error = %v", err)
	}
	if len(rows) != 1 || rows[0].Name != "Main" {
		t.Fatalf("FindSymbols() = %#v, want exported Main", rows)
	}

	packages, err := store.FindPackages(ctx, PackageImportPathLike("project"), PackageLimit(10))
	if err != nil {
		t.Fatalf("FindPackages() error = %v", err)
	}
	if len(packages) != 1 || packages[0].Name != "project" {
		t.Fatalf("FindPackages() = %#v, want project", packages)
	}

	files, err := store.FindFiles(ctx, FilePathLike("main"), FileLimit(10))
	if err != nil {
		t.Fatalf("FindFiles() error = %v", err)
	}
	if len(files) != 1 || files[0].Path != "main.go" {
		t.Fatalf("FindFiles() = %#v, want main.go", files)
	}

	refs, err := store.FindRefs(ctx, RefFrom("sym:example.com/project.func.Main"), RefKind("call"), RefLimit(10))
	if err != nil {
		t.Fatalf("FindRefs() error = %v", err)
	}
	if len(refs) != 1 || refs[0].ToSymbolID != "sym:example.com/project.func.helper" {
		t.Fatalf("FindRefs() = %#v, want call to helper", refs)
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
