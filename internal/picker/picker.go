// Package picker lets the user choose one item out of a list.
//
// It mirrors Rust kon's select_or_list_* helpers: the full-screen TUI is
// used when both streams are terminals, and otherwise the items are printed
// so that scripts can consume them.
package picker

import (
	"fmt"
	"io"
	"os"
	"slices"

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
	// Style carries the theme the TUI renders from.
	Style style.Style
	// In is the input stream of the interactive TUI.
	In io.Reader
	// Out is the stream the TUI and the item list are written to.
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

// SelectWithOptions presents items and reports the outcome. The TUI runs
// when both streams are terminals; otherwise the items are listed.
func SelectWithOptions(items []string, opts Options) (Result, string, error) {
	if len(items) == 0 {
		return Cancelled, "", fmt.Errorf("no items to select")
	}

	opts = opts.withDefaults()

	sorted := sortedItems(items)

	// Rust kubie reverses the sorted names before handing them to the
	// interactive menu, so the menu lists them in reverse order.
	menu := slices.Clone(sorted)
	slices.Reverse(menu)

	if attachedToTerminal(opts) {
		return runTui(menu, opts)
	}

	return listItems(sorted, opts)
}

// listItems prints the sorted items one per line and reports them as listed.
func listItems(items []string, opts Options) (Result, string, error) {
	for _, item := range items {
		if _, err := fmt.Fprintln(opts.Out, item); err != nil {
			return Cancelled, "", fmt.Errorf("could not print the item list: %w", err)
		}
	}

	return Listed, "", nil
}

// attachedToTerminal reports whether both picker streams are terminals.
func attachedToTerminal(opts Options) bool {
	in, okIn := opts.In.(*os.File)
	out, okOut := opts.Out.(*os.File)

	return okIn && okOut && isTTY(in) && isTTY(out)
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

	if o.Style == (style.Style{}) {
		o.Style = style.Default()
	}

	return o
}
