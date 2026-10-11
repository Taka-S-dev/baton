package tui

import (
	"fmt"
	"strings"

	"github.com/Taka-S-dev/baton/internal/slot"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	n := m.sp.valueSkipRow()
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
	case sp.canSkip && i == sp.valueSkipRow():
		return "→ skip — resolve at run time"
	case i == len(sp.filtered):
		if sp.search == "" {
			return ""
		}
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
	input.Placeholder = "Filter or type a value…"
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
				line = pickerSelectedRow(" ▶ "+label, p.w)
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

// pickerContextLines is what a choice depends on: the step being
// answered, and the command as it would run with the hovered value.
// It is all that accompanies either picker, beside the box on a wide
// terminal and above it on a narrow one, so the values get the rows.
func (m Model) pickerContextLines(w int) []string {
	sp := m.sp
	lines := []string{}
	if sp.contextNames != nil {
		step := fmt.Sprintf("▶ %d. %s", sp.contextIdx+1, sp.contextNames[sp.contextIdx])
		if n := len(sp.contextNames); n > 1 {
			step += "   " + fmt.Sprintf("%d of %d", sp.contextIdx+1, n)
		}
		lines = append(lines, " "+accentBold(truncate(step, max(1, w-2))))
	}
	if sp.currentCmd != nil {
		hovered := ""
		if sp.cursor >= 0 && sp.cursor < len(sp.filtered) {
			hovered = sp.filtered[sp.cursor].Value
		} else if sp.cursor == len(sp.filtered) && sp.search != "" {
			hovered = sp.search
		}
		if c := sp.completion; c != nil {
			hovered = ""
			if !c.loading && c.err == nil && c.cursor >= 0 && c.cursor < len(c.entries) && m.pathCandidateSelectable(c.entries[c.cursor]) {
				hovered = c.entries[c.cursor].value
			}
		}
		picked := sp.picked
		if sp.completion != nil {
			picked = sp.completion.draft
		}
		preview := func(text string) string {
			for k, v := range sp.resolvedSoFar {
				text = slot.Replace(text, k, v)
			}
			switch {
			case sp.variadic && len(picked) > 0:
				text = slot.Replace(text, sp.slotName, highlight(strings.Join(picked, " ")))
			case hovered != "":
				text = slot.Replace(text, sp.slotName, highlight(hovered))
			default:
				text = slot.Replace(text, sp.slotName, slotVar(sp.placeholder()))
			}
			return slot.ApplyVars(text, m.vars)
		}
		lines = append(lines, " "+gray("$")+" "+preview(sp.currentCmd.Cmd))
		if sp.currentCmd.Dir != "" {
			lines = append(lines, " "+gray("workdir:")+" "+preview(sp.currentCmd.Dir))
		}
	}
	return lines
}

func (m Model) pickerFrame(w int) (int, int, string) {
	if splitPanels(w) && m.termHeight() >= 6 {
		return w / 2, 0, ""
	}
	lines := m.pickerContextLines(w)
	for len(lines) > 0 && m.termHeight()-len(lines) < 7 {
		lines = lines[:len(lines)-1]
	}
	for i := range lines {
		lines[i] = ansi.Truncate(lines[i], max(1, w), "…")
	}
	return w, len(lines), strings.Join(lines, "\n")
}

func (m Model) finishPickerFrame(w, leftW int, left string) string {
	if leftW == w {
		return left
	}
	rightW := w - leftW - 1
	lines := m.pickerContextLines(rightW)
	for i := range lines {
		lines[i] = ansi.Truncate(lines[i], max(1, rightW-4), "…")
	}
	right := box("Command preview", strings.Join(lines, "\n"), rightW, m.termHeight(), false)
	return lipgloss.JoinHorizontal(lipgloss.Top, left, " ", right)
}

func (m Model) slotValuesScreen(w int) (string, *actionPopupLayout) {
	leftW, anchor, bg := m.pickerFrame(w)
	p := m.slotValuesLayout(leftW, anchor)
	return m.finishPickerFrame(w, leftW, m.overlaySlotValues(leftW, bg, anchor)), &p
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

// Empty custom input is not a selectable row in a list picker.
func (sp *slotPickState) valueSkipRow() int {
	n := len(sp.filtered)
	if sp.search != "" || sp.pathDef.PathKind != "" {
		n++
	}
	return n
}

// pickerSelectedRow renders the cursor row of a picker box across its
// whole inner width, so the highlight reaches both borders rather than
// stopping where the label ends.
func pickerSelectedRow(text string, width int) string {
	inner := max(0, width-2)
	text = ansi.Truncate(text, inner, "…")
	return sRowActive.Render(text + strings.Repeat(" ", max(0, inner-ansi.StringWidth(text))))
}
