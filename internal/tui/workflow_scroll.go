package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func (m Model) workflowDetailLines(w int) ([]string, string) {
	f := m.wfFiltered()
	if m.listCursor < 0 || m.listCursor >= len(f) {
		return []string{"Choose a workflow."}, ""
	}
	wf := m.workflows[f[m.listCursor]]
	text := ""
	if m.rows()-m.bannerRows >= 20 {
		text = "Saved order · all steps run by default\n\n"
	}
	orderW := len(fmt.Sprint(len(wf.Commands)))
	nameW := 0
	for _, name := range wf.Commands {
		nameW = max(nameW, ansi.StringWidth(name))
	}
	// Measure all steps so scrolling keeps the command column stationary.
	nameW = min(nameW, max(1, (w-orderW-5)/3))
	for i, name := range wf.Commands {
		body := "not found"
		if c, ok := m.workflowStepCommand(name); ok {
			body = "$ " + c.Cmd
		}
		label := ansi.Truncate(name, nameW, "…")
		label += strings.Repeat(" ", max(0, nameW-ansi.StringWidth(label)))
		text += fmt.Sprintf("%*d. %s  %s\n", orderW, i+1, label, body)
	}
	if len(wf.Commands) == 0 {
		text += "No steps."
	}
	return strings.Split(ansi.Hardwrap(strings.TrimSuffix(text, "\n"), max(1, w), true), "\n"), wf.Name + "\n" + text
}

func (m Model) workflowScrollAreas() []runScrollArea {
	w, h := m.viewWidth(), m.workflowBodyHeight()
	lw := w
	if splitPanels(w) {
		lw = m.runLeftWidth(w)
	}
	rows := max(1, h-3)
	n := len(m.wfFiltered())
	offset := windowStart(rows, n, m.listCursor)
	if m.wfScrollManual {
		offset = min(max(0, m.wfScrollOffset), max(0, n-rows))
	}
	areas := []runScrollArea{{4, 1, 2, lw - 2, rows, n, offset}}
	if splitPanels(w) {
		rw := w - lw - 1
		lines, key := m.workflowDetailLines(rw - 3)
		off := 0
		if key == m.wfDetailKey {
			off = m.wfDetailOffset
		}
		rows = max(1, h-2)
		areas = append(areas, runScrollArea{5, lw + 2, 1, rw - 2, rows, len(lines), min(off, max(0, len(lines)-rows))})
	}
	return areas
}

func (m Model) updateWorkflowHomeMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.handleRunScrollMouse(msg) {
		return m, nil
	}
	a := m.workflowScrollAreas()[0]
	if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft && msg.X >= a.x && msg.X < a.x+a.width-1 && msg.Y >= a.y && msg.Y < a.y+a.rows {
		idx := a.offset + msg.Y - a.y
		if idx < a.total {
			m.listCursor = idx
			m.updateStepsViewport()
			m.wfDetailOffset = 0
			m.wfDetailKey = ""
		}
	}
	return m, nil
}
