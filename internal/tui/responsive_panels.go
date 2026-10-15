package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

func splitPanels(w int) bool      { return w >= 100 }
func (m Model) shortPanels() bool { return m.termHeight() < 20 }
func joinPanels(left, right string, w, leftW, h int) string {
	a, b := strings.Split(left, "\n"), strings.Split(right, "\n")
	lines := make([]string, h)
	for i := range lines {
		l, r := "", ""
		if i < len(a) {
			l = ansi.Truncate(a[i], leftW, "…")
		}
		if i < len(b) {
			r = ansi.Truncate(b[i], max(1, w-leftW-3), "…")
		}
		lines[i] = l + strings.Repeat(" ", max(0, leftW-ansi.StringWidth(l))) + gray(" │ ") + r
	}
	return strings.Join(lines, "\n")
}

func (m Model) shortRunQueue(w int) string {
	n := len(m.msSelected)
	rows := max(1, m.termHeight()-1)
	start := m.queueStart(rows)
	title := fmt.Sprintf("Run queue · %d selected", n)
	if n > rows {
		title += fmt.Sprintf(" · %d–%d", start+1, min(start+rows, n))
	}
	var lines []string
	for i := 0; i < n; i++ {
		line := m.queueCommandLabel(i, max(1, w-3))
		if i == m.runQueueCursor && m.runQueueFocused {
			line = m.selectedSyntaxRow("▶ "+line, w-1)
		} else {
			line = "  " + line
		}
		lines = append(lines, line)
	}
	if n == 0 {
		lines = append(lines, gray("Tab: add commands"))
	}
	heading := white(title)
	if m.runQueueFocused {
		heading = accentBold(title)
	}
	return heading + "\n" + scrollRows(lines, start, rows, w)
}

func (m Model) shortWorkflowSteps(w int, hitRow ...func(int, int)) string {
	p := m.wfp
	if p == nil || p.wfIdx >= len(m.workflows) {
		return ""
	}
	wf := m.workflows[p.wfIdx]
	rows := max(1, m.termHeight()-1)
	start := windowStart(rows, len(wf.Commands), p.cursor)
	lines := []string{accentBold(fmt.Sprintf("Steps · %d selected", p.count()))}
	for i := start; i < min(start+rows, len(wf.Commands)); i++ {
		for _, hit := range hitRow {
			hit(i, len(lines))
		}
		check := "[ ]"
		if p.picked[i] {
			check = "[✓]"
		}
		line := fmt.Sprintf("%s %d. %s", check, i+1, wf.Commands[i])
		if i == p.cursor {
			line = m.selectedSyntaxRow("▶ "+line, w)
		} else {
			line = "  " + line
		}
		lines = append(lines, ansi.Truncate(line, w, "…"))
	}
	return strings.Join(lines, "\n")
}

func (m Model) shortWorkflowDetail(w int) string {
	lines := []string{gray("Command details")}
	p := m.wfp
	if p != nil && p.wfIdx < len(m.workflows) {
		wf := m.workflows[p.wfIdx]
		if p.cursor >= 0 && p.cursor < len(wf.Commands) {
			name := wf.Commands[p.cursor]
			lines = append(lines, name)
			if c, ok := m.workflowStepCommand(name); ok {
				lines = append(lines, m.syntaxText("$ "+c.Cmd), gray("In: "+m.confirmWorkdir(c.Dir)))
			} else {
				lines = append(lines, warn("Command not found"))
			}
		}
	}
	for i := range lines {
		lines[i] = ansi.Truncate(lines[i], w, "…")
	}
	return strings.Join(lines[:min(len(lines), m.termHeight())], "\n")
}
