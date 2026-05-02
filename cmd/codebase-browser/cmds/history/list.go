package history

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/wesen/codebase-browser/internal/history"
)

func newListCmd() *cobra.Command {
	var dbPath string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List indexed commits in the history database",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			store, err := history.Open(dbPath)
			if err != nil {
				return fmt.Errorf("open history db: %w", err)
			}
			defer func() { _ = store.Close() }()

			commits, err := store.ListCommits(ctx)
			if err != nil {
				return err
			}

			if len(commits) == 0 {
				fmt.Println("No commits indexed yet. Run 'codebase-browser history scan' first.")
				return nil
			}

			fmt.Printf("%-10s %-19s %-6s %s\n", "HASH", "DATE", "SYMS", "MESSAGE")
			for _, c := range commits {
				symCount, _ := store.SymbolCountAtCommit(ctx, c.Hash)
				date := time.Unix(c.AuthorTime, 0).Format("2006-01-02 15:04:05")
				errMark := ""
				if c.Error != "" {
					errMark = " [ERROR]"
				}
				fmt.Printf("%-10s %-19s %-6d %s%s\n",
					c.ShortHash, date, symCount, c.Message, errMark)
			}
			fmt.Printf("\n%d commit(s)\n", len(commits))
			return nil
		},
	}

	cmd.Flags().StringVar(&dbPath, "db", "history.db", "Path to history database")
	return cmd
}
