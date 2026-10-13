package tui

import (
	"fmt"
	"strings"

	mdl "github.com/Taka-S-dev/baton/internal/model"
	"github.com/Taka-S-dev/baton/internal/slot"
	"github.com/charmbracelet/x/ansi"
)

func (m Model) runDetailCommand() *mdl.Command {
	index := -1
	if m.runQueueFocused && len(m.msSelected) > 0 {
		index = m.msSelected[min(m.runQueueCursor, len(m.msSelected)-1)]
	} else {
		f := m.msFiltered()
		if m.msCursor >= 0 && m.msCursor < len(f) {
			index = f[m.msCursor]
		}
	}
	if index < 0 || index >= len(m.msItems) {
		return nil
	}
	return m.msItems[index].cmd
}

func (m Model) runDetailLines(w int) ([]string, string) {
	c := m.runDetailCommand()
	if c == nil {
		return nil, ""
	}
	cmd := slot.ApplyVarsToCommand(*c, m.vars)
	text := "$ " + cmd.Cmd + "\n\nworkdir: " + m.confirmWorkdir(cmd.Dir) + "\nshell: " + shellChoiceLabel(cmd.Shell)
	if cmd.Template != "" {
		text += "\ntemplate: " + cmd.Template
		for _, k := range sortedKeys(cmd.Values) {
			text += "\n" + k + " = " + cmd.Values[k]
		}
	}
	// Hard wrapping preserves shell whitespace and long unbroken paths.
	return strings.Split(ansi.Hardwrap(text, max(1, w), true), "\n"), c.Name + "\x00" + text
}

func (m Model) detailSize() (int, int) {
	w, h := m.viewWidth(), m.termHeight()
	if m.runDetailOpen {
		p := m.runDetailPopup()
		return p.w, max(3, p.h-1)
	}
	if splitPanels(w) {
		w = w - m.runLeftWidth(w) - 1
	}
	return w, max(3, h-max(5, h/2)-2)
}

func (m *Model) scrollRunDetail(delta int) {
	w, h := m.detailSize()
	lines, key := m.runDetailLines(w - 3)
	if key != m.runDetailKey {
		m.runDetailOffset = 0
	}
	m.runDetailKey = key
	m.runDetailOffset = min(max(0, len(lines)-(h-2)), max(0, m.runDetailOffset+delta))
}

// A reserved last column prevents text from reflowing when the bar appears.
func scrollRows(lines []string, start, rows, width int) string {
	start = min(max(0, start), max(0, len(lines)-rows))
	var out []string
	top, thumb := scrollbarThumb(len(lines), rows, start)
	for i := 0; i < rows; i++ {
		line := ""
		if start+i < len(lines) {
			line = ansi.Truncate(lines[start+i], max(0, width-1), "…")
		}
		line += strings.Repeat(" ", max(0, width-1-ansi.StringWidth(line)))
		bar := " "
		if len(lines) > rows {
			bar = gray("│")
			if i >= top && i < top+thumb {
				bar = accent("┃")
			}
		}
		out = append(out, line+bar)
	}
	return strings.Join(out, "\n")
}

func (m Model) runDetailBox(w, h int) string {
	lines, key := m.runDetailLines(w - 3)
	offset := m.runDetailOffset
	if key != m.runDetailKey {
		offset = 0
	}
	offset = min(offset, max(0, len(lines)-max(1, h-2)))
	title := "Command details"
	if c := m.runDetailCommand(); c != nil {
		title += " · " + c.Name
	}
	if len(lines) > h-2 {
		title += fmt.Sprintf(" · %d–%d/%d", min(offset+1, len(lines)), min(offset+h-2, len(lines)), len(lines))
	}
	return box(title, scrollRows(lines, offset, max(1, h-2), w-2), w, h, m.runDetailOpen)
}
