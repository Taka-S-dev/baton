package tui

import tea "github.com/charmbracelet/bubbletea"

func (m Model) valueDraftDirty() bool {
	switch m.screen {
	case ScreenEditList:
		le := m.le
		if le.adding {
			return le.addVal != "" || le.addLbl != ""
		}
		if le.editing {
			e := le.entries[le.cursor]
			return le.editValTI.Value() != e.Value || le.editLblTI.Value() != e.Label
		}
	case ScreenVarForm:
		if m.ve.mode == 1 {
			return m.nameInput.Value() != m.ve.oldValue
		}
		return m.ve.name != "" || m.nameInput.Value() != ""
	case ScreenNameInput:
		return m.nameInput.Value() != ""
	}
	return false
}

func (m *Model) closeExploreValue() {
	m.le, m.ve = nil, nil
	m.nameInput.Blur()
	m.returnFromFlow()
}

func (m *Model) saveValueBeforeLeaving() tea.Cmd {
	var next tea.Model
	var cmd tea.Cmd
	switch m.screen {
	case ScreenEditList:
		if m.le.adding {
			if m.le.addVal == "" {
				m.errMsg = "Value must not be empty"
				return nil
			}
			m.le.addFld = 1
		} else if m.le.editing {
			m.le.editFld = 1
		}
		next, cmd = m.updateEditList(tea.KeyMsg{Type: tea.KeyEnter})
	case ScreenVarForm:
		if m.ve.mode == 0 && m.ve.fieldIdx == 0 {
			m.errMsg = "Enter a variable value before saving. Your draft is kept."
			return nil
		}
		next, cmd = m.updateVarForm(tea.KeyMsg{Type: tea.KeyEnter})
	case ScreenNameInput:
		next, cmd = m.updateNameInput(tea.KeyMsg{Type: tea.KeyEnter})
	}
	if next == nil {
		return cmd
	}
	*m = next.(Model)
	if m.screen == ScreenEditList && !m.le.adding && !m.le.editing {
		m.closeExploreValue()
	} else if m.screen == ScreenEditList && m.le.adding && m.le.addVal == "" && m.le.addLbl == "" {
		m.closeExploreValue()
	}
	return cmd
}
