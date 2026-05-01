package review

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/wesen/codebase-browser/internal/gitutil"
	"github.com/wesen/codebase-browser/internal/indexer"
)

func TestLoadForExport(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := Create(dbPath)
	if err != nil {
		t.Fatalf("create review db: %v", err)
	}

	// Insert two commits with different snapshots.
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
		Packages: []indexer.Package{
			{ID: "pkg:test", ImportPath: "test", Name: "test"},
		},
		Files: []indexer.File{
			{ID: "file:a.go", Path: "a.go", PackageID: "pkg:test"},
		},
		Symbols: []indexer.Symbol{
			{ID: "sym:test.func.Foo", Kind: "func", Name: "Foo", PackageID: "pkg:test", FileID: "file:a.go", Signature: "func Foo()"},
		},
		Refs: []indexer.Ref{},
	}
	idx2 := &indexer.Index{
		Version: "1",
		Packages: []indexer.Package{
			{ID: "pkg:test", ImportPath: "test", Name: "test"},
		},
		Files: []indexer.File{
			{ID: "file:a.go", Path: "a.go", PackageID: "pkg:test"},
		},
		Symbols: []indexer.Symbol{
			{ID: "sym:test.func.Foo", Kind: "func", Name: "Foo", PackageID: "pkg:test", FileID: "file:a.go", Signature: "func Foo(x int)"},
		},
		Refs: []indexer.Ref{},
	}

	if err := store.History.LoadSnapshot(context.Background(), commit1, idx1, ""); err != nil {
		t.Fatalf("load snapshot 1: %v", err)
	}
	if err := store.History.LoadSnapshot(context.Background(), commit2, idx2, ""); err != nil {
		t.Fatalf("load snapshot 2: %v", err)
	}

	store.Close()

	out, err := LoadForExport(dbPath)
	if err != nil {
		t.Fatalf("load for export: %v", err)
	}

	if len(out.Commits) != 2 {
		t.Fatalf("expected 2 commits, got %d", len(out.Commits))
	}
	if len(out.Diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(out.Diffs))
	}
	if len(out.Histories) != 1 {
		t.Fatalf("expected 1 history, got %d", len(out.Histories))
	}

	// Verify diff contains the signature change.
	for k, diff := range out.Diffs {
		t.Logf("diff %s: symbols=%d", k, len(diff.Symbols))
		found := false
		for _, s := range diff.Symbols {
			if s.SymbolID == "sym:test.func.Foo" {
				found = true
				if s.ChangeType != "signature-changed" {
					t.Fatalf("expected signature-changed, got %s", s.ChangeType)
				}
			}
		}
		if !found {
			t.Fatal("expected Foo symbol in diff")
		}
	}
}
