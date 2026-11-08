// Package picker lets the user choose one item out of a list.
//
// It mirrors Rust kon's select_or_list_* helpers: with a single item it is
// returned right away, the full-screen TUI is used when it is enabled and
// both streams are terminals, the numbered terminal prompt is used when
// stdout is a terminal, and otherwise the items are printed so that scripts
// can consume them.
package picker

import (
	"fmt"
	"io"
	"os"
	"slices"

	"github.com/charmbracelet/lipgloss"
	"github.com/khanghua1505/kon/internal/settings"
	"github.com/khanghua1505/kon/internal/tui/style"
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

// Options tunes SelectWithOptions.
type Options struct {
	// Tui carries the picker settings: the enabled toggle and colors.
	Tui settings.TuiSettings
	// In is the stream the terminal prompt reads the choice from.
	In io.Reader
	// Out is the stream the item list and the prompt are written to.
	Out io.Writer
}

// isTTY reports whether the file is attached to a terminal. It is a variable
// so tests can force either branch of the fallback chain.
var isTTY = func(f *os.File) bool {
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

// Select presents items and reports the outcome. It is the Go counterpart of
// Rust kon's select_or_list_* helpers.
func Select(items []string) (Result, string, error) {
	return SelectWithOptions(items, Options{})
}

// SelectWithOptions presents items and reports the outcome. The fallback
// chain is: a single item is returned without any interaction, then the TUI
// when enabled and both streams are terminals, then the numbered prompt when
// stdout is a terminal, then the plain item list.
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
	case opts.Tui.IsEnabled() && attachedToTerminal(opts):
		return runTui(menu, opts)
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

	return okIn && okOut && isTTY(in) && isTTY(out)
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

	return isTTY(file)
}

// resolveColors turns the configured color strings into terminal colors,
// falling back to the defaults when they cannot be resolved.
func resolveColors(tui settings.TuiSettings) (selected, border lipgloss.TerminalColor) {
	selected, border = lipgloss.Color("212"), lipgloss.Color("238")

	selectedText, borderText := tui.ColorValues()

	if color, err := style.Resolve(selectedText); err == nil {
		selected = color
	}

	if color, err := style.Resolve(borderText); err == nil {
		border = color
	}

	return selected, border
}
