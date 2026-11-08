package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeSettings(t *testing.T, includeGlob string) {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := os.MkdirAll(filepath.Join(home, ".kube"), 0o755); err != nil {
		t.Fatal(err)
	}

	content := fmt.Sprintf("configs:\n  include:\n    - %q\n", includeGlob)
	if err := os.WriteFile(filepath.Join(home, ".kube", "kon.yaml"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeSettingsWithBehavior(t *testing.T, includeGlob, printContextInExec string) {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := os.MkdirAll(filepath.Join(home, ".kube"), 0o755); err != nil {
		t.Fatal(err)
	}

	content := fmt.Sprintf("configs:\n  include:\n    - %q\nbehavior:\n  print_context_in_exec: %s\n", includeGlob, printContextInExec)
	if err := os.WriteFile(filepath.Join(home, ".kube", "kon.yaml"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func runRoot(t *testing.T, args ...string) (string, error) {
	t.Helper()

	root := NewRootCmd()
	RegisterCommands(root)

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(args)

	err := root.Execute()

	return out.String(), err
}

func TestLintMatchesGolden(t *testing.T) {
	writeSettings(t, "testdata/lint/kubeconfig/*.yaml")

	out, err := runRoot(t, "lint")
	if err != nil {
		t.Fatalf("lint failed: %v", err)
	}

	golden, err := os.ReadFile("testdata/lint/golden.txt")
	if err != nil {
		t.Fatal(err)
	}

	if out != string(golden) {
		t.Fatalf("lint output mismatch\n--- got ---\n%s--- want ---\n%s", out, golden)
	}
}

func TestExportMatchesGolden(t *testing.T) {
	writeSettings(t, "testdata/export/kubeconfig/*.yaml")

	out, err := runRoot(t, "export", "ctx-a", "my-ns")
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}

	path := strings.TrimSpace(out)
	if path == "" {
		t.Fatal("export printed no path")
	}

	exported, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("exported kubeconfig is missing: %v", err)
	}

	golden, err := os.ReadFile("testdata/export/golden.yaml")
	if err != nil {
		t.Fatal(err)
	}

	if string(exported) != string(golden) {
		t.Fatalf("exported kubeconfig mismatch\n--- got ---\n%s--- want ---\n%s", exported, golden)
	}
}

func TestExportWithoutMatchingContext(t *testing.T) {
	writeSettings(t, "testdata/export/kubeconfig/*.yaml")

	_, err := runRoot(t, "export", "missing", "my-ns")
	if err == nil {
		t.Fatal("export of an unknown context should fail")
	}

	if err.Error() != "No context matching missing" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecForwardsExitCode(t *testing.T) {
	writeSettings(t, "testdata/export/kubeconfig/*.yaml")

	_, err := runRoot(t, "exec", "ctx-a", "my-ns", "--", "sh", "-c", "exit 7")
	if err == nil {
		t.Fatal("exec should report the failing command")
	}

	var exitErr *ExecExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected an ExitError, got %v", err)
	}

	if exitErr.ExitCode() != 7 {
		t.Fatalf("expected exit code 7, got %d", exitErr.ExitCode())
	}
}

func TestExecSuccess(t *testing.T) {
	writeSettings(t, "testdata/export/kubeconfig/*.yaml")

	out, err := runRoot(t, "exec", "ctx-a", "my-ns", "--", "sh", "-c", "echo hello")
	if err != nil {
		t.Fatalf("exec failed: %v", err)
	}

	if strings.TrimSpace(out) != "hello" {
		t.Fatalf("unexpected command output: %q", out)
	}
}

func TestExecSetsKubeEnvironmentAndContextHeader(t *testing.T) {
	writeSettingsWithBehavior(t, "testdata/export/kubeconfig/*.yaml", "always")
	t.Setenv("KON_DEPTH", "")

	out, err := runRoot(t, "exec", "ctx-a", "my-ns", "--", "sh", "-c", "printf '%s|%s|%s\\n' \"$KUBECONFIG\" \"$KON_ACTIVE\" \"$KON_DEPTH\"")
	if err != nil {
		t.Fatalf("exec failed: %v", err)
	}

	header := "CONTEXT => ctx-a\n"
	separator := strings.Repeat("-", 20)

	headerIndex := strings.Index(out, header)
	if headerIndex < 0 {
		t.Fatalf("missing context header in %q", out)
	}

	separatorIndex := strings.Index(out, separator)
	if separatorIndex < 0 {
		t.Fatalf("missing context separator in %q", out)
	}

	if separatorIndex < headerIndex {
		t.Fatalf("separator printed before the header: %q", out)
	}

	var kubeconfigPath, active, depth string
	for _, line := range strings.Split(out, "\n") {
		parts := strings.SplitN(line, "|", 3)
		if len(parts) != 3 {
			continue
		}

		kubeconfigPath, active, depth = parts[0], parts[1], parts[2]
	}

	if active != "1" {
		t.Fatalf("expected KON_ACTIVE=1, got %q", active)
	}

	if depth != "1" {
		t.Fatalf("expected KON_DEPTH=1, got %q", depth)
	}

	base := filepath.Base(kubeconfigPath)
	if !strings.HasPrefix(base, "kon-config") || !strings.HasSuffix(base, ".yaml") {
		t.Fatalf("unexpected KUBECONFIG path %q", kubeconfigPath)
	}
}
