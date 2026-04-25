package tui

import (
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
)

// ── Project select ────────────────────────────────────────────────────────────

func (m Model) updateProjectSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.listCursor > 0 {
			m.listCursor--
		}
	case "down", "j":
		if m.listCursor < len(m.listItems)-1 {
			m.listCursor++
		}
	case "enter":
		dir := filepath.Join(m.projectsDir, m.projects[m.listCursor])
		if err := m.loadProject(dir); err != nil {
			m.errMsg = err.Error()
		} else {
			m.gotoMainMenu()
		}
	case "esc", "q":
		return m, tea.Quit
	}
	return m, nil
}

// ── Main menu ─────────────────────────────────────────────────────────────────

func (m Model) updateMainMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.listCursor > 0 {
			m.listCursor--
		}
	case "down", "j":
		if m.listCursor < len(m.listItems)-1 {
			m.listCursor++
		}
	case "enter":
		m.mainMenuCursor = m.listCursor
		switch m.listItems[m.listCursor] {
		case "Run workflow":
			m.screen = ScreenRunWorkflow
			names := make([]string, len(m.workflows))
			for i, w := range m.workflows {
				names[i] = w.Name
			}
			m.listItems = names
			m.listCursor = 0
			m.stepsFocused = false
			if m.lastWorkflow != "" {
				for i, w := range m.workflows {
					if w.Name == m.lastWorkflow {
						m.listCursor = i
						break
					}
				}
			}
			m.updateStepsViewport()
		case "Run manually":
			m.screen = ScreenRunManually
			return m, m.setupMultiSelect(true)
		case "Create workflow":
			m.screen = ScreenCreateWorkflow
			return m, m.setupMultiSelectCmdsOnly()
		case "Edit workflow":
			m.screen = ScreenEditWorkflow
			names := make([]string, len(m.workflows))
			for i, w := range m.workflows {
				names[i] = w.Name
			}
			m.listItems = names
			m.listCursor = 0
			m.updateStepsViewport()
		case "Delete workflow":
			m.screen = ScreenDeleteWorkflow
			names := make([]string, len(m.workflows))
			for i, w := range m.workflows {
				names[i] = w.Name
			}
			m.listItems = names
			m.listCursor = 0
			m.updateStepsViewport()
		case "Manage aliases":
			m.screen = ScreenAliasMgmt
			m.listItems = []string{"Create alias", "Edit alias", "Delete alias"}
			m.listCursor = 0
		case "Manage lists":
			m.gotoManageLists()
		case "Switch config":
			m.screen = ScreenSwitchConfig
			m.listItems = m.projects
			m.listCursor = 0
		case "Exit":
			return m, tea.Quit
		}
	case "esc":
		return m, tea.Quit
	}
	return m, nil
}

// ── Switch config ─────────────────────────────────────────────────────────────

func (m Model) updateSwitchConfig(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.listCursor > 0 {
			m.listCursor--
		}
	case "down", "j":
		if m.listCursor < len(m.listItems)-1 {
			m.listCursor++
		}
	case "enter":
		dir := filepath.Join(m.projectsDir, m.projects[m.listCursor])
		if err := m.loadProject(dir); err != nil {
			m.errMsg = err.Error()
		} else {
			m.gotoMainMenu()
		}
	case "esc":
		m.gotoMainMenu()
	}
	return m, nil
}
