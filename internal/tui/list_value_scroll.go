package tui

import "strings"

func (m Model) listValueScrollArea() runScrollArea {
	l := m.exploreLayout(m.viewWidth())
	rows := max(1, l.editH-2)
	n := len(m.le.entries) + 1
	return runScrollArea{7, l.rightX + 1, l.fieldY, l.rightW - 2, rows, n, min(max(0, m.le.scrollOffset), max(0, n-rows))}
}
func (m Model) writeListValueRows(b *strings.Builder, render func(int) string) {
	a := m.listValueScrollArea()
	for row := 0; row < a.rows; row++ {
		line := ""
		if a.offset+row < a.total {
			line = render(a.offset + row)
		}
		b.WriteString(candidateRow(line, a.width, a.total, a.rows, a.offset, row) + "\n")
	}
}
