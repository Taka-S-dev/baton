package tui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

func (m Model) runDetailPopup() actionPopupLayout {
	w, h := m.viewWidth(), m.termHeight()
	pw := min(90, max(4, w-4))
	if w < 8 {
		pw = w
	}
	lines, _ := m.runDetailLines(pw - 3)
	ph := min(max(4, len(lines)+3), max(4, h-2))
	ph = min(ph, h)
	p := actionPopupLayout{x: max(0, (w-pw)/2), y: max(0, (h-ph)/2), w: pw, h: ph}
	bg := m
	bg.runDetailOpen = false
	target, cursor := 3, m.msCursor
	if m.runQueueFocused {
		target = 1
		cursor = m.runQueueCursor
	}
	for _, a := range bg.runScrollAreas() {
		if a.target != target || cursor < a.offset || cursor >= a.offset+a.rows {
			continue
		}
		row := a.y + cursor - a.offset
		below, above := h-row-1, row
		if below >= ph {
			p.y = row + 1
			return p
		}
		if above >= ph {
			p.y = row - ph
			return p
		}
		if splitPanels(w) {
			left := m.runLeftWidth(w)
			if target == 1 {
				p.x = 0
				p.w = left
			} else {
				p.x = left + 1
				p.w = w - p.x
			}
			lines, _ = m.runDetailLines(p.w - 3)
			p.h = min(h, max(4, len(lines)+3))
			p.y = max(0, min(row, h-p.h))
			return p
		}
		if max(above, below) >= 4 {
			if below >= above {
				p.y = row + 1
				p.h = below
			} else {
				p.y = 0
				p.h = above
			}
			return p
		}
	}
	return p
}

func (m Model) viewRunDetailPopup(w int) string {
	bg := m
	bg.runDetailOpen = false
	background := bg.viewMultiSelect(w)
	p := m.runDetailPopup()
	panel := m.runDetailBox(p.w, max(3, p.h-1)) + "\n" + gray(ansi.Truncate("↑↓ Scroll · Esc Close", p.w, "…"))
	// Preserve the list underneath without introducing a persistent footer.
	lines := strings.Split(background, "\n")
	for len(lines) < m.termHeight() {
		lines = append(lines, "")
	}
	return overlayBox(strings.Join(lines, "\n"), panel, p.x, p.y, w)
}
