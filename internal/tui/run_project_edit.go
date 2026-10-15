package tui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

const runProjectEditLabel = "[e Edit]"

func (m Model) candidateHeading(title string, w int) (string, int) {
	prefix := "  " + title
	if m.shortPanels() {
		prefix = title
	}
	start := ansi.StringWidth(prefix) + 2
	if !m.runHomeVisible() || start+len(runProjectEditLabel) > w {
		return white(ansi.Truncate(prefix, w, "…")), -1
	}
	return white(prefix) + "  " + gray(runProjectEditLabel), start
}

func (m Model) writeCandidateHeading(b *strings.Builder, title string, w int) {
	if m.runHomeVisible() {
		return
	}
	var original strings.Builder
	m.writeHeading(&original, title, w)
	lines := strings.Split(original.String(), "\n")
	row := 0
	if m.termHeight() >= 12 {
		row = 1
	}
	lines[row], _ = m.candidateHeading(title, w)
	b.WriteString(strings.Join(lines, "\n"))
}

func (m Model) runProjectEditHit(x, y int) bool {
	if m.runHomeVisible() || m.runDetailOpen {
		return false
	}
	w := m.viewWidth()
	if splitPanels(w) {
		w = m.runLeftWidth(w)
	} else if m.runQueueFocused {
		return false
	}
	_, start := m.candidateHeading("Select commands", w)
	row := 1
	if m.shortPanels() {
		row = 0
	}
	return start >= 0 && y == row && x >= start && x < start+len(runProjectEditLabel)
}
