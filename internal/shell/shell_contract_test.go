package shell

import (
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/khanghua1505/kon/internal/settings"
	"github.com/khanghua1505/kon/internal/state"
	"github.com/khanghua1505/kon/internal/vars"
)

var updateGolden = flag.Bool("update", false, "rewrite the golden files in testdata")

const (
	goldenConfigPath  = "/tmp/kubie-test/kubeconfig"
	goldenSessionPath = "/tmp/kubie-test/session.json"
	goldenStatePath   = "/tmp/kubie-test/state.json"
	goldenExePath     = "/usr/local/bin/kon"
	goldenDepth       = 4
	goldenParentDepth = goldenDepth - 1
)

// goldenInfo builds the spawn info the golden tests work on.
func goldenInfo(t *testing.T, kind Kind) *spawnInfo {
	t.Helper()

	t.Setenv(state.EnvState, goldenStatePath)
	t.Setenv(vars.EnvDepth, strconv.Itoa(goldenParentDepth))

	s, err := settings.Default()
	if err != nil {
		t.Fatalf("could not load the default settings: %v", err)
	}

	depth := vars.Depth() + 1

	return &spawnInfo{
		settings: s,
		envVars:  buildEnvVars(s, kind, goldenConfigPath, goldenSessionPath, depth),
		prompt:   generatePS1(s, depth, kind, goldenExePath),
		kind:     kind,
	}
}

// renderEnv renders the child environment in insertion order.
func renderEnv(env EnvVars) string {
	var b strings.Builder
	for _, entry := range env.entries {
		b.WriteString(entry.name + "=" + entry.value + "\n")
	}

	return b.String()
}

// compareGolden compares content with the golden file in testdata.
func compareGolden(t *testing.T, name, got string) {
	t.Helper()

	path := filepath.Join("testdata", name)
	if *updateGolden {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatalf("could not create the testdata directory: %v", err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("could not write the golden file %s: %v", name, err)
		}
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("could not read the golden file %s: %v", name, err)
	}

	if got != string(want) {
		t.Errorf("golden %s mismatch\n--- got ---\n%s\n--- want ---\n%s", name, got, string(want))
	}
}

// captureNuCommand returns the nu command line kubie would run.
func captureNuCommand(t *testing.T, info *spawnInfo) string {
	t.Helper()

	dir := t.TempDir()
	argsFile := filepath.Join(dir, "nu-args.txt")
	writeFake(t, dir, "nu", "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$NU_ARGS_FILE\"\n")
	t.Setenv("NU_ARGS_FILE", argsFile)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	if _, err := spawnNu(info); err != nil {
		t.Fatalf("spawnNu() error = %v", err)
	}

	data, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("could not read the recorded nu arguments: %v", err)
	}

	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) != 2 || lines[0] != "-e" {
		t.Fatalf("nu arguments = %q, want the -e flag and the command", lines)
	}

	return lines[1]
}

// TestRCFileGoldens checks the startup script of every shell kubie spawns.
func TestRCFileGoldens(t *testing.T) {
	cases := []struct {
		kind   Kind
		golden string
		rc     func(t *testing.T, info *spawnInfo) string
	}{
		{
			kind:   Bash,
			golden: "bashrc.golden",
			rc: func(_ *testing.T, info *spawnInfo) string {
				return bashRCContent(info)
			},
		},
		{
			kind:   Zsh,
			golden: "zshrc.golden",
			rc: func(_ *testing.T, info *spawnInfo) string {
				return zshRCContent(info)
			},
		},
		{
			kind:   Fish,
			golden: "fishrc.golden",
			rc: func(_ *testing.T, info *spawnInfo) string {
				return fishRCContent(info)
			},
		},
		{
			kind:   Xonsh,
			golden: "xonshrc.golden",
			rc: func(_ *testing.T, info *spawnInfo) string {
				return xonshRCContent(info)
			},
		},
		{
			kind:   Nu,
			golden: "nurc.golden",
			rc:     captureNuCommand,
		},
	}

	for _, tc := range cases {
		t.Run(tc.kind.String(), func(t *testing.T) {
			info := goldenInfo(t, tc.kind)
			rc := tc.rc(t, info)

			if strings.Contains(rc, promptPlaceholder) {
				t.Errorf("%s startup script keeps the prompt placeholder", tc.kind)
			}

			env := renderEnv(info.envVars)
			for _, want := range []string{
				"KUBIE_KUBECONFIG=" + goldenConfigPath + "\n",
				"KUBIE_DEPTH=" + strconv.Itoa(goldenDepth) + "\n",
			} {
				if !strings.Contains(env, want) {
					t.Errorf("%s environment misses %q", tc.kind, want)
				}
			}

			compareGolden(t, tc.golden, "=== env ===\n"+env+"=== rc ===\n"+rc)
		})
	}
}

// TestBuildEnvVarsPerShell checks the environment kubie hands to every shell.
func TestBuildEnvVarsPerShell(t *testing.T) {
	t.Setenv(state.EnvState, goldenStatePath)
	t.Setenv(vars.EnvDepth, "3")

	s, err := settings.Default()
	if err != nil {
		t.Fatalf("could not load the default settings: %v", err)
	}

	// The child shell runs one level deeper than the current one.
	depth := vars.Depth() + 1

	for _, kind := range []Kind{Bash, Zsh, Fish, Nu, Xonsh} {
		t.Run(kind.String(), func(t *testing.T) {
			env := buildEnvVars(s, kind, goldenConfigPath, goldenSessionPath, depth)

			want := map[string]string{
				"KUBECONFIG":                   goldenConfigPath,
				"KUBIE_ACTIVE":                 "1",
				"KUBIE_DEPTH":                  strconv.Itoa(depth),
				"KUBIE_KUBECONFIG":             goldenConfigPath,
				"KUBIE_SESSION":                goldenSessionPath,
				"KUBIE_STATE":                  goldenStatePath,
				"KUBIE_PROMPT_DISABLE":         "0",
				"KUBIE_ZSH_USE_RPS1":           "0",
				"KUBIE_FISH_USE_RPROMPT":       "0",
				"KUBIE_XONSH_USE_RIGHT_PROMPT": "0",
				"KUBIE_SHELL":                  kind.String(),
			}

			if len(env.entries) != len(want) {
				t.Errorf("environment has %d entries, want %d: %v", len(env.entries), len(want), renderEnv(env))
			}

			got := map[string]string{}
			for _, entry := range env.entries {
				got[entry.name] = entry.value
			}

			for name, value := range want {
				if got[name] != value {
					t.Errorf("%s = %q, want %q", name, got[name], value)
				}
			}

			if env.entries[0].name != "KUBECONFIG" {
				t.Errorf("first environment entry = %q, want KUBECONFIG", env.entries[0].name)
			}

			cmd := exec.Command("true")
			cmd.Env = []string{}
			env.Apply(cmd)

			for name, value := range want {
				entry := name + "=" + value
				if !containsEnv(cmd.Env, entry) {
					t.Errorf("child environment misses %q", entry)
				}
			}
		})
	}
}

// containsEnv reports whether the environment holds the entry.
func containsEnv(env []string, entry string) bool {
	for _, candidate := range env {
		if candidate == entry {
			return true
		}
	}

	return false
}

// TestGeneratePS1PerShell checks the prompt of every shell kubie spawns.
func TestGeneratePS1PerShell(t *testing.T) {
	cases := []struct {
		kind Kind
		want string
	}{
		{
			kind: Bash,
			want: `[\[\e[31m\]$(/usr/local/bin/kon info ctx)\[\e[0m\]|\[\e[32m\]$(/usr/local/bin/kon info ns)\[\e[0m\]|\[\e[34m\]4\[\e[0m\]]`,
		},
		{
			kind: Zsh,
			want: `[%{\e[31m%}$(/usr/local/bin/kon info ctx)%{\e[0m%}|%{\e[32m%}$(/usr/local/bin/kon info ns)%{\e[0m%}|%{\e[34m%}4%{\e[0m%}]`,
		},
		{
			kind: Fish,
			want: `[\e[31m(/usr/local/bin/kon info ctx)\e[0m|\e[32m(/usr/local/bin/kon info ns)\e[0m|\e[34m4\e[0m]`,
		},
		{
			kind: Xonsh,
			want: `[\033[31m$(/usr/local/bin/kon info ctx)\033[0m|\033[32m$(/usr/local/bin/kon info ns)\033[0m|\033[34m4\033[0m]`,
		},
		{
			kind: Nu,
			want: `[\e[31m$(/usr/local/bin/kon info ctx)\e[0m|\e[32m$(/usr/local/bin/kon info ns)\e[0m|\e[34m4\e[0m]`,
		},
	}

	s := &settings.Settings{Prompt: settings.Prompt{ShowDepth: true}}

	for _, tc := range cases {
		t.Run(tc.kind.String(), func(t *testing.T) {
			got := generatePS1(s, goldenDepth, tc.kind, goldenExePath)
			if got != tc.want {
				t.Errorf("prompt = %q, want %q", got, tc.want)
			}
		})
	}

	t.Run("depth hidden on the first level", func(t *testing.T) {
		got := generatePS1(s, 1, Bash, goldenExePath)
		if strings.Contains(got, "34") {
			t.Errorf("prompt = %q, want no depth", got)
		}
	})

	t.Run("depth hidden by the setting", func(t *testing.T) {
		without := &settings.Settings{Prompt: settings.Prompt{ShowDepth: false}}
		got := generatePS1(without, goldenDepth, Bash, goldenExePath)
		if strings.Contains(got, "34") {
			t.Errorf("prompt = %q, want no depth", got)
		}
	})

	t.Run("disabled prompt drops the prompt from the startup script", func(t *testing.T) {
		disabled := &settings.Settings{Prompt: settings.Prompt{Disable: true}}
		info := &spawnInfo{settings: disabled, prompt: "KUBIE_PROMPT"}

		for name, content := range map[string]string{
			"bash":  bashRCContent(info),
			"zsh":   zshRCContent(info),
			"xonsh": xonshRCContent(info),
		} {
			if strings.Contains(content, "KUBIE_PROMPT") {
				t.Errorf("%s startup script keeps the prompt", name)
			}
		}
	})
}
