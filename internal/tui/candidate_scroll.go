package tui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

func (m Model) candidateStart(rows, n int) int {
	if m.msScrollManual {
		return min(max(0, m.msScrollOffset), max(0, n-rows))
	}
	return windowStart(rows, n, m.msCursor)
}

func candidateRow(line string, w, n, rows, start, row int) string {
	tail := "…"
	if ansi.StringWidth(strings.TrimRight(ansi.Strip(line), " ")) <= max(0, w-1) {
		tail = ""
	}
	line = ansi.Truncate(line, max(0, w-1), tail)
	line += strings.Repeat(" ", max(0, w-1-ansi.StringWidth(line)))
	bar := " "
	if n > rows {
		top, size := scrollbarThumb(n, rows, start)
		bar = gray("│")
		if row >= top && row < top+size {
			bar = accent("┃")
		}
	}
	return line + bar
}

func (m Model) writeCandidateWindow(b *strings.Builder, rows, n, w int, render func(int) string) {
	start := m.candidateStart(rows, n)
	for row := 0; row < rows; row++ {
		line := ""
		if start+row < n {
			line = render(start + row)
		} else if n == 0 && row == 0 {
			line = gray("  No results.")
		}
		b.WriteString(candidateRow(line, w, n, rows, start, row) + "\n")
	}
}

func (m Model) candidateScrollArea() runScrollArea {
	w := m.viewWidth()
	split := splitPanels(w)
	if split {
		w = m.runLeftWidth(w)
	} else if m.runQueueFocused {
		return runScrollArea{}
	}
	first, last := -1, -1
	m.viewMultiSelectCandidates(w, !split, func(i, y int) {
		if i >= 0 {
			if first < 0 {
				first = y
			}
			last = y
		}
	})
	rows := last - first + 1
	n := len(m.msFiltered())
	if first < 0 {
		return runScrollArea{}
	}
	return runScrollArea{3, 0, first, w, rows, n, m.candidateStart(rows, n)}
}
