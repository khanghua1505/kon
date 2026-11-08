// Package cmd holds kubie's command line interface.
package cmd

import (
	"github.com/spf13/cobra"
)

// Version is the kubie-go release version.
const Version = "0.28.0-go"

// NewRootCmd builds kubie's root command.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:     "kon",
		Short:   "A more powerful alternative to kubectx and kubens.",
		Version: Version,

		SilenceUsage:  true,
		SilenceErrors: true,

		// Upstream kubie has no `completion` command; completion scripts
		// ship via `kubie generate-completion` instead.
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
	}

	RegisterPortCommands(root)

	return root
}
