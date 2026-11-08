package picker

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/khanghua1505/kon/internal/settings"
)

// fzfArgs maps the fzf display settings onto fzf command line flags.
func fzfArgs(fzf settings.Fzf) []string {
	args := []string{"--no-multi"}

	if !fzf.Mouse {
		args = append(args, "--no-mouse")
	}

	if fzf.Reverse {
		args = append(args, "--reverse")
	}

	if fzf.IgnoreCase {
		args = append(args, "--case=ignore")
	}

	if fzf.InfoHidden {
		args = append(args, "--no-info")
	}

	if fzf.Height != nil && *fzf.Height != "" {
		args = append(args, "--height", *fzf.Height)
	}

	if fzf.Prompt != nil && *fzf.Prompt != "" {
		args = append(args, "--prompt", *fzf.Prompt)
	}

	if fzf.Color != nil && *fzf.Color != "" {
		args = append(args, "--color", *fzf.Color)
	}

	return args
}

// fzfAvailable reports whether fzf is installed.
func fzfAvailable() bool {
	_, err := exec.LookPath("fzf")

	return err == nil
}

// runFzf pipes the items into fzf and returns the selected one. fzf aborting
// is not an error, it ends the selection.
func runFzf(items []string, opts Options) (Result, string, error) {
	bin, err := exec.LookPath("fzf")
	if err != nil {
		return Cancelled, "", fmt.Errorf("could not find fzf: %w", err)
	}

	cmd := exec.Command(bin, fzfArgs(opts.Fzf)...)
	cmd.Stdin = strings.NewReader(strings.Join(items, "\n") + "\n")
	cmd.Stderr = os.Stderr

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			return Cancelled, "", fmt.Errorf("could not run fzf: %w", err)
		}

		// 130 is fzf's abort code, 1 means that nothing matched.
		if code := exitErr.ExitCode(); code != 130 && code != 1 {
			return Cancelled, "", fmt.Errorf("fzf failed: %w", err)
		}

		return Cancelled, "", nil
	}

	selection := strings.TrimSpace(out.String())
	if index := strings.IndexByte(selection, '\n'); index >= 0 {
		selection = selection[:index]
	}

	if selection == "" {
		return Cancelled, "", nil
	}

	return Selected, selection, nil
}
