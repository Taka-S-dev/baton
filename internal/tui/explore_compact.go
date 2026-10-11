package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// tabPresentation returns the tab labels and the gap between them, so
// drawing and hit-testing use the same labels, spacing and cell widths.
func (m Model) tabPresentation(w int) ([]string, string) {
	labels := make([]string, tabCount)
	for i, name := range tabNames {
		labels[i] = fmt.Sprintf("[%d %s]", i+1, name)
	}
	if lipgloss.Width("  "+strings.Join(labels, "  "))+3 <= w {
		return labels, "  "
	}
	short := []string{"Flows", "Cmds", "Lists", "Vars"}
	for i, name := range short {
		labels[i] = fmt.Sprintf("%d %s", i+1, name)
		if i == m.exploreTab {
			labels[i] = "[" + labels[i] + "]"
		}
	}
	if lipgloss.Width(" "+strings.Join(labels, " ")) <= w {
		return labels, " "
	}
	for i := range labels {
		labels[i] = fmt.Sprintf("%d", i+1)
		if i == m.exploreTab {
			labels[i] = fmt.Sprintf("[%d %s]", i+1, short[i])
		}
	}
	return labels, " "
}

func (m Model) compactFooter(w int) string {
	full := m.exploreKeys() + "  " + m.helpHint()
	if lipgloss.Width(full)+2 <= w {
		return "  " + full
	}
	parts := []string{"Enter Open", "a Add", "F1 Help", "v Details", "Esc Back"}
	if !m.exploreListFocused() || m.exploreSearching {
		parts = []string{"F1 Help"}
		parts = append(parts, strings.Split(m.exploreKeys(), "  ")...)
	}
	line := ""
	for _, part := range parts {
		candidate := part
		if line != "" {
			candidate = line + "  " + part
		}
		if lipgloss.Width(candidate) <= w {
			line = candidate
		}
	}
	return line
}
