package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Taka-S-dev/baton/internal/store"
)

// A workflow is edited in the box beside the list as the ordered steps
// it is: move one, take one out, add one from the commands, open the
// command a step names. The steps are a working copy until Ctrl+S, so
// a reorder that went wrong is an Esc away from never having happened.
// Taking a step out and deleting the command it names are different
// things, and only the first happens here.
type workflowEditState struct {
	idx    int      // which workflow, in m.workflows
	steps  []string // the working copy
	cursor int
	dirty  bool

	// Adding a step: a picker of the commands opens in the box, with a
	// search over them; the choice goes in after the cursor.
	adding    bool
	addCursor int
	addSearch string
}

// openWorkflowEditor starts editing the named workflow in the box.
func (m *Model) openWorkflowEditor(name string) {
	for i, wf := range m.workflows {
		if wf.Name == name {
			m.wfe = &workflowEditState{idx: i, steps: append([]string{}, wf.Commands...)}
			return
		}
	}
}

// addCandidates lists the commands a step can name, narrowed by the
// search typed over them.
func (m Model) addCandidates() []string {
	terms := strings.Fields(strings.ToLower(m.wfe.addSearch))
	var out []string
	for _, c := range m.config.AllCommands() {
		if matchesAllTerms(strings.ToLower(c.Name+" "+c.Cmd), terms) {
			out = append(out, c.Name)
		}
	}
	return out
}

func (m Model) updateWorkflowEditor(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.leaveAsk != leaveNone {
		return m, m.updateLeave(msg)
	}
	e := m.wfe
	key := msg.String()

	if e.adding {
		cands := m.addCandidates()
		switch key {
		case "up":
			e.addCursor = max(0, e.addCursor-1)
		case "down":
			e.addCursor = min(max(0, len(cands)-1), e.addCursor+1)
		case "enter":
			if e.addCursor < len(cands) {
				at := min(e.cursor+1, len(e.steps))
				if len(e.steps) == 0 {
					at = 0
				}
				e.steps = append(e.steps[:at], append([]string{cands[e.addCursor]}, e.steps[at:]...)...)
				e.cursor = at
				e.dirty = true
			}
			e.adding, e.addSearch = false, ""
		case "esc":
			e.adding, e.addSearch = false, ""
		case "backspace":
			if r := []rune(e.addSearch); len(r) > 0 {
				e.addSearch = string(r[:len(r)-1])
			}
			e.addCursor = 0
		default:
			if msg.Type == tea.KeyRunes {
				e.addSearch += key
				e.addCursor = 0
			}
		}
		return m, nil
	}

	n := len(e.steps)
	switch key {
	case "up":
		e.cursor = max(0, e.cursor-1)
	case "down":
		e.cursor = min(max(0, n-1), e.cursor+1)
	case "K", "shift+up":
		if e.cursor > 0 {
			e.steps[e.cursor-1], e.steps[e.cursor] = e.steps[e.cursor], e.steps[e.cursor-1]
			e.cursor--
			e.dirty = true
		}
	case "J", "shift+down":
		if e.cursor < n-1 {
			e.steps[e.cursor+1], e.steps[e.cursor] = e.steps[e.cursor], e.steps[e.cursor+1]
			e.cursor++
			e.dirty = true
		}
	case "a":
		e.adding, e.addCursor, e.addSearch = true, 0, ""
	case "x":
		if n > 0 {
			e.steps = append(e.steps[:e.cursor], e.steps[e.cursor+1:]...)
			e.cursor = min(e.cursor, max(0, len(e.steps)-1))
			e.dirty = true
		}
	case "enter":
		if n == 0 {
			return m, nil
		}
		name := e.steps[e.cursor]
		if _, ok := m.workflowStepCommand(name); !ok {
			m.errMsg = "command not found: " + name
			return m, nil
		}
		m.openExploreLink(tabCommands, name)

	case "ctrl+s":
		return m, m.saveWorkflowSteps()
	case "esc":
		if !m.askToLeave(leaveWorkflow) {
			m.wfe = nil
		}
	}
	return m, nil
}

// saveWorkflowSteps writes the working copy over the workflow and saves
// workflows.json. A name baton built from the old steps is left as it
// is: it is still the name you have been using, and r renames.
func (m *Model) saveWorkflowSteps() tea.Cmd {
	e := m.wfe
	if e == nil || e.idx < 0 || e.idx >= len(m.workflows) {
		m.wfe = nil
		return nil
	}
	original := m.workflows[e.idx].Commands
	m.workflows[e.idx].Commands = append([]string{}, e.steps...)
	if err := store.SaveWorkflows(m.projectDir, m.workflows); err != nil {
		m.workflows[e.idx].Commands = original
		m.errMsg = "failed to save: " + err.Error()
		return nil
	}
	m.successMsg = "saved workflow \"" + m.workflows[e.idx].Name + "\""
	m.wfe = nil
	m.updateStepsViewport()
	return nil
}
