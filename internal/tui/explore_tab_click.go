package tui

import tea "github.com/charmbracelet/bubbletea"

func (m *Model) clickExploreTab(tab int) tea.Cmd {
	if tab == m.exploreTab {
		return nil
	}
	what := leaveNone
	switch {
	case m.exploreTemplateFlow():
		what = leaveTemplate
	case m.exploreValueFlow():
		what = leaveValue
	case m.cf != nil:
		what = leaveForm
	case m.wfe != nil:
		what = leaveWorkflow
	}
	if what != leaveNone && m.askToLeave(what) {
		m.explorePendingTab = tab + 1
		return nil
	}
	switch what {
	case leaveForm:
		m.closeCommandForm()
	case leaveWorkflow:
		m.wfe = nil
	case leaveTemplate:
		m.closeExploreTemplate()
	case leaveValue:
		m.closeExploreValue()
	}
	m.switchTab(tab)
	return nil
}
