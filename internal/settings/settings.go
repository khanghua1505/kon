// Package settings loads kubie's own configuration from ~/.kube/kubie.yaml.
package settings

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/khanghua1505/kubie/internal/tty"
)

const configFileName = "kubie.yaml"

// Settings is the parsed ~/.kube/kubie.yaml file.
type Settings struct {
	Shell         *string
	DefaultEditor *string
	Configs       Configs
	Prompt        Prompt
	Behavior      Behavior
	Hooks         Hooks
	Fzf           Fzf
}

// Configs holds the kubeconfig file globs.
type Configs struct {
	Include []string
	Exclude []string
}

// Prompt holds the prompt customization flags.
type Prompt struct {
	Disable             bool
	ShowDepth           bool
	ZshUseRPS1          bool
	FishUseRprompt      bool
	XonshUseRightPrompt bool
}

// Behavior holds the namespace and exec behaviors.
type Behavior struct {
	ValidateNamespaces           ValidateNamespaces
	PrintContextInExec           ContextHeaderBehavior
	AllowMultipleContextPatterns bool
}

// Hooks holds the optional start and stop hooks.
type Hooks struct {
	StartCtx string
	StopCtx  string
}

// Fzf holds the dropdown menu options.
type Fzf struct {
	Mouse      bool
	Reverse    bool
	IgnoreCase bool
	InfoHidden bool
	Height     *string
	Prompt     *string
	Color      *string
}

// Default returns the settings kubie uses when no config file exists.
func Default() (*Settings, error) {
	home, err := homeDir()
	if err != nil {
		return nil, err
	}
	return &Settings{
		Configs: Configs{Include: defaultIncludePatterns(home)},
		Prompt:  Prompt{ShowDepth: true},
		Behavior: Behavior{
			ValidateNamespaces: ValidateNamespacesTrue,
			PrintContextInExec: ContextHeaderAuto,
		},
		Fzf: Fzf{Mouse: true},
	}, nil
}

// Path returns the path of kubie's own config file.
func Path() (string, error) {
	home, err := homeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".kube", configFileName), nil
}

// Load reads kubie's config file, falling back to the defaults when the file
// does not exist. The config file itself is always excluded from the
// kubeconfig globs.
func Load() (*Settings, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}

	settings, err := Default()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	switch {
	case err == nil:
		if err := yaml.Unmarshal(data, settings); err != nil {
			return nil, fmt.Errorf("could not parse kubie config: %w", err)
		}
	case !os.IsNotExist(err):
		return nil, err
	}

	settings.Configs.Exclude = append(settings.Configs.Exclude, path)

	return settings, nil
}

// GetKubeConfigsPaths resolves the include globs and removes the exclude
// globs from the result.
func (s *Settings) GetKubeConfigsPaths() ([]string, error) {
	paths := map[string]struct{}{}

	for _, pattern := range s.Configs.Include {
		matches, err := glob(pattern)
		if err != nil {
			return nil, err
		}

		for _, match := range matches {
			paths[match] = struct{}{}
		}
	}

	for _, pattern := range s.Configs.Exclude {
		matches, err := glob(pattern)
		if err != nil {
			return nil, err
		}

		for _, match := range matches {
			delete(paths, match)
		}
	}

	resolved := make([]string, 0, len(paths))
	for path := range paths {
		resolved = append(resolved, path)
	}

	sort.Strings(resolved)

	return resolved, nil
}

// ExpandUser expands a leading "~/" with the home directory.
func ExpandUser(path string) (string, error) {
	if !strings.HasPrefix(path, "~/") {
		return path, nil
	}

	home, err := homeDir()
	if err != nil {
		return "", err
	}

	return home + "/" + path[2:], nil
}

// ShouldPrintHeaders reports whether "kubie exec" prints context headers.
func (b ContextHeaderBehavior) ShouldPrintHeaders() bool {
	switch b {
	case ContextHeaderAlways:
		return true
	case ContextHeaderNever:
		return false
	default:
		return tty.IsTerminal(os.Stdout)
	}
}

// CanListNamespaces reports whether namespaces may be listed with kubectl.
func (v ValidateNamespaces) CanListNamespaces() bool {
	return v == ValidateNamespacesTrue || v == ValidateNamespacesPartial
}

func glob(pattern string) ([]string, error) {
	expanded, err := ExpandUser(pattern)
	if err != nil {
		return nil, err
	}

	matches, err := filepath.Glob(expanded)
	if err != nil {
		return nil, fmt.Errorf("could not resolve kubeconfig pattern %q: %w", pattern, err)
	}

	return matches, nil
}

func homeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not get home directory path: %w", err)
	}

	return home, nil
}

func defaultIncludePatterns(home string) []string {
	return []string{
		filepath.Join(home, ".kube", "config"),
		filepath.Join(home, ".kube", "*.yml"),
		filepath.Join(home, ".kube", "*.yaml"),
		filepath.Join(home, ".kube", "configs", "*.yml"),
		filepath.Join(home, ".kube", "configs", "*.yaml"),
		filepath.Join(home, ".kube", "kubie", "*.yml"),
		filepath.Join(home, ".kube", "kubie", "*.yaml"),
	}
}
