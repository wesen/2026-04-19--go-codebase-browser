package history

import (
	"github.com/spf13/cobra"
)

func Register(root *cobra.Command) error {
	cmd := &cobra.Command{
		Use:   "history",
		Short: "Git-aware codebase history: index commits, diff symbols, track changes",
	}

	cmd.AddCommand(newScanCmd())
	cmd.AddCommand(newListCmd())

	root.AddCommand(cmd)
	return nil
}
