// Package session tracks the context and namespace history of a single kubie
// shell.
//
// It mirrors /tmp/kubie-src/src/session.rs: the history lives in the JSON file
// referenced by $KUBIE_SESSION.
package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/khanghua1505/kon/internal/state"
	"github.com/khanghua1505/kon/internal/vars"
)

// HistoryEntry is a single visited context with its optional namespace.
type HistoryEntry struct {
	Context   string  `json:"context"`
	Namespace *string `json:"namespace"`
}

// Session holds the history of the contexts entered in one kubie shell.
type Session struct {
	History []HistoryEntry `json:"history"`
}

// Load reads the session file referenced by $KUBIE_SESSION. A missing session
// file yields an empty session, a malformed one an error.
func Load() (Session, error) {
	sessionPath, ok := vars.SessionPath()
	if !ok {
		return Session{}, nil
	}

	data, err := os.ReadFile(sessionPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Session{}, nil
		}
		return Session{}, fmt.Errorf("could not read session file %s: %w", sessionPath, err)
	}

	var loaded Session
	if err := json.Unmarshal(data, &loaded); err != nil {
		return Session{}, fmt.Errorf("could not parse session file %s: %w", sessionPath, err)
	}
	return loaded, nil
}

// Save writes the session to the file referenced by $KUBIE_SESSION.
func (s *Session) Save() error {
	sessionPath, ok := vars.SessionPath()
	if !ok {
		return errors.New("KUBIE_SESSION env variable missing")
	}
	if s.History == nil {
		s.History = []HistoryEntry{}
	}

	data, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("could not encode session: %w", err)
	}
	if err := os.WriteFile(sessionPath, data, 0o600); err != nil {
		return fmt.Errorf("could not write session file %s: %w", sessionPath, err)
	}
	return nil
}

// AddHistoryEntry appends a visited context and namespace to the history.
func (s *Session) AddHistoryEntry(context string, namespace *string) {
	s.History = append(s.History, HistoryEntry{Context: context, Namespace: namespace})
}

// RecordContextEntry records a visited context in the session history and
// mirrors it into the shared state.
func (s *Session) RecordContextEntry(context string, namespace *string) error {
	s.AddHistoryEntry(context, namespace)

	return state.Modify(func(current *state.State) error {
		if namespace != nil {
			current.NamespaceHistory[context] = namespace
		}
		lastContext := context
		current.LastContext = &lastContext
		return nil
	})
}

// GetLastContext returns the most recent entry that belongs to another context
// than the current one, skipping consecutive entries of the same context. It
// returns nil when there is no such entry.
func (s *Session) GetLastContext() *HistoryEntry {
	if len(s.History) == 0 {
		return nil
	}

	current := s.History[len(s.History)-1]
	for i := len(s.History) - 2; i >= 0; i-- {
		if s.History[i].Context != current.Context {
			return &s.History[i]
		}
	}
	return nil
}

// GetLastNamespace returns the namespace of the previous entry in the chain of
// entries of the current context, when it differs from the current namespace.
// It returns nil when the chain holds no differing namespace.
func (s *Session) GetLastNamespace() *string {
	if len(s.History) == 0 {
		return nil
	}

	current := s.History[len(s.History)-1]
	for i := len(s.History) - 2; i >= 0; i-- {
		entry := s.History[i]
		if entry.Context != current.Context {
			return nil
		}
		if !sameNamespace(entry.Namespace, current.Namespace) {
			return entry.Namespace
		}
	}
	return nil
}

func sameNamespace(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}
