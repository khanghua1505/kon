package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/khanghua1505/kubie/internal/kubeconfig"
	"github.com/khanghua1505/kubie/internal/settings"
)

func newLintCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "lint",
		Short: "Check the kubeconfig files for errors.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := settings.Load()
			if err != nil {
				return err
			}

			installed, err := kubeconfig.GetInstalledContexts(config.Configs.Include, config.Configs.Exclude)
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			lintClusters(installed, out)
			lintUsers(installed, out)
			lintContexts(installed, out)

			return nil
		},
	}
}

func lintClusters(installed *kubeconfig.Installed, out io.Writer) {
	seen := map[string]bool{}

	for _, cluster := range installed.Clusters {
		name := cluster.Item.Name
		source := cluster.Source

		if len(installed.FindContextsByCluster(name, source)) == 0 {
			fmt.Fprintf(out, "Cluster '%s' has no context referencing it in file %s\n", name, source)
		}

		key := name + "\x00" + source
		if seen[key] {
			fmt.Fprintf(out, "A cluster named '%s' appears more than once in file %s\n", name, source)
		} else {
			seen[key] = true
		}
	}
}

func lintUsers(installed *kubeconfig.Installed, out io.Writer) {
	seen := map[string]bool{}

	for _, user := range installed.Users {
		name := user.Item.Name
		source := user.Source

		if len(installed.FindContextsByUser(name, source)) == 0 {
			fmt.Fprintf(out, "User '%s' has no context referencing it in file %s\n", name, source)
		}

		key := name + "\x00" + source
		if seen[key] {
			fmt.Fprintf(out, "A user named '%s' appears more than once in file %s\n", name, source)
		} else {
			seen[key] = true
		}
	}
}

func lintContexts(installed *kubeconfig.Installed, out io.Writer) {
	seen := map[string]bool{}

	for _, context := range installed.Contexts {
		name := context.Item.Name
		source := context.Source

		if installed.FindClusterByName(context.Item.Context.Cluster, source) == nil {
			fmt.Fprintf(out, "Context '%s' references unknown cluster '%s' in file %s\n", name, context.Item.Context.Cluster, source)
		}

		if installed.FindUserByName(context.Item.Context.User, source) == nil {
			fmt.Fprintf(out, "Context '%s' references unknown users '%s' in file %s\n", name, context.Item.Context.User, source)
		}

		if seen[name] {
			fmt.Fprintf(out, "A context name '%s' appears more than once in file %s\n", name, source)
		} else {
			seen[name] = true
		}
	}
}
