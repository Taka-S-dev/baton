package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Leaving something half-edited is the one moment the screen must not
// be quiet. Esc on the form or the workflow editor, or a click away
// from either, asks what to do with the changes when there are any,
// and does nothing at all when there are none — the question is only
// worth asking when an answer could lose work.
//
// leaveForm and leaveWorkflow say what is being left; the three choices
// are the same for both.
const (
	leaveNone = iota
	leaveForm
	leaveWorkflow
	leaveTemplate
	leaveValue
)

var leaveChoices = []exploreAction{
	{"k", "Keep editing"},
	{"s", "Save and leave"},
	{"d", "Discard changes"},
}

// formDirty reports whether the form holds anything that is not saved:
// a field that differs from what was opened, with the field under the
// caret read live.
func (m Model) formDirty() bool {
	if m.cf == nil {
		return false
	}
	f := m.cf.fields
	f[m.cf.fieldIdx] = strings.TrimSpace(m.nameInput.Value())
	return f != m.cf.original
}

// askToLeave opens the question for what is being left, or reports that
// there was nothing to ask about so the caller can just leave.
func (m *Model) askToLeave(what int) bool {
	switch what {
	case leaveForm:
		if !m.formDirty() {
			return false
		}
	case leaveValue:
		if !m.valueDraftDirty() {
			return false
		}
	case leaveTemplate:
		if !m.templateDraftDirty() {
			return false
		}
	case leaveWorkflow:
		if m.wfe == nil || !m.wfe.dirty {
			return false
		}
	default:
		return false
	}
	m.leaveAsk, m.leaveCursor = what, 0
	return true
}

// updateLeave handles a key while the question is up. It answers with
// whether the key was for the question, so the caller can go on with
// its own keys otherwise — which never happens while it is up, since
// every key is either a choice or ignored.
func (m *Model) updateLeave(msg tea.KeyMsg) tea.Cmd {
	key := msg.String()
	switch key {
	case "up":
		m.leaveCursor = max(0, m.leaveCursor-1)
		return nil
	case "down":
		m.leaveCursor = min(len(leaveChoices)-1, m.leaveCursor+1)
		return nil
	case "enter":
		key = leaveChoices[m.leaveCursor].key
	case "esc":
		key = "k"
	}
	what := m.leaveAsk
	switch key {
	case "s":
		m.leaveAsk = leaveNone
		if what == leaveValue {
			return m.saveValueBeforeLeaving()
		}
		if what == leaveTemplate {
			return m.saveTemplateBeforeLeaving()
		}
		if what == leaveWorkflow {
			return m.saveWorkflowSteps()
		}
		m.cf.fields[m.cf.fieldIdx] = strings.TrimSpace(m.nameInput.Value())
		_, cmd := m.saveCommandForm()
		return cmd
	case "d":
		m.leaveAsk = leaveNone
		if what == leaveValue {
			m.closeExploreValue()
			return nil
		}
		if what == leaveTemplate {
			m.closeExploreTemplate()
			return nil
		}
		if what == leaveWorkflow {
			m.wfe = nil
			return nil
		}
		m.closeCommandForm()
		return nil
	case "k":
		m.leaveAsk = leaveNone
	}
	return nil
}

// writeLeaveAsk draws the question and its three answers, the chosen
// one marked, each with the key that gives it.
func (m Model) writeLeaveAsk(b *strings.Builder) {
	b.WriteString(" " + warn("Unsaved changes.") + " " + gray("What should happen to them?") + "\n\n")
	for i, c := range leaveChoices {
		marker, label := "   ", c.label
		if i == m.leaveCursor {
			marker, label = " ▶ ", white(c.label)
		}
		b.WriteString(marker + accentBold(c.key) + "  " + label + "\n")
	}
}
