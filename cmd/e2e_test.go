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

func buildBin(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "kubie")
	if b, err := exec.Command("go", "build", "-o", bin, "..").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, b)
	}
	return bin
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
	if err := os.WriteFile(filepath.Join(home, ".kube", "kubie.yaml"), config, 0o600); err != nil {
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
	home := t.TempDir()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmdE := exec.CommandContext(ctx, bin, "ctx", "-f", "testdata/e2e/kubeconfig/a.yaml", "ctx-a")
	cmdE.Env = append(os.Environ(), "HOME="+home, "PATH="+filepath.Dir(bin)+":"+os.Getenv("PATH"), "SHELL=/bin/sh", "KUBIE_PROMPT_DISABLE=1")
	cmdE.Stdin = strings.NewReader("kubie info ctx\nkubie info ns\nkubie export\nexit 0\n")
	out, _ := cmdE.CombinedOutput()
	s := string(out)
	if !strings.Contains(s, "ctx-a") {
		t.Fatalf("no ctx-a:\n%s", s)
	}
	if !strings.Contains(s, "default") {
		t.Fatalf("no default ns:\n%s", s)
	}
}
