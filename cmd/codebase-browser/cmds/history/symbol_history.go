package history

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/wesen/codebase-browser/internal/history"
)

func newSymbolHistoryCmd() *cobra.Command {
	var (
		dbPath   string
		symbolID string
		name     string
		limit    int
	)

	cmd := &cobra.Command{
		Use:   "symbol-history",
		Short: "Show the history of a symbol across indexed commits",
		Long: `List every commit where a symbol appeared, showing its body hash and line range.

Provide --symbol with a full symbol ID, or --name to search by name.

Examples:
  codebase-browser history symbol-history --symbol "sym:github.com/.../func.Extract"
  codebase-browser history symbol-history --name Extract --limit 10`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			store, err := history.Open(dbPath)
			if err != nil {
				return fmt.Errorf("open history db: %w", err)
			}
			defer store.Close()

			targetID := symbolID
			if targetID == "" && name != "" {
				// Find in the latest indexed commit.
				commits, err := store.ListCommits(ctx)
				if err != nil {
					return err
				}
				if len(commits) == 0 {
					return fmt.Errorf("no commits indexed yet")
				}
				targetID, err = findSymbolIDByName(ctx, store, commits[0].Hash, name)
				if err != nil {
					return err
				}
			}
			if targetID == "" {
				return fmt.Errorf("provide --symbol or --name")
			}

			rows, err := store.DB().QueryContext(ctx, `
SELECT c.hash, c.short_hash, c.message, c.author_time,
       s.body_hash, s.start_line, s.end_line, s.signature, s.kind
FROM   snapshot_symbols s
JOIN   commits c ON c.hash = s.commit_hash
WHERE  s.id = ?
ORDER BY c.author_time DESC
LIMIT  ?`, targetID, limit)
			if err != nil {
				return err
			}
			defer rows.Close()

			fmt.Printf("Symbol: %s\n\n", targetID)
			fmt.Printf("%-10s %-19s %-10s %-12s %s\n",
				"HASH", "DATE", "LINES", "BODY_HASH", "MESSAGE")

			count := 0
			for rows.Next() {
				var hash, shortHash, message, bodyHash, signature, kind string
				var authorTime int64
				var startLine, endLine int
				if err := rows.Scan(&hash, &shortHash, &message, &authorTime,
					&bodyHash, &startLine, &endLine, &signature, &kind); err != nil {
					return err
				}
				date := time.Unix(authorTime, 0).Format("2006-01-02 15:04")
				bodyShort := ""
				if len(bodyHash) > 7 {
					bodyShort = bodyHash[:7]
				}
				fmt.Printf("%-10s %-19s %-10d %-12s %s\n",
					shortHash, date, endLine-startLine+1, bodyShort, message)
				count++
			}
			fmt.Printf("\n%d commit(s)\n", count)
			return rows.Err()
		},
	}

	cmd.Flags().StringVar(&dbPath, "db", "history.db", "Path to history database")
	cmd.Flags().StringVar(&symbolID, "symbol", "", "Full symbol ID")
	cmd.Flags().StringVar(&name, "name", "", "Search symbol by name")
	cmd.Flags().IntVar(&limit, "limit", 50, "Max commits to show")

	return cmd
}
