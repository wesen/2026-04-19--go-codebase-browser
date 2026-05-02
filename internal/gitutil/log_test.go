package gitutil_test

import (
	"context"
	"testing"

	"github.com/go-go-golems/codebase-browser/internal/gitutil"
)

func TestLogCommits(t *testing.T) {
	dir := setupTestRepo(t)
	ctx := context.Background()

	commits, err := gitutil.LogCommits(ctx, dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) != 3 {
		t.Fatalf("expected 3 commits, got %d", len(commits))
	}

	// Most recent first.
	if commits[0].Message != "third commit" {
		t.Errorf("first commit message = %q, want %q", commits[0].Message, "third commit")
	}
	if commits[2].Message != "first commit" {
		t.Errorf("last commit message = %q, want %q", commits[2].Message, "first commit")
	}

	for _, c := range commits {
		if c.Hash == "" {
			t.Error("empty hash")
		}
		if c.ShortHash == "" {
			t.Error("empty short hash")
		}
		if len(c.ShortHash) > len(c.Hash) {
			t.Errorf("short hash %q longer than full hash %q", c.ShortHash, c.Hash)
		}
		if c.AuthorName != "Test" {
			t.Errorf("author = %q, want %q", c.AuthorName, "Test")
		}
	}
}

func TestLogCommitsRange(t *testing.T) {
	dir := setupTestRepo(t)
	ctx := context.Background()

	commits, err := gitutil.LogCommits(ctx, dir, "HEAD~1..HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) != 1 {
		t.Fatalf("expected 1 commit in range, got %d", len(commits))
	}
	if commits[0].Message != "third commit" {
		t.Errorf("commit message = %q, want %q", commits[0].Message, "third commit")
	}
}

func TestChangedFiles(t *testing.T) {
	dir := setupTestRepo(t)
	ctx := context.Background()

	commits, err := gitutil.LogCommits(ctx, dir, "")
	if err != nil {
		t.Fatal(err)
	}

	// Most recent commit should have changed main.go.
	files, err := gitutil.ChangedFiles(ctx, dir, commits[0].Hash)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("expected at least 1 changed file")
	}
	found := false
	for _, f := range files {
		if f == "main.go" {
			found = true
		}
	}
	if !found {
		t.Errorf("main.go not in changed files: %v", files)
	}
}

func TestResolveRef(t *testing.T) {
	dir := setupTestRepo(t)
	ctx := context.Background()

	hash, err := gitutil.ResolveRef(ctx, dir, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if len(hash) < 40 {
		t.Errorf("hash too short: %q", hash)
	}
}
