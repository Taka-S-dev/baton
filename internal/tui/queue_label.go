package tui

import (
	"fmt"
	"strings"

	"github.com/Taka-S-dev/baton/internal/slot"
	"github.com/charmbracelet/x/ansi"
)

// Queue rows prioritize execution order and name; full commands and paths live in details.
func (m Model) queueCommandLabel(order, w int) string {
	item := m.msItems[m.msSelected[order]]
	orderW := len(fmt.Sprint(len(m.msSelected)))
	nameW := 0
	for _, idx := range m.msSelected {
		nameW = max(nameW, ansi.StringWidth(m.msItems[idx].name()))
	}
	// Reserve most of the row for the executable text; measure the entire queue
	// so scrolling cannot move the command column.
	nameW = min(nameW, max(1, (w-orderW-5)/3))
	label := ansi.Truncate(item.name(), nameW, "…")
	label += strings.Repeat(" ", max(0, nameW-ansi.StringWidth(label)))
	name := fmt.Sprintf("%*d. %s", orderW, order+1, label)
	if item.cmd == nil {
		return ansi.Truncate(name, w, "…")
	}
	cmd := slot.ApplyVarsToCommand(*item.cmd, m.vars)
	body := strings.ReplaceAll(strings.ReplaceAll(cmd.Cmd, "\r", ""), "\n", " ↵ ")
	return ansi.Truncate(white(name)+"  "+gray("$ ")+m.syntaxText(body), w, "…")
}
