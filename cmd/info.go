package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/khanghua1505/kon/internal/kubeconfig"
	"github.com/khanghua1505/kon/internal/vars"
)

// newInfoCmd builds the info command and its subcommands.
func newInfoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info",
		Short: "View info about the current kubie shell, such as the context name and the current namespace.",
	}

	cmd.AddCommand(newInfoCtxCmd(), newInfoNsCmd(), newInfoDepthCmd())

	return cmd
}

// newInfoCtxCmd builds the info ctx subcommand.
func newInfoCtxCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ctx",
		Short: "Get the current shell's context name.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := vars.EnsureActive(); err != nil {
				return err
			}

			config, err := currentConfig()
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), config.CurrentContext)

			return nil
		},
	}
}

// newInfoNsCmd builds the info ns subcommand.
func newInfoNsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ns",
		Short: "Get the current shell's namespace.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := vars.EnsureActive(); err != nil {
				return err
			}

			config, err := currentConfig()
			if err != nil {
				return err
			}
			if len(config.Contexts) == 0 {
				return errors.New("The current kubeconfig has no contexts")
			}

			namespace := config.Contexts[0].Context.Namespace
			if namespace == "" {
				namespace = "default"
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), namespace)

			return nil
		},
	}
}

// newInfoDepthCmd builds the info depth subcommand.
func newInfoDepthCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "depth",
		Short: "Get the current shell's depth.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := vars.EnsureActive(); err != nil {
				return err
			}

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), vars.Depth())

			return nil
		},
	}
}

// currentConfig reads the kubeconfig of the running kubie shell.
func currentConfig() (*kubeconfig.KubeConfig, error) {
	path, ok := vars.KubeconfigPath()
	if !ok {
		return nil, errors.New("KUBIE_KUBECONFIG env variable missing")
	}

	return kubeconfig.Load(path)
}
