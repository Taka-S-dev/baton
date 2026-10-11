package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// The values of a list slot are offered in the same box as the files of
// a path slot: a title with the count, a search line, the rows and a
// footer, drawn over the step context and the command preview. One
// picker to learn, whatever the slot holds.

const slotValueRowOffset = 3 // border, search, separator

// slotValueRows is how many rows the picker lists: the matching
// entries, the typed value, and the skip row while creating a command.
func (m Model) slotValueRows() int {
	n := len(m.sp.filtered) + 1
	if m.sp.canSkip {
		n++
	}
	return n
}

// slotValuesLayout places the box under the background's anchor line and
// gives it the rest of the terminal.
func (m Model) slotValuesLayout(w, anchor int) actionPopupLayout {
	height := max(1, m.termHeight())
	h := max(1, height-anchor)
	reserved := 4 // borders, search line, separator
	if m.termHeight() >= 18 {
		reserved++ // footer
	}
	if m.sp.reuseFrom != "" {
		reserved++
	}
	p := actionPopupLayout{x: 0, y: anchor, w: max(1, w), h: h, rows: max(1, h-reserved)}
	p.start = windowStart(p.rows, m.slotValueRows(), m.sp.cursor)
	return p
}

// slotValueLabel is the text of row i without any styling.
func (m Model) slotValueLabel(i, labelCol int) string {
	sp := m.sp
	switch {
	case sp.canSkip && i == len(sp.filtered)+1:
		return "→ skip — resolve at run time"
	case i == len(sp.filtered):
		return sp.typedValueLabel()
	}
	e := sp.filtered[i]
	text := e.Value
	if e.Label != "" {
		text += strings.Repeat(" ", max(1, labelCol-len(e.Value))) + "·  " + e.Label
	}
	if sp.variadic {
		if sp.isPicked(e.Value) {
			return "[x] " + text
		}
		return "[ ] " + text
	}
	return text
}

func (m Model) overlaySlotValues(w int, background string, anchor int) string {
	sp := m.sp
	p := m.slotValuesLayout(w, anchor)
	inner := max(1, p.w-4)
	var b strings.Builder

	input := sp.input
	input.Prompt = "Search / "
	input.Placeholder = "Filter values…"
	if input.Value() != sp.search {
		input.SetValue(sp.search)
		input.CursorEnd()
	}
	input.Width = max(1, inner-1-len(input.Prompt))
	b.WriteString(" " + ansi.Truncate(input.View(), inner, "…") + "\n")
	if sp.reuseFrom != "" {
		b.WriteString(" " + dim(truncate("same as "+sp.reuseFrom+" — Enter keeps it", inner)) + "\n")
	}
	b.WriteString(" " + hline(inner) + "\n")

	labelCol := 0
	for _, e := range sp.filtered {
		if e.Label != "" && len(e.Value)+2 > labelCol {
			labelCol = len(e.Value) + 2
		}
	}
	total := m.slotValueRows()
	for row := 0; row < p.rows; row++ {
		i := p.start + row
		line := ""
		if i < total {
			label := truncate(m.slotValueLabel(i, labelCol), max(1, p.w-6))
			switch {
			case i == sp.cursor:
				line = sRowActive.Render(" ▶ " + label)
			case i >= len(sp.filtered):
				line = "   " + dim(label)
			default:
				line = "   " + label
			}
		}
		b.WriteString(line + "\n")
	}

	footer := "↑↓ Move  Enter Select  Esc Back"
	if sp.variadic {
		footer = "↑↓ Move  Tab Check  Enter Select  Esc Back"
	}
	if sp.search != "" {
		footer = strings.ReplaceAll(footer, "Esc Back", "Esc Clear filter")
	}
	if len([]rune(footer)) > inner {
		footer = strings.TrimPrefix(footer, "↑↓ Move  ")
	}
	if m.termHeight() >= 18 {
		b.WriteString(" " + truncate(footer, inner))
	}

	title := fmt.Sprintf("%s · %d candidates", sp.placeholder(), len(sp.filtered))
	if len(sp.entries) == 0 {
		title = sp.placeholder() + " · type a value"
	}
	if sp.variadic {
		title = fmt.Sprintf("%s · %d selected · %d candidates", sp.placeholder(), len(sp.picked), len(sp.filtered))
	}
	background += strings.Repeat("\n", max(0, m.termHeight()-strings.Count(background, "\n")-1))
	return overlayBox(background, box(title, b.String(), p.w, p.h, true), p.x, p.y, w)
}

// updateSlotValuesMouse selects the clicked row, and chooses it on a
// second click; a click elsewhere in the box does nothing.
func (m Model) updateSlotValuesMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return m, nil
	}
	_, p := m.slotPickRender(m.viewWidth())
	if p == nil {
		return m, nil
	}
	offset := slotValueRowOffset
	if m.sp.reuseFrom != "" {
		offset++
	}
	row := msg.Y - p.y - offset
	if msg.X < p.x || msg.X >= p.x+p.w || row < 0 || row >= p.rows || p.start+row >= m.slotValueRows() {
		return m, nil
	}
	if m.sp.cursor == p.start+row {
		return m.updateSlotPick(tea.KeyMsg{Type: tea.KeyEnter})
	}
	m.sp.cursor = p.start + row
	return m, nil
}
