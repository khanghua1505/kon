// Package state stores the kubie state that is shared between all shells.
//
// It mirrors /tmp/kubie-src/src/state.rs: the state file is referenced by
// $KON_STATE, guarded by an exclusive file lock and rewritten atomically.
package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// EnvState is the environment variable holding the path of the state file.
const EnvState = "KON_STATE"

// lockFileName matches the lock file name used by the Rust implementation.
const lockFileName = ".state.json.lock"

// State is the kubie state shared between all shells.
type State struct {
	// LastContext is the most recently entered context across all sessions.
	LastContext *string `json:"last_context"`

	// NamespaceHistory maps a context to the last namespace it was used in, so
	// the namespace can be restored when the context is entered again.
	NamespaceHistory map[string]*string `json:"namespace_history"`
}

// Path returns the path of the state file referenced by $KON_STATE.
func Path() (string, error) {
	path, ok := os.LookupEnv(EnvState)
	if !ok || path == "" {
		return "", errors.New("KON_STATE env variable missing")
	}
	return path, nil
}

// Load reads the state file while holding the exclusive lock.
func Load() (State, error) {
	var loaded State
	err := withLock(func() error {
		state, err := readState()
		if err != nil {
			return err
		}
		loaded = state
		return nil
	})
	if err != nil {
		return State{}, err
	}
	return loaded, nil
}

// Modify applies fn to the state while holding the exclusive lock and writes
// the result back to the state file. The state is not written when fn fails.
func Modify(fn func(*State) error) error {
	return withLock(func() error {
		current, err := readState()
		if err != nil {
			return err
		}
		if err := fn(&current); err != nil {
			return err
		}
		return writeState(&current)
	})
}

// withLock runs fn while holding an exclusive lock on the lock file next to
// the state file.
func withLock(fn func() error) error {
	statePath, err := Path()
	if err != nil {
		return err
	}

	dir := filepath.Dir(statePath)
	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("could not access state directory %s: %w", dir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("state directory %s is not a directory", dir)
	}

	lockPath := filepath.Join(dir, lockFileName)
	lock, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("could not open lock file %s: %w", lockPath, err)
	}
	defer func() { _ = lock.Close() }()

	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("could not lock file %s: %w", lockPath, err)
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN) //nolint:errcheck // unlocking a closed lock is not actionable

	return fn()
}

// readState reads and parses the state file. A missing state file yields the
// default state, a malformed one an error.
func readState() (State, error) {
	statePath, err := Path()
	if err != nil {
		return State{}, err
	}

	current := State{NamespaceHistory: map[string]*string{}}
	data, err := os.ReadFile(statePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return current, nil
		}
		return State{}, fmt.Errorf("failed to read state from '%s': %w", statePath, err)
	}

	if err := json.Unmarshal(data, &current); err != nil {
		return State{}, fmt.Errorf("failed to read state from '%s': %w", statePath, err)
	}
	if current.NamespaceHistory == nil {
		current.NamespaceHistory = map[string]*string{}
	}
	return current, nil
}

// writeState writes the state file atomically: a temporary file in the same
// directory is renamed over the state file.
func writeState(current *State) error {
	statePath, err := Path()
	if err != nil {
		return err
	}
	if current.NamespaceHistory == nil {
		current.NamespaceHistory = map[string]*string{}
	}

	data, err := json.Marshal(current)
	if err != nil {
		return fmt.Errorf("failed to write state to '%s': %w", statePath, err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(statePath), ".state.json.tmp")
	if err != nil {
		return fmt.Errorf("failed to write state to '%s': %w", statePath, err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }() // no-op once the rename succeeded

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("failed to write state to '%s': %w", statePath, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to write state to '%s': %w", statePath, err)
	}
	if err := os.Rename(tmpPath, statePath); err != nil {
		return fmt.Errorf("failed to write state to '%s': %w", statePath, err)
	}
	return nil
}
