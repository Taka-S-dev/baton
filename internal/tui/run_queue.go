package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
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
	if m.shortPanels() {
		if splitPanels(w) {
			return joinPanels(m.viewCompactCandidates(w/2), m.shortRunQueue(w-w/2-3), w, w/2, m.termHeight())
		}
		if m.runQueueFocused {
			return m.shortRunQueue(w)
		}
		return m.viewCompactCandidates(w)
	}
	if !splitPanels(w) {
		if m.runQueueFocused {
			return m.viewRunQueue(w)
		}
		return m.viewMultiSelectList(w)
	}
	leftW := w / 2
	left := strings.TrimRight(m.viewMultiSelectCandidates(leftW, false), "\n")
	right := m.viewRunQueue(w - leftW - 1)
	leftLines := strings.Split(left, "\n")
	rightLines := strings.Split(right, "\n")
	leftLines = leftLines[:max(0, len(leftLines)-2)]
	rightLines = rightLines[:max(0, len(rightLines)-1)]
	rows := max(1, m.termHeight()-2)
	for len(leftLines) < rows {
		leftLines = append(leftLines, "")
	}
	for len(rightLines) < rows {
		rightLines = append(rightLines, "")
	}
	body := lipgloss.JoinHorizontal(lipgloss.Top, strings.Join(leftLines[:rows], "\n"), " ", strings.Join(rightLines[:rows], "\n"))
	return body + "\n" + hline(w) + "\n" + gray(truncate(m.runSelectKeys(), w))
}

func (m Model) viewRunQueue(w int) string {
	h := m.termHeight()
	queueH := max(5, h/2)
	var b strings.Builder
	cursor := min(m.runQueueCursor, max(0, len(m.msSelected)-1))
	writeWindowedList(&b, max(1, queueH-3), len(m.msSelected), cursor, "Tab adds from the list; Enter there reviews one command.", func(i int) string {
		label := " " + m.queueCommandLabel(i, max(1, w-3))
		if i == cursor && m.runQueueFocused {
			return m.selectedSyntaxRow(label, max(1, w-2))
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
	footer := m.runSelectKeys()
	lines := strings.Split(top+"\n"+bottom, "\n")
	return strings.Join(lines[:min(max(0, h-1), len(lines))], "\n") + "\n" + gray(truncate(footer, w))
}

func (m Model) runSelectKeys() string {
	if m.runHomeVisible() {
		return m.runHomeKeys()
	}
	if m.runQueueFocused {
		return "↑↓ Select  J/K Reorder  Del Remove  Enter Review  F2 List  Esc List  F1 Help"
	}
	return "↑↓ Select  Tab Toggle  Enter Review  F2 Queue  F3 Sort  Esc Back  F1 Help"
}
