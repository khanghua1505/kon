// Package picker lets the user choose one item out of a list.
//
// It mirrors Rust kon's select_or_list_* helpers: fzf is used when it is
// installed and stdout is a terminal, otherwise the items are printed so
// that scripts can consume them.
package picker

import (
	"fmt"
	"io"
	"os"
	"slices"

	"github.com/khanghua1505/kon/internal/settings"
	"github.com/khanghua1505/kon/internal/tty"
)

// Result reports how a selection ended.
type Result int

const (
	// Cancelled means the user aborted the selection.
	Cancelled Result = iota
	// Listed means the items were printed for the caller to consume.
	Listed
	// Selected means the user picked one of the items.
	Selected
)

// Mode selects the picker implementation.
type Mode int

const (
	// ModeAuto uses fzf when it is installed and stdout is a terminal, and
	// falls back to the numbered terminal prompt otherwise.
	ModeAuto Mode = iota
	// ModeFzf uses fzf when it is installed.
	ModeFzf
	// ModeTerminal always uses the numbered terminal prompt.
	ModeTerminal
	// ModeTui uses the full-screen bubbletea picker when both streams are
	// terminals, and falls back to the automatic behavior otherwise.
	ModeTui
)

// Options tunes SelectWithOptions.
type Options struct {
	// Mode selects the picker implementation.
	Mode Mode
	// Fzf holds the display options that are passed to fzf.
	Fzf settings.Fzf
	// In is the stream the terminal prompt reads the choice from.
	In io.Reader
	// Out is the stream the item list and the prompt are written to.
	Out io.Writer
}

// Select presents items and reports the outcome. It is the Go counterpart of
// Rust kon's select_or_list_* helpers.
func Select(items []string, fzf *settings.Fzf) (Result, string, error) {
	opts := Options{}
	if fzf != nil {
		opts.Fzf = *fzf
	}

	return SelectWithOptions(items, opts)
}

// SelectWithOptions presents items and reports the outcome.
func SelectWithOptions(items []string, opts Options) (Result, string, error) {
	if len(items) == 0 {
		return Cancelled, "", fmt.Errorf("no items to select")
	}

	if len(items) == 1 {
		return Selected, items[0], nil
	}

	opts = opts.withDefaults()

	sorted := sortedItems(items)

	// Rust kubie reverses the sorted names before handing them to the
	// interactive menu, so the menu lists them in reverse order.
	menu := slices.Clone(sorted)
	slices.Reverse(menu)

	switch {
	case opts.Mode == ModeTerminal:
		return promptList(menu, opts)
	case opts.Mode == ModeTui && attachedToTerminal(opts):
		return runTui(menu, opts)
	case fzfAvailable() && (opts.Mode == ModeFzf || isTerminal(opts.Out)):
		return runFzf(menu, opts)
	case isTerminal(opts.Out):
		return promptList(menu, opts)
	default:
		return listItems(sorted, opts)
	}
}

// attachedToTerminal reports whether both picker streams are terminals.
func attachedToTerminal(opts Options) bool {
	in, okIn := opts.In.(*os.File)
	out, okOut := opts.Out.(*os.File)

	return okIn && okOut && tty.IsTerminal(in) && tty.IsTerminal(out)
}

// listItems prints the items one per line so that scripts can consume them.
func listItems(sorted []string, opts Options) (Result, string, error) {
	for _, item := range sorted {
		if _, err := fmt.Fprintln(opts.Out, item); err != nil {
			return Listed, "", fmt.Errorf("could not print the item list: %w", err)
		}
	}

	return Listed, "", nil
}

func sortedItems(items []string) []string {
	sorted := slices.Clone(items)
	slices.Sort(sorted)

	return sorted
}

func (o Options) withDefaults() Options {
	if o.In == nil {
		o.In = os.Stdin
	}

	if o.Out == nil {
		o.Out = os.Stdout
	}

	return o
}

// isTerminal reports whether the output stream is attached to a terminal.
func isTerminal(out io.Writer) bool {
	file, ok := out.(*os.File)
	if !ok {
		return false
	}

	return tty.IsTerminal(file)
}
