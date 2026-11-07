package cmd

import "github.com/spf13/cobra"

// RegisterCommands adds the ported commands to the root command.
//
// It lives in its own file so other ports can register their commands without
// touching root.go.
func RegisterCommands(root *cobra.Command) {
	root.AddCommand(
		newExecCmd(),
		newLintCmd(),
		newEditCmd(),
		newEditConfigCmd(),
		newDeleteCmd(),
		newExportCmd(),
		newUpdateCmd(),
	)
}
