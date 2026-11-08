package cmd

import (
	"github.com/spf13/cobra"

	"github.com/khanghua1505/kon/internal/kubeconfig"
	"github.com/khanghua1505/kon/internal/settings"
)

func newDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete [CONTEXT]",
		Short: "Delete a context, its cluster and its user.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := settings.Load()
			if err != nil {
				return err
			}

			installed, err := kubeconfig.GetInstalledContexts(config.Configs.Include, config.Configs.Exclude)
			if err != nil {
				return err
			}

			name := ""
			if len(args) == 1 {
				name = args[0]
			} else {
				name, err = selectContext(cmd, installed, config)
				if err != nil {
					return err
				}
				if name == "" {
					return nil
				}
			}

			return installed.DeleteContext(name)
		},
	}
}
