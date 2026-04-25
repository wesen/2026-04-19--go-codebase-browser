package gitutil_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/wesen/codebase-browser/internal/gitutil"
)

func TestCreateAndRemoveWorktree(t *testing.T) {
	dir := setupTestRepo(t)
	ctx := context.Background()

	commits, err := gitutil.LogCommits(ctx, dir, "")
	if err != nil {
		t.Fatal(err)
	}

	// Create worktree at the second commit.
	wt, err := gitutil.CreateWorktree(ctx, dir, commits[1].Hash)
	if err != nil {
		t.Fatal(err)
	}

	// Worktree should exist and contain main.go.
	mainGo := filepath.Join(wt, "main.go")
	data, err := os.ReadFile(mainGo)
	if err != nil {
		t.Fatalf("read main.go from worktree: %v", err)
	}
	// Second commit has "v2".
	content := string(data)
	if !contains(content, "v2") {
		t.Errorf("worktree content doesn't match commit: %s", content)
	}

	// Remove worktree.
	if err := gitutil.RemoveWorktree(ctx, dir, wt); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(wt); !os.IsNotExist(err) {
		t.Error("worktree directory still exists after remove")
	}
}

func TestShowFile(t *testing.T) {
	dir := setupTestRepo(t)
	ctx := context.Background()

	commits, err := gitutil.LogCommits(ctx, dir, "")
	if err != nil {
		t.Fatal(err)
	}

	// Show main.go at the first (oldest) commit.
	data, err := gitutil.ShowFile(ctx, dir, commits[2].Hash, "main.go")
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !contains(content, "v1") {
		t.Errorf("expected v1 content, got: %s", content)
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
