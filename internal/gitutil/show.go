package gitutil

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// ShowFile reads a file's content at a specific commit.
// Equivalent to: git show <hash>:<path>
func ShowFile(ctx context.Context, repoRoot, commitHash, filePath string) ([]byte, error) {
	ref := commitHash + ":" + filePath
	cmd := exec.CommandContext(ctx, "git", "show", ref)
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git show %s: %w", ref, err)
	}
	return out, nil
}

// FileBlobHash returns the git blob hash for a file at a commit.
// Equivalent to: git rev-parse <hash>:<path>
func FileBlobHash(ctx context.Context, repoRoot, commitHash, filePath string) (string, error) {
	ref := commitHash + ":" + filePath
	cmd := exec.CommandContext(ctx, "git", "rev-parse", ref)
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse %s: %w", ref, err)
	}
	return strings.TrimSpace(string(out)), nil
}
