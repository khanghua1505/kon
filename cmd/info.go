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
		Short: "View info about the current kon shell, such as the context name and the current namespace.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			contextName, err := infoContextName()
			if err != nil {
				return err
			}

			namespace, err := infoNamespace()
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "context: %s\nnamespace: %s\ndepth: %d\n", contextName, namespace, vars.Depth())

			return nil
		},
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
			contextName, err := infoContextName()
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), contextName)

			return nil
		},
	}
}

// infoContextName returns the context name of the running kon shell.
func infoContextName() (string, error) {
	if err := vars.EnsureActive(); err != nil {
		return "", err
	}

	config, err := currentConfig()
	if err != nil {
		return "", err
	}

	return config.CurrentContext, nil
}

// newInfoNsCmd builds the info ns subcommand.
func newInfoNsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ns",
		Short: "Get the current shell's namespace.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			namespace, err := infoNamespace()
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), namespace)

			return nil
		},
	}
}

// infoNamespace returns the namespace of the running kon shell, defaulting
// to default when the current context has none.
func infoNamespace() (string, error) {
	if err := vars.EnsureActive(); err != nil {
		return "", err
	}

	config, err := currentConfig()
	if err != nil {
		return "", err
	}
	if len(config.Contexts) == 0 {
		return "", errors.New("The current kubeconfig has no contexts")
	}

	namespace := config.Contexts[0].Context.Namespace
	if namespace == "" {
		namespace = "default"
	}

	return namespace, nil
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

// currentConfig reads the kubeconfig of the running kon shell.
func currentConfig() (*kubeconfig.KubeConfig, error) {
	path, ok := vars.KubeconfigPath()
	if !ok {
		return nil, errors.New("KON_KUBECONFIG env variable missing")
	}

	return kubeconfig.Load(path)
}
