package cmd

import "github.com/spf13/cobra"

// RegisterCommands adds the ported commands to the root command.
//
// It lives in its own file so other ports can register their commands without
// touching root.go.
func RegisterCommands(root *cobra.Command) {
	existing := map[string]bool{}
	for _, command := range root.Commands() {
		existing[command.Name()] = true
	}

	for _, command := range []*cobra.Command{
		newExecCmd(),
		newLintCmd(),
		newEditCmd(),
		newEditConfigCmd(),
		newDeleteCmd(),
		newExportCmd(),
		newUpdateCmd(),
	} {
		if existing[command.Name()] {
			continue
		}

		root.AddCommand(command)
	}
}
