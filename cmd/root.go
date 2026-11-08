// Package cmd holds kon's command line interface.
package cmd

import (
	"github.com/spf13/cobra"
)

// Version is the kon-go release version.
const Version = "0.28.0-go"

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
