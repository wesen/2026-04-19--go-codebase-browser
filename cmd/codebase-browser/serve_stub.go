package main

import (
	"github.com/spf13/cobra"

	"github.com/wesen/codebase-browser/cmd/codebase-browser/cmds/serve"
)

func registerServe(root *cobra.Command) {
	cobra.CheckErr(serve.Register(root))
}
