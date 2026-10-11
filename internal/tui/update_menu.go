package tui

import (
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
)

// ── Project select ────────────────────────────────────────────────────────────

func (m Model) updateProjectSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) { return m.updateProjects(msg) }

// ── Switch config ─────────────────────────────────────────────────────────────

func (m Model) updateSwitchConfig(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "down":
		m.moveListCursor(msg.String(), len(m.listItems))
	case "enter":
		dir := filepath.Join(m.projectsDir, m.projects[m.listCursor])
		if err := m.loadProject(dir); err != nil {
			m.errMsg = err.Error()
		} else {
			m.gotoPostLoad()
		}
	case "esc":
		m.gotoRunHome()
	}
	return m, nil
}
