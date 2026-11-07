// Package tty reports whether a file is attached to a terminal.
package tty

import "os"

// IsTerminal reports whether f is a terminal.
func IsTerminal(f *os.File) bool {
	if f == nil {
		return false
	}

	return isTerminal(f.Fd())
}
