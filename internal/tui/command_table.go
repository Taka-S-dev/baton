package tui

import (
	"strings"

	mdl "github.com/Taka-S-dev/baton/internal/model"
	"github.com/Taka-S-dev/baton/internal/slot"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func commandKind(c mdl.Command) string {
	if c.Template != "" {
		return "From: " + c.Template
	}
	if slot.HasPlaceholders(c) {
		return "Template"
	}
	return "Direct"
}

// Use the same column widths for headers and rows. Narrow panes prioritize
// identity and type; group and command remain available in the detail pane.
func commandTableLine(it wsItem, w int) string {
	available := max(1, w-3)
	kindW := min(20, max(8, available/2))
	nameW := max(1, available-kindW-2)
	groupW, cmdW := 0, 0
	if available >= 64 {
		nameW = 16
		kindW = 20
		groupW = 12
		cmdW = available - nameW - kindW - groupW - 6
	}
	cell := func(s string, n int) string {
		s = ansi.Truncate(s, n, "…")
		return s + strings.Repeat(" ", max(0, n-lipgloss.Width(s)))
	}
	line := cell(it.name, nameW) + "  "
	if groupW > 0 {
		line += cell(displayOr(it.group, "-"), groupW) + "  "
	}
	line += cell(it.commandType, kindW)
	if cmdW > 0 {
		line += "  " + cell(strings.TrimPrefix(it.note, "$ "), cmdW)
	}
	return ansi.Truncate("   "+line, w, "…")
}

func commandTableRow(it wsItem, selected, focused bool, w int) string {
	line := commandTableLine(it, w)
	if selected {
		line = " ▶ " + strings.TrimPrefix(line, "   ")
		style := sRowInactive
		if focused {
			style = sRowActive
		}
		return style.Render(line)
	}
	if it.draft {
		return highlight(line)
	}
	return line
}

func (m Model) writeCommandIdentity(b *strings.Builder, c mdl.Command, w int) {
	kind := commandKind(c)
	if c.Template != "" {
		kind = "From template"
	}
	b.WriteString("\n " + gray("Type > ") + kind + "\n")
	if c.Template == "" {
		return
	}
	b.WriteString(" " + dim("Enter → Change values edits fixed values") + "\n")
	b.WriteString(" " + gray("Template > ") + truncate(c.Template, max(1, w-13)) + "\n")
	for _, name := range sortedKeys(c.Values) {
		b.WriteString(" " + gray("Fixed > ") + truncate(name+" = "+c.Values[name], max(1, w-10)) + "\n")
	}
}

// Candidate rows use explicit labels, leaving execution order to the queue.
func selectCommandColumns(name, group, kind string, w int) string {
	groupW := 0
	if w >= 48 {
		groupW = 12
	}
	kindW := min(20, max(8, w/2))
	nameW := max(1, w-kindW-2)
	if groupW > 0 {
		nameW = max(1, nameW-groupW-2)
	}
	cell := func(value string, width int) string {
		value = ansi.Truncate(value, width, "…")
		return value + strings.Repeat(" ", max(0, width-lipgloss.Width(value)))
	}
	line := cell(name, nameW) + "  "
	if groupW > 0 {
		line += cell(displayOr(group, "-"), groupW) + "  "
	}
	return ansi.Truncate(line+cell(kind, kindW), w, "…")
}
