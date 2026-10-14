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
	if m.runDetailOpen {
		return m.viewRunDetailPopup(w)
	}
	if m.msEscArmed && len(m.msSelected) > 0 {
		names := []string{}
		for _, i := range m.msSelected {
			names = append(names, m.msItems[i].name())
		}
		return m.viewDiscardWindow(w, names)
	}
	if m.shortPanels() {
		if splitPanels(w) {
			return lipgloss.JoinHorizontal(lipgloss.Top, m.viewCompactCandidates(m.runLeftWidth(w)), m.runDivider(m.termHeight()), m.shortRunQueue(w-m.runLeftWidth(w)-1))
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
	leftW := m.runLeftWidth(w)
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
	body := lipgloss.JoinHorizontal(lipgloss.Top, strings.Join(leftLines[:rows], "\n"), m.runDivider(rows), strings.Join(rightLines[:rows], "\n"))
	return body + "\n" + hline(w) + "\n" + gray(truncate(m.runSelectKeys(), w))
}

func (m Model) viewRunQueue(w int) string {
	h := m.termHeight()
	queueH := max(5, h/2)
	cursor := min(m.runQueueCursor, max(0, len(m.msSelected)-1))
	rows := max(1, queueH-3)
	start := m.queueStart(rows)
	var queue []string
	for i := range m.msSelected {
		label := " " + m.queueCommandLabel(i, max(1, w-4))
		if i == cursor && m.runQueueFocused {
			label = m.selectedSyntaxRow(label, max(1, w-3))
		}
		queue = append(queue, label)
	}
	if len(queue) == 0 {
		queue = []string{"Tab adds from the list."}
	}
	top := box("Run queue · execution order", scrollRows(queue, start, rows, w-2), w, queueH, m.runQueueFocused)
	bottom := m.runDetailBox(w, max(3, h-queueH-2))
	footer := m.runSelectKeys()
	lines := strings.Split(top+"\n"+bottom, "\n")
	return strings.Join(lines[:min(max(0, h-1), len(lines))], "\n") + "\n" + gray(truncate(footer, w))
}

func (m Model) runSelectKeys() string {
	if m.runHomeVisible() {
		return m.runHomeKeys()
	}
	if m.runQueueFocused {
		return "↑↓ Select  J/K Reorder  Del Remove  d Details · PgUp/PgDn Scroll  Enter Review  F2 List  Esc List  F1 Help"
	}
	return "↑↓ Select  Tab Toggle  Enter Review  F2 Queue  d Details  s Sort  / Search  Esc Back  F1 Help"
}
