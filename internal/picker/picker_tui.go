package picker

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/khanghua1505/kon/internal/tui/style"
)

// tuiMaxListRows caps the visible rows of the full-screen picker.
const tuiMaxListRows = 20

// tuiModel is the bubbletea model behind the full-screen picker.
type tuiModel struct {
	items    []string
	query    string
	filtered []string
	cursor   int
	choice   string
	quitting bool
	width    int
	height   int
	title    lipgloss.Style
	queryS   lipgloss.Style
	item     lipgloss.Style
	dim      lipgloss.Style
	selected lipgloss.Style
	border   lipgloss.Style
}

// newTuiModel builds the model with the lipgloss styles derived from the theme.
func newTuiModel(items []string, theme style.Style) tuiModel {
	return tuiModel{
		items:    items,
		filtered: items,
		title:    lipgloss.NewStyle().Bold(true).Foreground(theme.Secondary),
		queryS:   lipgloss.NewStyle().Foreground(theme.Foreground),
		item:     lipgloss.NewStyle().Foreground(theme.Foreground),
		dim:      lipgloss.NewStyle().Foreground(theme.Muted),
		selected: lipgloss.NewStyle().Bold(true).Foreground(theme.Primary),
		border:   lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(theme.Border).Padding(0, 1),
	}
}

// Init satisfies tea.Model.
func (m tuiModel) Init() tea.Cmd {
	return nil
}

// Update handles keys and window sizes.
func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			if len(m.filtered) > 0 {
				m.choice = m.filtered[m.cursor]
			} else {
				m.quitting = true
			}
			return m, tea.Quit
		case "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
		case "backspace":
			if m.query != "" {
				m.query = m.query[:len(m.query)-1]
				m.refilter()
			}
		default:
			if msg.Text != "" {
				m.query += msg.Text
				m.refilter()
			}
		}
	}

	return m, nil
}

// View renders the framed picker.
func (m tuiModel) View() tea.View {
	if m.choice != "" || m.quitting {
		return tea.NewView("")
	}

	selectedStyle := m.selected

	var rows []string
	start, visible := tuiWindow(m.filtered, m.cursor, m.listRows())
	for i, item := range visible {
		if start+i == m.cursor {
			rows = append(rows, selectedStyle.Render("❯ "+item))
		} else {
			rows = append(rows, m.item.Render("  "+item))
		}
	}
	if len(rows) == 0 {
		rows = []string{m.dim.Render("  no matches")}
	}

	divider := m.dim.Render(strings.Repeat("─", 32))
	lines := append(
		rows,
		[]string{
			divider,
			m.queryS.Render("❯ " + m.query + "▌"),
		}...,
	)
	box := m.border.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
	help := m.dim.Render("  ↑/↓ move · enter select · esc cancel")

	return tea.NewView(lipgloss.JoinVertical(lipgloss.Left, box, help))
}

// listRows returns how many rows fit between the frame chrome and the cap.
func (m tuiModel) listRows() int {
	rows := tuiMaxListRows
	if m.height > 0 {
		if fit := m.height - 8; fit < rows {
			rows = fit
		}
	}
	if rows < 1 {
		rows = 1
	}

	return rows
}

// refilter recomputes the visible items after the query changed.
func (m *tuiModel) refilter() {
	m.filtered = tuiFilter(m.items, m.query)
	m.cursor = 0
}

// tuiFilter returns the items matching the query, case-insensitively.
func tuiFilter(items []string, query string) []string {
	if query == "" {
		return items
	}

	needle := strings.ToLower(query)
	matched := make([]string, 0, len(items))
	for _, item := range items {
		if strings.Contains(strings.ToLower(item), needle) {
			matched = append(matched, item)
		}
	}

	return matched
}

// tuiWindow returns the visible slice of items and its start offset.
func tuiWindow(items []string, cursor, size int) (int, []string) {
	if size < 1 || len(items) == 0 {
		return 0, nil
	}
	if len(items) <= size {
		return 0, items
	}

	start := cursor - size/2
	if start < 0 {
		start = 0
	}
	if start > len(items)-size {
		start = len(items) - size
	}

	return start, items[start : start+size]
}

// runTui shows the full-screen picker and returns the outcome.
func runTui(items []string, opts Options) (Result, string, error) {
	program := tea.NewProgram(newTuiModel(items, opts.Style), tea.WithInput(opts.In), tea.WithOutput(opts.Out))
	final, err := program.Run()
	if err != nil {
		return Cancelled, "", fmt.Errorf("could not run the picker: %w", err)
	}

	model, ok := final.(tuiModel)
	if !ok || model.choice == "" {
		return Cancelled, "", nil
	}

	return Selected, model.choice, nil
}
