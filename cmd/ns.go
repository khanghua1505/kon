package cmd

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/khanghua1505/kon/internal/kubectl"
	"github.com/khanghua1505/kon/internal/picker"
	"github.com/khanghua1505/kon/internal/session"
	"github.com/khanghua1505/kon/internal/settings"
	"github.com/khanghua1505/kon/internal/shell"
	"github.com/khanghua1505/kon/internal/state"
	"github.com/khanghua1505/kon/internal/tty"
	"github.com/khanghua1505/kon/internal/vars"
)

// nsOptions holds the flags of the ns command.
type nsOptions struct {
	recursive bool
	unset     bool
}

// newNsCmd builds the ns command.
func newNsCmd() *cobra.Command {
	opts := &nsOptions{}

	cmd := &cobra.Command{
		Use:   "ns [NAME|-]",
		Short: "Change the namespace in which the current shell operates. The namespace change does not affect other shells.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runNs(cmd, opts, args)
		},
	}

	cmd.Flags().BoolVarP(&opts.recursive, "recursive", "r", false, "Enter the namespace by spawning a new recursive shell")
	cmd.Flags().BoolVarP(&opts.unset, "unset", "u", false, "Unset the namespace in the currently active context")

	return cmd
}

// runNs changes the namespace of the running kon shell.
func runNs(cmd *cobra.Command, opts *nsOptions, args []string) error {
	if err := vars.EnsureActive(); err != nil {
		return err
	}

	cfg, err := settings.Load()
	if err != nil {
		return err
	}

	sess, err := session.Load()
	if err != nil {
		return fmt.Errorf("Could not load session file: %w", err)
	}

	if len(args) == 0 && opts.unset {
		return enterNamespace(cfg, &sess, opts.recursive, nil)
	}

	var namespace *string
	switch {
	case len(args) == 1 && args[0] == "-":
		last := sess.GetLastNamespace()
		if last == nil {
			return errors.New("There is not previous namespace to switch to")
		}
		namespace = last
	case len(args) == 1:
		namespace, err = resolveNamespace(cmd, cfg, args[0])
		if err != nil {
			return err
		}
	default:
		namespaces, err := kubectl.GetNamespaces(nil)
		if err != nil {
			return fmt.Errorf("Could not get namespaces: %w", err)
		}
		namespace, err = pickNamespace(cmd, cfg, namespaces)
		if err != nil {
			return err
		}
	}

	if namespace == nil {
		return nil
	}

	return enterNamespace(cfg, &sess, opts.recursive, namespace)
}

// resolveNamespace validates the requested namespace against the namespaces of
// the current context.
//
// A nil namespace means the user cancelled the selection.
func resolveNamespace(cmd *cobra.Command, cfg *settings.Settings, name string) (*string, error) {
	if cfg.Behavior.ValidateNamespaces == settings.ValidateNamespacesFalse {
		return &name, nil
	}

	namespaces, err := kubectl.GetNamespaces(nil)
	if err != nil {
		return nil, err
	}

	resolved, candidates, err := matchNamespace(cfg.Behavior.ValidateNamespaces, name, namespaces)
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return &resolved, nil
	}

	return pickNamespace(cmd, cfg, candidates)
}

// matchNamespace matches the requested namespace against the available ones.
//
// The returned candidates hold the partial matches when the request is
// ambiguous and the user has to pick one.
func matchNamespace(behavior settings.ValidateNamespaces, name string, namespaces []string) (string, []string, error) {
	if slices.Contains(namespaces, name) {
		return name, nil, nil
	}

	if behavior == settings.ValidateNamespacesTrue {
		return "", nil, fmt.Errorf("'%s' is not a valid namespace for the context", name)
	}

	candidates := make([]string, 0, len(namespaces))
	for _, namespace := range namespaces {
		if strings.Contains(namespace, name) {
			candidates = append(candidates, namespace)
		}
	}

	switch len(candidates) {
	case 0:
		return "", nil, fmt.Errorf("'%s' is not a valid namespace for the context", name)
	case 1:
		return candidates[0], nil, nil
	default:
		return "", candidates, nil
	}
}

// pickNamespace lets the user pick one of the given namespaces.
//
// A nil namespace means the namespace was not picked.
func pickNamespace(cmd *cobra.Command, cfg *settings.Settings, namespaces []string) (*string, error) {
	sorted := slices.Clone(namespaces)
	slices.Sort(sorted)

	if len(sorted) == 0 {
		return nil, errors.New("No namespaces found")
	}

	out := cmd.OutOrStdout()
	if file, ok := out.(*os.File); !ok || !tty.IsTerminal(file) {
		for _, namespace := range sorted {
			_, _ = fmt.Fprintln(out, namespace)
		}
		return nil, nil
	}

	result, selected, err := picker.SelectWithOptions(sorted, picker.Options{
		Tui: cfg.Tui,
		In:  cmd.InOrStdin(),
		Out: out,
	})
	if err != nil {
		return nil, err
	}
	if result != picker.Selected {
		return nil, nil
	}

	return &selected, nil
}

// enterNamespace sets the namespace of the current context and records it in
// the session and in the shared state.
func enterNamespace(cfg *settings.Settings, sess *session.Session, recursive bool, namespace *string) error {
	if err := ensureStatePath(); err != nil {
		return err
	}

	config, err := currentConfig()
	if err != nil {
		return err
	}
	if len(config.Contexts) == 0 {
		return errors.New("The current kubeconfig has no contexts")
	}

	config.Contexts[0].Context.Namespace = ""
	if namespace != nil {
		config.Contexts[0].Context.Namespace = *namespace
	}
	contextName := config.Contexts[0].Name

	if err := state.Modify(func(current *state.State) error {
		current.NamespaceHistory[contextName] = namespace
		return nil
	}); err != nil {
		return err
	}

	sess.AddHistoryEntry(contextName, namespace)

	if recursive {
		_, err := shell.Shell(cfg, config, sess)
		return err
	}

	path, ok := vars.KubeconfigPath()
	if !ok {
		return errors.New("KON_KUBECONFIG env variable missing")
	}
	if err := config.WriteToFile(path); err != nil {
		return err
	}

	return sess.Save()
}
