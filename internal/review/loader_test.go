package review

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/wesen/codebase-browser/internal/gitutil"
	"github.com/wesen/codebase-browser/internal/indexer"
)

func TestLoadLatestSnapshot(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := Create(dbPath)
	if err != nil {
		t.Fatalf("create review db: %v", err)
	}
	defer store.Close()

	// Insert a commit and snapshot via the history store.
	commit := gitutil.Commit{
		Hash:        "abc123def456",
		ShortHash:   "abc123",
		Message:     "test commit",
		AuthorName:  "Test",
		AuthorEmail: "test@example.com",
		AuthorTime:  time.Now(),
	}
	idx := &indexer.Index{
		Version: "1",
		Module:  "github.com/test/module",
		Packages: []indexer.Package{
			{ID: "pkg:test", ImportPath: "github.com/test/module", Name: "module", FileIDs: []string{"file:a.go"}, SymbolIDs: []string{"sym:test.func.Main"}},
		},
		Files: []indexer.File{
			{ID: "file:a.go", Path: "a.go", PackageID: "pkg:test"},
		},
		Symbols: []indexer.Symbol{
			{ID: "sym:test.func.Main", Kind: "func", Name: "Main", PackageID: "pkg:test", FileID: "file:a.go"},
		},
		Refs: []indexer.Ref{},
	}

	if err := store.History.LoadSnapshot(context.Background(), commit, idx, ""); err != nil {
		t.Fatalf("load snapshot: %v", err)
	}

	// Now load the latest snapshot.
	loaded, err := LoadLatestSnapshot(context.Background(), store)
	if err != nil {
		t.Fatalf("load latest snapshot: %v", err)
	}

	// Module is not stored in snapshot tables, so it will be empty.
	// The important fields are packages, files, symbols.
	if len(loaded.Index.Packages) != 1 {
		t.Fatalf("expected 1 package, got %d", len(loaded.Index.Packages))
	}
	if len(loaded.Index.Symbols) != 1 {
		t.Fatalf("expected 1 symbol, got %d", len(loaded.Index.Symbols))
	}

	// Verify lookup maps are populated.
	sym, ok := loaded.Symbol("sym:test.func.Main")
	if !ok {
		t.Fatal("expected to find symbol sym:test.func.Main")
	}
	if sym.Name != "Main" {
		t.Fatalf("expected symbol name Main, got %s", sym.Name)
	}
}

func TestLoadLatestSnapshotEmptyDB(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := Create(dbPath)
	if err != nil {
		t.Fatalf("create review db: %v", err)
	}
	defer store.Close()

	_, err = LoadLatestSnapshot(context.Background(), store)
	if err == nil {
		t.Fatal("expected error for empty database")
	}
}

func TestLoadLatestSnapshotMultipleCommits(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := Create(dbPath)
	if err != nil {
		t.Fatalf("create review db: %v", err)
	}
	defer store.Close()

	// Insert two commits — the later one should be loaded.
	commit1 := gitutil.Commit{
		Hash:        "aaa111",
		ShortHash:   "aaa",
		Message:     "first",
		AuthorName:  "Test",
		AuthorEmail: "test@example.com",
		AuthorTime:  time.Now().Add(-time.Hour),
	}
	commit2 := gitutil.Commit{
		Hash:        "bbb222",
		ShortHash:   "bbb",
		Message:     "second",
		AuthorName:  "Test",
		AuthorEmail: "test@example.com",
		AuthorTime:  time.Now(),
	}

	idx1 := &indexer.Index{
		Version: "1",
		Module:  "module-v1",
		Packages: []indexer.Package{
			{ID: "pkg:v1", ImportPath: "v1", Name: "v1"},
		},
	}
	idx2 := &indexer.Index{
		Version: "1",
		Module:  "module-v2",
		Packages: []indexer.Package{
			{ID: "pkg:v2", ImportPath: "v2", Name: "v2"},
		},
	}

	if err := store.History.LoadSnapshot(context.Background(), commit1, idx1, ""); err != nil {
		t.Fatalf("load snapshot 1: %v", err)
	}
	if err := store.History.LoadSnapshot(context.Background(), commit2, idx2, ""); err != nil {
		t.Fatalf("load snapshot 2: %v", err)
	}

	loaded, err := LoadLatestSnapshot(context.Background(), store)
	if err != nil {
		t.Fatalf("load latest snapshot: %v", err)
	}

	// Module is not stored in snapshot tables.
	if len(loaded.Index.Packages) != 1 || loaded.Index.Packages[0].ID != "pkg:v2" {
		t.Fatalf("expected pkg:v2, got %+v", loaded.Index.Packages)
	}
}
