package tui

import (
	"path/filepath"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// Use all rows for stable columns while searching or scrolling.
func (m Model) recentColumns(entry recentRow, width int, selected bool) string {
	markerWidth, nameWidth, projectWidth := 4, 0, 0
	for _, row := range m.recentRows {
		markerWidth = max(markerWidth, ansi.StringWidth(m.recentMarker(row.ref)))
		nameWidth = max(nameWidth, ansi.StringWidth(row.ref.Command))
		projectWidth = max(projectWidth, ansi.StringWidth(filepath.Base(row.ref.Project)))
	}
	available := max(1, width-markerWidth)
	projectWidth = min(projectWidth, max(1, available/3))
	nameWidth = min(nameWidth, max(1, available-projectWidth-2))
	marker := m.recentMarker(entry.ref)
	marker += strings.Repeat(" ", max(0, markerWidth-ansi.StringWidth(marker)))
	name := ansi.Truncate(entry.ref.Command, nameWidth, "…")
	project := filepath.Base(entry.ref.Project)
	if entry.problem != "" {
		project += " [unavailable]"
	}
	project = ansi.Truncate(project, max(0, available-nameWidth-2), "…")
	if !selected {
		project = gray(project)
	}
	return marker + name + strings.Repeat(" ", max(0, nameWidth-ansi.StringWidth(name))+2) + project
}
