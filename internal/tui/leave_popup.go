package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"strings"
)

// Layout uses the unmodified form's frame for both rendering and hit testing.
func (m Model) leavePopup(w int) actionPopupLayout {
	bg := m
	bg.leaveAsk = leaveNone
	height := min(m.rows(), frameRows(bg.renderScreen(w)))
	p := actionPopupLayout{w: min(52, w), h: min(8, height)}
	p.rows = max(1, p.h-2)
	if p.h >= 7 {
		p.rows -= 2
		p.footer = true
	}
	p.start = windowStart(p.rows, len(leaveChoices), m.leaveCursor)
	p.x, p.y = max(0, (w-p.w)/2), max(0, (height-p.h)/2)
	return p
}

func (m Model) overlayLeaveAsk(background string, w int) string {
	p := m.leavePopup(w)
	title := "Unsaved changes"
	if m.cf != nil {
		name := m.cf.fields[0]
		if m.cf.fieldIdx == 0 {
			name = m.nameInput.Value()
		}
		if name != "" {
			title += " · " + name
		}
	} else if m.wfe != nil && m.wfe.idx < len(m.workflows) {
		title += " · " + m.workflows[m.wfe.idx].Name
	}
	var b strings.Builder
	for i := p.start; i < len(leaveChoices) && i < p.start+p.rows; i++ {
		row := "   " + leaveChoices[i].key + "  " + leaveChoices[i].label
		if i == m.leaveCursor {
			row = sRowActive.Render(" ▶ " + leaveChoices[i].key + "  " + leaveChoices[i].label)
		}
		b.WriteString(row + "\n")
	}
	if p.footer {
		b.WriteString("\n ↑↓ Select  Enter Confirm  Esc Keep editing")
	}
	return overlayBox(background, box(title, b.String(), p.w, p.h, true), p.x, p.y, w)
}

func (m Model) updateLeaveMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	p := m.leavePopup(m.viewWidth())
	inside := msg.X >= p.x && msg.X < p.x+p.w && msg.Y >= p.y && msg.Y < p.y+p.h
	if msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress {
		if !inside {
			return m, m.updateLeave(tea.KeyMsg{Type: tea.KeyEsc})
		}
		row := msg.Y - p.y - 1
		if msg.X > p.x && msg.X < p.x+p.w-1 && row >= 0 && row < p.rows && p.start+row < len(leaveChoices) {
			m.leaveCursor = p.start + row
			return m, m.updateLeave(tea.KeyMsg{Type: tea.KeyEnter})
		}
	}
	return m, nil
}
