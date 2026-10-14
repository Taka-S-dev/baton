package tui

func (m Model) recentStart(rows, total int) int {
	if m.recentManual {
		return min(max(0, m.recentOffset), max(0, total-rows))
	}
	return windowStart(rows, total, min(m.recentCursor, max(0, total-1)))
}
