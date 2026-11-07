package shell

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Detect returns the shell kubie was started from.
//
// The process tree is walked upwards until the command of a parent process
// matches a known shell, the same way the Rust implementation does it.
func Detect() (Kind, error) {
	pid := fmt.Sprintf("%d", os.Getpid())
	parent, err := parentOf(pid)
	if err != nil {
		return 0, err
	}

	for {
		if parent == "1" {
			return 0, fmt.Errorf("Could not detect shell in use")
		}

		command, err := commandOf(parent)
		if err != nil {
			return 0, err
		}

		if kind, ok := KindFromString(parseCommand(command)); ok {
			return kind, nil
		}

		parent, err = parentOf(parent)
		if err != nil {
			return 0, err
		}
	}
}

// KindFromString maps the name of a shell to its kind.
func KindFromString(name string) (Kind, bool) {
	switch name {
	case "bash", "dash":
		return Bash, true
	case "fish":
		return Fish, true
	case "xonsh", "python":
		return Xonsh, true
	case "zsh":
		return Zsh, true
	case "nu":
		return Nu, true
	default:
		return 0, false
	}
}

// parentOf returns the parent process id of the given process.
func parentOf(pid string) (string, error) {
	output, err := runPS("-o", "ppid=", pid)
	if err != nil {
		return "", err
	}
	if len(output) == 0 {
		return "", fmt.Errorf("Could not detect parent process")
	}
	return output[0], nil
}

// commandOf returns the command line of the given process.
func commandOf(pid string) (string, error) {
	output, err := runPS("-o", "args=", pid)
	if err != nil {
		return "", err
	}
	if len(output) == 0 {
		return "", fmt.Errorf("Could not detect command")
	}
	return output[0], nil
}

// runPS calls ps with the given arguments and returns its output lines.
func runPS(args ...string) ([]string, error) {
	cmd := exec.Command("ps", args...)
	stderr := &strings.Builder{}
	cmd.Stderr = stderr

	output, err := cmd.Output()
	if err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			return nil, fmt.Errorf("Error calling ps: %v", err)
		}
		return nil, fmt.Errorf("Error calling ps: %s", message)
	}

	var lines []string
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines, nil
}

// parseCommand returns the name of the binary of a command line.
//
// The leading dashes of a login shell and the version suffix of an
// interpreter are removed, so "-bash" and "python3.8" become "bash" and
// "python".
func parseCommand(command string) string {
	firstSpace := strings.Index(command, " ")
	if firstSpace == -1 {
		firstSpace = len(command)
	}

	binaryPath := command[:firstSpace]
	separator := strings.LastIndex(binaryPath, "/")
	binary := binaryPath
	if separator != -1 {
		binary = binaryPath[separator+1:]
	}

	binary = strings.TrimLeft(binary, "-")
	binary = strings.TrimRight(binary, "0123456789.")
	return binary
}
