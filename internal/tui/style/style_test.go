package style

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func Test_Resolve_returns_mapped_color_for_supported_value(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  lipgloss.TerminalColor
	}{
		{"black", "black", lipgloss.ANSIColor(0)},
		{"red", "red", lipgloss.ANSIColor(1)},
		{"green", "green", lipgloss.ANSIColor(2)},
		{"yellow", "yellow", lipgloss.ANSIColor(3)},
		{"blue", "blue", lipgloss.ANSIColor(4)},
		{"magenta", "magenta", lipgloss.ANSIColor(5)},
		{"cyan", "cyan", lipgloss.ANSIColor(6)},
		{"white", "white", lipgloss.ANSIColor(7)},
		{"brightblack", "brightblack", lipgloss.ANSIColor(8)},
		{"brightred", "brightred", lipgloss.ANSIColor(9)},
		{"brightgreen", "brightgreen", lipgloss.ANSIColor(10)},
		{"brightyellow", "brightyellow", lipgloss.ANSIColor(11)},
		{"brightblue", "brightblue", lipgloss.ANSIColor(12)},
		{"brightmagenta", "brightmagenta", lipgloss.ANSIColor(13)},
		{"brightcyan", "brightcyan", lipgloss.ANSIColor(14)},
		{"brightwhite", "brightwhite", lipgloss.ANSIColor(15)},
		{"name uppercase", "BLACK", lipgloss.ANSIColor(0)},
		{"name mixed case", "BrightWhite", lipgloss.ANSIColor(15)},
		{"decimal", "212", lipgloss.Color("212")},
		{"hex", "#ff5fd7", lipgloss.Color("#ff5fd7")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Resolve(tt.input)
			if err != nil {
				t.Fatalf("Resolve(%q) returned error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("Resolve(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func Test_Resolve_returns_error_for_unsupported_value(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"unknown name", "neon"},
		{"decimal above range", "256"},
		{"negative decimal", "-1"},
		{"short hex", "#fff"},
		{"invalid hex digits", "#12345g"},
		{"empty", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Resolve(tt.input)
			if err == nil {
				t.Fatalf("Resolve(%q) = %v, want error", tt.input, got)
			}
			if got != nil {
				t.Errorf("Resolve(%q) color = %v, want nil", tt.input, got)
			}
		})
	}
}
