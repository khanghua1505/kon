package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// newGenerateCompletionCmd builds the generate-completion command.
func newGenerateCompletionCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "generate-completion [SHELL]",
		Aliases: []string{"generate-completions"},
		Short:   "Generate a completion script.",
		Long: "Generate a completion script. Enable completion using\n" +
			"`source <(kubie generate-completion)`. This can be added to your shell's\n" +
			"configuration file to enable completion automatically.",
		Args:      cobra.MaximumNArgs(1),
		ValidArgs: []string{"bash", "fish", "powershell", "zsh"},
		RunE: func(cmd *cobra.Command, args []string) error {
			shell := ""
			if len(args) == 1 {
				shell = args[0]
			}

			return generateCompletion(cmd, shell)
		},
	}
}

// generateCompletion writes the completion script of the given shell.
func generateCompletion(cmd *cobra.Command, shell string) error {
	if shell == "" {
		detected, ok := detectShell(os.Getenv("SHELL"))
		if !ok {
			return errors.New("Could not determine shell from environment. Please specify the shell.")
		}
		shell = detected
	}

	out := cmd.OutOrStdout()
	root := cmd.Root()

	switch shell {
	case "bash":
		return root.GenBashCompletionV2(out, true)
	case "fish":
		return root.GenFishCompletion(out, true)
	case "powershell":
		return root.GenPowerShellCompletionWithDesc(out)
	case "zsh":
		return root.GenZshCompletion(out)
	default:
		return fmt.Errorf("unsupported shell %q, supported shells are bash, fish, powershell and zsh", shell)
	}
}

// detectShell returns the completion shell matching the given shell path.
func detectShell(path string) (string, bool) {
	switch filepath.Base(path) {
	case "bash":
		return "bash", true
	case "elvish":
		return "elvish", true
	case "nu":
		return "nushell", true
	case "fish":
		return "fish", true
	case "powershell", "pwsh":
		return "powershell", true
	case "zsh":
		return "zsh", true
	default:
		return "", false
	}
}
