// Package settings loads kon's own configuration from ~/.kube/kon.yaml or
// ~/.kube/kon.yaml.
package settings

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/khanghua1505/kon/internal/tty"
	"github.com/khanghua1505/kon/internal/tui/style"
)

const configFileName = "kon.yaml"
const legacyConfigFileName = "kubie.yaml"

// Settings is the parsed ~/.kube/kon.yaml file.
type Settings struct {
	Shell         *string
	DefaultEditor *string
	Tui           TuiSettings
	Configs       Configs
	Prompt        Prompt
	Behavior      Behavior
	Hooks         Hooks
}

// TuiSettings feeds the TUI theme. The shape is being redefined, so it is
// empty for now and FromSettings applies the default palette.
type TuiSettings struct{}

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
		Tui: TuiSettings{},
	}, nil
}

// Path returns the path of kon's own config file: kon.yaml when it exists,
// otherwise kubie.yaml for backward compatibility.
func Path() (string, error) {
	home, err := homeDir()
	if err != nil {
		return "", err
	}

	kubiePath := filepath.Join(home, ".kube", legacyConfigFileName)
	if _, err := os.Stat(kubiePath); err == nil {
		return kubiePath, nil
	}
	return filepath.Join(home, ".kube", configFileName), nil
}

// Load reads kon's config file, falling back to the defaults when the file
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

	settings.Configs.Exclude = append(settings.Configs.Exclude,
		filepath.Join(filepath.Dir(path), configFileName),
		filepath.Join(filepath.Dir(path), legacyConfigFileName),
	)

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

// GetStyle resolves Style
func (s *Settings) GetStyle() style.Style {
	return style.Default()
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
