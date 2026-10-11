package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// The picker owns its filter; the background only supplies command context.
func (m Model) pathCompletionBackground(w int) string {
	bg := m
	sp := *m.sp
	sp.completion = nil
	sp.previewOnly = true
	sp.picked = append([]string(nil), m.sp.completion.draft...)
	bg.sp = &sp
	lines := strings.Split(bg.renderScreen(w), "\n")
	end := len(lines)
	for i, line := range lines {
		plain := ansi.Strip(line)
		if strings.Contains(plain, "Select value") && !strings.Contains(plain, "[") || strings.HasPrefix(strings.TrimSpace(plain), "/ ") {
			end = i
			break
		}
	}
	lines = lines[:end]
	for len(lines) > 0 && strings.TrimSpace(ansi.Strip(lines[len(lines)-1])) == "" {
		lines = lines[:len(lines)-1]
	}
	maxContext := max(0, m.termHeight()-6)
	if len(lines) > maxContext {
		compact := []string{accentBold("  " + sp.placeholder())}
		for _, line := range lines {
			if strings.HasPrefix(strings.TrimSpace(ansi.Strip(line)), "$ ") {
				compact = append(compact, line)
				break
			}
		}
		lines = compact[:min(len(compact), maxContext)]
	}
	for i := range lines {
		lines[i] = ansi.Truncate(lines[i], max(1, w), "…")
	}
	return strings.Join(lines, "\n")
}

func (m Model) pathCompletionAnchor(w int) (int, int) {
	bg := m.pathCompletionBackground(w)
	if bg == "" {
		return 0, 0
	}
	return 0, strings.Count(bg, "\n") + 1
}

func (m Model) pathCompletionLayout(w int) actionPopupLayout {
	x, anchor := m.pathCompletionAnchor(w)
	height := max(1, m.termHeight())
	h := max(1, height-anchor)
	reserved := 4
	if m.termHeight() >= 18 {
		reserved++
	}
	p := actionPopupLayout{x: x, y: anchor, w: max(1, w), h: h, rows: max(1, h-reserved)}
	p.start = windowStart(p.rows, len(m.sp.completion.entries), m.sp.completion.cursor)
	return p
}

const pathCandidateRowOffset = 3 // border, filter, separator

func (m Model) overlayPathCompletion(w int) string {
	p := m.pathCompletionLayout(w)
	c := m.sp.completion
	var b strings.Builder
	input := c.input
	input.Prompt = "Search / "
	input.Placeholder = "Filter files…"
	if m.sp.pathDef.PathKind == "dir" {
		input.Placeholder = "Filter folders…"
	}
	input.Width = max(1, p.w-5-ansi.StringWidth(input.Prompt))
	b.WriteString(" " + ansi.Truncate(input.View(), max(1, p.w-4), "…") + "\n")
	b.WriteString(" " + hline(max(1, p.w-4)) + "\n")
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
	footer := "↑↓ Move  Enter Select  Esc Back"
	if m.sp.pathDef.PathKind == "dir" {
		footer = "↑↓ Move  → Open  Enter Select  Esc Back"
	}
	if m.sp.variadic {
		footer = "Tab Check  Enter Select  Esc Back"
		if p.w >= 60 {
			footer = "↑↓ Move  Tab Check  Enter Select  Esc Back"
			if m.sp.pathDef.PathKind == "dir" {
				footer = "↑↓ Move  Tab Check  → Open  Enter Select  Esc Back"
			}
		}
	}
	if !c.loading && c.err == nil && len(c.entries) > 0 && !m.pathCandidateSelectable(c.entries[c.cursor]) {
		footer = "↑↓ Move  Enter/→ Open folder  Esc Back"
	}
	if len(c.entries) > 0 && c.entries[c.cursor].parent {
		footer = "Enter/← Parent folder  Esc Back"
	}
	if m.sp.search != "" {
		footer = strings.ReplaceAll(footer, "Esc Back", "Esc Clear filter")
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
	background := m.pathCompletionBackground(w)
	background += strings.Repeat("\n", max(0, m.termHeight()-strings.Count(background, "\n")-1))
	return overlayBox(background, box(title, b.String(), p.w, p.h, true), p.x, p.y, w)
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
		if msg.Y == p.y+1 && msg.X >= p.x && msg.X < p.x+p.w {
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
