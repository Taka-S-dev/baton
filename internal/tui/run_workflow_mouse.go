package tui

import tea "github.com/charmbracelet/bubbletea"

func (m Model) updateWorkflowStepsMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft || m.wfp == nil {
		return m, nil
	}
	w := m.viewWidth()
	if m.workflowDetailsInline() {
		w /= 2
	} else if m.runWorkflowDetail {
		return m, nil
	}
	if msg.X < 0 || msg.X >= w || msg.Y < 0 {
		return m, nil
	}
	index := -1
	m.viewRunWorkflowStepsList(w, func(i, y int) {
		if msg.Y == y {
			index = i
		}
	})
	if index < 0 {
		return m, nil
	}
	m.wfp.cursor = index
	if msg.X >= 2 && msg.X < 5 {
		return m.updateRunWorkflowSteps(tea.KeyMsg{Type: tea.KeyTab})
	}
	return m, nil
}
