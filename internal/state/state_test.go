package state_test

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"testing"

	"github.com/khanghua1505/kubie/internal/state"
)

func pointer(value string) *string {
	return &value
}

func unsetEnv(t *testing.T, key string) {
	t.Helper()
	t.Setenv(key, "")
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("could not unset %s: %v", key, err)
	}
}

func TestModifyWritesTheUpstreamJsonShape(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	t.Setenv("KUBIE_STATE", statePath)

	err := state.Modify(func(current *state.State) error {
		current.LastContext = pointer("ctx")
		current.NamespaceHistory["ctx"] = pointer("ns")
		return nil
	})
	if err != nil {
		t.Fatalf("Modify() error = %v, want nil", err)
	}

	data, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("could not read state file: %v", err)
	}
	want := `{"last_context":"ctx","namespace_history":{"ctx":"ns"}}`
	if string(data) != want {
		t.Fatalf("state file = %s, want %s", data, want)
	}
}

func TestModifySeesThePreviousState(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	t.Setenv("KUBIE_STATE", statePath)

	err := state.Modify(func(current *state.State) error {
		current.LastContext = pointer("first")
		return nil
	})
	if err != nil {
		t.Fatalf("Modify() error = %v, want nil", err)
	}

	err = state.Modify(func(current *state.State) error {
		if current.LastContext == nil || *current.LastContext != "first" {
			t.Fatalf("Modify() saw last context %v, want %q", current.LastContext, "first")
		}
		current.LastContext = pointer("second")
		current.NamespaceHistory["first"] = pointer("ns")
		return nil
	})
	if err != nil {
		t.Fatalf("Modify() error = %v, want nil", err)
	}

	loaded, err := state.Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if loaded.LastContext == nil || *loaded.LastContext != "second" {
		t.Fatalf("Load() last context = %v, want %q", loaded.LastContext, "second")
	}
	if len(loaded.NamespaceHistory) != 1 {
		t.Fatalf("Load() namespace history = %+v, want 1 entry", loaded.NamespaceHistory)
	}
}

func TestModifyKeepsTheFileWhenTheCallbackFails(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	t.Setenv("KUBIE_STATE", statePath)

	err := state.Modify(func(current *state.State) error {
		current.LastContext = pointer("ctx")
		return os.ErrInvalid
	})
	if err == nil {
		t.Fatal("Modify() = nil, want error from the callback")
	}

	if _, err := os.Stat(statePath); !os.IsNotExist(err) {
		t.Fatalf("state file exists after a failed Modify(), stat error = %v", err)
	}
}

func TestLoadMissingFile(t *testing.T) {
	t.Setenv("KUBIE_STATE", filepath.Join(t.TempDir(), "state.json"))

	loaded, err := state.Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if loaded.LastContext != nil {
		t.Fatalf("Load() last context = %v, want nil", *loaded.LastContext)
	}
	if len(loaded.NamespaceHistory) != 0 {
		t.Fatalf("Load() namespace history = %+v, want empty", loaded.NamespaceHistory)
	}
}

func TestLoadMalformedFile(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(statePath, []byte(`{"last_context":`), 0o600); err != nil {
		t.Fatalf("could not write state file: %v", err)
	}
	t.Setenv("KUBIE_STATE", statePath)

	if _, err := state.Load(); err == nil {
		t.Fatal("Load() = nil, want error for a malformed state file")
	}
}

func TestMissingStateVariableErrorsWithoutPanic(t *testing.T) {
	unsetEnv(t, "KUBIE_STATE")

	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("Modify() panicked: %v", recovered)
		}
	}()

	err := state.Modify(func(current *state.State) error { return nil })
	if err == nil {
		t.Fatal("Modify() = nil, want error when KUBIE_STATE is missing")
	}
}

func TestMissingStateDirectoryErrorsWithoutPanic(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "missing", "state.json")
	t.Setenv("KUBIE_STATE", statePath)

	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("Modify() panicked: %v", recovered)
		}
	}()

	err := state.Modify(func(current *state.State) error { return nil })
	if err == nil {
		t.Fatal("Modify() = nil, want error when the state directory is missing")
	}

	if _, err := state.Load(); err == nil {
		t.Fatal("Load() = nil, want error when the state directory is missing")
	}
}

func TestModifySerializesConcurrentWriters(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	t.Setenv("KUBIE_STATE", statePath)

	const goroutines = 2
	const writesPerGoroutine = 25

	var mutex sync.Mutex
	observed := make([]int, 0, goroutines*writesPerGoroutine)
	failures := make(chan error, goroutines)

	var wait sync.WaitGroup
	for range goroutines {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for range writesPerGoroutine {
				err := state.Modify(func(current *state.State) error {
					count := 0
					if stored, ok := current.NamespaceHistory["counter"]; ok && stored != nil {
						parsed, err := strconv.Atoi(*stored)
						if err != nil {
							return err
						}
						count = parsed
					}
					next := strconv.Itoa(count + 1)
					current.NamespaceHistory["counter"] = &next

					mutex.Lock()
					observed = append(observed, count+1)
					mutex.Unlock()
					return nil
				})
				if err != nil {
					failures <- err
					return
				}
			}
		}()
	}
	wait.Wait()
	close(failures)

	for err := range failures {
		t.Fatalf("Modify() error = %v, want nil", err)
	}

	sort.Ints(observed)
	for i, value := range observed {
		if value != i+1 {
			t.Fatalf("observed writes = %v, want the serialized sequence 1..%d", observed, goroutines*writesPerGoroutine)
		}
	}
	t.Logf("serialized write sequence across %d goroutines: %v", goroutines, observed)

	loaded, err := state.Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	stored, ok := loaded.NamespaceHistory["counter"]
	if !ok || stored == nil {
		t.Fatalf("Load() namespace history = %+v, want a counter entry", loaded.NamespaceHistory)
	}
	if want := strconv.Itoa(goroutines * writesPerGoroutine); *stored != want {
		t.Fatalf("Load() counter = %q, want %q", *stored, want)
	}
}
