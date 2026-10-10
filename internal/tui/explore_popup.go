package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

type actionPopupLayout struct {
	x, y, w, h, start, rows int
	footer                  bool
}

// One geometry definition drives painting and mouse hit testing.
func (m Model) actionPopup(w int, count int) actionPopupLayout {
	l := m.exploreLayout(w)
	p := actionPopupLayout{w: min(44, w), h: min(count+4, max(3, l.bodyH))}
	p.footer = p.h >= count+4
	p.rows = max(1, p.h-2)
	if p.footer {
		p.rows -= 2
	}
	p.start = windowStart(p.rows, count, m.exploreMenuCursor)
	p.x, p.y = max(0, (w-p.w)/2), max(0, l.bodyY+(l.bodyH-p.h)/2)
	if m.actionAtMouse {
		p.x = min(max(0, m.contextX), max(0, w-p.w))
		p.y = min(max(0, m.contextY), max(0, m.termHeight()-p.h))
	}
	return p
}

func (m Model) overlayExploreActions(background string, w int) string {
	it, ok := m.exploreCurrent()
	if !ok {
		return background
	}
	actions := m.exploreActions(it)
	p := m.actionPopup(w, len(actions))
	var b strings.Builder
	for i := p.start; i < len(actions) && i < p.start+p.rows; i++ {
		marker := "   "
		if i == m.exploreMenuCursor {
			marker = " ▶ "
		}
		row := marker + actions[i].key + "  " + actions[i].label
		if i == m.exploreMenuCursor {
			row = sRowActive.Render(row + strings.Repeat(" ", max(0, p.w-2-lipgloss.Width(row))))
		}
		b.WriteString(row + "\n")
	}
	if p.footer {
		b.WriteString("\n ↑↓ j/k Select  Enter Confirm  Esc Close")
	}
	return overlayBox(background, box("Actions · "+it.name, b.String(), p.w, p.h, true), p.x, p.y, w)
}

func overlayBox(background, content string, x, top, w int) string {
	popup := strings.Split(content, "\n")
	lines := strings.Split(background, "\n")
	for i, line := range popup {
		y := top + i
		if y >= len(lines) {
			break
		}
		base := lines[y]
		base += strings.Repeat(" ", max(0, w-lipgloss.Width(base)))
		lines[y] = ansi.Cut(base, 0, x) + line + ansi.Cut(base, x+lipgloss.Width(line), w)
	}
	return strings.Join(lines, "\n")
}

func (m Model) updateExploreActionMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	it, ok := m.exploreCurrent()
	if !ok {
		m.exploreMenu = false
		return m, nil
	}
	actions := m.exploreActions(it)
	p := m.actionPopup(m.viewWidth(), len(actions))
	inside := msg.X >= p.x && msg.X < p.x+p.w && msg.Y >= p.y && msg.Y < p.y+p.h
	if msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress {
		if !inside {
			m.exploreMenu = false
			return m, nil
		}
		row := msg.Y - p.y - 1
		i := p.start + row
		if msg.X > p.x && msg.X < p.x+p.w-1 && row >= 0 && row < p.rows && i < len(actions) {
			m.exploreMenuCursor = i
			return m, m.runExploreAction(actions[i].key, it)
		}
	}
	if inside {
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			m.exploreMenuCursor = max(0, m.exploreMenuCursor-1)
		case tea.MouseButtonWheelDown:
			m.exploreMenuCursor = min(len(actions)-1, m.exploreMenuCursor+1)
		}
	}
	return m, nil
}
