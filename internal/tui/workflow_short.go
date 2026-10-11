package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

func (m Model) viewShortWorkflowSplit(w int) string {
	h := m.termHeight()
	leftW := w / 3
	rightW := w - leftW - 3
	left := strings.Split(strings.TrimRight(m.viewRunWorkflowList(leftW), "\n"), "\n")
	right := []string{gray("Execution steps")}
	filtered := m.wfFiltered()
	if m.listCursor >= 0 && m.listCursor < len(filtered) {
		wf := m.workflows[filtered[m.listCursor]]
		right[0] = gray("Execution steps · " + wf.Name)
		count := len(wf.Commands)
		rows := max(1, h-1)
		if count > rows {
			rows = max(1, h-2)
		}
		start := windowStart(rows, count, m.stepsVP.YOffset)
		for i := start; i < min(start+rows, count); i++ {
			name := wf.Commands[i]
			label := fmt.Sprintf("%d. %s", i+1, name)
			if _, ok := m.workflowStepCommand(name); !ok {
				label += " · not found"
				right = append(right, warn(label))
			} else {
				right = append(right, label)
			}
		}
		if count == 0 {
			right = append(right, gray("No steps."))
		}
		if count > rows {
			right = append(right, gray(fmt.Sprintf("%d–%d / %d · F2 Details", start+1, min(start+rows, count), count)))
		}
	} else {
		right = append(right, gray("Choose a workflow."))
	}
	lines := make([]string, h)
	for y := 0; y < h; y++ {
		l, r := "", ""
		if y < len(left) {
			l = ansi.Truncate(left[y], leftW, "…")
		}
		if y < len(right) {
			r = ansi.Truncate(right[y], rightW, "…")
		}
		lines[y] = l + strings.Repeat(" ", max(0, leftW-ansi.StringWidth(l))) + gray(" │ ") + r
	}
	return strings.Join(lines, "\n")
}
