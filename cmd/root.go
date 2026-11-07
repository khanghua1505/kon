// Package cmd holds kubie's command line interface.
package cmd

import (
	"github.com/spf13/cobra"
)

// Version is the kubie-go release version.
const Version = "0.28.0-go"

// NewRootCmd builds kubie's root command.
func NewRootCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "kubie",
		Short:         "A more powerful alternative to kubectx and kubens.",
		Version:       Version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
}

// Execute runs kubie's root command.
func Execute() error {
	return NewRootCmd().Execute()
}
