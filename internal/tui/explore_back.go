package tui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

const exploreBackLabel = "[← Back]"

func (m Model) exploreBackRow() int {
	if m.termHeight() < 12 {
		return 0
	}
	return 1
}

func (m Model) writeExploreHeading(b *strings.Builder, w int) {
	var original strings.Builder
	m.writeHeading(&original, m.exploreTitle(), w)
	lines := strings.Split(original.String(), "\n")
	row := m.exploreBackRow()
	lines[row] = ansi.Truncate("  "+gray(exploreBackLabel)+"  "+accentBold(m.exploreTitle()), w, "…")
	b.WriteString(strings.Join(lines, "\n"))
}

func (m Model) exploreBackHit(x, y int) bool {
	return m.viewWidth() >= 10 && y == m.exploreBackRow() && x >= 2 && x < 2+ansi.StringWidth(exploreBackLabel)
}
