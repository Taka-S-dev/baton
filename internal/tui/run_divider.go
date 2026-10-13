package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) runPaneIndex() int {
	if m.screen == ScreenRunWorkflow {
		return 1
	}
	return 0
}
func (m Model) runLeftWidth(w int) int {
	want := m.runPaneWidths[m.runPaneIndex()]
	if want == 0 {
		want = w / 2
		if m.screen == ScreenRunWorkflow {
			want = w / 3
		}
	}
	return min(max(want, 28), w-29)
}
func (m Model) runDivider(rows int) string {
	lines := make([]string, max(1, rows))
	for i := range lines {
		lines[i] = " "
		if m.runPaneDragging {
			lines[i] = accent("│")
		}
	}
	lines[len(lines)/2] = gray("↔")
	if m.runPaneDragging {
		lines[len(lines)/2] = accent("↔")
	}
	return strings.Join(lines, "\n")
}
func (m *Model) handleRunDivider(msg tea.MouseMsg) bool {
	if m.runPaneDragging {
		if msg.Action == tea.MouseActionMotion && msg.Button == tea.MouseButtonLeft {
			m.runPaneWidths[m.runPaneIndex()] = min(max(msg.X, 28), m.viewWidth()-29)
			return true
		}
		m.runPaneDragging = false
		if msg.Action == tea.MouseActionRelease {
			return true
		}
	}
	if !splitPanels(m.viewWidth()) || m.runDetailOpen || msg.Y < 0 || msg.Y >= m.termHeight()-2 || msg.X != m.runLeftWidth(m.viewWidth()) {
		return false
	}
	if msg.Action == tea.MouseActionPress {
		if msg.Button == tea.MouseButtonLeft {
			m.runPaneDragging = true
			return true
		}
		if msg.Button == tea.MouseButtonRight {
			m.runPaneWidths[m.runPaneIndex()] = 0
			return true
		}
	}
	return false
}
