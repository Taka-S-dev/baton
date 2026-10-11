package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func (m Model) updateRunSelectMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.msEscArmed || msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return m, nil
	}
	w := m.viewWidth()
	if msg.X < 0 || msg.X >= w || msg.Y < 0 {
		return m, nil
	}
	split := splitPanels(w)
	leftW := w
	if split {
		leftW = w / 2
	}
	if (split && msg.X > leftW) || (!split && m.runQueueFocused) {
		queueH := max(5, m.termHeight()/2)
		if m.shortPanels() {
			queueH = m.termHeight() + 1
		}
		if msg.Y < 1 || msg.Y >= queueH-1 {
			return m, nil
		}
		rows := max(1, queueH-3)
		if m.shortPanels() {
			rows = max(1, m.termHeight()-1)
		}
		start := windowStart(rows, len(m.msSelected), min(m.runQueueCursor, max(0, len(m.msSelected)-1)))
		row := msg.Y - 1
		index := start + row
		if row < rows && index < len(m.msSelected) {
			m.runQueueFocused = true
			m.runQueueCursor = index
		}
		return m, nil
	}
	if msg.X >= leftW {
		return m, nil
	}
	selected := -1
	headerY := -1
	searchY := -1
	rendered := m.viewMultiSelectCandidates(leftW, !split, func(index, y int) {
		if index == -2 {
			searchY = y
			return
		}
		if index < 0 {
			headerY = y
			return
		}
		if msg.Y == y {
			selected = index
		}
	})
	if msg.Y == searchY {
		m.runQueueFocused = false
		return m, m.msSearchTI.Focus()
	}
	if msg.Y == headerY {
		lines := strings.Split(ansi.Strip(rendered), "\n")
		if headerY >= 0 && headerY < len(lines) {
			line := lines[headerY]
			for _, col := range []struct{ label, mode string }{{"Run", "run"}, {"Name", "name"}, {"Group", "group"}, {"Type", "type"}} {
				start := strings.Index(line, col.label)
				if start >= 0 {
					start = lipgloss.Width(line[:start])
				}
				heading := m.sortHeading(col.mode, col.label)
				if col.mode == "run" {
					heading = "Run"
				}
				if start >= 0 && msg.X >= start && msg.X < start+lipgloss.Width(heading) {
					mode := col.mode
					if m.commandSort == mode {
						mode += "-desc"
					} else if m.commandSort == mode+"-desc" {
						mode = "definition"
					}
					m.setCommandSort(mode)
					m.runQueueFocused = false
					return m, nil
				}
			}
		}
	}
	if selected >= 0 {
		m.msCursor = selected
		m.runQueueFocused = false
	}
	return m, nil
}
