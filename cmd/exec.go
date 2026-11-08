package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/khanghua1505/kon/internal/kubeconfig"
	"github.com/khanghua1505/kon/internal/settings"
	"github.com/khanghua1505/kon/internal/vars"
)

type execOptions struct {
	exitEarly      bool
	contextHeaders string
}

// ExecExitError carries the exit code of a command run through kon exec.
type ExecExitError struct {
	Code int
}

func (e *ExecExitError) Error() string {
	return fmt.Sprintf("command exited with code %d", e.Code)
}

// ExitCode reports the process exit code.
func (e *ExecExitError) ExitCode() int {
	return e.Code
}

func newExecCmd() *cobra.Command {
	opts := &execOptions{}

	cmd := &cobra.Command{
		Use:   "exec [CONTEXT] [NAMESPACE] -- COMMAND [ARGS...]",
		Short: "Execute a command inside of the given context and namespace, defaulting to the current ones.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExec(cmd, opts, args)
		},
	}

	cmd.Flags().BoolVarP(&opts.exitEarly, "exit-early", "e", false, "Exit early if a command fails when using a wildcard context.")
	cmd.Flags().StringVar(&opts.contextHeaders, "context-headers", "", "Overrides behavior.print_context_in_exec in kon settings file.")
	cmd.Flags().SetInterspersed(false)

	return cmd
}

func runExec(cmd *cobra.Command, opts *execOptions, args []string) error {
	head, command := splitExecArgs(cmd, args)
	if len(command) == 0 {
		return fmt.Errorf("exec requires a command")
	}

	config, err := settings.Load()
	if err != nil {
		return err
	}

	installed, err := kubeconfig.GetInstalledContexts(config.Configs.Include, config.Configs.Exclude)
	if err != nil {
		return err
	}

	contextName := ""
	if len(head) > 0 {
		contextName = head[0]
	} else {
		contextName, err = currentContextName()
		if err != nil {
			return err
		}
	}

	var namespace *string
	if len(head) > 1 {
		namespace = &head[1]
	} else if current, ok := currentNamespace(); ok {
		namespace = &current
	}

	matching := installed.GetContextsMatching(contextName, config.Behavior.AllowMultipleContextPatterns)
	if len(matching) == 0 {
		return fmt.Errorf("No context matching %s", contextName)
	}

	printContext, err := shouldPrintContextHeaders(config, opts.contextHeaders)
	if err != nil {
		return err
	}

	failed := 0

	for _, context := range matching {
		if printContext {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "CONTEXT => %s\n", context.Item.Name)
		}

		isolated, err := installed.MakeKubeconfigForContext(context.Item.Name, namespace)
		if err != nil {
			return err
		}

		code, err := runInContext(cmd, isolated, command)
		if err != nil {
			return err
		}

		if printContext {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), strings.Repeat("-", 20))
		}

		if code != 0 {
			if failed == 0 {
				failed = code
			}

			if opts.exitEarly {
				return &ExecExitError{Code: code}
			}
		}
	}

	if failed != 0 {
		return &ExecExitError{Code: failed}
	}

	return nil
}

// splitExecArgs splits the arguments of exec into the optional context and
// namespace arguments and the command. When the command is introduced by --,
// everything before it belongs to the optional arguments; otherwise the
// first two arguments are the context and the namespace.
func splitExecArgs(cmd *cobra.Command, args []string) ([]string, []string) {
	if dashLen := cmd.ArgsLenAtDash(); dashLen >= 0 {
		return args[:dashLen], args[dashLen:]
	}

	for index, arg := range args {
		if arg == "--" && index <= 2 {
			return args[:index], args[index+1:]
		}
	}

	if len(args) >= 2 {
		return args[:2], args[2:]
	}

	return args, nil
}

// currentContextName returns the context of the running kon shell, used when
// exec is called without a context.
func currentContextName() (string, error) {
	if err := vars.EnsureActive(); err != nil {
		return "", fmt.Errorf("cannot fall back to the current context: %w", err)
	}

	kubecfg, err := currentConfig()
	if err != nil {
		return "", err
	}

	if kubecfg.CurrentContext == "" {
		return "", fmt.Errorf("the current kubeconfig has no current context")
	}

	return kubecfg.CurrentContext, nil
}

// currentNamespace returns the namespace of the running kon shell and whether
// it is known. Outside a kon shell the namespace is unknown and the context
// falls back to its default namespace.
func currentNamespace() (string, bool) {
	if !vars.Active() {
		return "", false
	}

	kubecfg, err := currentConfig()
	if err != nil || len(kubecfg.Contexts) == 0 {
		return "", false
	}

	return kubecfg.Contexts[0].Context.Namespace, true
}

func shouldPrintContextHeaders(config *settings.Settings, override string) (bool, error) {
	if override == "" {
		return config.Behavior.PrintContextInExec.ShouldPrintHeaders(), nil
	}

	switch settings.ContextHeaderBehavior(override) {
	case settings.ContextHeaderAlways:
		return true, nil
	case settings.ContextHeaderNever:
		return false, nil
	case settings.ContextHeaderAuto:
		return settings.ContextHeaderAuto.ShouldPrintHeaders(), nil
	default:
		return false, fmt.Errorf("invalid value for --context-headers: %s", override)
	}
}

func terminatingSignal(sig os.Signal) bool {
	switch sig {
	case syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT:
		return true
	default:
		return false
	}
}

func runInContext(cmd *cobra.Command, config *kubeconfig.KubeConfig, command []string) (int, error) {
	path, err := config.WriteTemp()
	if err != nil {
		return 0, err
	}
	defer func() { _ = os.Remove(path) }()

	child := exec.Command(command[0], command[1:]...)
	child.Stdin = cmd.InOrStdin()
	child.Stdout = cmd.OutOrStdout()
	child.Stderr = cmd.ErrOrStderr()
	depth := strconv.Itoa(vars.Depth() + 1)
	child.Env = append(os.Environ(),
		"KUBECONFIG="+path,
		"KON_KUBE_CONFIG="+path,
		"KON_ACTIVE=1",
		"KON_DEPTH="+depth,
	)

	if err := child.Start(); err != nil {
		return 0, err
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals,
		syscall.SIGHUP,
		syscall.SIGTERM,
		syscall.SIGINT,
		syscall.SIGQUIT,
		syscall.SIGWINCH,
		syscall.SIGUSR1,
		syscall.SIGUSR2,
	)
	defer signal.Stop(signals)

	done := make(chan struct{})
	defer close(done)

	go func() {
		for {
			select {
			case sig := <-signals:
				if err := child.Process.Signal(sig); err != nil {
					fmt.Fprintf(os.Stderr, "kon: failed to forward signal to child: %v\n", err)
				}

				if terminatingSignal(sig) {
					signal.Stop(signals)
					if err := syscall.Kill(syscall.Getpid(), sig.(syscall.Signal)); err != nil {
						fmt.Fprintf(os.Stderr, "kon: failed to re-raise signal: %v\n", err)
					}
					return
				}
			case <-done:
				return
			}
		}
	}()

	if err := child.Wait(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			code := exitErr.ExitCode()
			if code < 0 {
				return 0, nil
			}

			return code, nil
		}

		return 0, err
	}

	return 0, nil
}
