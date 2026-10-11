package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// Short terminals devote space to search and candidates; F1 exposes help.
func (m Model) viewRecentCompact(w int) string {
	p := m.recentGeometry(w)
	filtered := m.recentFiltered()
	h := m.termHeight()
	search := "/ " + m.recentSearch + "_"
	if m.recentSearch == "" {
		search = gray("/ Search recent commands…")
	}
	title := fmt.Sprintf("Recent · %d/%d", len(filtered), len(m.recentRows))
	if len(filtered) > p.rows {
		title += fmt.Sprintf(" · %d–%d", p.start+1, min(p.start+p.rows, len(filtered)))
	}
	lines := []string{accentBold(title), search}
	for row := 0; row < p.rows; row++ {
		i := p.start + row
		if i >= len(filtered) {
			hint := "No matching commands"
			if len(m.recentRows) == 0 {
				hint = "No recent commands"
			}
			lines = append(lines, gray("  "+hint))
			continue
		}
		entry := m.recentRows[filtered[i]]
		label := entry.ref.Command
		if entry.problem != "" {
			label += " [unavailable]"
		}
		selected := i == m.recentCursor
		label = recentHighlight(label, m.recentSearch, selected)
		if selected {
			label = sRowActive.Width(p.width).Render(ansi.Truncate("▶ "+label, p.width, "…"))
		} else {
			label = "  " + label
		}
		lines = append(lines, label)
	}
	lines = append(lines, recentBrowseRow(p.width, m.recentCursor == len(filtered)))
	if m.recentNotice != "" {
		lines[0] = gray(m.recentNotice)
	}
	if h < 4 {
		lines = []string{gray("Enlarge terminal · F1 Help")}
	} else if len(lines) > h {
		lines = lines[:h]
	}
	for i := range lines {
		lines[i] = strings.Repeat(" ", p.x) + ansi.Truncate(lines[i], p.width, "…")
	}
	return strings.Join(lines, "\n")
}

func recentBrowseRow(width int, selected bool) string {
	const label = "→ Browse projects…"
	if selected {
		return sRowActive.Width(width).Render(ansi.Truncate("▶ "+label, width, "…"))
	}
	return "  " + accent(label)
}
