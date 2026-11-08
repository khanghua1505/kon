package style

import (
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// Style is the theme every kon TUI component renders from. It mirrors the
// design-token sets of frontend systems (primary/background/foreground plus
// semantic colors): components never hardcode colors, they read tokens.
type Style struct {
	// Primary is the brand color: selection highlight, titles, emphasis.
	Primary   ansi.Color
	Secondary ansi.Color
	// Accent marks highlights that must stand apart from Primary.
	Accent     ansi.Color
	Background ansi.Color
	Foreground ansi.Color
	// Muted is dimmed text: help lines, dividers, hints.
	Muted   ansi.Color
	Border  ansi.Color
	Error   ansi.Color
	Warning ansi.Color
	Success ansi.Color
	Info    ansi.Color
}

// defaultStyle is the palette used whenever TuiSettings leaves a token unset.
// Values are xterm 256-color indexes; the xterm name is kept in the comment.
var defaultStyle = Style{
	Primary:    lipgloss.ANSIColor(212), // orchid2
	Secondary:  lipgloss.ANSIColor(39),  // deepskyblue1
	Accent:     lipgloss.ANSIColor(50),  // cyan2
	Background: lipgloss.ANSIColor(236), // grey23
	Foreground: lipgloss.ANSIColor(252), // grey85
	Muted:      lipgloss.ANSIColor(241), // grey58
	Border:     lipgloss.ANSIColor(238), // grey27
	Error:      lipgloss.ANSIColor(196), // red1
	Warning:    lipgloss.ANSIColor(214), // orange1
	Success:    lipgloss.ANSIColor(46),  // green1
	Info:       lipgloss.ANSIColor(81),  // steelblue1
}

// Default returns the default palette.
func Default() Style {
	return defaultStyle
}
