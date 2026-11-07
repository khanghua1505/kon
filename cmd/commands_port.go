package cmd

import "github.com/spf13/cobra"

// RegisterPortCommands adds the context, namespace, info and completion
// commands to the root command.
func RegisterPortCommands(root *cobra.Command) {
	root.AddCommand(
		newCtxCmd(),
		newNsCmd(),
		newInfoCmd(),
		newGenerateCompletionCmd(),
	)
}
