package picker

import (
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/khanghua1505/kon/internal/settings"
)

// ttyInOut returns pipe-backed stdin and stdout files so the TUI branch of the
// fallback chain can be reached from a test.
func ttyInOut(t *testing.T) (*os.File, *os.File) {
	t.Helper()

	inReader, inWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	outReader, outWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}

	t.Cleanup(func() {
		_ = inReader.Close()
		_ = inWriter.Close()
		_ = outReader.Close()
		_ = outWriter.Close()
	})

	return inWriter, outWriter
}

func TestSelectSingleItemSkipsTui(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	forceTTY(t)
	in, out := ttyInOut(t)

	result, item, err := SelectWithOptions([]string{"only"}, Options{
		Tui: settings.TuiSettings{Enabled: true},
		In:  in,
		Out: out,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result != Selected || item != "only" {
		t.Errorf("result = %v item = %q, want Selected and only", result, item)
	}
}

func TestSelectTuiEnabledNonFileInputFallsBackToPrompt(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	forceTTY(t)
	out, readOut := promptOut(t)

	result, _, err := SelectWithOptions([]string{"alpha", "beta"}, Options{
		Tui: settings.TuiSettings{Enabled: true},
		In:  strings.NewReader("1\n"),
		Out: out,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result != Selected {
		t.Fatalf("expected Selected, got %v", result)
	}

	if got := readOut(); !strings.Contains(got, "1) ") {
		t.Errorf("expected the numbered prompt menu, got %q", got)
	}
}

func strp(s string) *string { return &s }

func TestResolveColors(t *testing.T) {
	tests := []struct {
		name string
		tui  settings.TuiSettings
		want [2]lipgloss.TerminalColor
	}{
		{
			name: "zero value falls back to defaults",
			tui:  settings.TuiSettings{},
			want: [2]lipgloss.TerminalColor{lipgloss.Color("212"), lipgloss.Color("238")},
		},
		{
			name: "valid named color is used",
			tui:  settings.TuiSettings{Enabled: true, SelectedColor: strp("magenta"), BorderColor: strp("7")},
			want: [2]lipgloss.TerminalColor{lipgloss.ANSIColor(5), lipgloss.Color("7")},
		},
		{
			name: "valid hex color is used",
			tui:  settings.TuiSettings{Enabled: true, SelectedColor: strp("#ff00ff")},
			want: [2]lipgloss.TerminalColor{lipgloss.Color("#ff00ff"), lipgloss.Color("238")},
		},
		{
			name: "invalid color falls back to default",
			tui:  settings.TuiSettings{Enabled: true, SelectedColor: strp("#zzz"), BorderColor: strp("notacolor")},
			want: [2]lipgloss.TerminalColor{lipgloss.Color("212"), lipgloss.Color("238")},
		},
	}

	for _, tt := range tests {
		selected, border := resolveColors(tt.tui)
		if selected != tt.want[0] || border != tt.want[1] {
			t.Errorf("%s: got (%v, %v), want (%v, %v)", tt.name, selected, border, tt.want[0], tt.want[1])
		}
	}
}
