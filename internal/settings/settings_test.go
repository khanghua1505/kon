package settings

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultMatchesUpstream(t *testing.T) {
	s, err := Default()
	if err != nil {
		t.Fatal(err)
	}

	if s.Tui.IsEnabled() != true {
		t.Error("tui default must be enabled")
	}

	sel, border := s.Tui.ColorValues()
	if sel != "212" || border != "238" {
		t.Errorf("tui default colors = %q/%q, want 212/238", sel, border)
	}

	if !s.Prompt.ShowDepth {
		t.Error("prompt.show_depth default must be true")
	}

	if s.Prompt.Disable || s.Prompt.ZshUseRPS1 || s.Prompt.FishUseRprompt || s.Prompt.XonshUseRightPrompt {
		t.Error("prompt boolean defaults must all be false except show_depth")
	}

	if len(s.Configs.Include) != 7 {
		t.Fatalf("configs.include default must hold 7 globs, got %d: %v", len(s.Configs.Include), s.Configs.Include)
	}

	if len(s.Configs.Exclude) != 0 {
		t.Errorf("configs.exclude default must be empty, got %v", s.Configs.Exclude)
	}

	if s.Behavior.ValidateNamespaces != ValidateNamespacesTrue {
		t.Errorf("behavior.validate_namespaces default must be true, got %q", s.Behavior.ValidateNamespaces)
	}

	if s.Behavior.PrintContextInExec != ContextHeaderAuto {
		t.Errorf("behavior.print_context_in_exec default must be auto, got %q", s.Behavior.PrintContextInExec)
	}

	if s.Behavior.AllowMultipleContextPatterns {
		t.Error("behavior.allow_multiple_context_patterns default must be false")
	}

	if s.Shell != nil || s.DefaultEditor != nil {
		t.Error("shell and default_editor default must be unset")
	}
}

func TestExpandUser(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	got, err := ExpandUser("~/hello/world/*.foo")
	if err != nil {
		t.Fatalf("ExpandUser: %v", err)
	}

	want := filepath.ToSlash(home) + "/hello/world/*.foo"
	if got != want {
		t.Errorf("ExpandUser = %q, want %q", got, want)
	}

	unchanged, err := ExpandUser("~notashome/file")
	if err != nil {
		t.Fatalf("ExpandUser: %v", err)
	}

	if unchanged != "~notashome/file" {
		t.Errorf("ExpandUser must only strip a ~/ prefix, got %q", unchanged)
	}
}

func TestGetKubeConfigsPathsExcludesMatches(t *testing.T) {
	dir := t.TempDir()

	for _, name := range []string{"a.yaml", "b.yaml", "c.yml", "skipped.yaml"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("apiVersion: v1\n"), 0o600); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
	}

	s, err := Default()
	if err != nil {
		t.Fatal(err)
	}
	s.Configs.Include = []string{filepath.Join(dir, "*.yaml"), filepath.Join(dir, "*.yml")}
	s.Configs.Exclude = []string{filepath.Join(dir, "skipped.yaml")}

	paths, err := s.GetKubeConfigsPaths()
	if err != nil {
		t.Fatalf("GetKubeConfigsPaths: %v", err)
	}

	want := []string{
		filepath.Join(dir, "a.yaml"),
		filepath.Join(dir, "b.yaml"),
		filepath.Join(dir, "c.yml"),
	}

	if len(paths) != len(want) {
		t.Fatalf("GetKubeConfigsPaths = %v, want %v", paths, want)
	}

	for i, path := range paths {
		if path != want[i] {
			t.Errorf("path %d = %q, want %q", i, path, want[i])
		}
	}
}

func TestLoadReportsParseErrors(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := os.MkdirAll(filepath.Join(home, ".kube"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	broken := filepath.Join(home, ".kube", "kon.yaml")
	if err := os.WriteFile(broken, []byte("shell: [unterminated\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := Load()
	if err == nil {
		t.Fatal("Load must fail on malformed yaml")
	}

	if !strings.Contains(err.Error(), "could not parse kubie config") {
		t.Errorf("error %q must mention 'could not parse kubie config'", err)
	}
}

func TestLoadAppliesOverridesAndExcludesSelf(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := os.MkdirAll(filepath.Join(home, ".kube"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	config := "" +
		"shell: bash\n" +
		"default_editor: nano\n" +
		"tui: true\n" +
		"prompt:\n" +
		"  disable: true\n" +
		"  show_depth: false\n" +
		"  zsh_use_rps1: true\n" +
		"behavior:\n" +
		"  validate_namespaces: partial\n" +
		"  print_context_in_exec: always\n" +
		"  allow_multiple_context_patterns: true\n" +
		"hooks:\n" +
		"  start_ctx: echo start\n" +
		"fzf:\n" +
		"  mouse: false\n"

	path := filepath.Join(home, ".kube", "kon.yaml")
	if err := os.WriteFile(path, []byte(config), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	s, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if s.Shell == nil || *s.Shell != "bash" {
		t.Errorf("shell = %v, want bash", s.Shell)
	}

	if s.DefaultEditor == nil || *s.DefaultEditor != "nano" {
		t.Errorf("default_editor = %v, want nano", s.DefaultEditor)
	}

	if !s.Prompt.Disable || s.Prompt.ShowDepth || !s.Prompt.ZshUseRPS1 {
		t.Errorf("prompt overrides not applied: %+v", s.Prompt)
	}

	if !s.Tui.IsEnabled() {
		t.Error("tui = false, want true")
	}

	sel, border := s.Tui.ColorValues()
	if sel != "212" || border != "238" {
		t.Errorf("tui colors = %q/%q, want default 212/238", sel, border)
	}

	if s.Behavior.ValidateNamespaces != ValidateNamespacesPartial {
		t.Errorf("validate_namespaces = %q, want partial", s.Behavior.ValidateNamespaces)
	}

	if !s.Behavior.ValidateNamespaces.CanListNamespaces() {
		t.Error("partial must allow listing namespaces")
	}

	if s.Behavior.PrintContextInExec != ContextHeaderAlways {
		t.Errorf("print_context_in_exec = %q, want always", s.Behavior.PrintContextInExec)
	}

	if !s.Behavior.AllowMultipleContextPatterns {
		t.Error("allow_multiple_context_patterns must be true")
	}

	if s.Hooks.StartCtx != "echo start" || s.Hooks.StopCtx != "" {
		t.Errorf("hooks = %+v", s.Hooks)
	}

	found := false
	for _, exclude := range s.Configs.Exclude {
		if exclude == path {
			found = true
		}
	}

	if !found {
		t.Errorf("configs.exclude %v must contain the settings file %q", s.Configs.Exclude, path)
	}
}

func TestValidateNamespacesAcceptsBoolAndPartial(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := os.MkdirAll(filepath.Join(home, ".kube"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	for value, want := range map[string]ValidateNamespaces{
		"false":   ValidateNamespacesFalse,
		"true":    ValidateNamespacesTrue,
		"partial": ValidateNamespacesPartial,
	} {
		path := filepath.Join(home, ".kube", "kon.yaml")
		config := "behavior:\n  validate_namespaces: " + value + "\n"

		if err := os.WriteFile(path, []byte(config), 0o600); err != nil {
			t.Fatalf("write config: %v", err)
		}

		s, err := Load()
		if err != nil {
			t.Fatalf("Load(%s): %v", value, err)
		}

		if s.Behavior.ValidateNamespaces != want {
			t.Errorf("validate_namespaces %q = %q, want %q", value, s.Behavior.ValidateNamespaces, want)
		}
	}
}

func TestValidateNamespacesFalseBlocksListing(t *testing.T) {
	if ValidateNamespacesFalse.CanListNamespaces() {
		t.Error("false must not allow listing namespaces")
	}

	if !ValidateNamespacesTrue.CanListNamespaces() {
		t.Error("true must allow listing namespaces")
	}
}

func TestLoadWithoutConfigFileUsesDefaults(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	s, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if !s.Tui.IsEnabled() || !s.Prompt.ShowDepth {
		t.Error("missing config file must yield upstream defaults")
	}

	if len(s.Configs.Include) != 7 {
		t.Errorf("configs.include = %v, want 7 default globs", s.Configs.Include)
	}
}

func TestLoadTuiDualForm(t *testing.T) {
	for _, tc := range []struct {
		name        string
		config      string
		wantEnabled bool
		wantSel     string
		wantBorder  string
	}{
		{
			name:        "scalar true",
			config:      "tui: true\n",
			wantEnabled: true,
			wantSel:     "212",
			wantBorder:  "238",
		},
		{
			name:        "scalar false",
			config:      "tui: false\n",
			wantEnabled: false,
			wantSel:     "212",
			wantBorder:  "238",
		},
		{
			name:        "map enabled with colors",
			config:      "tui:\n  enabled: true\n  selected_color: pink\n  border_color: '#00ff00'\n",
			wantEnabled: true,
			wantSel:     "pink",
			wantBorder:  "#00ff00",
		},
		{
			name:        "map disabled",
			config:      "tui:\n  enabled: false\n",
			wantEnabled: false,
			wantSel:     "212",
			wantBorder:  "238",
		},
		{
			name:        "map partial colors",
			config:      "tui:\n  selected_color: '42'\n",
			wantEnabled: true,
			wantSel:     "42",
			wantBorder:  "238",
		},
		{
			name:        "stale fzf key ignored",
			config:      "tui: true\nfzf:\n  mouse: false\n",
			wantEnabled: true,
			wantSel:     "212",
			wantBorder:  "238",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			if err := os.MkdirAll(filepath.Join(home, ".kube"), 0o755); err != nil {
				t.Fatalf("mkdir: %v", err)
			}
			path := filepath.Join(home, ".kube", "kon.yaml")
			if err := os.WriteFile(path, []byte(tc.config), 0o600); err != nil {
				t.Fatalf("write config: %v", err)
			}

			s, err := Load()
			if err != nil {
				t.Fatalf("Load: %v", err)
			}

			if s.Tui.IsEnabled() != tc.wantEnabled {
				t.Errorf("enabled = %v, want %v", s.Tui.IsEnabled(), tc.wantEnabled)
			}

			sel, border := s.Tui.ColorValues()
			if sel != tc.wantSel || border != tc.wantBorder {
				t.Errorf("colors = %q/%q, want %q/%q", sel, border, tc.wantSel, tc.wantBorder)
			}
		})
	}
}
