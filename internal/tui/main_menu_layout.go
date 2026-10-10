package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func (m Model) viewMainMenu(w int) string {
	contentW := min(w, 120)
	offset := 0
	lines := strings.Split(strings.TrimRight(m.viewMainMenuContent(contentW), "\n"), "\n")
	if len(lines) >= 2 {
		lines = lines[:len(lines)-2]
	}
	h := max(1, m.termHeight()-2)
	var b strings.Builder
	for i := 0; i < h; i++ {
		if i < len(lines) {
			b.WriteString(strings.Repeat(" ", offset) + ansi.Truncate(lines[i], contentW, "…"))
		}
		b.WriteByte('\n')
	}
	b.WriteString(hline(w) + "\n" + gray(truncate("  ↑↓ Move  Enter Open  Esc Quit  F1 Help", w)))
	return b.String()
}

func (m Model) updateMainMenuMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return m, nil
	}
	w := m.viewWidth()
	left := 0
	if msg.X < left || msg.X >= left+min(32, w) {
		return m, nil
	}
	lines := strings.Split(ansi.Strip(m.View()), "\n")
	if msg.Y < 0 || msg.Y >= len(lines) {
		return m, nil
	}
	line := []rune(lines[msg.Y])
	if len(line) <= left {
		return m, nil
	}
	end := min(len(line), left+min(32, w))
	label := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(string(line[left:end])), "▶"))
	for i, item := range m.listItems {
		if label == item {
			m.listCursor = i
			return m.updateMainMenu(tea.KeyMsg{Type: tea.KeyEnter})
		}
	}
	return m, nil
}
