package history

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/wesen/codebase-browser/internal/gitutil"
	"github.com/wesen/codebase-browser/internal/history"
)

func newScanCmd() *cobra.Command {
	var (
		dbPath       string
		rangeSpec    string
		branch       string
		incremental  bool
		worktrees    bool
		fileFilters  []string
	)

	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Index git commits into the history database",
		Long: `Scan git commits and extract a per-commit codebase index.

For each commit in the range, the indexer creates a git worktree (when --worktrees
is set), runs the Go AST extractor, and stores the resulting snapshot (packages,
files, symbols, refs) in the history database.

Examples:
  codebase-browser history scan --range "HEAD~5..HEAD" --worktrees
  codebase-browser history scan --range "main..feature-branch" --worktrees --incremental
  codebase-browser history scan --range "HEAD~10..HEAD" --worktrees --filter internal/server/`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			repoRoot, err := findRepoRoot()
			if err != nil {
				return err
			}

			store, err := openOrCreateStore(dbPath)
			if err != nil {
				return err
			}
			defer store.Close()

			// Resolve range spec.
			if rangeSpec == "" {
				rangeSpec = "HEAD"
			}

			// Scan commits.
			scanOpts := history.ScanOptions{
				RepoRoot:    repoRoot,
				Range:       rangeSpec,
				Branch:      branch,
				FileFilter:  fileFilters,
				Incremental: incremental,
			}
			scanResult, err := history.ScanCommits(ctx, store, scanOpts)
			if err != nil {
				return fmt.Errorf("scan commits: %w", err)
			}

			fmt.Fprintf(os.Stderr, "scan: %d commits to index, %d skipped, %d filtered\n",
				len(scanResult.Commits), scanResult.Skipped, scanResult.Filtered)

			if len(scanResult.Commits) == 0 {
				fmt.Println("No commits to index.")
				return nil
			}

			// Index commits.
			idxOpts := history.IndexOptions{
				RepoRoot:     repoRoot,
				Commits:      scanResult.Commits,
				Patterns:     defaultPatterns(),
				IncludeTests: true,
				Worktrees:    worktrees,
				OnProgress: func(done, total int, shortHash, message string) {
					fmt.Fprintf(os.Stderr, "  [%d/%d] %s %s\n", done, total, shortHash, message)
				},
			}
			result, err := history.IndexCommits(ctx, store, idxOpts)
			if err != nil {
				return err
			}

			fmt.Fprintf(os.Stderr, "\nDone in %s: %d indexed, %d failed\n",
				result.Duration.Round(time.Millisecond), result.Indexed, result.Failed)
			for _, idxErr := range result.Errors {
				fmt.Fprintf(os.Stderr, "  ERROR %s: %v\n", idxErr.ShortHash, idxErr.Err)
			}

			if result.Failed > 0 {
				os.Exit(1)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&dbPath, "db", "history.db", "Path to history database")
	cmd.Flags().StringVar(&rangeSpec, "range", "", "Git log range spec (e.g. HEAD~20..HEAD)")
	cmd.Flags().StringVar(&branch, "branch", "", "Branch name for metadata")
	cmd.Flags().BoolVar(&incremental, "incremental", false, "Skip already-indexed commits")
	cmd.Flags().BoolVar(&worktrees, "worktrees", false, "Use git worktrees for per-commit extraction")
	cmd.Flags().StringArrayVar(&fileFilters, "filter", nil, "Only index commits touching these paths")

	return cmd
}

func openOrCreateStore(dbPath string) (*history.Store, error) {
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return history.Create(dbPath)
	}
	return history.Open(dbPath)
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	// Validate it's a git repo by resolving HEAD.
	_, err = gitutil.ResolveRef(context.Background(), dir, "HEAD")
	if err != nil {
		return "", fmt.Errorf("%s is not a git repo: %w", dir, err)
	}
	return dir, nil
}

func defaultPatterns() []string {
	return []string{"./cmd/...", "./internal/..."}
}
