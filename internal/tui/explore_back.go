package tui

import "strings"

const exploreBackLabel = "[← Back]"

func (m Model) exploreBackText() string {
	if m.runHome.ready && m.exploreListFocused() && !m.exploreSearching && m.exploreSearch == "" && len(m.exploreHistory) == 0 {
		return "[← Back to Run]"
	}
	return exploreBackLabel
}
func (m Model) writeExploreHeading(b *strings.Builder, w int) {
	line, _ := workspaceHeading(m.exploreTitle(), m.exploreBackText(), w)
	b.WriteString(line + "\n")
	if m.workspaceTabRow() == 2 {
		b.WriteString(hline(w) + "\n")
	}
}
func (m Model) exploreBackHit(x, y int) bool {
	_, start := workspaceHeading(m.exploreTitle(), m.exploreBackText(), m.viewWidth())
	return start >= 0 && y == 0 && x >= start && x < m.viewWidth()-2
}
