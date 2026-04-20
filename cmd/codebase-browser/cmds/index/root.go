package index

import (
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/spf13/cobra"
)

// Register adds `index build` and `index stats` under a parent `index` group.
func Register(root *cobra.Command) error {
	group := &cobra.Command{
		Use:   "index",
		Short: "Manage the codebase index",
	}

	build, err := NewBuildCommand()
	if err != nil {
		return err
	}
	buildCobra, err := cli.BuildCobraCommandFromCommand(build)
	if err != nil {
		return err
	}
	group.AddCommand(buildCobra)

	stats, err := NewStatsCommand()
	if err != nil {
		return err
	}
	statsCobra, err := cli.BuildCobraCommandFromCommand(stats)
	if err != nil {
		return err
	}
	group.AddCommand(statsCobra)

	root.AddCommand(group)
	return nil
}
