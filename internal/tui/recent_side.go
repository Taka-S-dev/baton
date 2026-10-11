package tui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

func (m Model) recentSidePreview(w int) string {
	lines := []string{gray("Selected command")}
	filtered := m.recentFiltered()
	if m.recentCursor < len(filtered) {
		row := m.recentRows[filtered[m.recentCursor]]
		lines = append(lines, row.ref.Command)
		if row.problem != "" {
			lines = append(lines, warn(row.problem))
		} else {
			lines = append(lines, "$ "+recentHighlight(row.command.Cmd, m.recentSearch, false), gray("In: "+m.confirmWorkdir(row.command.Dir)))
		}
	} else {
		lines = append(lines, gray("Browse projects to choose a command."))
	}
	for i := range lines {
		lines[i] = ansi.Truncate(lines[i], w, "…")
	}
	return strings.Join(lines[:min(len(lines), m.termHeight())], "\n")
}
