package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

const (
	installBeginMarker = "# >>> kon >>>"
	installEndMarker   = "# <<< kon <<<"
)

// newInstallCmd builds install command.
func newInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:       "install [SHELL]",
		Short:     "Install completion, kubectl aliases and the kubectl-kon plugin.",
		Args:      cobra.MaximumNArgs(1),
		ValidArgs: []string{"bash", "zsh", "fish", "nushell"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := installKubectlPlugin(cmd); err != nil {
				return err
			}

			shell := ""
			if len(args) == 1 {
				shell = args[0]
			}
			return installConfig(cmd, shell)
		},
	}
}

// installConfig appends or replaces the kon configuration block in the
// shell's configuration file.
func installConfig(cmd *cobra.Command, shell string) error {
	if shell == "" {
		detected, ok := detectShell(os.Getenv("SHELL"))
		if !ok {
			return errors.New("Could not determine shell environment. Please specify shell.")
		}
		shell = detected
	}

	bin := cmd.Root().Name()
	var lines []string
	switch shell {
	case "bash", "zsh":
		lines = []string{
			fmt.Sprintf("source <(%s generate-completion %s)", bin, shell),
			fmt.Sprintf("source <(%s kubectl-alias %s)", bin, shell),
		}
	case "fish":
		lines = []string{
			fmt.Sprintf("%s generate-completion fish | source", bin),
			fmt.Sprintf("%s kubectl-alias fish | source", bin),
		}
	case "nushell":
		// ponytail: nushell has no completion script generator in cobra,
		// so only aliases are installed, generated once into a file.
		aliasPath, err := writeNushellAliases(cmd, bin)
		if err != nil {
			return err
		}
		lines = []string{fmt.Sprintf("source %s", aliasPath)}
	default:
		return fmt.Errorf("unsupported shell %q, supported shells are bash, zsh, fish and nushell", shell)
	}

	rcPath, err := shellRcPath(shell)
	if err != nil {
		return err
	}
	block := installBeginMarker + "\n" + strings.Join(lines, "\n") + "\n" + installEndMarker
	if err := replaceRcBlock(rcPath, block); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Installed kon configuration to %s\n", rcPath) //nolint:errcheck
	return nil
}

// writeNushellAliases generates the kubectl aliases for nushell into
// ~/.config/kon and returns the file path.
func writeNushellAliases(cmd *cobra.Command, bin string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".config", bin)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	aliasPath := filepath.Join(dir, "kubectl-aliases.nu")
	file, err := os.Create(aliasPath)
	if err != nil {
		return "", err
	}
	defer file.Close() //nolint:errcheck

	aliasCmd := &cobra.Command{Use: bin}
	aliasCmd.SetOut(file)
	if err := generateKubectlAliases(aliasCmd, "nushell"); err != nil {
		return "", err
	}
	return aliasPath, nil
}

// shellRcPath returns the configuration file path of the shell.
func shellRcPath(shell string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	switch shell {
	case "bash":
		return filepath.Join(home, ".bashrc"), nil
	case "zsh":
		return filepath.Join(home, ".zshrc"), nil
	case "fish":
		return filepath.Join(home, ".config", "fish", "config.fish"), nil
	case "nushell":
		return filepath.Join(home, ".config", "nushell", "config.nu"), nil
	default:
		return "", fmt.Errorf("unsupported shell %q", shell)
	}
}

// replaceRcBlock writes the block into the file between the kon markers,
// replacing any previous kon block.
func replaceRcBlock(path, block string) error {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	content := ""
	if data, err := os.ReadFile(path); err == nil {
		content = string(data)
	}

	content = stripRcBlock(content)
	content = strings.TrimRight(content, "\n")
	if content != "" {
		content += "\n\n"
	}
	content += block + "\n"

	return os.WriteFile(path, []byte(content), 0o644)
}

// stripRcBlock removes the kon marker block from content, up to the end of
// the file when the end marker is missing.
func stripRcBlock(content string) string {
	var kept []string
	inBlock := false
	for _, line := range strings.Split(content, "\n") {
		switch {
		case strings.TrimSpace(line) == installBeginMarker:
			inBlock = true
		case strings.TrimSpace(line) == installEndMarker:
			inBlock = false
		case !inBlock:
			kept = append(kept, line)
		}
	}
	return strings.Join(kept, "\n")
}

// installKubectlPlugin links kubectl-kon next to the kon binary so kubectl
// runs kon as a plugin, e.g. kubectl kon ctx.
func installKubectlPlugin(cmd *cobra.Command) error {
	binPath, err := os.Executable()
	if err != nil {
		return err
	}

	scripts, err := ensurePluginScripts(binPath)
	if err != nil {
		return fmt.Errorf("could not install kubectl plugins: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Installed kubectl plugins %s (use: kubectl ctx, kubectl ns)\n", strings.Join(scripts, ", ")) //nolint:errcheck
	return nil
}

// ensurePluginScripts writes the kubectl-ctx and kubectl-ns plugin scripts
// next to the kon binary, delegating to kon ctx and kon ns.
func ensurePluginScripts(binPath string) ([]string, error) {
	if resolved, err := filepath.EvalSymlinks(binPath); err == nil {
		binPath = resolved
	}

	plugins := map[string]string{
		"kubectl-on":  "",
		"kubectl-ctx": "ctx",
		"kubectl-ns":  "ns",
	}

	var paths []string
	for name, sub := range plugins {
		path := filepath.Join(filepath.Dir(binPath), name)
		content := fmt.Sprintf("#!/bin/sh\nexec \"$(dirname \"$0\")/kon\" %s \"$@\"\n", sub)
		if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}
	sort.Strings(paths)

	return paths, nil
}
