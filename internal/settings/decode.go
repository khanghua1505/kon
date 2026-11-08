package settings

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// ValidateNamespaces selects the namespace validation behavior.
type ValidateNamespaces string

const (
	ValidateNamespacesTrue    ValidateNamespaces = "true"
	ValidateNamespacesFalse   ValidateNamespaces = "false"
	ValidateNamespacesPartial ValidateNamespaces = "partial"
)

// ContextHeaderBehavior selects when "kubie exec" prints context headers.
type ContextHeaderBehavior string

const (
	ContextHeaderAuto   ContextHeaderBehavior = "auto"
	ContextHeaderAlways ContextHeaderBehavior = "always"
	ContextHeaderNever  ContextHeaderBehavior = "never"
)

func (s *Settings) UnmarshalYAML(value *yaml.Node) error {
	type raw struct {
		Shell         *string      `yaml:"shell"`
		DefaultEditor *string      `yaml:"default_editor"`
		Tui           *TuiSettings `yaml:"tui"`
		Configs       *Configs     `yaml:"configs"`
		Prompt        *Prompt      `yaml:"prompt"`
		Behavior      *Behavior    `yaml:"behavior"`
		Hooks         *Hooks       `yaml:"hooks"`
	}

	var parsed raw
	if err := value.Decode(&parsed); err != nil {
		return err
	}

	def, err := Default()
	if err != nil {
		return err
	}
	*s = *def

	if parsed.Shell != nil {
		s.Shell = parsed.Shell
	}

	if parsed.DefaultEditor != nil {
		s.DefaultEditor = parsed.DefaultEditor
	}

	if parsed.Tui != nil {
		s.Tui = *parsed.Tui
	}

	if parsed.Configs != nil {
		s.Configs = *parsed.Configs
	}

	if parsed.Prompt != nil {
		s.Prompt = *parsed.Prompt
	}

	if parsed.Behavior != nil {
		s.Behavior = *parsed.Behavior
	}

	if parsed.Hooks != nil {
		s.Hooks = *parsed.Hooks
	}

	return nil
}

func (c *Configs) UnmarshalYAML(value *yaml.Node) error {
	type raw struct {
		Include *[]string `yaml:"include"`
		Exclude *[]string `yaml:"exclude"`
	}

	var parsed raw
	if err := value.Decode(&parsed); err != nil {
		return err
	}

	home, err := homeDir()
	if err != nil {
		return err
	}
	*c = Configs{Include: defaultIncludePatterns(home), Exclude: []string{}}

	if parsed.Include != nil {
		c.Include = *parsed.Include
	}

	if parsed.Exclude != nil {
		c.Exclude = *parsed.Exclude
	}

	return nil
}

func (p *Prompt) UnmarshalYAML(value *yaml.Node) error {
	type raw struct {
		Disable             *bool `yaml:"disable"`
		ShowDepth           *bool `yaml:"show_depth"`
		ZshUseRPS1          *bool `yaml:"zsh_use_rps1"`
		FishUseRprompt      *bool `yaml:"fish_use_rprompt"`
		XonshUseRightPrompt *bool `yaml:"xonsh_use_right_prompt"`
	}

	var parsed raw
	if err := value.Decode(&parsed); err != nil {
		return err
	}

	*p = Prompt{ShowDepth: true}

	if parsed.Disable != nil {
		p.Disable = *parsed.Disable
	}

	if parsed.ShowDepth != nil {
		p.ShowDepth = *parsed.ShowDepth
	}

	if parsed.ZshUseRPS1 != nil {
		p.ZshUseRPS1 = *parsed.ZshUseRPS1
	}

	if parsed.FishUseRprompt != nil {
		p.FishUseRprompt = *parsed.FishUseRprompt
	}

	if parsed.XonshUseRightPrompt != nil {
		p.XonshUseRightPrompt = *parsed.XonshUseRightPrompt
	}

	return nil
}

func (b *Behavior) UnmarshalYAML(value *yaml.Node) error {
	type raw struct {
		ValidateNamespaces           *ValidateNamespaces    `yaml:"validate_namespaces"`
		PrintContextInExec           *ContextHeaderBehavior `yaml:"print_context_in_exec"`
		AllowMultipleContextPatterns *bool                  `yaml:"allow_multiple_context_patterns"`
	}

	var parsed raw
	if err := value.Decode(&parsed); err != nil {
		return err
	}

	*b = Behavior{
		ValidateNamespaces: ValidateNamespacesTrue,
		PrintContextInExec: ContextHeaderAuto,
	}

	if parsed.ValidateNamespaces != nil {
		b.ValidateNamespaces = *parsed.ValidateNamespaces
	}

	if parsed.PrintContextInExec != nil {
		b.PrintContextInExec = *parsed.PrintContextInExec
	}

	if parsed.AllowMultipleContextPatterns != nil {
		b.AllowMultipleContextPatterns = *parsed.AllowMultipleContextPatterns
	}

	return nil
}

func (h *Hooks) UnmarshalYAML(value *yaml.Node) error {
	type raw struct {
		StartCtx *string `yaml:"start_ctx"`
		StopCtx  *string `yaml:"stop_ctx"`
	}

	var parsed raw
	if err := value.Decode(&parsed); err != nil {
		return err
	}

	*h = Hooks{}

	if parsed.StartCtx != nil {
		h.StartCtx = *parsed.StartCtx
	}

	if parsed.StopCtx != nil {
		h.StopCtx = *parsed.StopCtx
	}

	return nil
}

func (t *TuiSettings) UnmarshalYAML(value *yaml.Node) error {
	// TuiSettings carries no fields yet. Legacy `tui: true|false` and
	// `tui: {selected_color: ...}` shapes are accepted and ignored so old
	// config files keep loading; token fields decode here once redefined.
	*t = TuiSettings{}
	return nil
}

func (v *ValidateNamespaces) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode && value.Tag == "!!bool" {
		*v = ValidateNamespaces(strings.ToLower(value.Value))

		return nil
	}

	var text string
	if err := value.Decode(&text); err != nil {
		return fmt.Errorf("could not parse validate_namespaces: %w", err)
	}

	switch candidate := ValidateNamespaces(text); candidate {
	case ValidateNamespacesTrue, ValidateNamespacesFalse, ValidateNamespacesPartial:
		*v = candidate

		return nil
	default:
		return fmt.Errorf("unknown validate_namespaces value %q", text)
	}
}

func (b *ContextHeaderBehavior) UnmarshalYAML(value *yaml.Node) error {
	var text string
	if err := value.Decode(&text); err != nil {
		return fmt.Errorf("could not parse print_context_in_exec: %w", err)
	}

	switch candidate := ContextHeaderBehavior(text); candidate {
	case ContextHeaderAuto, ContextHeaderAlways, ContextHeaderNever:
		*b = candidate

		return nil
	default:
		return fmt.Errorf("unknown print_context_in_exec value %q", text)
	}
}
