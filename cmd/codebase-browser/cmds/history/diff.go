package history

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/wesen/codebase-browser/internal/gitutil"
	"github.com/wesen/codebase-browser/internal/history"
)

func newDiffCmd() *cobra.Command {
	var (
		dbPath   string
		format   string
		onlyType string
	)

	cmd := &cobra.Command{
		Use:   "diff [old-ref] [new-ref]",
		Short: "Diff two commits at the file and symbol level",
		Long: `Compare two commits and show which files and symbols changed.

Refs can be commit hashes, branch names, or any git ref (HEAD, HEAD~3, etc.).

Examples:
  codebase-browser history diff HEAD~1 HEAD
  codebase-browser history diff abc1234 def5678
  codebase-browser history diff main feature-branch --only modified`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			repoRoot, err := findRepoRoot()
			if err != nil {
				return err
			}

			// Resolve refs to full hashes.
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
			defer store.Close()

			diff, err := store.DiffCommits(ctx, oldHash, newHash)
			if err != nil {
				return err
			}

			switch format {
			case "json":
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(diff)
			default:
				printDiff(diff, onlyType)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&dbPath, "db", "history.db", "Path to history database")
	cmd.Flags().StringVar(&format, "format", "table", "Output format: table, json")
	cmd.Flags().StringVar(&onlyType, "only", "", "Only show symbols of this change type (added, removed, modified, moved)")

	return cmd
}

func printDiff(diff *history.CommitDiff, onlyType string) {
	fmt.Printf("Diff: %s..%s\n\n", diff.OldHash[:7], diff.NewHash[:7])

	fmt.Printf("Files: +%d -%d ~%d\n", diff.Stats.FilesAdded, diff.Stats.FilesRemoved, diff.Stats.FilesModified)
	fmt.Printf("Symbols: +%d -%d ~%d →%d\n\n",
		diff.Stats.SymbolsAdded, diff.Stats.SymbolsRemoved,
		diff.Stats.SymbolsModified, diff.Stats.SymbolsMoved)

	if len(diff.Files) > 0 {
		fmt.Println("Files:")
		for _, f := range diff.Files {
			if f.ChangeType == "unchanged" {
				continue
			}
			fmt.Printf("  %-12s %s\n", f.ChangeType, f.Path)
		}
		fmt.Println()
	}

	if len(diff.Symbols) > 0 {
		fmt.Println("Symbols:")
		for _, s := range diff.Symbols {
			if onlyType != "" && string(s.ChangeType) != onlyType {
				continue
			}
			lineInfo := ""
			switch s.ChangeType {
			case history.ChangeAdded:
				lineInfo = fmt.Sprintf("lines %d-%d", s.NewStartLine, s.NewEndLine)
			case history.ChangeRemoved:
				lineInfo = fmt.Sprintf("lines %d-%d", s.OldStartLine, s.OldEndLine)
			case history.ChangeModified, history.ChangeMoved:
				lineInfo = fmt.Sprintf("lines %d-%d → %d-%d",
					s.OldStartLine, s.OldEndLine, s.NewStartLine, s.NewEndLine)
			}
			fmt.Printf("  %-16s %-8s %-30s %s\n", s.ChangeType, s.Kind, s.Name, lineInfo)
		}
	}
}
