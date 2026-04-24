package main

import (
	"os"

	"github.com/go-go-golems/glazed/pkg/cmds/logging"
	"github.com/go-go-golems/glazed/pkg/help"
	help_cmd "github.com/go-go-golems/glazed/pkg/help/cmd"
	"github.com/spf13/cobra"

	"github.com/wesen/codebase-browser/cmd/codebase-browser/cmds/doc"
	"github.com/wesen/codebase-browser/cmd/codebase-browser/cmds/index"
	"github.com/wesen/codebase-browser/cmd/codebase-browser/cmds/query"
	"github.com/wesen/codebase-browser/cmd/codebase-browser/cmds/symbol"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:     "codebase-browser",
	Short:   "Index, query, and serve the Go source of this very binary",
	Version: version,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return logging.InitLoggerFromCobra(cmd)
	},
}

func main() {
	cobra.CheckErr(logging.AddLoggingSectionToRootCommand(rootCmd, "codebase-browser"))

	helpSystem := help.NewHelpSystem()
	help_cmd.SetupCobraRootCommand(helpSystem, rootCmd)

	cobra.CheckErr(index.Register(rootCmd))
	cobra.CheckErr(symbol.Register(rootCmd))
	cobra.CheckErr(query.Register(rootCmd))
	cobra.CheckErr(doc.Register(rootCmd))
	registerServe(rootCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
