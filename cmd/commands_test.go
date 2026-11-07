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
	if err := os.WriteFile(filepath.Join(home, ".kube", "kubie.yaml"), []byte(content), 0o600); err != nil {
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
