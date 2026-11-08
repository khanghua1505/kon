package cmd

import "github.com/spf13/cobra"

// RegisterCommands adds the commands to the root command.
func RegisterCommands(root *cobra.Command) {
	root.AddCommand(
		newCtxCmd(),
		newNsCmd(),
		newInfoCmd(),
		newExecCmd(),
		newLintCmd(),
		newEditCmd(),
		newEditConfigCmd(),
		newDeleteCmd(),
		newExportCmd(),
		newUpdateCmd(),
		newGenerateCompletionCmd(),
		newKubectlAliasCmd(),
		newInstallCmd(),
	)
}
