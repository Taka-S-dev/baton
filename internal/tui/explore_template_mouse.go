package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) templateDraftDirty() bool {
	if m.sce == nil || m.screen == ScreenEditCommandMode {
		return false
	}
	if len(m.sce.currentValues) > 0 || m.sce.currentSlotIdx > 0 {
		return true
	}
	if m.sp != nil && (m.sp.search != "" || len(m.sp.picked) > 0) {
		return true
	}
	if m.screen == ScreenCreateCommandName {
		return true
	}
	if m.screen == ScreenEditCommandName || m.screen == ScreenNameInput {
		return m.nameInput.Value() != m.sce.name
	}
	if m.sce.mode == 1 && m.sce.editIdx >= 0 && m.sce.editIdx < len(m.config.Commands) {
		candidates := m.templateCandidates()
		i := m.sce.templateRefIdx
		return i >= 0 && i < len(candidates) && candidates[i].Name != m.config.Commands[m.sce.editIdx].Template
	}
	return false
}

func (m *Model) closeExploreTemplate() {
	m.sce, m.sp = nil, nil
	m.nameInput.Blur()
	m.returnFromFlow()
}

func (m *Model) saveTemplateBeforeLeaving() tea.Cmd {
	var result tea.Model
	var cmd tea.Cmd
	switch m.screen {
	case ScreenCreateCommandName:
		result, cmd = m.updateCreateCommand(tea.KeyMsg{Type: tea.KeyEnter})
	case ScreenEditCommandName:
		result, cmd = m.updateEditCommand(tea.KeyMsg{Type: tea.KeyEnter})
	case ScreenNameInput:
		result, cmd = m.updateNameInput(tea.KeyMsg{Type: tea.KeyEnter})
	default:
		m.errMsg = "Finish choosing values and enter a name before saving. Your draft is kept."
		return nil
	}
	switch next := result.(type) {
	case Model:
		*m = next
	case *Model:
		*m = *next
	}
	return cmd
}

func (m Model) updateExploreTemplateMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	l := m.exploreLayout(m.viewWidth())
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return m, nil
	}
	onList := msg.X >= 0 && msg.X < l.leftW && msg.Y >= l.bodyY && msg.Y < l.bodyY+l.bodyH
	onBack := msg.Y == l.editY && msg.X >= l.rightX+3 && msg.X < l.rightX+9
	if onList || onBack {
		if m.askToLeave(leaveTemplate) {
			return m, nil
		}
		m.closeExploreTemplate()
		if onList && l.addY >= 0 && msg.Y == l.addY {
			m.exploreSearching = false
			return m.updateExplore(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
		}
		if onList {
			row := msg.Y - l.listY
			if row >= 0 && row < l.listH {
				i := l.start + row
				if i < len(l.items) {
					m.exploreCursor[m.exploreTab] = i
				}
			}
		}
		return m, nil
	}
	if msg.X <= l.rightX || msg.X >= l.rightX+l.rightW-1 || msg.Y < l.fieldY || msg.Y >= l.editY+l.editH-1 {
		return m, nil
	}
	rows := max(1, l.editH-5)
	row := msg.Y - l.fieldY
	switch m.screen {
	case ScreenEditCommandMode:
		start := windowStart(rows, len(m.listItems), m.listCursor)
		if row >= 0 && row < rows && start+row < len(m.listItems) {
			m.listCursor = start + row
			return m.updateEditCommandMode(tea.KeyMsg{Type: tea.KeyEnter})
		}
	case ScreenEditCommandTemplate:
		items := m.templateCandidates()
		start := windowStart(rows, len(items), m.sce.templateRefIdx)
		if row >= 0 && row < rows && start+row < len(items) {
			m.sce.templateRefIdx = start + row
			return m.startCommandSlots()
		}
	case ScreenSlotPick:
		row-- // search line
		total := len(m.sp.filtered) + 1
		if m.sp.canSkip {
			total++
		}
		start := windowStart(rows, total, m.sp.cursor)
		if row >= 0 && row < rows && start+row < total {
			m.sp.cursor = start + row
			return m.updateSlotPick(tea.KeyMsg{Type: tea.KeyEnter})
		}
	}
	return m, nil
}
