package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/khanghua1505/kubie/internal/kubeconfig"
	"github.com/khanghua1505/kubie/internal/settings"
)

func newExportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "export CONTEXT NAMESPACE",
		Short: "Export a context and namespace as a temporary kubeconfig file.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := settings.Load()
			if err != nil {
				return err
			}

			installed, err := kubeconfig.GetInstalledContexts(config.Configs.Include, config.Configs.Exclude)
			if err != nil {
				return err
			}

			matching := installed.GetContextsMatching(args[0], config.Behavior.AllowMultipleContextPatterns)
			if len(matching) == 0 {
				return fmt.Errorf("No context matching %s", args[0])
			}

			namespace := args[1]
			for _, context := range matching {
				isolated, err := installed.MakeKubeconfigForContext(context.Item.Name, &namespace)
				if err != nil {
					return err
				}

				path, err := isolated.WriteTemp()
				if err != nil {
					return err
				}

				_, _ = fmt.Fprintln(cmd.OutOrStdout(), path)
			}

			return nil
		},
	}
}
