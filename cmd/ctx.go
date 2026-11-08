package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"

	"github.com/spf13/cobra"

	"github.com/khanghua1505/kon/internal/kubeconfig"
	"github.com/khanghua1505/kon/internal/kubectl"
	"github.com/khanghua1505/kon/internal/picker"
	"github.com/khanghua1505/kon/internal/session"
	"github.com/khanghua1505/kon/internal/settings"
	"github.com/khanghua1505/kon/internal/shell"
	"github.com/khanghua1505/kon/internal/state"
	"github.com/khanghua1505/kon/internal/vars"
)

// ctxOptions holds the flags of the ctx command.
type ctxOptions struct {
	namespace   string
	kubeconfigs []string
	recursive   bool
}

// newCtxCmd builds the ctx command.
func newCtxCmd() *cobra.Command {
	opts := &ctxOptions{}

	cmd := &cobra.Command{
		Use:   "ctx [NAME|-]",
		Short: "Spawn a shell in the given context. The shell is isolated from other shells.",
		Long: "Spawn a shell in the given context. The shell is isolated from other shells.\n" +
			"Kon shells can be spawned recursively without any issue.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCtx(cmd, opts, args)
		},
	}

	cmd.Flags().StringVarP(&opts.namespace, "namespace", "n", "", "Namespace to use in the selected context")
	cmd.Flags().StringArrayVarP(&opts.kubeconfigs, "kubeconfig", "f", nil, "Files to load contexts from instead of the installed ones")
	cmd.Flags().BoolVarP(&opts.recursive, "recursive", "r", false, "Enter the context by spawning a new recursive shell")

	return cmd
}

// runCtx enters the requested context.
func runCtx(cmd *cobra.Command, opts *ctxOptions, args []string) error {
	cfg, err := settings.Load()
	if err != nil {
		return err
	}

	installed, err := loadInstalledContexts(cfg, opts.kubeconfigs)
	if err != nil {
		return err
	}

	contextName := ""
	if len(args) == 1 {
		contextName = args[0]
	} else {
		names := contextNames(installed)
		if len(names) == 0 {
			return errors.New("No contexts found")
		}
		mode := picker.ModeAuto
		if cfg.Tui {
			mode = picker.ModeTui
		}
		result, selected, err := picker.SelectWithOptions(names, picker.Options{
			Mode: mode,
			In:   cmd.InOrStdin(),
			Out:  cmd.OutOrStdout(),
		})
		if err != nil {
			return err
		}
		if result != picker.Selected {
			return nil
		}
		contextName = selected
	}

	return enterContext(cfg, installed, contextName, opts.namespace, opts.recursive, cmd.ErrOrStderr())
}

// loadInstalledContexts loads the contexts of the given kubeconfig files, or
// of every installed kubeconfig when no file is given.
func loadInstalledContexts(cfg *settings.Settings, kubeconfigs []string) (*kubeconfig.Installed, error) {
	paths := kubeconfigs
	if len(paths) == 0 {
		resolved, err := cfg.GetKubeConfigsPaths()
		if err != nil {
			return nil, err
		}
		paths = resolved
	}

	installed := kubeconfig.LoadKubeconfigs(paths)
	if len(installed.Contexts) == 0 {
		if len(kubeconfigs) == 0 {
			return nil, errors.New("Could not find any contexts in the Kon kubeconfig directories!")
		}

		return nil, errors.New("Could not find any contexts in the given set of files!")
	}

	return installed, nil
}

// contextNames returns the sorted names of every installed context.
func contextNames(installed *kubeconfig.Installed) []string {
	names := make([]string, 0, len(installed.Contexts))
	for _, context := range installed.Contexts {
		names = append(names, context.Item.Name)
	}
	slices.Sort(names)

	return names
}

// enterContext enters the given context, either by rewriting the kubeconfig of
// the running kon shell or by spawning a new shell.
func enterContext(cfg *settings.Settings, installed *kubeconfig.Installed, contextName, namespaceName string, recursive bool, stderr io.Writer) error {
	if err := ensureStatePath(); err != nil {
		return err
	}

	currentState, err := state.Load()
	if err != nil {
		return err
	}

	sess, err := session.Load()
	if err != nil {
		return err
	}

	config, err := contextKubeconfig(installed, currentState, &sess, contextName, namespaceName)
	if err != nil {
		return err
	}

	namespace := config.Contexts[0].Context.Namespace
	var namespacePtr *string
	if namespace != "" {
		namespacePtr = &namespace
	}

	if err := sess.RecordContextEntry(config.Contexts[0].Name, namespacePtr); err != nil {
		return err
	}

	if cfg.Behavior.ValidateNamespaces.CanListNamespaces() && namespaceName != "" {
		namespaces, err := kubectl.GetNamespaces(config)
		if err != nil {
			return err
		}
		if !slices.Contains(namespaces, namespaceName) {
			_, _ = fmt.Fprintf(stderr, "Warning: namespace %s does not exist.\n", namespaceName)
		}
	}

	if vars.Active() && !recursive {
		path, ok := vars.KubeconfigPath()
		if !ok {
			return errors.New("KON_KUBECONFIG env variable missing")
		}
		if err := config.WriteToFile(path); err != nil {
			return err
		}
		return sess.Save()
	}

	_, err = shell.Shell(cfg, config, &sess)

	return err
}

// contextKubeconfig builds the isolated kubeconfig of the requested context.
//
// The "-" context switches back to the previous context of the session, or to
// the last context used in any session.
func contextKubeconfig(installed *kubeconfig.Installed, currentState state.State, sess *session.Session, contextName, namespaceName string) (*kubeconfig.KubeConfig, error) {
	if contextName == "-" {
		if previous := sess.GetLastContext(); previous != nil {
			return installed.MakeKubeconfigForContext(previous.Context, namespaceOr(previous.Namespace, namespaceName))
		}
		if currentState.LastContext != nil {
			last := *currentState.LastContext
			return installed.MakeKubeconfigForContext(last, namespaceOr(currentState.NamespaceHistory[last], namespaceName))
		}
		return nil, errors.New("There is no previous context to switch to.")
	}

	return installed.MakeKubeconfigForContext(contextName, namespaceOr(currentState.NamespaceHistory[contextName], namespaceName))
}

// namespaceOr returns the requested namespace when it is set, the remembered one otherwise.
func namespaceOr(remembered *string, requested string) *string {
	if requested != "" {
		return &requested
	}

	return remembered
}

// ensureStatePath points the state package at a usable state file.
//
// The environment wins, the platform data directory is the fallback used when
// kon runs outside of a kon shell.
func ensureStatePath() error {
	path := os.Getenv(state.EnvState)
	if path == "" {
		dir, err := localDataDir()
		if err != nil {
			return err
		}
		path = filepath.Join(dir, "kon", "state.json")
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("could not create state directory: %w", err)
	}

	return os.Setenv(state.EnvState, path)
}

// localDataDir returns the platform local data directory.
func localDataDir() (string, error) {
	if runtime.GOOS == "darwin" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "Library", "Application Support"), nil
	}

	if dir := os.Getenv("XDG_DATA_HOME"); dir != "" {
		return dir, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".local", "share"), nil
}
