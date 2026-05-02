package history

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/wesen/codebase-browser/internal/gitutil"
	"github.com/wesen/codebase-browser/internal/history"
)

func newSymbolDiffCmd() *cobra.Command {
	var (
		dbPath   string
		symbolID string
		name     string
	)

	cmd := &cobra.Command{
		Use:   "symbol-diff [old-ref] [new-ref]",
		Short: "Diff a single symbol's body between two commits",
		Long: `Show the body diff of a specific function, method, or type between two commits.

Provide --symbol with a full symbol ID, or --name to search by name.

Examples:
  codebase-browser history symbol-diff HEAD~1 HEAD --symbol "sym:github.com/.../func.Extract"
  codebase-browser history symbol-diff HEAD~3 HEAD --name Extract`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			repoRoot, err := findRepoRoot()
			if err != nil {
				return err
			}

			oldHash, err := gitutil.ResolveRef(ctx, repoRoot, args[0])
			if err != nil {
				return fmt.Errorf("resolve %s: %w", args[0], err)
			}
			newHash, err := gitutil.ResolveRef(ctx, repoRoot, args[1])
			if err != nil {
				return fmt.Errorf("resolve %s: %w", args[1], err)
			}

			store, err := history.Open(dbPath)
			if err != nil {
				return fmt.Errorf("open history db: %w", err)
			}
			defer func() { _ = store.Close() }()

			// Resolve name to symbol ID if needed.
			targetID := symbolID
			if targetID == "" && name != "" {
				targetID, err = findSymbolIDByName(ctx, store, newHash, name)
				if err != nil {
					return err
				}
			}
			if targetID == "" {
				return fmt.Errorf("provide --symbol or --name")
			}

			result, err := history.DiffSymbolBodyWithContent(ctx, store, repoRoot, oldHash, newHash, targetID)
			if err != nil {
				return err
			}

			fmt.Printf("Symbol: %s (%s)\n", result.Name, targetID)
			fmt.Printf("Old: %s %s\n", result.OldCommit[:7], result.OldRange)
			fmt.Printf("New: %s %s\n\n", result.NewCommit[:7], result.NewRange)

			if result.UnifiedDiff != "" {
				fmt.Println(result.UnifiedDiff)
			} else {
				fmt.Println("(no body diff computed)")
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&dbPath, "db", "history.db", "Path to history database")
	cmd.Flags().StringVar(&symbolID, "symbol", "", "Full symbol ID")
	cmd.Flags().StringVar(&name, "name", "", "Search symbol by name in new commit")

	return cmd
}

func findSymbolIDByName(ctx context.Context, store *history.Store, commitHash, name string) (string, error) {
	var id string
	err := store.DB().QueryRowContext(ctx, `
SELECT id FROM snapshot_symbols
WHERE commit_hash = ? AND name = ?
ORDER BY kind, package_id
LIMIT 1`, commitHash, name).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("symbol %q not found at %s", name, commitHash[:7])
	}
	return id, nil
}
