package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

func (m Model) viewCompactCandidates(w int, hitRow ...func(int, int)) string {
	filtered := m.msFiltered()
	title := "Select commands"
	if m.screen == ScreenEditWorkflowCommands {
		title = "Edit commands"
	}
	title += fmt.Sprintf(" · %d selected", len(m.msSelected))
	rows := max(1, m.termHeight()-2)
	start := windowStart(rows, len(filtered), m.msCursor)
	search := m.commandSearchLine(w, len(filtered))
	lines := []string{accentBold(ansi.Truncate(title, w, "…")), ansi.Truncate(search, w, "…")}
	terms := strings.Fields(m.msSearchTI.Value())
	for row := 0; row < rows; row++ {
		i := start + row
		if i >= len(filtered) {
			if row == 0 {
				lines = append(lines, gray("No results."))
			} else {
				lines = append(lines, " ")
			}
			continue
		}
		for _, hit := range hitRow {
			hit(i, len(lines))
		}
		idx := filtered[i]
		item := m.msItems[idx]
		check := "[ ]"
		for order, selected := range m.msSelected {
			if selected == idx {
				check = fmt.Sprintf("[%d]", order+1)
				break
			}
		}
		label := check + " " + item.name()
		if item.cmd != nil && w >= 60 {
			nameWidth := min(28, w/3)
			label = check + " " + ansi.Truncate(item.name(), nameWidth, "…")
			label += strings.Repeat(" ", max(2, nameWidth+6-ansi.StringWidth(label)))
			label += strings.Join(strings.Fields(item.cmd.Cmd), " ")
		}
		// Highlight only command text so a search cannot recolor the selection marker.
		label = searchHighlight(strings.TrimPrefix(label, check+" "), terms, false, true)
		if i == m.msCursor {
			label = check + " " + label
		} else if check == "[ ]" {
			label = gray(check) + " " + label
		} else {
			label = sSelNum.Render(check) + " " + label
		}
		selected := i == m.msCursor
		if selected {
			label = "▶ " + label
		} else {
			label = "  " + label
		}
		label = ansi.Truncate(label, w, "…")
		if selected {
			label = sRowActive.Width(w).Render(ansi.Strip(label))
		}
		if selected {
			label = searchHighlight(label, terms, true, true)
		}
		lines = append(lines, label)
	}
	return strings.Join(lines[:min(len(lines), m.termHeight())], "\n")
}
