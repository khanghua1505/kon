package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsurePluginScripts(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "kon")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	paths, err := ensurePluginScripts(bin)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 2 {
		t.Fatalf("got %d paths, want 2", len(paths))
	}

	resolvedDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		filepath.Join(resolvedDir, "kubectl-ctx"): "#!/bin/sh\nexec \"$(dirname \"$0\")/kon\" ctx \"$@\"\n",
		filepath.Join(resolvedDir, "kubectl-ns"):  "#!/bin/sh\nexec \"$(dirname \"$0\")/kon\" ns \"$@\"\n",
	}
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if want[path] != string(content) {
			t.Fatalf("script %s = %q, want %q", path, content, want[path])
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode()&0o111 == 0 {
			t.Fatalf("script %s is not executable", path)
		}
	}
}
