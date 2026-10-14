package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Keep descriptive columns aligned without allowing a long value to
// consume the label column. Widths use terminal cells, including wide text.
func writeExploreColumns(b *strings.Builder, first, second string, rows [][2]string, w int) {
	if len(rows) == 0 {
		b.WriteString(" " + dim("(none)") + "\n")
		return
	}
	available := max(1, w-2)
	left := lipgloss.Width(first)
	for _, row := range rows {
		left = max(left, lipgloss.Width(row[0]))
	}
	left = min(left, max(1, (available-2)/2))
	right := available - left - 2
	line := func(a, c string, header bool) {
		if right < 4 {
			b.WriteString(" " + truncate(a, available) + "\n")
			return
		}
		a = truncate(a, left)
		padding := strings.Repeat(" ", max(0, left-lipgloss.Width(a))+2)
		c = truncate(c, right)
		if header {
			b.WriteString(" " + gray(a+padding+c) + "\n")
		} else {
			b.WriteString(" " + a + padding + dim(c) + "\n")
		}
	}
	line(first, second, true)
	for _, row := range rows {
		line(row[0], row[1], false)
	}
}
