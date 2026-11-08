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
		Use:   "exec CONTEXT NAMESPACE -- COMMAND [ARGS...]",
		Short: "Execute a command inside of the given context and namespace.",
		Args:  cobra.MinimumNArgs(2),
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
	head, command := splitExecArgs(args)
	if len(head) < 2 {
		return fmt.Errorf("exec requires a context and a namespace")
	}

	if len(command) == 0 {
		return nil
	}

	config, err := settings.Load()
	if err != nil {
		return err
	}

	installed, err := kubeconfig.GetInstalledContexts(config.Configs.Include, config.Configs.Exclude)
	if err != nil {
		return err
	}

	matching := installed.GetContextsMatching(head[0], config.Behavior.AllowMultipleContextPatterns)
	if len(matching) == 0 {
		return fmt.Errorf("No context matching %s", head[0])
	}

	printContext, err := shouldPrintContextHeaders(config, opts.contextHeaders)
	if err != nil {
		return err
	}

	namespace := head[1]
	failed := 0

	for _, context := range matching {
		if printContext {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "CONTEXT => %s\n", context.Item.Name)
		}

		isolated, err := installed.MakeKubeconfigForContext(context.Item.Name, &namespace)
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

func splitExecArgs(args []string) ([]string, []string) {
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
