package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	mdl "github.com/Taka-S-dev/baton/internal/model"
	"github.com/Taka-S-dev/baton/internal/store"
)

// ── Workflow management ───────────────────────────────────────────────────────

// maxWorkflowNameLen bounds a suggested name so it stays readable in the
// workflow list.
const maxWorkflowNameLen = 48

// workflowNameSegments splits steps into the names a suggested workflow
// name spells out and the "+N" tail counting whatever did not fit.
//
// The budget is length, not a step count: capping at three steps named a
// four-step workflow "build+clean+build-src+1" while the fourth name
// would have fit with room to spare, and that "+1" reads as a step
// called 1. Names are only ever dropped whole, so a name never carries
// half a step.
func workflowNameSegments(cmds []string) (shown []string, tail string) {
	for i, c := range cmds {
		next := c
		if len(shown) > 0 {
			next = strings.Join(shown, "+") + "+" + c
		}
		room := ""
		if rest := len(cmds) - i - 1; rest > 0 {
			room = fmt.Sprintf("+%d", rest)
		}
		if len(shown) > 0 && len([]rune(next+room)) > maxWorkflowNameLen {
			break
		}
		shown = append(shown, c)
	}
	if rest := len(cmds) - len(shown); rest > 0 {
		tail = fmt.Sprintf("+%d", rest)
	}
	return shown, tail
}

// suggestWorkflowName proposes "cmd1+cmd2+cmd3" (a "+n" tail counts any
// step that did not fit) from the picked commands, so workflow names
// show what they run. A numeric suffix resolves collisions with existing
// workflow names.
func (m Model) suggestWorkflowName() string {
	return m.suggestWorkflowNameFor(m.pendingWorkflowCmds, -1)
}

// suggestWorkflowNameFor builds the name for the given steps.
// excludeIdx is the workflow being renamed, which does not collide with
// itself.
func (m Model) suggestWorkflowNameFor(cmds []string, excludeIdx int) string {
	if len(cmds) == 0 {
		return ""
	}
	shown, tail := workflowNameSegments(cmds)
	base := strings.Join(shown, "+") + tail
	// A single step whose own name blows the budget is the one case
	// where dropping whole names cannot help.
	base = truncate(base, maxWorkflowNameLen)
	taken := func(n string) bool {
		for i, wf := range m.workflows {
			if i != excludeIdx && wf.Name == n {
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
	// .last_workflow references workflows by name, so it follows the
	// rename — otherwise the Run workflow cursor loses its position.
	if m.lastWorkflow == m.workflows[idx].Name {
		m.lastWorkflow = name
		store.SaveLastWorkflow(m.projectDir, name)
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
