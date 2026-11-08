package session_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/khanghua1505/kon/internal/session"
	"github.com/khanghua1505/kon/internal/vars"
)

func namespace(value string) *string {
	return &value
}

func unsetEnv(t *testing.T, key string) {
	t.Helper()
	t.Setenv(key, "")
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("could not unset %s: %v", key, err)
	}
}

func TestGetLastContext(t *testing.T) {
	tests := []struct {
		name    string
		history []session.HistoryEntry
		want    string
		found   bool
	}{
		{name: "empty history"},
		{name: "single entry", history: []session.HistoryEntry{{Context: "a"}}},
		{
			name:    "previous context",
			history: []session.HistoryEntry{{Context: "a"}, {Context: "b"}},
			want:    "a",
			found:   true,
		},
		{
			name:    "skips consecutive entries of the same context",
			history: []session.HistoryEntry{{Context: "a"}, {Context: "b"}, {Context: "b"}},
			want:    "a",
			found:   true,
		},
		{
			name:    "last distinct context",
			history: []session.HistoryEntry{{Context: "a"}, {Context: "b"}, {Context: "a"}},
			want:    "b",
			found:   true,
		},
		{
			name:    "only one context",
			history: []session.HistoryEntry{{Context: "a"}, {Context: "a"}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			current := session.Session{History: tc.history}

			got := current.GetLastContext()
			if !tc.found {
				if got != nil {
					t.Fatalf("GetLastContext() = %+v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatal("GetLastContext() = nil, want entry")
			}
			if got.Context != tc.want {
				t.Fatalf("GetLastContext().Context = %q, want %q", got.Context, tc.want)
			}
		})
	}
}

func TestGetLastNamespace(t *testing.T) {
	tests := []struct {
		name    string
		history []session.HistoryEntry
		want    string
		found   bool
	}{
		{name: "empty history"},
		{name: "single entry", history: []session.HistoryEntry{{Context: "a", Namespace: namespace("x")}}},
		{
			name: "same namespace",
			history: []session.HistoryEntry{
				{Context: "a", Namespace: namespace("x")},
				{Context: "a", Namespace: namespace("x")},
			},
		},
		{
			name: "different namespace",
			history: []session.HistoryEntry{
				{Context: "a", Namespace: namespace("x")},
				{Context: "a", Namespace: namespace("y")},
			},
			want:  "x",
			found: true,
		},
		{
			name: "previous entry without namespace",
			history: []session.HistoryEntry{
				{Context: "a"},
				{Context: "a", Namespace: namespace("x")},
			},
		},
		{
			name: "namespace chain",
			history: []session.HistoryEntry{
				{Context: "a", Namespace: namespace("x")},
				{Context: "a", Namespace: namespace("y")},
				{Context: "a", Namespace: namespace("z")},
			},
			want:  "y",
			found: true,
		},
		{
			name: "context switch ends the chain",
			history: []session.HistoryEntry{
				{Context: "a", Namespace: namespace("x")},
				{Context: "b", Namespace: namespace("y")},
			},
		},
		{
			name: "context switch after a namespace chain",
			history: []session.HistoryEntry{
				{Context: "a", Namespace: namespace("x")},
				{Context: "a", Namespace: namespace("y")},
				{Context: "b", Namespace: namespace("z")},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			current := session.Session{History: tc.history}

			got := current.GetLastNamespace()
			if !tc.found {
				if got != nil {
					t.Fatalf("GetLastNamespace() = %q, want nil", *got)
				}
				return
			}
			if got == nil {
				t.Fatal("GetLastNamespace() = nil, want namespace")
			}
			if *got != tc.want {
				t.Fatalf("GetLastNamespace() = %q, want %q", *got, tc.want)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	t.Run("session variable unset", func(t *testing.T) {
		unsetEnv(t, vars.EnvSession)

		loaded, err := session.Load()
		if err != nil {
			t.Fatalf("Load() error = %v, want nil", err)
		}
		if len(loaded.History) != 0 {
			t.Fatalf("Load() history = %+v, want empty", loaded.History)
		}
	})

	t.Run("session file missing", func(t *testing.T) {
		t.Setenv(vars.EnvSession, filepath.Join(t.TempDir(), "session.json"))

		loaded, err := session.Load()
		if err != nil {
			t.Fatalf("Load() error = %v, want nil", err)
		}
		if len(loaded.History) != 0 {
			t.Fatalf("Load() history = %+v, want empty", loaded.History)
		}
	})

	t.Run("session file with history", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "session.json")
		content := `{"history":[{"context":"ctx","namespace":"team"},{"context":"other","namespace":null}]}`
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("could not write session file: %v", err)
		}
		t.Setenv(vars.EnvSession, path)

		loaded, err := session.Load()
		if err != nil {
			t.Fatalf("Load() error = %v, want nil", err)
		}
		if len(loaded.History) != 2 {
			t.Fatalf("Load() history = %+v, want 2 entries", loaded.History)
		}
		if loaded.History[0].Context != "ctx" {
			t.Fatalf("Load() history[0].Context = %q, want %q", loaded.History[0].Context, "ctx")
		}
		if loaded.History[0].Namespace == nil || *loaded.History[0].Namespace != "team" {
			t.Fatalf("Load() history[0].Namespace = %v, want %q", loaded.History[0].Namespace, "team")
		}
		if loaded.History[1].Namespace != nil {
			t.Fatalf("Load() history[1].Namespace = %v, want nil", *loaded.History[1].Namespace)
		}
	})

	t.Run("malformed session file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "session.json")
		if err := os.WriteFile(path, []byte("{\"history\":"), 0o600); err != nil {
			t.Fatalf("could not write session file: %v", err)
		}
		t.Setenv(vars.EnvSession, path)

		loaded, err := session.Load()
		if err == nil {
			t.Fatal("Load() = nil error, want error for malformed session file")
		}
		if len(loaded.History) != 0 {
			t.Fatalf("Load() history = %+v, want empty", loaded.History)
		}
	})
}

func TestSave(t *testing.T) {
	t.Run("session variable unset", func(t *testing.T) {
		unsetEnv(t, vars.EnvSession)

		current := session.Session{}
		if err := current.Save(); err == nil {
			t.Fatal("Save() = nil, want error when KON_SESSION is missing")
		}
	})

	t.Run("writes the upstream json shape", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "session.json")
		t.Setenv(vars.EnvSession, path)

		current := session.Session{}
		current.AddHistoryEntry("ctx", nil)
		if err := current.Save(); err != nil {
			t.Fatalf("Save() error = %v, want nil", err)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("could not read session file: %v", err)
		}
		want := `{"history":[{"context":"ctx","namespace":null}]}`
		if string(data) != want {
			t.Fatalf("session file = %s, want %s", data, want)
		}
	})

	t.Run("empty history is an empty json array", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "session.json")
		t.Setenv(vars.EnvSession, path)

		current := session.Session{}
		if err := current.Save(); err != nil {
			t.Fatalf("Save() error = %v, want nil", err)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("could not read session file: %v", err)
		}
		want := `{"history":[]}`
		if string(data) != want {
			t.Fatalf("session file = %s, want %s", data, want)
		}
	})

	t.Run("round trip", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "session.json")
		t.Setenv(vars.EnvSession, path)

		current := session.Session{}
		current.AddHistoryEntry("ctx", namespace("team"))
		current.AddHistoryEntry("ctx", nil)
		if err := current.Save(); err != nil {
			t.Fatalf("Save() error = %v, want nil", err)
		}

		loaded, err := session.Load()
		if err != nil {
			t.Fatalf("Load() error = %v, want nil", err)
		}
		if len(loaded.History) != 2 {
			t.Fatalf("Load() history = %+v, want 2 entries", loaded.History)
		}
		if loaded.History[0].Namespace == nil || *loaded.History[0].Namespace != "team" {
			t.Fatalf("Load() history[0].Namespace = %v, want %q", loaded.History[0].Namespace, "team")
		}
		if loaded.History[1].Namespace != nil {
			t.Fatalf("Load() history[1].Namespace = %v, want nil", *loaded.History[1].Namespace)
		}
	})
}

func TestAddHistoryEntry(t *testing.T) {
	current := session.Session{}
	current.AddHistoryEntry("first", namespace("team"))
	current.AddHistoryEntry("second", nil)

	if len(current.History) != 2 {
		t.Fatalf("history = %+v, want 2 entries", current.History)
	}
	if current.History[0].Context != "first" {
		t.Fatalf("history[0].Context = %q, want %q", current.History[0].Context, "first")
	}
	if current.History[1].Context != "second" {
		t.Fatalf("history[1].Context = %q, want %q", current.History[1].Context, "second")
	}
}

func TestRecordContextEntry(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(vars.EnvSession, filepath.Join(dir, "session.json"))
	t.Setenv("KON_STATE", filepath.Join(dir, "state.json"))

	current := session.Session{}
	if err := current.RecordContextEntry("ctx", namespace("team")); err != nil {
		t.Fatalf("RecordContextEntry() error = %v, want nil", err)
	}

	if len(current.History) != 1 {
		t.Fatalf("history = %+v, want 1 entry", current.History)
	}
	if current.History[0].Context != "ctx" {
		t.Fatalf("history[0].Context = %q, want %q", current.History[0].Context, "ctx")
	}

	data, err := os.ReadFile(filepath.Join(dir, "state.json"))
	if err != nil {
		t.Fatalf("could not read state file: %v", err)
	}
	want := `{"last_context":"ctx","namespace_history":{"ctx":"team"}}`
	if string(data) != want {
		t.Fatalf("state file = %s, want %s", data, want)
	}
}
