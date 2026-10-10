package tui

import (
	"strings"

	"github.com/Taka-S-dev/baton/internal/slot"
	"github.com/charmbracelet/x/ansi"
)

func (m Model) listValueDetailBox(l exploreLayout) string {
	le := m.le
	title := "Value details"
	value, label := "", ""
	selected := false
	switch {
	case le.adding:
		title = "New value · unsaved"
		value, label, selected = le.addVal, le.addLbl, true
	case le.editing:
		title = "Value details · editing"
		value, label, selected = le.editValTI.Value(), le.editLblTI.Value(), true
	case le.cursor >= 0 && le.cursor < len(le.entries):
		entry := le.entries[le.cursor]
		value, label, selected = entry.Value, entry.Label, true
	}
	var b strings.Builder
	write := func(name, text string) {
		line := gray(name) + text
		b.WriteString(" " + strings.ReplaceAll(ansi.Hardwrap(line, max(1, l.rightW-4), false), "\n", "\n ") + "\n")
	}
	if selected {
		write("Value > ", m.syntaxText(displayOr(value, "(empty)")))
		write("Label > ", displayOr(label, "(none)"))
		if resolved := slot.ApplyVars(value, m.vars); resolved != value {
			write("Resolved > ", resolved)
		}
		m.writeSyntaxWarnings(&b, value)
	} else {
		title = "Add value"
		write("", "Enter or a to add a value to this list.")
	}
	if selected {
		refs := m.listValueRefs(le.name, value)
		b.WriteString("\n" + hlineLabel(l.rightW-2, "Commands with this fixed value") + "\n")
		if len(refs) == 0 {
			write("", "No command has this value saved for this list.")
		} else {
			for _, ref := range refs {
				write("", ref)
			}
		}
		write("", "Saved values only; run-time choices are not tracked.")
	}
	return box(title, b.String(), l.rightW, l.prevH, false)
}
