package tui

import (
	"fmt"
	"strings"

	"github.com/Taka-S-dev/baton/internal/slot"
	"github.com/charmbracelet/x/ansi"
)

func (m Model) queueCommandLabel(order, w int) string {
	item := m.msItems[m.msSelected[order]]
	prefix := fmt.Sprintf("%d. $ ", order+1)
	if item.cmd == nil {
		return ansi.Truncate(prefix+item.name(), w, "…")
	}
	cmd := slot.ApplyVarsToCommand(*item.cmd, m.vars)
	// Preserve spacing inside shell arguments, including quoted values.
	body := strings.ReplaceAll(strings.ReplaceAll(cmd.Cmd, "\r", ""), "\n", " ↵ ")
	dir := m.confirmWorkdir(cmd.Dir)
	differing := false
	for _, idx := range m.msSelected {
		other := m.msItems[idx].cmd
		if other != nil && m.confirmWorkdir(slot.ApplyVarsToCommand(*other, m.vars).Dir) != dir {
			differing = true
			break
		}
	}
	suffix := ""
	if differing {
		suffix = " · In: " + dir
	}
	if suffix != "" {
		suffix = ansi.Truncate(suffix, max(1, w/2), "…")
	}
	bodyWidth := max(1, w-ansi.StringWidth(prefix)-ansi.StringWidth(suffix))
	return ansi.Truncate(prefix+m.syntaxText(ansi.Truncate(body, bodyWidth, "…"))+gray(suffix), w, "…")
}
