package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	mdl "github.com/Taka-S-dev/baton/internal/model"
	"github.com/Taka-S-dev/baton/internal/store"
)

// ── Workflow management ───────────────────────────────────────────────────────

// suggestWorkflowName proposes "cmd1+cmd2+cmd3" (a "+n" tail counts the
// rest) from the picked commands, so workflow names show what they run.
// A numeric suffix resolves collisions with existing workflow names.
func (m Model) suggestWorkflowName() string {
	cmds := m.pendingWorkflowCmds
	if len(cmds) == 0 {
		return ""
	}
	shown := cmds
	if len(shown) > 3 {
		shown = shown[:3]
	}
	base := strings.Join(shown, "+")
	if rest := len(cmds) - len(shown); rest > 0 {
		base += fmt.Sprintf("+%d", rest)
	}
	if r := []rune(base); len(r) > 48 {
		base = strings.TrimRight(string(r[:48]), "+-")
	}
	taken := func(n string) bool {
		for _, wf := range m.workflows {
			if wf.Name == n {
				return true
			}
		}
		return false
	}
	name := base
	for i := 2; taken(name); i++ {
		name = fmt.Sprintf("%s-%d", base, i)
	}
	return name
}

// gotoWorkflowMgmt opens the Manage workflows submenu.
func (m *Model) gotoWorkflowMgmt() {
	m.screen = ScreenWorkflowMgmt
	m.listItems = []string{"Create workflow", "Edit workflow", "Delete workflow"}
	m.listCursor = 0
}

func (m Model) updateWorkflowMgmt(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "down":
		m.moveListCursor(msg.String(), len(m.listItems))
	case "enter":
		switch m.listItems[m.listCursor] {
		case "Create workflow":
			m.screen = ScreenCreateWorkflow
			return m, m.setupMultiSelect()
		case "Edit workflow":
			m.screen = ScreenEditWorkflow
			m.setWorkflowPickBase()
			m.listCursor = 0
			m.updateStepsViewport()
		case "Delete workflow":
			m.screen = ScreenDeleteWorkflow
			m.setWorkflowPickBase()
			m.listCursor = 0
			m.updateStepsViewport()
		}
	case "esc":
		m.gotoMainMenu()
	}
	return m, nil
}

// ── Edit workflow ─────────────────────────────────────────────────────────────

// setWorkflowPickBase fills the pick filter with the workflow names,
// searchable like Run workflow: name, step names and step bodies.
func (m *Model) setWorkflowPickBase() {
	names := make([]string, len(m.workflows))
	texts := make([]string, len(m.workflows))
	for i, wf := range m.workflows {
		names[i] = wf.Name
		texts[i] = m.wfSearchText(wf)
	}
	m.setPickBase(names, texts)
}

func (m Model) updateEditWorkflow(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "down":
		if m.moveListCursor(msg.String(), len(m.listItems)) {
			m.updateStepsViewport()
		}
	case "enter":
		if len(m.listItems) == 0 {
			break
		}
		m.editTargetIdx = m.pickOrig(m.listCursor)
		m.screen = ScreenEditWorkflowMode
		m.listItems = []string{"Rename", "Change commands"}
		m.listCursor = 0
	case "esc":
		if m.pickSearch != "" {
			m.clearPickFilter()
			m.updateStepsViewport()
			break
		}
		m.gotoWorkflowMgmt()
	default:
		m.handlePickTyping(msg, m.updateStepsViewport)
	}
	return m, nil
}

func (m Model) updateEditWorkflowMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "down":
		m.moveListCursor(msg.String(), len(m.listItems))
	case "enter":
		switch m.listItems[m.listCursor] {
		case "Rename":
			m.nameInput.SetValue(m.workflows[m.editTargetIdx].Name)
			m.nameInput.Prompt = "Name > "
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
		m.setWorkflowPickBase()
		m.listCursor = m.editTargetIdx
		m.updateStepsViewport()
	}
	return m, nil
}

// ── Delete workflow ───────────────────────────────────────────────────────────

func (m Model) updateDeleteWorkflow(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	return m.updateDeleteList(msg, len(m.listItems),
		m.updateStepsViewport,
		m.gotoWorkflowMgmt,
		func(indices []int) {
			for _, i := range indices {
				m.workflows = append(m.workflows[:i], m.workflows[i+1:]...)
			}
			if err := store.SaveWorkflows(m.projectDir, m.workflows); err != nil {
				m.errMsg = "failed to save workflows: " + err.Error()
			} else {
				m.successMsg = fmt.Sprintf("deleted %d workflow(s)", len(indices))
			}
		})
}

// ── Save / rename workflow ────────────────────────────────────────────────────

func (m Model) saveWorkflow(name string) (tea.Model, tea.Cmd) {
	for _, w := range m.workflows {
		if w.Name == name {
			m.nameInputErr = "name already in use: " + name
			return m, nil
		}
	}
	m.workflows = append(m.workflows, mdl.Workflow{Name: name, Commands: m.pendingWorkflowCmds})
	if err := store.SaveWorkflows(m.projectDir, m.workflows); err != nil {
		m.errMsg = "failed to save workflows: " + err.Error()
	} else {
		m.successMsg = "created workflow \"" + name + "\""
	}
	m.pendingWorkflowCmds = nil
	m.gotoWorkflowMgmt()
	return m, nil
}

func (m Model) renameWorkflow(idx int, name string) (tea.Model, tea.Cmd) {
	if idx < 0 || idx >= len(m.workflows) {
		m.gotoWorkflowMgmt()
		return m, nil
	}
	for i, w := range m.workflows {
		if i != idx && w.Name == name {
			m.nameInputErr = "name already in use: " + name
			return m, nil
		}
	}
	m.workflows[idx].Name = name
	if err := store.SaveWorkflows(m.projectDir, m.workflows); err != nil {
		m.errMsg = "failed to save workflows: " + err.Error()
	} else {
		m.successMsg = "renamed workflow to \"" + name + "\""
	}
	m.gotoWorkflowMgmt()
	return m, nil
}
