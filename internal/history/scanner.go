package history

import (
	"context"
	"fmt"

	"github.com/wesen/codebase-browser/internal/gitutil"
)

// ScanOptions controls how commits are discovered for indexing.
type ScanOptions struct {
	RepoRoot    string
	Range       string // git log range spec (e.g. "HEAD~20..HEAD")
	Branch      string // branch name for metadata
	FileFilter  []string
	Incremental bool // skip already-indexed commits
}

// ScanResult describes what the scanner found.
type ScanResult struct {
	Commits  []gitutil.Commit
	Skipped  int // already indexed
	Filtered int // didn't touch tracked files
}

// ScanCommits discovers commits to index. It lists commits from git,
// optionally filters them by file paths, and optionally skips commits
// already present in the history store.
func ScanCommits(ctx context.Context, store *Store, opts ScanOptions) (*ScanResult, error) {
	commits, err := gitutil.LogCommits(ctx, opts.RepoRoot, opts.Range)
	if err != nil {
		return nil, fmt.Errorf("list commits: %w", err)
	}

	result := &ScanResult{}

	for _, commit := range commits {
		// Skip already-indexed commits if incremental mode is enabled.
		if opts.Incremental {
			has, err := store.HasCommit(ctx, commit.Hash)
			if err != nil {
				return nil, fmt.Errorf("check commit %s: %w", commit.ShortHash, err)
			}
			if has {
				result.Skipped++
				continue
			}
		}

		// Filter by file paths if specified.
		if len(opts.FileFilter) > 0 {
			changed, err := gitutil.ChangedFiles(ctx, opts.RepoRoot, commit.Hash)
			if err != nil {
				// Merge commits with no changed files (e.g., root commits) are fine.
				changed = nil
			}
			if !anyFileMatches(changed, opts.FileFilter) {
				result.Filtered++
				continue
			}
		}

		result.Commits = append(result.Commits, commit)
	}

	return result, nil
}

// anyFileMatches returns true if any file in changed has a prefix matching
// any of the filters.
func anyFileMatches(changed, filters []string) bool {
	for _, f := range changed {
		for _, filter := range filters {
			if len(f) >= len(filter) && f[:len(filter)] == filter {
				return true
			}
		}
	}
	return false
}
