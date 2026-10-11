package tui

import (
	"fmt"
	"strings"

	"github.com/Taka-S-dev/baton/internal/slot"
	"github.com/charmbracelet/x/ansi"
)

func (m Model) confirmCommonDirectory() (string, bool) {
	dir := ""
	for i, item := range m.confirmRunItems {
		if item.Cmd == nil {
			return "", false
		}
		cmd := slot.ApplyVarsToCommand(*item.Cmd, m.vars)
		current := m.confirmWorkdir(cmd.Dir)
		if i == 0 {
			dir = current
		} else if dir != current {
			return "", false
		}
	}
	return dir, len(m.confirmRunItems) > 0
}

func (m Model) viewCompactConfirm(w int) string {
	n := len(m.confirmRunItems)
	h := m.termHeight()
	dir, same := m.confirmCommonDirectory()
	lines := []string{accentBold(fmt.Sprintf("Review · %d commands", n))}
	if same {
		lines = append(lines, gray("In: "+dir))
	} else {
		lines = append(lines, gray("Working directories shown per command"))
	}
	per := m.confirmRunPerPage()
	start := min(max(0, m.confirmRunScroll), max(0, n-per))
	end := min(n, start+per)
	numberWidth := len(fmt.Sprint(max(1, n)))
	nameWidth := 0
	for _, item := range m.confirmRunItems {
		nameWidth = max(nameWidth, ansi.StringWidth(item.Name))
	}
	nameWidth = min(nameWidth, max(1, min(24, (w-numberWidth-7)/3)))
	for i := start; i < end; i++ {
		item := m.confirmRunItems[i]
		name := ansi.Truncate(item.Name, nameWidth, "…")
		name += strings.Repeat(" ", max(0, nameWidth-ansi.StringWidth(name)))
		line := gray(fmt.Sprintf("%*d. %s │ ", numberWidth, i+1, name))
		if item.Cmd != nil {
			cmd := slot.ApplyVarsToCommand(*item.Cmd, m.vars)
			line += white("$ " + strings.ReplaceAll(strings.ReplaceAll(cmd.Cmd, "\r", ""), "\n", " ↵ "))
			lines = append(lines, line)
			if !same {
				lines = append(lines, gray("   In: "+m.confirmWorkdir(cmd.Dir)))
			}
		} else {
			lines = append(lines, line)
			if !same {
				lines = append(lines, "")
			}
		}
	}
	for len(lines) < h-3 {
		lines = append(lines, "")
	}
	status := ""
	if n > per {
		status = fmt.Sprintf("%d–%d / %d · ↑↓ Scroll", start+1, end, n)
	}
	lines = append(lines, gray(status))
	run, back := "  Run  ", "  "+m.confirmBackLabel()+"  "
	if m.confirmRunBtn == 0 {
		run = sRowActive.Render("[ Run ]")
	} else {
		back = sRowActive.Render("[ " + m.confirmBackLabel() + " ]")
	}
	lines = append(lines, run+"  "+back, gray("Tab Switch · Enter Select · Esc "+m.confirmBackLabel()))
	for i := range lines {
		lines[i] = ansi.Truncate(lines[i], max(1, w), "…")
	}
	return strings.Join(lines, "\n")
}
