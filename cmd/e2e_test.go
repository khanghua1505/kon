package cmd

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// konEnv drops kon and kubie session variables from base so spawned
// commands start outside any live kon shell.
func konEnv(base []string) []string {
	clean := make([]string, 0, len(base))
	for _, kv := range base {
		name := kv
		if i := strings.Index(kv, "="); i >= 0 {
			name = kv[:i]
		}
		if strings.HasPrefix(name, "KON_") || strings.HasPrefix(name, "KUBIE_") || name == "KUBECONFIG" {
			continue
		}
		clean = append(clean, kv)
	}
	return clean
}

func buildBin(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "kon")
	if b, err := exec.Command("go", "build", "-o", bin, "./kon").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, b)
	}
	return bin
}

func prepareE2EHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".kube"), 0o700); err != nil {
		t.Fatalf("create config directory: %v", err)
	}
	fixture, err := os.ReadFile("testdata/e2e/kubeconfig/a.yaml")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, ".kube", "config"), fixture, 0o600); err != nil {
		t.Fatalf("write kubeconfig: %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, ".kube", "kon.yaml"), []byte("shell: bash\n"), 0o600); err != nil {
		t.Fatalf("write settings: %v", err)
	}
	t.Setenv("HOME", home)
	t.Setenv("KON_STATE", filepath.Join(home, "state.json"))
	t.Setenv("KON_ACTIVE", "")
	t.Setenv("KON_DEPTH", "0")
	t.Setenv("KON_SESSION", "")
	t.Setenv("KON_KUBECONFIG", "")
	return home
}

func TestE2ELintGolden(t *testing.T) {
	bin := buildBin(t)
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".kube"), 0o700); err != nil {
		t.Fatalf("create config directory: %v", err)
	}
	fixture, err := filepath.Abs("testdata/e2e/kubeconfig/a.yaml")
	if err != nil {
		t.Fatalf("resolve fixture: %v", err)
	}
	config := []byte("configs:\n  include:\n    - " + fixture + "\n")
	if err := os.WriteFile(filepath.Join(home, ".kube", "kon.yaml"), config, 0o600); err != nil {
		t.Fatalf("write settings: %v", err)
	}

	cmd := exec.Command(bin, "lint")
	cmd.Env = append(os.Environ(), "HOME="+home)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("lint: %v\n%s", err, out)
	}
}

func TestE2ECorruptYAML(t *testing.T) {
	bin := buildBin(t)
	out, err := exec.Command(bin, "ctx", "-f", "testdata/e2e/corrupt/corrupted.yaml", "ctx-a").CombinedOutput()
	if err == nil {
		t.Fatalf("expected error, got: %s", out)
	}
	if strings.Contains(strings.ToLower(string(out)), "panic") {
		t.Fatalf("panic: %s", out)
	}
}

func TestE2EExportCtx(t *testing.T) {
	bin := buildBin(t)
	home := prepareE2EHome(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmdE := exec.CommandContext(ctx, bin, "ctx", "-f", "testdata/e2e/kubeconfig/a.yaml", "ctx-a")
	cmdE.Env = append(konEnv(os.Environ()),
		"HOME="+home,
		"PATH="+filepath.Dir(bin)+":"+os.Getenv("PATH"),
		"SHELL=/bin/bash",
		"KON_PROMPT_DISABLE=1",
		"KON_DEPTH=0",
		"KON_STATE="+os.Getenv("KON_STATE"),
	)
	cmdE.Stdin = strings.NewReader(strings.Join([]string{
		"kon info ctx",
		"kon info ns",
		"printf 'DEPTH='",
		"kon info depth",
		"export_path=$(kon export ctx-a default) || exit 1",
		`printf 'EXPORT=%s\n' "$export_path"`,
		`test -f "$export_path" || exit 1`,
		"exit 0",
	}, "\n") + "\n")
	out, err := cmdE.CombinedOutput()
	if err != nil {
		t.Fatalf("ctx shell: %v\n%s", err, out)
	}
	s := string(out)
	if !strings.Contains(s, "ctx-a") {
		t.Fatalf("no ctx-a:\n%s", s)
	}
	if !strings.Contains(s, "default") {
		t.Fatalf("no default ns:\n%s", s)
	}
	if !strings.Contains(s, "DEPTH=1") {
		t.Fatalf("no depth 1:\n%s", s)
	}
	const exportPrefix = "EXPORT="
	exportAt := strings.Index(s, exportPrefix)
	if exportAt < 0 {
		t.Fatalf("no exported path:\n%s", s)
	}
	exportPath := strings.TrimSpace(s[exportAt+len(exportPrefix):])
	if exportPath == "" {
		t.Fatalf("empty exported path:\n%s", s)
	}
	if _, err := os.Stat(exportPath); err != nil {
		t.Fatalf("exported path %q: %v", exportPath, err)
	}
}

func TestE2ECtxNamespaceRewrite(t *testing.T) {
	bin := buildBin(t)
	prepareE2EHome(t)

	fakeDir := t.TempDir()
	fakeKubectl := filepath.Join(fakeDir, "kubectl")
	fakeScript := "#!/bin/sh\ncat <<'EOF'\nNAME\n----\ndefault\nother-ns\nEOF\n"
	if err := os.WriteFile(fakeKubectl, []byte(fakeScript), 0o755); err != nil {
		t.Fatalf("write fake kubectl: %v", err)
	}
	t.Setenv("PATH", strings.Join([]string{fakeDir, filepath.Dir(bin), os.Getenv("PATH")}, string(os.PathListSeparator)))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "ctx", "-f", "testdata/e2e/kubeconfig/a.yaml", "ctx-a")
	cmd.Stdin = strings.NewReader(strings.Join([]string{
		"printf 'BEFORE='",
		"kon info ns || exit 1",
		"kon ns other-ns || exit 1",
		"printf 'AFTER='",
		"kon info ns || exit 1",
		"exit 0",
	}, "\n") + "\n")
	cmd.Env = append(konEnv(os.Environ()), "KON_STATE="+os.Getenv("KON_STATE"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("ctx namespace rewrite: %v\n%s", err, out)
	}
	s := string(out)
	if !strings.Contains(s, "BEFORE=default") {
		t.Fatalf("no initial namespace:\n%s", s)
	}
	if !strings.Contains(s, "AFTER=other-ns") {
		t.Fatalf("namespace not rewritten:\n%s", s)
	}
}

func TestE2ECtxBackSwitch(t *testing.T) {
	bin := buildBin(t)
	home := prepareE2EHome(t)

	sessionPath := filepath.Join(home, "session.json")
	sessionJSON := []byte(`{"history":[{"context":"ctx-a","namespace":null},{"context":"ctx-b","namespace":null}]}`)
	if err := os.WriteFile(sessionPath, sessionJSON, 0o600); err != nil {
		t.Fatalf("write session: %v", err)
	}
	activeConfig := filepath.Join(home, "active.yaml")
	fixture, err := os.ReadFile("testdata/e2e/kubeconfig/a.yaml")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if err := os.WriteFile(activeConfig, fixture, 0o600); err != nil {
		t.Fatalf("write active kubeconfig: %v", err)
	}
	t.Setenv("KON_ACTIVE", "1")
	t.Setenv("KON_DEPTH", "1")
	t.Setenv("KON_SESSION", sessionPath)
	t.Setenv("KON_KUBECONFIG", activeConfig)
	t.Setenv("KUBECONFIG", activeConfig)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if out, err := exec.CommandContext(ctx, bin, "ctx", "-").CombinedOutput(); err != nil {
		t.Fatalf("ctx back-switch: %v\n%s", err, out)
	}
	out, err := exec.CommandContext(ctx, bin, "info", "ctx").CombinedOutput()
	if err != nil {
		t.Fatalf("info ctx: %v\n%s", err, out)
	}
	if got := strings.TrimSpace(string(out)); got != "ctx-a" {
		t.Fatalf("context after back-switch = %q, want ctx-a", got)
	}
}
