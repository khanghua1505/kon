package picker

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/khanghua1505/kon/internal/settings"
)

// fakeFzf installs an fzf shim in dir and returns the file that receives the
// arguments the shim was called with.
func fakeFzf(t *testing.T, dir, body string) string {
	t.Helper()

	argsFile := filepath.Join(dir, "args.txt")
	script := "#!/bin/sh\ncat > /dev/null\n" +
		"printf '%s\\n' \"$@\" > \"$FZF_ARGS_FILE\"\n" +
		body

	if err := os.WriteFile(filepath.Join(dir, "fzf"), []byte(script), 0o755); err != nil {
		t.Fatalf("could not write the fzf shim: %v", err)
	}

	t.Setenv("FZF_ARGS_FILE", argsFile)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	return argsFile
}

func TestSelectFzfReturnsSelection(t *testing.T) {
	dir := t.TempDir()
	fakeFzf(t, dir, "printf 'beta\\n'\n")

	var out bytes.Buffer

	result, item, err := SelectWithOptions([]string{"alpha", "beta"}, Options{Mode: ModeFzf, Out: &out})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result != Selected {
		t.Fatalf("expected Selected, got %v", result)
	}

	if item != "beta" {
		t.Fatalf("expected beta, got %q", item)
	}
}

func TestSelectFzfAbortedIsNotAnError(t *testing.T) {
	dir := t.TempDir()
	fakeFzf(t, dir, "exit 130\n")

	result, item, err := SelectWithOptions([]string{"alpha", "beta"}, Options{Mode: ModeFzf})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result != Cancelled {
		t.Fatalf("expected Cancelled, got %v", result)
	}

	if item != "" {
		t.Fatalf("expected no item, got %q", item)
	}
}

func TestSelectFzfNoMatchIsCancelled(t *testing.T) {
	dir := t.TempDir()
	fakeFzf(t, dir, "exit 1\n")

	result, _, err := SelectWithOptions([]string{"alpha", "beta"}, Options{Mode: ModeFzf})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result != Cancelled {
		t.Fatalf("expected Cancelled, got %v", result)
	}
}

func TestSelectFzfGetsGoldenArgs(t *testing.T) {
	dir := t.TempDir()
	argsFile := fakeFzf(t, dir, "printf 'beta\\n'\n")

	height := "50%"
	prompt := "ctx>"
	color := "fg:#fff"
	fzf := settings.Fzf{
		Mouse:      false,
		Reverse:    true,
		IgnoreCase: true,
		InfoHidden: true,
		Height:     &height,
		Prompt:     &prompt,
		Color:      &color,
	}

	if _, _, err := SelectWithOptions([]string{"alpha", "beta"}, Options{Mode: ModeFzf, Fzf: fzf}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	got, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("could not read the recorded arguments: %v", err)
	}

	want, err := os.ReadFile(filepath.Join("testdata", "fzf_args_full.golden"))
	if err != nil {
		t.Fatalf("could not read the golden file: %v", err)
	}

	if string(got) != string(want) {
		t.Fatalf("arguments mismatch:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestFzfArgsGolden(t *testing.T) {
	height := "50%"
	prompt := "ctx>"
	color := "fg:#fff"

	cases := []struct {
		name   string
		fzf    settings.Fzf
		golden string
	}{
		{
			name:   "default",
			fzf:    settings.Fzf{Mouse: true},
			golden: "fzf_args_default.golden",
		},
		{
			name: "full",
			fzf: settings.Fzf{
				Mouse:      false,
				Reverse:    true,
				IgnoreCase: true,
				InfoHidden: true,
				Height:     &height,
				Prompt:     &prompt,
				Color:      &color,
			},
			golden: "fzf_args_full.golden",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := strings.Join(fzfArgs(tc.fzf), "\n") + "\n"

			want, err := os.ReadFile(filepath.Join("testdata", tc.golden))
			if err != nil {
				t.Fatalf("could not read the golden file: %v", err)
			}

			if got != string(want) {
				t.Fatalf("arguments mismatch:\ngot:\n%s\nwant:\n%s", got, want)
			}
		})
	}
}

func TestSelectListsItemsWhenNotATerminal(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	var out bytes.Buffer

	result, item, err := SelectWithOptions([]string{"beta", "alpha"}, Options{Out: &out})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result != Listed {
		t.Fatalf("expected Listed, got %v", result)
	}

	if item != "" {
		t.Fatalf("expected no item, got %q", item)
	}

	if got := out.String(); got != "alpha\nbeta\n" {
		t.Fatalf("expected the sorted items, got %q", got)
	}
}

func TestSelectTerminalPromptReadsNumber(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	var out bytes.Buffer

	result, item, err := SelectWithOptions([]string{"alpha", "beta"}, Options{
		Mode: ModeTerminal,
		In:   strings.NewReader("2\n"),
		Out:  &out,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result != Selected {
		t.Fatalf("expected Selected, got %v", result)
	}

	if item != "alpha" {
		t.Fatalf("expected alpha, got %q", item)
	}

	if !strings.Contains(out.String(), "1) beta") || !strings.Contains(out.String(), "2) alpha") {
		t.Fatalf("expected the reversed menu, got %q", out.String())
	}
}

func TestSelectTerminalPromptRepromptsOnInvalidNumber(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	var out bytes.Buffer

	result, item, err := SelectWithOptions([]string{"alpha", "beta"}, Options{
		Mode: ModeTerminal,
		In:   strings.NewReader("abc\n9\n1\n"),
		Out:  &out,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result != Selected {
		t.Fatalf("expected Selected, got %v", result)
	}

	if item != "beta" {
		t.Fatalf("expected beta, got %q", item)
	}

	if got := strings.Count(out.String(), "Invalid selection:"); got != 2 {
		t.Fatalf("expected 2 invalid selections, got %d in %q", got, out.String())
	}
}

func TestSelectTerminalPromptEndsOnEOF(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	var out bytes.Buffer

	result, item, err := SelectWithOptions([]string{"alpha", "beta"}, Options{
		Mode: ModeTerminal,
		In:   strings.NewReader(""),
		Out:  &out,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result != Cancelled {
		t.Fatalf("expected Cancelled, got %v", result)
	}

	if item != "" {
		t.Fatalf("expected no item, got %q", item)
	}
}

func TestSelectReturnsSingleItemImmediately(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	result, item, err := Select([]string{"only"}, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result != Selected {
		t.Fatalf("expected Selected, got %v", result)
	}

	if item != "only" {
		t.Fatalf("expected only, got %q", item)
	}
}

func TestSelectWithoutItemsFails(t *testing.T) {
	if _, _, err := Select(nil, nil); err == nil {
		t.Fatal("expected an error for an empty item list")
	}
}
