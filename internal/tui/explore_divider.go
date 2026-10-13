package tui

import "strings"

func (m Model) exploreDivider(l exploreLayout) string {
	lines := make([]string, max(0, l.bodyH))
	for i := range lines {
		lines[i] = " "
		if m.exploreDragging {
			lines[i] = accent("│")
		}
	}
	if len(lines) > 0 {
		mark := gray("↔")
		if m.exploreDragging {
			mark = accent("↔")
		}
		lines[len(lines)/2] = mark
	}
	return strings.Join(lines, "\n")
}
