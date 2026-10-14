package tui

import tea "github.com/charmbracelet/bubbletea"

// Keep the viewport anchored; keyboard navigation only reveals an off-screen row.
func revealRow(start, rows, total, cursor int) int {
	if cursor < start {
		start = cursor
	} else if cursor >= start+rows {
		start = cursor - rows + 1
	}
	return min(max(0, start), max(0, total-rows))
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	before := m
	if m.screen == ScreenRunCommands {
		bg := m
		bg.runDetailOpen = false
		for _, a := range bg.runScrollAreas() {
			if a.target == 1 {
				m.runQueueOffset = a.offset
				m.runQueueManual = true
			}
		}
	}
	switch m.screen {
	case ScreenRecent:
		p := m.recentGeometry(m.viewWidth())
		m.recentOffset = p.start
		m.recentManual = true
	case ScreenExplore:
		l := m.exploreLayout(m.viewWidth())
		m.exploreScroll[m.exploreTab] = l.start
		m.exploreScrollManual[m.exploreTab] = true
	case ScreenRunCommands, ScreenCreateWorkflow, ScreenEditWorkflowCommands:
		a := m.candidateScrollArea()
		if a.rows > 0 {
			m.msScrollOffset = a.offset
			m.msScrollManual = true
		}
	case ScreenRunWorkflow:
		a := m.workflowScrollAreas()[0]
		m.wfScrollOffset = a.offset
		m.wfScrollManual = true
	}
	next, cmd := m.updateFrame(msg)
	var result *Model
	switch v := next.(type) {
	case Model:
		result = &v
	case *Model:
		result = v
	default:
		return next, cmd
	}
	_, keyboard := msg.(tea.KeyMsg)
	if result.screen == before.screen && keyboard {
		switch result.screen {
		case ScreenRecent:
			p := result.recentGeometry(result.viewWidth())
			result.recentOffset = revealRow(p.start, p.rows, len(result.recentFiltered()), min(result.recentCursor, max(0, len(result.recentFiltered())-1)))
			result.recentManual = true
		case ScreenEditList:
			if result.le != nil && !result.le.adding && !result.le.editing && result.exploreValueFlow() {
				a := result.listValueScrollArea()
				result.le.scrollOffset = revealRow(a.offset, a.rows, a.total, result.le.cursor)
			}
		case ScreenExplore:
			if result.exploreTab == before.exploreTab && result.exploreListFocused() {
				l := result.exploreLayout(result.viewWidth())
				result.exploreScroll[result.exploreTab] = revealRow(l.start, l.listH, len(l.items), l.cursor)
			}
		case ScreenRunCommands, ScreenCreateWorkflow, ScreenEditWorkflowCommands:
			if result.runQueueFocused {
				bg := *result
				bg.runDetailOpen = false
				for _, q := range bg.runScrollAreas() {
					if q.target == 1 {
						result.runQueueOffset = revealRow(q.offset, q.rows, q.total, result.runQueueCursor)
					}
				}
			}
			a := result.candidateScrollArea()
			if a.rows > 0 && !result.runQueueFocused {
				result.msScrollOffset = revealRow(a.offset, a.rows, a.total, result.msCursor)
			}
		case ScreenRunWorkflow:
			a := result.workflowScrollAreas()[0]
			result.wfScrollOffset = revealRow(a.offset, a.rows, a.total, result.listCursor)
		}
	}
	return *result, cmd
}

func (m Model) queueStart(rows int) int {
	if m.runQueueManual {
		return min(max(0, m.runQueueOffset), max(0, len(m.msSelected)-rows))
	}
	return windowStart(rows, len(m.msSelected), m.runQueueCursor)
}
