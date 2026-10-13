package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) viewRunWorkflow(w int, searchRow ...func(int)) string {
	m.width = w
	areas := m.workflowScrollAreas()
	a := areas[0]
	for _, hit := range searchRow {
		hit(1)
	}
	f := m.wfFiltered()
	var list []string
	for i, idx := range f {
		wf := m.workflows[idx]
		label := fmt.Sprintf("  %s · %d steps", wf.Name, len(wf.Commands))
		if i == m.listCursor {
			label = m.selectedSyntaxRow("▶ "+strings.TrimPrefix(label, "  "), a.width-1)
		}
		list = append(list, label)
	}
	if len(list) == 0 {
		list = []string{"No workflows."}
	}
	left := box("Run workflow", m.runSearchView(m.wfSearchTI)+"\n"+scrollRows(list, a.offset, a.rows, a.width), a.width+2, m.termHeight()-1, true)
	body := left
	if len(areas) > 1 {
		d := areas[1]
		lines, _ := m.workflowDetailLines(d.width - 1)
		right := box("Execution steps", scrollRows(lines, d.offset, d.rows, d.width), d.width+2, m.termHeight()-1, false)
		body = lipgloss.JoinHorizontal(lipgloss.Top, left, m.runDivider(m.termHeight()-1), right)
	}
	return body + "\n" + gray(truncate(m.runHomeKeys(), w))
}

func (m Model) workflowDetailsInline() bool { return splitPanels(m.viewWidth()) }

func (m Model) viewRunWorkflowSteps(w int) string {
	if m.wfp == nil {
		return ""
	}
	if m.shortPanels() {
		if splitPanels(w) {
			return joinPanels(m.shortWorkflowSteps(w/2), m.shortWorkflowDetail(w-w/2-3), w, w/2, m.termHeight())
		}
		if m.runWorkflowDetail {
			return m.shortWorkflowDetail(w)
		}
		return m.shortWorkflowSteps(w)
	}
	if !splitPanels(w) {
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
	footer := "F2 Steps · Esc Back"
	if m.workflowDetailsInline() {
		footer = ""
	}
	return box("Command details", b.String(), w, max(3, m.termHeight()-1), m.runWorkflowDetail && !m.workflowDetailsInline()) + "\n" + gray(truncate(footer, w))
}
