package shell

import (
	"os"
	"strconv"
	"strings"

	"github.com/khanghua1505/kon/internal/settings"
)

// The colors used by the kubie prompt.
const (
	colorRed   = 31
	colorGreen = 32
	colorBlue  = 34
)

// executablePath returns the path of the running binary. It is a variable so
// the tests can pin it.
var executablePath = os.Executable

// generatePS1 builds the prompt shown by the kubie shell.
//
// The prompt holds the current context, the current namespace and, when the
// setting is enabled, the depth of the shell.
func generatePS1(s *settings.Settings, depth int, kind Kind, exePath string) string {
	parts := []string{
		colorize(kind, colorRed, command(kind, exePath+" info ctx")),
		colorize(kind, colorGreen, command(kind, exePath+" info ns")),
	}

	if s.Prompt.ShowDepth && depth > 1 {
		parts = append(parts, colorize(kind, colorBlue, strconv.Itoa(depth)))
	}

	return "[" + strings.Join(parts, "|") + "]"
}

// command wraps a command in the substitution syntax of the shell.
func command(kind Kind, content string) string {
	if kind == Fish {
		return "(" + content + ")"
	}
	return "$(" + content + ")"
}

// colorize wraps the content in the color escape sequence of the shell.
//
// Only the escape sequences are isolated, the content is left alone so the
// shell still runs the command substitution inside it.
func colorize(kind Kind, color int, content string) string {
	return isolate(kind, startColor(kind, color)) + content + isolate(kind, endColor(kind))
}

// startColor returns the escape sequence starting a color.
func startColor(kind Kind, color int) string {
	if kind == Xonsh {
		return "\\033[" + strconv.Itoa(color) + "m"
	}
	return "\\e[" + strconv.Itoa(color) + "m"
}

// endColor returns the escape sequence resetting the color.
func endColor(kind Kind) string {
	if kind == Xonsh {
		return "\\033[0m"
	}
	return "\\e[0m"
}

// isolate marks the escape sequences as non printable for the shell.
func isolate(kind Kind, content string) string {
	switch kind {
	case Zsh:
		return "%{" + content + "%}"
	case Bash:
		return "\\[" + content + "\\]"
	default:
		return content
	}
}
