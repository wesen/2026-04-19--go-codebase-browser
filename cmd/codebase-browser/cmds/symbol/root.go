package symbol

import (
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/spf13/cobra"
)

func Register(root *cobra.Command) error {
	group := &cobra.Command{
		Use:   "symbol",
		Short: "Query symbols in the index",
	}

	show, err := NewShowCommand()
	if err != nil {
		return err
	}
	showCobra, err := cli.BuildCobraCommandFromCommand(show)
	if err != nil {
		return err
	}
	group.AddCommand(showCobra)

	find, err := NewFindCommand()
	if err != nil {
		return err
	}
	findCobra, err := cli.BuildCobraCommandFromCommand(find)
	if err != nil {
		return err
	}
	group.AddCommand(findCobra)

	root.AddCommand(group)
	return nil
}
