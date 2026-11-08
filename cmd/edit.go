package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/khanghua1505/kon/internal/kubeconfig"
	"github.com/khanghua1505/kon/internal/picker"
	"github.com/khanghua1505/kon/internal/settings"
)

type editorCommand struct {
	executable string
	args       []string
}

func (e editorCommand) String() string {
	if len(e.args) == 0 {
		return e.executable
	}

	return e.executable + " " + strings.Join(e.args, " ")
}

func parseEditorCommand(raw string) (editorCommand, error) {
	parts := strings.Fields(raw)
	if len(parts) == 0 {
		return editorCommand{}, fmt.Errorf("executable is empty")
	}

	return editorCommand{executable: parts[0], args: parts[1:]}, nil
}

func getEditor(config *settings.Settings) (editorCommand, error) {
	if config.DefaultEditor != nil {
		editor, err := parseEditorCommand(*config.DefaultEditor)
		if err != nil {
			return editorCommand{}, fmt.Errorf("unable to parse default_editor command %s: %w", *config.DefaultEditor, err)
		}

		return editor, nil
	}

	if raw, ok := os.LookupEnv("EDITOR"); ok {
		editor, err := parseEditorCommand(raw)
		if err != nil {
			return editorCommand{}, fmt.Errorf("unable to parse EDITOR command %s: %w", raw, err)
		}

		return editor, nil
	}

	for _, candidate := range []string{"nvim", "vim", "emacs", "vi", "nano"} {
		if path, err := exec.LookPath(candidate); err == nil {
			return editorCommand{executable: path}, nil
		}
	}

	return editorCommand{}, fmt.Errorf("Could not find any editor to use")
}

func runEditor(editor editorCommand, path string) error {
	args := append(append([]string{}, editor.args...), path)

	command := exec.Command(editor.executable, args...)
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	if err := command.Run(); err != nil {
		return fmt.Errorf("Failed to spawn editor command '%s': %w", editor, err)
	}

	return nil
}

func newEditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "edit [CONTEXT]",
		Short: "Edit the kubeconfig file containing the given context.",
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

			sort.SliceStable(installed.Contexts, func(i, j int) bool {
				return installed.Contexts[i].Item.Name < installed.Contexts[j].Item.Name
			})

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

			context := installed.FindContextByName(name)
			if context == nil {
				return fmt.Errorf("Could not find context %s", name)
			}

			editor, err := getEditor(config)
			if err != nil {
				return err
			}

			return runEditor(editor, context.Source)
		},
	}
}

func newEditConfigCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "edit-config",
		Short: "Edit kubie's config file.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := settings.Load()
			if err != nil {
				return err
			}

			path, err := settings.Path()
			if err != nil {
				return err
			}

			editor, err := getEditor(config)
			if err != nil {
				return err
			}

			return runEditor(editor, path)
		},
	}
}

func selectContext(cmd *cobra.Command, installed *kubeconfig.Installed, config *settings.Settings) (string, error) {
	names := make([]string, 0, len(installed.Contexts))
	for _, context := range installed.Contexts {
		names = append(names, context.Item.Name)
	}

	result, selected, err := picker.SelectWithOptions(names, picker.Options{
		Mode: picker.ModeAuto,
		Fzf:  config.Fzf,
		In:   cmd.InOrStdin(),
		Out:  cmd.OutOrStdout(),
	})
	if err != nil {
		return "", err
	}

	if result != picker.Selected {
		return "", nil
	}

	return selected, nil
}
