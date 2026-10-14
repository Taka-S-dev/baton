package tui

import tea "github.com/charmbracelet/bubbletea"

// Coordinates are local to the content below the Run tabs.
type runScrollArea struct{ target, x, y, width, rows, total, offset int }

func scrollbarThumb(total, rows, offset int) (int, int) {
	size := min(rows, max(1, rows*rows/max(1, total)))
	return min(max(0, offset), max(0, total-rows)) * (rows - size) / max(1, total-rows), size
}

func (m Model) runScrollAreas() []runScrollArea {
	if m.screen == ScreenRecent {
		p := m.recentGeometry(m.viewWidth())
		return []runScrollArea{{8, p.x, p.listY, p.width, p.rows, len(m.recentFiltered()), p.start}}
	}
	if m.screen == ScreenEditList && m.le != nil && !m.le.adding && !m.le.editing && m.exploreValueFlow() {
		return []runScrollArea{m.listValueScrollArea()}
	}
	if m.screen == ScreenExplore {
		l := m.exploreLayout(m.viewWidth())
		x, w := 1, l.leftW-2
		if l.plain {
			x = 0
			w = l.leftW
		}
		return []runScrollArea{{6, x, l.listY, w, l.listH, len(l.items), l.start}}
	}
	if m.screen == ScreenRunWorkflow {
		return m.workflowScrollAreas()
	}
	w, h := m.viewWidth(), m.termHeight()
	x := 0
	if m.runDetailOpen {
		p := m.runDetailPopup()
		lines, _ := m.runDetailLines(p.w - 3)
		return []runScrollArea{{2, p.x + 1, p.y + 1, p.w - 2, max(1, p.h-3), len(lines), m.runDetailOffset}}
	}
	if splitPanels(w) {
		x = m.runLeftWidth(w) + 1
		w -= x
	} else if !m.runQueueFocused {
		return []runScrollArea{m.candidateScrollArea()}
	}
	qh := max(5, h/2)
	rows := max(1, qh-3)
	qx, qy, qw := x+1, 1, w-2
	if m.shortPanels() {
		rows = max(1, h-1)
		qx = x
		qy = 1
		qw = w
		if splitPanels(m.viewWidth()) {
			qx = m.runLeftWidth(m.viewWidth()) + 1
			qw = m.viewWidth() - qx
		}
	}
	a := []runScrollArea{{1, qx, qy, qw, rows, len(m.msSelected), m.queueStart(rows)}}
	if !m.shortPanels() {
		lines, key := m.runDetailLines(w - 3)
		offset := m.runDetailOffset
		if key != m.runDetailKey {
			offset = 0
		}
		a = append(a, runScrollArea{2, x + 1, qh + 1, w - 2, max(1, h-qh-4), len(lines), offset})
	}
	a = append(a, m.candidateScrollArea())
	return a
}

func (m *Model) setRunScroll(a runScrollArea, offset int) {
	offset = min(max(0, offset), max(0, a.total-a.rows))
	if a.target == 8 {
		m.recentManual = true
		m.recentOffset = offset
		return
	}
	if a.target == 7 {
		m.le.scrollOffset = offset
		return
	}
	if a.target == 6 {
		m.exploreScrollManual[m.exploreTab] = true
		m.exploreScroll[m.exploreTab] = offset
		return
	}
	if a.target == 4 {
		m.wfScrollManual = true
		m.wfScrollOffset = offset
		return
	}
	if a.target == 5 {
		m.wfDetailOffset = offset
		_, m.wfDetailKey = m.workflowDetailLines(a.width - 1)
		return
	}
	if a.target == 3 {
		m.msScrollManual = true
		m.msScrollOffset = offset
		return
	}
	if a.target == 1 {
		m.runQueueOffset = offset
		m.runQueueManual = true
		m.runQueueFocused = true
		m.runQueueCursor = min(max(0, a.total-1), offset+a.rows/2)
	} else {
		m.scrollRunDetail(offset - a.offset)
	}
}

func (m *Model) handleRunScrollMouse(msg tea.MouseMsg) bool {
	if msg.Action == tea.MouseActionRelease {
		was := m.runScrollDrag != 0
		m.runScrollDrag = 0
		return was
	}
	for _, a := range m.runScrollAreas() {
		if m.runScrollDrag == a.target && msg.Action == tea.MouseActionMotion {
			_, size := scrollbarThumb(a.total, a.rows, a.offset)
			track := a.rows - size
			if track > 0 {
				pos := min(max(0, msg.Y-a.y-m.runScrollGrab), track)
				m.setRunScroll(a, (pos*(a.total-a.rows)+track/2)/track)
			}
			return true
		}
		if msg.X < a.x || msg.X >= a.x+a.width || msg.Y < a.y || msg.Y >= a.y+a.rows {
			continue
		}
		delta := 0
		if msg.Button == tea.MouseButtonWheelUp {
			delta = -3
		}
		if msg.Button == tea.MouseButtonWheelDown {
			delta = 3
		}
		if delta != 0 {
			if a.target == 8 {
				if delta > 0 {
					delta = min(3, a.rows)
				} else {
					delta = -min(3, a.rows)
				}
			}
			if a.target == 3 || a.target == 4 || a.target == 5 || a.target == 6 || a.target == 7 || a.target == 8 {
				m.setRunScroll(a, a.offset+delta)
			} else if a.target == 1 {
				m.runQueueFocused = true
				m.runQueueCursor = min(max(0, a.total-1), max(0, m.runQueueCursor+delta))
				m.runQueueManual = true
				m.runQueueOffset = revealRow(a.offset, a.rows, a.total, m.runQueueCursor)
			} else {
				m.scrollRunDetail(delta)
			}
			return true
		}
		if a.total > a.rows && msg.X == a.x+a.width-1 && msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			top, size := scrollbarThumb(a.total, a.rows, a.offset)
			row := msg.Y - a.y
			if row >= top && row < top+size {
				m.runScrollDrag = a.target
				m.runScrollGrab = row - top
			} else if row < top {
				m.setRunScroll(a, a.offset-a.rows)
			} else {
				m.setRunScroll(a, a.offset+a.rows)
			}
			return true
		}
	}
	return false
}
