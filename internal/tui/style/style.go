// Package style resolves user-supplied color strings into lipgloss terminal colors.
package style

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var ansiNames = map[string]lipgloss.ANSIColor{
	"black":         0,
	"red":           1,
	"green":         2,
	"yellow":        3,
	"blue":          4,
	"magenta":       5,
	"cyan":          6,
	"white":         7,
	"brightblack":   8,
	"brightred":     9,
	"brightgreen":   10,
	"brightyellow":  11,
	"brightblue":    12,
	"brightmagenta": 13,
	"brightcyan":    14,
	"brightwhite":   15,
}

// Resolve converts a color name, decimal ANSI index, or #RRGGBB hex value into a
// lipgloss color. Names are case-insensitive. Unsupported values return an error so
// callers can fall back to their own default.
func Resolve(text string) (lipgloss.TerminalColor, error) {
	value := strings.TrimSpace(text)
	if color, ok := ansiNames[strings.ToLower(value)]; ok {
		return color, nil
	}
	if _, err := strconv.ParseUint(value, 10, 8); err == nil {
		return lipgloss.Color(value), nil
	}
	if len(value) == 7 && value[0] == '#' {
		if _, err := hex.DecodeString(value[1:]); err == nil {
			return lipgloss.Color(value), nil
		}
	}
	return nil, fmt.Errorf("unknown color %q", text)
}
