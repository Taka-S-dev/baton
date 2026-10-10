package tui

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"strings"
)

func (m Model) viewRunWorkflow(w int) string {
	if w < 100 || m.termHeight() < 20 {
		return m.viewRunWorkflowList(w)
	}
	leftW := w / 3
	var b strings.Builder
	b.WriteString(m.wfSearchTI.View() + "\n")
	filtered := m.wfFiltered()
	writeWindowedList(&b, max(1, m.termHeight()-5), len(filtered), m.listCursor, "No workflows.", func(i int) string {
		wf := m.workflows[filtered[i]]
		label := fmt.Sprintf("%s · %d steps", wf.Name, len(wf.Commands))
		label = searchHighlight(truncate(label, leftW-4), strings.Fields(m.wfSearchTI.Value()), false, true)
		if i == m.listCursor {
			return accentBold("▶ ") + label
		}
		return "  " + label
	})
	left := box("Run workflow", b.String(), leftW, m.termHeight()-1, !m.stepsFocused)
	rightW := w - leftW - 1
	var detail strings.Builder
	detail.WriteString("Saved order · all steps run by default\n\n")
	if m.listCursor < len(filtered) {
		wf := m.workflows[filtered[m.listCursor]]
		writeWindowedList(&detail, max(1, m.termHeight()-9), len(wf.Commands), m.stepsVP.YOffset, "No steps.", func(i int) string {
			name := wf.Commands[i]
			if c, ok := m.workflowStepCommand(name); ok {
				return truncate(fmt.Sprintf("%d  %s  $ %s", i+1, name, c.Cmd), rightW-2)
			}
			return warn(truncate(fmt.Sprintf("%d  %s · not found", i+1, name), rightW-2))
		})
	}
	right := box("Execution steps", detail.String(), rightW, m.termHeight()-1, m.stepsFocused)
	footer := "↑↓ Select · → Steps · F2 Details · Enter Review · Esc Back · F1 Help"
	return lipgloss.JoinHorizontal(lipgloss.Top, left, " ", right) + "\n" + gray(truncate(footer, w)) + "\n"
}
func (m Model) viewRunWorkflowSteps(w int) string {
	if m.wfp == nil {
		return ""
	}
	if w < 100 || m.termHeight() < 20 {
		if m.runWorkflowDetail {
			return m.workflowRunDetail(w)
		}
		return m.viewRunWorkflowStepsList(w)
	}
	leftW := w / 2
	return lipgloss.JoinHorizontal(lipgloss.Top, strings.TrimRight(m.viewRunWorkflowStepsList(leftW), "\n"), " ", m.workflowRunDetail(w-leftW-1))
}
func (m Model) workflowRunDetail(w int) string {
	var b strings.Builder
	p := m.wfp
	if p != nil && p.wfIdx < len(m.workflows) {
		wf := m.workflows[p.wfIdx]
		if p.cursor < len(wf.Commands) {
			if c, ok := m.workflowStepCommand(wf.Commands[p.cursor]); ok {
				m.writeCommandHover(&b, &c, max(1, w-4))
			} else {
				b.WriteString(warn("Command not found"))
			}
		}
	}
	b.WriteString("\nThis run only. Saved workflow is unchanged.\n")
	return box("Command details", b.String(), w, max(3, m.termHeight()-1), m.runWorkflowDetail) + "\n" + gray(truncate("F2 Details / Steps · Esc Back", w))
}
