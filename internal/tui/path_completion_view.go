package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// Keep the original input and command context in place. The completion list
// is anchored to that input, not a second screen with its own search field.
func (m Model) pathCompletionBackground(w int) string {
	bg := m
	sp := *m.sp
	sp.completion = nil
	sp.previewOnly = true
	sp.picked = append([]string(nil), m.sp.completion.draft...)
	bg.sp = &sp
	background := bg.renderScreen(w)
	if frameRows(background) > m.termHeight() {
		input := "/ " + sp.search + "_"
		if sp.search == "" {
			input = "/ type to filter…"
		}
		background = accentBold("  "+sp.placeholder()) + "\n  " + input
	}
	lines := strings.Split(strings.TrimRight(background, "\n"), "\n")
	for i, line := range lines {
		plain := ansi.Strip(line)
		if strings.Contains(plain, "Tab: open choices") || strings.Contains(plain, "Tab: choices  Enter:") {
			if strings.Contains(plain, "Tab: choices  Enter: confirm") && i > 0 {
				lines[i-1] = ""
			}
			line = ""
		}
		if sp.search == "" {
			line = strings.ReplaceAll(line, sp.inputHint(), "/ type to filter…")
		}
		lines[i] = ansi.Truncate(line, max(1, w), "…")
	}
	for len(lines) < m.termHeight() {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func (m Model) pathCompletionAnchor(w int) (int, int) {
	for y, line := range strings.Split(m.pathCompletionBackground(w), "\n") {
		plain := ansi.Strip(line)
		if i := strings.Index(plain, "/ "); i >= 0 && (strings.Contains(plain, "/ type to filter") || strings.Contains(plain, "/ "+m.sp.search+"_")) {
			return ansi.StringWidth(plain[:i]), y + 1
		}
	}
	return 2, max(1, m.termHeight()/2)
}

func (m Model) pathCompletionLayout(w int) actionPopupLayout {
	x, anchor := m.pathCompletionAnchor(w)
	height := max(1, m.termHeight())
	width := min(64, max(3, w-x-2))
	desired := min(10, max(4, len(m.sp.completion.entries)+3))
	y, h := anchor, min(desired, height-anchor)
	if h < 4 && anchor > height-anchor {
		h = min(desired, anchor-1)
		y = anchor - 1 - h
	}
	h = max(1, min(h, height))
	p := actionPopupLayout{x: min(x, max(0, w-width)), y: max(0, min(y, height-h)), w: min(width, w), h: h, rows: max(1, h-3)}
	if m.termHeight() < 18 {
		p.rows = max(1, p.h-2)
	}
	p.start = windowStart(p.rows, len(m.sp.completion.entries), m.sp.completion.cursor)
	return p
}

const pathCandidateRowOffset = 1 // only the border precedes the candidates

func (m Model) overlayPathCompletion(w int) string {
	p := m.pathCompletionLayout(w)
	c := m.sp.completion
	var b strings.Builder
	for row := 0; row < p.rows; row++ {
		line := ""
		i := p.start + row
		switch {
		case c.loading:
			if row == 0 {
				line = " Reading directory…"
			}
		case c.err != nil:
			if row == 0 {
				line = " " + truncate(c.err.Error(), max(1, p.w-4))
			}
		case len(c.entries) == 0:
			if row == 0 {
				line = " No matching files"
				if m.sp.pathDef.PathKind == "dir" {
					line = " No matching folders"
				}
			}
		case i < len(c.entries):
			label := c.entries[i].value
			if m.sp.variadic {
				mark := "[ ] "
				if !m.pathCandidateSelectable(c.entries[i]) {
					mark = " →  "
				}
				if c.selected(label) {
					mark = "[x] "
				}
				label = mark + label
			}
			if !m.sp.variadic && c.entries[i].directory {
				label = "→ " + label
			}
			if c.entries[i].parent {
				label = "← ../ (Parent folder)"
			}
			line = "   " + truncate(label, max(1, p.w-6))
			if i == c.cursor {
				line = sRowActive.Render(" ▶ " + truncate(label, max(1, p.w-6)))
			}
		}
		b.WriteString(line + "\n")
	}
	footer := "↑↓ Move  Enter Confirm  Esc Close"
	if m.sp.pathDef.PathKind == "dir" {
		footer = "↑↓ Move  → Open  Enter Confirm  Esc Close"
	}
	if m.sp.variadic {
		footer = "Tab Check  Enter Confirm  Esc Close"
		if p.w >= 60 {
			footer = "↑↓ Move  Tab Check  Enter Confirm  Esc Close"
			if m.sp.pathDef.PathKind == "dir" {
				footer = "↑↓ Move  Tab Check  → Open  Enter Confirm  Esc Close"
			}
		}
	}
	if !c.loading && c.err == nil && len(c.entries) > 0 && !m.pathCandidateSelectable(c.entries[c.cursor]) {
		footer = "↑↓ Move  Enter/→ Open folder  Esc Close"
	}
	if len(c.entries) > 0 && c.entries[c.cursor].parent {
		footer = "Enter/← Parent folder  Esc Close"
	}
	if m.sp.search != "" {
		footer = strings.ReplaceAll(footer, "Esc Close", "Esc Clear filter")
	}
	if len([]rune(footer)) > p.w-4 {
		footer = strings.TrimPrefix(footer, "↑↓ Move  ")
	}
	if m.termHeight() >= 18 {
		b.WriteString(" " + truncate(footer, max(1, p.w-4)))
	}
	kind := "Files"
	if m.sp.pathDef.PathKind == "dir" {
		kind = "Folders"
	}
	title := fmt.Sprintf("%s · %d candidates", kind, len(c.entries))
	if m.sp.variadic {
		title = fmt.Sprintf("%s · %d selected · %d candidates", kind, len(c.draft), len(c.entries))
	}
	return overlayBox(m.pathCompletionBackground(w), box(title, b.String(), p.w, p.h, true), p.x, p.y, w)
}

func (m Model) updatePathCompletionMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	p := m.pathCompletionLayout(m.viewWidth())
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		return m.updatePathCompletion(tea.KeyMsg{Type: tea.KeyUp})
	case tea.MouseButtonWheelDown:
		return m.updatePathCompletion(tea.KeyMsg{Type: tea.KeyDown})
	case tea.MouseButtonLeft:
		if msg.Action != tea.MouseActionPress {
			return m, nil
		}
		_, anchor := m.pathCompletionAnchor(m.viewWidth())
		if msg.Y == anchor-1 {
			return m, nil
		}
		if msg.X < p.x || msg.X >= p.x+p.w || msg.Y < p.y || msg.Y >= p.y+p.h {
			m.closePathCompletion()
			return m, nil
		}
		row := msg.Y - p.y - pathCandidateRowOffset
		c := m.sp.completion
		if !c.loading && c.err == nil && msg.X > p.x && msg.X < p.x+p.w-1 && row >= 0 && row < p.rows && p.start+row < len(c.entries) {
			c.cursor = p.start + row
			if m.sp.variadic && m.pathCandidateSelectable(c.entries[c.cursor]) {
				return m.updatePathCompletion(tea.KeyMsg{Type: tea.KeyTab})
			}
			if !m.sp.variadic && m.pathCandidateSelectable(c.entries[c.cursor]) {
				return m.updatePathCompletion(tea.KeyMsg{Type: tea.KeyTab})
			}
			return m.updatePathCompletion(tea.KeyMsg{Type: tea.KeyEnter})
		}
	}
	return m, nil
}
