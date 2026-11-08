// Package shell spawns the shell kubie runs inside.
//
// It mirrors /tmp/kubie-src/src/shell: the shell kind is detected from the
// process tree, the isolated kubeconfig and the session history are handed to
// the child shell through temporary files, and every shell gets its own rc
// file so the kubie prompt and the KUBECONFIG fixup hook are installed.
package shell

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/khanghua1505/kon/internal/kubeconfig"
	"github.com/khanghua1505/kon/internal/session"
	"github.com/khanghua1505/kon/internal/settings"
	"github.com/khanghua1505/kon/internal/state"
	"github.com/khanghua1505/kon/internal/vars"
)

// Kind is a shell kubie can spawn.
type Kind int

// The shells kubie knows how to spawn.
const (
	Bash Kind = iota
	Fish
	Xonsh
	Zsh
	Nu
)

// String returns the name of the shell executable.
func (k Kind) String() string {
	switch k {
	case Bash:
		return "bash"
	case Fish:
		return "fish"
	case Xonsh:
		return "xonsh"
	case Zsh:
		return "zsh"
	case Nu:
		return "nu"
	default:
		return ""
	}
}

// envEntry is a single environment variable handed to the child shell.
type envEntry struct {
	name  string
	value string
}

// EnvVars holds the environment variables exported to the child shell.
//
// The variables keep their insertion order so the child environment and the
// nu command line are reproducible.
type EnvVars struct {
	entries []envEntry
}

// Insert sets a variable, replacing an existing value with the same name.
func (e *EnvVars) Insert(name, value string) {
	for i := range e.entries {
		if e.entries[i].name == name {
			e.entries[i].value = value
			return
		}
	}
	e.entries = append(e.entries, envEntry{name: name, value: value})
}

// Apply exports the variables to cmd, replacing inherited values with the
// same name.
func (e *EnvVars) Apply(cmd *exec.Cmd) {
	env := cmd.Env
	if env == nil {
		env = os.Environ()
	}
	for _, entry := range e.entries {
		env = append(env, entry.name+"="+entry.value)
	}
	cmd.Env = env
}

// spawnInfo carries everything a shell spawner needs.
type spawnInfo struct {
	settings *settings.Settings
	envVars  EnvVars
	prompt   string
	kind     Kind
}

// Shell spawns the shell for the given settings and waits for it to exit.
//
// The kubeconfig and the session history are written to temporary files which
// are removed once the shell exits. The returned int is the exit code of the
// child shell, a shell killed by a signal reports 0.
func Shell(s *settings.Settings, config *kubeconfig.KubeConfig, sess *session.Session) (int, error) {
	kind, err := shellKind(s)
	if err != nil {
		return 0, err
	}

	configPath, err := config.WriteTemp()
	if err != nil {
		return 0, fmt.Errorf("could not write temporary kubeconfig: %w", err)
	}
	defer func() { _ = os.Remove(configPath) }()

	sessionPath, err := writeSessionTemp(sess)
	if err != nil {
		return 0, fmt.Errorf("could not write temporary session file: %w", err)
	}
	defer func() { _ = os.Remove(sessionPath) }()

	nextDepth := vars.Depth() + 1

	envVars := buildEnvVars(s, kind, configPath, sessionPath, nextDepth)

	exePath, err := executablePath()
	if err != nil {
		return 0, fmt.Errorf("could not get own binary path: %w", err)
	}

	info := &spawnInfo{
		settings: s,
		envVars:  envVars,
		prompt:   generatePS1(s, nextDepth, kind, exePath),
		kind:     kind,
	}

	switch kind {
	case Bash:
		return spawnBash(info)
	case Fish:
		return spawnFish(info)
	case Xonsh:
		return spawnXonsh(info)
	case Zsh:
		return spawnZsh(info)
	case Nu:
		return spawnNu(info)
	default:
		return 0, fmt.Errorf("unsupported shell: %s", kind)
	}
}

// shellKind returns the shell to spawn, honouring the shell setting and
// falling back to the detection of the shell kubie was started from.
func shellKind(s *settings.Settings) (Kind, error) {
	if s.Shell != nil {
		kind, ok := KindFromString(*s.Shell)
		if !ok {
			return 0, fmt.Errorf("Invalid shell setting: %s", *s.Shell)
		}
		return kind, nil
	}
	return Detect()
}

// buildEnvVars returns the environment variables exported to the child shell.
func buildEnvVars(s *settings.Settings, kind Kind, configPath, sessionPath string, depth int) EnvVars {
	var env EnvVars
	env.Insert("KUBECONFIG", configPath)
	depthStr := strconv.Itoa(depth)
	env.Insert("KON_ACTIVE", "1")
	env.Insert("KON_DEPTH", depthStr)
	env.Insert("KON_KUBECONFIG", configPath)
	env.Insert("KON_KUBE_CONFIG", configPath)
	env.Insert("KON_SESSION", sessionPath)
	env.Insert("KON_STATE", statePath())
	env.Insert("KON_PROMPT_DISABLE", boolFlag(s.Prompt.Disable))
	env.Insert("KON_ZSH_USE_RPS1", boolFlag(s.Prompt.ZshUseRPS1))
	env.Insert("KON_FISH_USE_RPROMPT", boolFlag(s.Prompt.FishUseRprompt))
	env.Insert("KON_XONSH_USE_RIGHT_PROMPT", boolFlag(s.Prompt.XonshUseRightPrompt))
	env.Insert("KON_SHELL", kind.String())
	env.Insert("KUBIE_ACTIVE", "1")
	env.Insert("KUBIE_DEPTH", depthStr)
	env.Insert("KUBIE_KUBECONFIG", configPath)
	env.Insert("KUBIE_SESSION", sessionPath)
	env.Insert("KUBIE_STATE", statePath())
	env.Insert("KUBIE_PROMPT_DISABLE", boolFlag(s.Prompt.Disable))
	env.Insert("KUBIE_ZSH_USE_RPS1", boolFlag(s.Prompt.ZshUseRPS1))
	env.Insert("KUBIE_FISH_USE_RPROMPT", boolFlag(s.Prompt.FishUseRprompt))
	env.Insert("KUBIE_XONSH_USE_RIGHT_PROMPT", boolFlag(s.Prompt.XonshUseRightPrompt))
	env.Insert("KUBIE_SHELL", kind.String())
	return env
}

// boolFlag renders a boolean setting the way the child shell expects it.
func boolFlag(value bool) string {
	if value {
		return "1"
	}
	return "0"
}

// statePath returns the path of the state file exported to the child shell.
//
// The environment wins so the shell and kubie itself share the same state
// file, the fallback is the platform data directory used by the Rust
// implementation.
func statePath() string {
	if path := os.Getenv(state.EnvState); path != "" {
		return path
	}
	return filepath.Join(dataLocalDir(), "kubie", "state.json")
}

// dataLocalDir returns the platform local data directory.
func dataLocalDir() string {
	if runtime.GOOS == "darwin" {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, "Library", "Application Support")
		}
	}
	if dir := os.Getenv("XDG_DATA_HOME"); dir != "" {
		return dir
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".local", "share")
	}
	return ""
}

// writeSessionTemp writes the session to a temporary file and returns its
// path. The file is removed by the caller.
func writeSessionTemp(sess *session.Session) (string, error) {
	file, err := os.CreateTemp("", "kubie-session*.json")
	if err != nil {
		return "", err
	}
	path := file.Name()

	history := sess.History
	if history == nil {
		history = []session.HistoryEntry{}
	}
	data, err := json.Marshal(session.Session{History: history})
	if err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return "", err
	}

	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return "", err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return "", err
	}
	return path, nil
}

// exitCode maps the result of running the child shell to an exit code.
//
// A shell terminated by a signal reports 0, matching the Rust implementation
// which reads the exit code of the child process.
func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if code := exitErr.ExitCode(); code >= 0 {
			return code
		}
	}
	return 0
}

// run executes the child shell with the standard streams of kubie attached.
func run(cmd *exec.Cmd) (int, error) {
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitCode(err), nil
		}
		return 0, err
	}
	return 0, nil
}
