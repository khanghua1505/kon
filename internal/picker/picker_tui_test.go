package picker

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/khanghua1505/kon/internal/tui/style"
)

func newTestModel(items []string) tuiModel {
	return newTuiModel(items, style.Default())
}

func updateKeys(t *testing.T, m tuiModel, keys ...tea.KeyPressMsg) tuiModel {
	t.Helper()

	for _, key := range keys {
		next, _ := m.Update(key)
		m = next.(tuiModel)
	}

	return m
}

func keyRunes(s string) tea.KeyPressMsg {
	return tea.KeyPressMsg{Text: s}
}

func TestTuiModelFilterSelects(t *testing.T) {
	m := newTestModel([]string{"dev-a", "prod-eu", "dev-b"})

	m = updateKeys(t, m, keyRunes("dev"), tea.KeyPressMsg{Code: tea.KeyDown}, tea.KeyPressMsg{Code: tea.KeyEnter})

	if m.choice != "dev-b" {
		t.Errorf("choice = %q, want %q", m.choice, "dev-b")
	}
}

func TestTuiModelEscCancels(t *testing.T) {
	m := newTestModel([]string{"dev-a"})

	m = updateKeys(t, m, tea.KeyPressMsg{Code: tea.KeyEsc})

	if m.choice != "" || !m.quitting {
		t.Errorf("choice = %q, quitting = %v, want cancel", m.choice, m.quitting)
	}
}

func TestTuiModelEmptyQueryListsAll(t *testing.T) {
	m := newTestModel([]string{"dev-a", "prod-eu"})

	if len(m.filtered) != 2 {
		t.Errorf("filtered = %v, want all items", m.filtered)
	}
}

func TestTuiModelBackspaceRestores(t *testing.T) {
	m := newTestModel([]string{"dev-a", "prod-eu"})

	m = updateKeys(t, m, keyRunes("zzz"), tea.KeyPressMsg{Code: tea.KeyBackspace}, tea.KeyPressMsg{Code: tea.KeyBackspace}, tea.KeyPressMsg{Code: tea.KeyBackspace})

	if m.query != "" || len(m.filtered) != 2 {
		t.Errorf("query = %q, filtered = %v, want empty query and all items", m.query, m.filtered)
	}
}

func TestTuiModelCursorClamps(t *testing.T) {
	m := newTestModel([]string{"a", "b", "c"})

	m = updateKeys(t, m, tea.KeyPressMsg{Code: tea.KeyUp})

	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0", m.cursor)
	}

	m = updateKeys(t, m, tea.KeyPressMsg{Code: tea.KeyDown}, tea.KeyPressMsg{Code: tea.KeyDown}, tea.KeyPressMsg{Code: tea.KeyDown})

	if m.cursor != 2 {
		t.Errorf("cursor = %d, want 2", m.cursor)
	}
}

func TestTuiFilterCaseInsensitive(t *testing.T) {
	got := tuiFilter([]string{"Dev-A", "PROD-EU"}, "dev")

	if len(got) != 1 || got[0] != "Dev-A" {
		t.Errorf("filter = %v, want [Dev-A]", got)
	}
}

func TestTuiWindowClampsToList(t *testing.T) {
	items := []string{"a", "b", "c"}
	start, visible := tuiWindow(items, 2, 5)

	if start != 0 || len(visible) != 3 {
		t.Errorf("window = %d %v, want 0 and the full list", start, visible)
	}

	start, visible = tuiWindow(items, 2, 2)

	if start != 1 || len(visible) != 2 || visible[0] != "b" {
		t.Errorf("window = %d %v, want 1 and [b c]", start, visible)
	}
}

func TestTuiViewShowsFrame(t *testing.T) {
	m := newTestModel([]string{"dev-a"})
	m = updateKeys(t, m, keyRunes("dev"))

	view := m.View().Content

	if !strings.Contains(view, "Select Context") || !strings.Contains(view, "❯ dev-a") {
		t.Errorf("view missing frame or selection:\n%s", view)
	}
}
