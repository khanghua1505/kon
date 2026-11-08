// Package cmd holds kon's command line interface.
package cmd

import (
	"github.com/spf13/cobra"
)

// Version is the kon release version.
const Version = "v0.29.1"

// NewRootCmd builds kon's root command.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:     "kon",
		Short:   "A more powerful alternative to kubectx and kubens.",
		Version: Version,

		SilenceUsage:  true,
		SilenceErrors: true,

		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
	}

	RegisterCommands(root)

	return root
}
