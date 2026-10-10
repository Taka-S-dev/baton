package tui

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"strings"
)

func (m Model) discardOptions() string {
	if m.discardCursor == 1 {
		return "  Keep selecting\n▶ Discard and go back"
	}
	return "▶ Keep selecting\n  Discard and go back"
}
func (m Model) viewMultiSelect(w int) string {
	if m.msEscArmed && len(m.msSelected) > 0 {
		names := []string{}
		for _, i := range m.msSelected {
			names = append(names, m.msItems[i].name())
		}
		return m.viewDiscardWindow(w, names)
	}
	if w < 100 || m.termHeight() < 20 {
		if m.runQueueFocused {
			return m.viewRunQueue(w)
		}
		return m.viewMultiSelectList(w)
	}
	leftW := w / 2
	left := strings.TrimRight(m.viewMultiSelectCandidates(leftW, false), "\n")
	right := m.viewRunQueue(w - leftW - 1)
	return lipgloss.JoinHorizontal(lipgloss.Top, left, " ", right)
}
func (m Model) viewRunQueue(w int) string {
	h := m.termHeight()
	queueH := max(5, h/2)
	var b strings.Builder
	cursor := min(m.runQueueCursor, max(0, len(m.msSelected)-1))
	nameWidth := 0
	for _, idx := range m.msSelected {
		nameWidth = max(nameWidth, lipgloss.Width(m.msItems[idx].name()))
	}
	nameWidth = min(nameWidth, max(4, (w-10)/3))
	writeWindowedList(&b, max(1, queueH-3), len(m.msSelected), cursor, "Tab selects commands for this queue.", func(i int) string {
		item := m.msItems[m.msSelected[i]]
		name := truncate(item.name(), nameWidth)
		name += strings.Repeat(" ", max(0, nameWidth-lipgloss.Width(name)))
		command := strings.Join(strings.Fields(item.cmd.Cmd), " ")
		label := ansi.Truncate(fmt.Sprintf(" %d  %s  $ ", i+1, name)+m.previewVars(command), max(1, w-2), "…")
		if i == cursor && m.runQueueFocused {
			return sRowActive.Render(label)
		}
		return label
	})
	top := box("Run queue · execution order", b.String(), w, queueH, m.runQueueFocused)
	var detail strings.Builder
	index := -1
	if m.runQueueFocused && len(m.msSelected) > 0 {
		index = m.msSelected[cursor]
	} else {
		filtered := m.msFiltered()
		if m.msCursor >= 0 && m.msCursor < len(filtered) {
			index = filtered[m.msCursor]
		}
	}
	if index >= 0 {
		m.writeCommandHover(&detail, m.msItems[index].cmd, w-2)
	}
	bottom := box("Command details", detail.String(), w, max(3, h-queueH-2), false)
	footer := "F2 Queue / List · Enter Review"
	if m.runQueueFocused {
		footer = "J/K Move · x Remove from queue · Esc List"
	}
	lines := strings.Split(top+"\n"+bottom, "\n")
	return strings.Join(lines[:min(max(0, h-1), len(lines))], "\n") + "\n" + gray(truncate(footer, w))
}
