package review

import "github.com/spf13/cobra"

func Register(root *cobra.Command) error {
	cmd := &cobra.Command{
		Use:   "review",
		Short: "Code review tool: index commits and markdown docs, serve review guides",
	}

	cmd.AddCommand(newIndexCmd())
	cmd.AddCommand(newServeCmd())
	cmd.AddCommand(newDBCmd())

	root.AddCommand(cmd)
	return nil
}
