package shell

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/khanghua1505/kon/internal/settings"
)

// writeFake installs an executable script in dir.
func writeFake(t *testing.T, dir, name, body string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatalf("could not write fake %s: %v", name, err)
	}
	return path
}

// fakePS installs a fake ps which reports the given parent pid and command
// and records every call in the log file.
func fakePS(t *testing.T, dir, parent, command, log string) {
	t.Helper()

	body := `#!/bin/sh
mode=""
pid=""
for arg in "$@"; do
  case "$arg" in
    ppid=) mode=ppid ;;
    args=) mode=args ;;
    -*) ;;
    *) pid="$arg" ;;
  esac
done
printf '%s %s\n' "$mode" "$pid" >> "` + log + `"
case "$mode" in
  ppid) printf '%s\n' "` + parent + `" ;;
  args) printf '%s\n' "` + command + `" ;;
esac
`
	writeFake(t, dir, "ps", body)
}

// TestDetectWalksProcessTree checks that the shell is found by walking the
// process tree upwards and that the parent is looked up before the command.
func TestDetectWalksProcessTree(t *testing.T) {
	dir := t.TempDir()
	log := filepath.Join(dir, "ps.log")
	fakePS(t, dir, "4242", "/bin/zsh -i", log)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	kind, err := Detect()
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if kind != Zsh {
		t.Fatalf("Detect() = %v, want zsh", kind)
	}

	data, err := os.ReadFile(log)
	if err != nil {
		t.Fatalf("could not read ps log: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	want := []string{"ppid " + strconv.Itoa(os.Getpid()), "args 4242"}
	if len(lines) != len(want) {
		t.Fatalf("ps calls = %v, want %v", lines, want)
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Fatalf("ps call %d = %q, want %q", i, lines[i], want[i])
		}
	}
}

// TestSpawnBashExportsEnvironmentAndPropagatesExitCode checks the environment
// handed to the child shell, the rcfile argument and the exit code.
func TestSpawnBashExportsEnvironmentAndPropagatesExitCode(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "env.txt")
	t.Setenv("FAKE_BASH_OUT", out)

	body := `#!/bin/sh
{
  for name in KUBECONFIG KON_ACTIVE KON_DEPTH KON_KUBECONFIG KON_SESSION KON_STATE KON_PROMPT_DISABLE KON_ZSH_USE_RPS1 KON_FISH_USE_RPROMPT KON_XONSH_USE_RIGHT_PROMPT KON_SHELL; do
    eval "value=\${$name-<unset>}"
    printf '%s=%s\n' "$name" "$value"
  done
  printf 'ARGS=%s\n' "$*"
} > "$FAKE_BASH_OUT"
exit 7
`
	writeFake(t, dir, "bash", body)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	oldExecutablePath := executablePath
	executablePath = func() (string, error) { return "/usr/local/bin/kon", nil }
	t.Cleanup(func() { executablePath = oldExecutablePath })

	s, err := settings.Default()
	if err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(dir, "kubeconfig")
	sessionPath := filepath.Join(dir, "session.json")

	info := &spawnInfo{
		settings: s,
		envVars:  buildEnvVars(s, Bash, configPath, sessionPath, 2),
		prompt:   generatePS1(s, 2, Bash, "/usr/local/bin/kon"),
		kind:     Bash,
	}

	code, err := spawnBash(info)
	if err != nil {
		t.Fatalf("spawnBash() error = %v", err)
	}
	if code != 7 {
		t.Fatalf("spawnBash() code = %d, want 7", code)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("could not read env dump: %v", err)
	}
	t.Logf("fake bash env dump:\n%s", data)

	values := map[string]string{}
	var args string
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		name, value, found := strings.Cut(line, "=")
		if !found {
			t.Fatalf("malformed env dump line %q", line)
		}
		if name == "ARGS" {
			args = value
			continue
		}
		values[name] = value
	}

	want := map[string]string{
		"KUBECONFIG":                 configPath,
		"KON_ACTIVE":                 "1",
		"KON_DEPTH":                  "2",
		"KON_KUBECONFIG":             configPath,
		"KON_SESSION":                sessionPath,
		"KON_PROMPT_DISABLE":         "0",
		"KON_ZSH_USE_RPS1":           "0",
		"KON_FISH_USE_RPROMPT":       "0",
		"KON_XONSH_USE_RIGHT_PROMPT": "0",
		"KON_SHELL":                  "bash",
	}
	for name, wantValue := range want {
		if got := values[name]; got != wantValue {
			t.Fatalf("%s = %q, want %q", name, got, wantValue)
		}
	}
	if values["KON_STATE"] == "" || values["KON_STATE"] == "<unset>" {
		t.Fatalf("KON_STATE = %q, want a path", values["KON_STATE"])
	}
	if !strings.Contains(args, "--rcfile") {
		t.Fatalf("child args = %q, want an --rcfile argument", args)
	}
}

// TestSpawnBashExitHookGate checks that the bash exit hook runs only when
// start_ctx is set, same as upstream bash.rs does.
func TestSpawnBashExitHookGate(t *testing.T) {
	dir := t.TempDir()
	log := filepath.Join(dir, "calls.txt")
	body := `#!/bin/sh
printf '%s\n' "$*" >> "` + log + `"
`
	writeFake(t, dir, "bash", body)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	oldExecutablePath := executablePath
	executablePath = func() (string, error) { return "/usr/local/bin/kon", nil }
	t.Cleanup(func() { executablePath = oldExecutablePath })

	s, err := settings.Default()
	if err != nil {
		t.Fatal(err)
	}
	s.Hooks.StopCtx = "echo stop"

	info := &spawnInfo{
		settings: s,
		envVars:  buildEnvVars(s, Bash, filepath.Join(dir, "kubeconfig"), filepath.Join(dir, "session.json"), 1),
		prompt:   generatePS1(s, 1, Bash, "/usr/local/bin/kon"),
		kind:     Bash,
	}

	if _, err := spawnBash(info); err != nil {
		t.Fatalf("spawnBash() error = %v", err)
	}

	data, err := os.ReadFile(log)
	if err != nil {
		t.Fatalf("could not read call log: %v", err)
	}
	if calls := strings.Count(string(data), "\n"); calls != 1 {
		t.Fatalf("bash calls = %d, want 1 without start_ctx", calls)
	}

	s.Hooks.StartCtx = "echo start"
	if _, err := spawnBash(info); err != nil {
		t.Fatalf("spawnBash() error = %v", err)
	}

	data, err = os.ReadFile(log)
	if err != nil {
		t.Fatalf("could not read call log: %v", err)
	}
	if calls := strings.Count(string(data), "\n"); calls != 3 {
		t.Fatalf("bash calls = %d, want 3 with start_ctx", calls)
	}
}

// TestBashRCContent checks that the rc file carries the startup files and the
// prompt of kubie.
func TestBashRCContent(t *testing.T) {
	s, err := settings.Default()
	if err != nil {
		t.Fatal(err)
	}
	info := &spawnInfo{
		settings: s,
		prompt:   "PROMPT",
	}

	content := bashRCContent(info)
	for _, want := range []string{`source "/etc/profile"`, `source "$HOME/.bash_profile"`, `trap '__kubie_cmd_pre_exec__' DEBUG`, `KON_PROMPT='PROMPT'`} {
		if !strings.Contains(content, want) {
			t.Fatalf("bash rc content misses %q", want)
		}
	}
}
