package settings_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/khanghua1505/kon/internal/settings"
)

func TestPathPrefersKonYaml(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, ".kube")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	t.Run("neither exists defaults to kon.yaml", func(t *testing.T) {
		path, err := settings.Path()
		if err != nil {
			t.Fatalf("Path: %v", err)
		}
		if want := filepath.Join(dir, "kon.yaml"); path != want {
			t.Fatalf("Path() = %q, want %q", path, want)
		}
	})

	t.Run("only kubie.yaml", func(t *testing.T) {
		if err := os.WriteFile(filepath.Join(dir, "kubie.yaml"), []byte("shell: bash\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		path, err := settings.Path()
		if err != nil {
			t.Fatalf("Path: %v", err)
		}
		if want := filepath.Join(dir, "kubie.yaml"); path != want {
			t.Fatalf("Path() = %q, want %q", path, want)
		}
	})

	t.Run("kubie.yaml still wins when both exist", func(t *testing.T) {
		if err := os.WriteFile(filepath.Join(dir, "kon.yaml"), []byte("shell: zsh\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		path, err := settings.Path()
		if err != nil {
			t.Fatalf("Path: %v", err)
		}
		if want := filepath.Join(dir, "kubie.yaml"); path != want {
			t.Fatalf("Path() = %q, want %q", path, want)
		}
	})

	t.Run("kon.yaml wins when kubie.yaml absent", func(t *testing.T) {
		if err := os.Remove(filepath.Join(dir, "kubie.yaml")); err != nil {
			t.Fatal(err)
		}
		path, err := settings.Path()
		if err != nil {
			t.Fatalf("Path: %v", err)
		}
		if want := filepath.Join(dir, "kon.yaml"); path != want {
			t.Fatalf("Path() = %q, want %q", path, want)
		}
		s, err := settings.Load()
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if s.Shell == nil || *s.Shell != "zsh" {
			t.Fatalf("Shell = %v, want zsh from kon.yaml", s.Shell)
		}
	})
}
