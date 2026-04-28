package tui

import (
	"sort"

	tea "github.com/charmbracelet/bubbletea"

	mdl "github.com/Taka-S-dev/baton/internal/model"
	"github.com/Taka-S-dev/baton/internal/store"
)

// ── Edit workflow ─────────────────────────────────────────────────────────────

func (m Model) updateEditWorkflow(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up":
		if m.listCursor > 0 {
			m.listCursor--
			m.updateStepsViewport()
		}
	case "down":
		if m.listCursor < len(m.listItems)-1 {
			m.listCursor++
			m.updateStepsViewport()
		}
	case "enter":
		if len(m.workflows) == 0 {
			break
		}
		m.editTargetIdx = m.listCursor
		m.screen = ScreenEditWorkflowMode
		m.listItems = []string{"Rename", "Change commands"}
		m.listCursor = 0
	case "esc":
		m.gotoMainMenu()
	}
	return m, nil
}

func (m Model) updateEditWorkflowMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up":
		if m.listCursor > 0 {
			m.listCursor--
		}
	case "down":
		if m.listCursor < len(m.listItems)-1 {
			m.listCursor++
		}
	case "enter":
		switch m.listItems[m.listCursor] {
		case "Rename":
			m.nameInput.SetValue(m.workflows[m.editTargetIdx].Name)
			m.nameInputMode = nameInputEditWorkflow
			m.nameInputErr = ""
			m.screen = ScreenNameInput
			return m, m.nameInput.Focus()
		case "Change commands":
			m.screen = ScreenEditWorkflowCommands
			return m, m.setupMultiSelectWithPreSelected(m.workflows[m.editTargetIdx].Commands)
		}
	case "esc":
		m.screen = ScreenEditWorkflow
		names := make([]string, len(m.workflows))
		for i, w := range m.workflows {
			names[i] = w.Name
		}
		m.listItems = names
		m.listCursor = m.editTargetIdx
	}
	return m, nil
}

// ── Delete workflow ───────────────────────────────────────────────────────────

func (m Model) updateDeleteWorkflow(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.deleteConfirm {
		switch msg.String() {
		case "tab", "left", "right", "h", "l":
			m.deleteBtn = 1 - m.deleteBtn
		case "enter":
			if m.deleteBtn == 1 {
				indices := m.deleteSelected
				sort.Sort(sort.Reverse(sort.IntSlice(indices)))
				for _, i := range indices {
					m.workflows = append(m.workflows[:i], m.workflows[i+1:]...)
				}
				if err := store.SaveWorkflows(m.projectDir, m.workflows); err != nil {
					m.errMsg = "failed to save workflows: " + err.Error()
				}
				m.deleteSelected = nil
				m.deleteConfirm = false
				m.deleteBtn = 0
				m.gotoMainMenu()
			} else {
				m.deleteConfirm = false
				m.deleteBtn = 0
			}
		case "esc":
			m.deleteConfirm = false
			m.deleteBtn = 0
		}
		return m, nil
	}
	switch msg.String() {
	case "up":
		if m.listCursor > 0 {
			m.listCursor--
			m.updateStepsViewport()
		}
	case "down":
		if m.listCursor < len(m.listItems)-1 {
			m.listCursor++
			m.updateStepsViewport()
		}
	case " ", "　":
		found := false
		for i, s := range m.deleteSelected {
			if s == m.listCursor {
				m.deleteSelected = append(m.deleteSelected[:i], m.deleteSelected[i+1:]...)
				found = true
				break
			}
		}
		if !found {
			m.deleteSelected = append(m.deleteSelected, m.listCursor)
		}
	case "enter":
		if len(m.workflows) == 0 {
			break
		}
		if len(m.deleteSelected) == 0 {
			m.deleteSelected = []int{m.listCursor}
		}
		m.deleteConfirm = true
		m.deleteBtn = 0
	case "esc":
		m.deleteSelected = nil
		m.gotoMainMenu()
	}
	return m, nil
}

// ── Save / rename workflow ────────────────────────────────────────────────────

func (m Model) saveWorkflow(name string) (tea.Model, tea.Cmd) {
	for _, w := range m.workflows {
		if w.Name == name {
			m.nameInputErr = "name already exists"
			return m, nil
		}
	}
	r := m.resolve
	var cmdNames []string
	for _, item := range r.rawItems {
		cmdNames = append(cmdNames, item.name())
	}
	wf := mdl.Workflow{Name: name, Commands: cmdNames}
	if len(r.workflowVars) > 0 {
		wf.Vars = r.workflowVars
	}
	m.workflows = append(m.workflows, wf)
	if err := store.SaveWorkflows(m.projectDir, m.workflows); err != nil {
		m.errMsg = "failed to save workflows: " + err.Error()
	}
	m.resolve = nil
	m.gotoMainMenu()
	return m, nil
}

func (m Model) renameWorkflow(idx int, name string) (tea.Model, tea.Cmd) {
	if idx < 0 || idx >= len(m.workflows) {
		m.gotoMainMenu()
		return m, nil
	}
	for i, w := range m.workflows {
		if i != idx && w.Name == name {
			m.nameInputErr = "name already exists"
			return m, nil
		}
	}
	m.workflows[idx].Name = name
	if err := store.SaveWorkflows(m.projectDir, m.workflows); err != nil {
		m.errMsg = "failed to save workflows: " + err.Error()
	}
	m.gotoMainMenu()
	return m, nil
}
