package review

import "github.com/spf13/cobra"

func Register(root *cobra.Command) error {
	cmd := &cobra.Command{
		Use:   "review",
		Short: "Code review tool: index commits and markdown docs, export static review guides",
	}

	cmd.AddCommand(newIndexCmd())
	cmd.AddCommand(newDBCmd())
	cmd.AddCommand(newExportCmd())

	root.AddCommand(cmd)
	return nil
}
