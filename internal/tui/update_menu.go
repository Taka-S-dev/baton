package tui

import (
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
)

// ── Project select ────────────────────────────────────────────────────────────

func (m Model) updateProjectSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "down":
		m.moveListCursor(msg.String(), len(m.listItems))
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
	case "up", "down":
		m.moveListCursor(msg.String(), len(m.listItems))
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
			m.wfSearchTI = newMSTI("/ ")
			if m.lastWorkflow != "" {
				for i, w := range m.workflows {
					if w.Name == m.lastWorkflow {
						m.listCursor = i
						break
					}
				}
			}
			m.updateStepsViewport()
			return m, m.wfSearchTI.Focus()
		case "Run commands":
			m.screen = ScreenRunCommands
			return m, m.setupMultiSelect(true)
		case "Manage workflows":
			m.gotoWorkflowMgmt()
		case "Manage commands":
			m.screen = ScreenManageCommands
			m.listCursor = 0
		case "Manage aliases":
			m.gotoAliasMgmt()
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
	case "up", "down":
		m.moveListCursor(msg.String(), len(m.listItems))
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
