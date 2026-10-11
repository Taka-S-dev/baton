package tui

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/Taka-S-dev/baton/internal/store"
	tea "github.com/charmbracelet/bubbletea"
)

func recentAge(used, now time.Time) string {
	if used.IsZero() {
		return ""
	}
	d := now.Sub(used)
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
	if d < 7*24*time.Hour {
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
	return used.Local().Format("2006-01-02")
}
func recentMatchMask(text, query string) []bool {
	r := []rune(text)
	lower := []rune(strings.Map(unicode.ToLower, text))
	mask := make([]bool, len(r))
	for _, term := range strings.Fields(strings.Map(unicode.ToLower, query)) {
		t := []rune(term)
		for i := 0; i+len(t) <= len(lower); i++ {
			if string(lower[i:i+len(t)]) == term {
				for j := i; j < i+len(t); j++ {
					mask[j] = true
				}
			}
		}
	}
	return mask
}
func recentHighlight(text, query string, selected bool) string {
	r := []rune(text)
	mask := recentMatchMask(text, query)
	var b strings.Builder
	for i := 0; i < len(r); {
		j := i + 1
		for j < len(r) && mask[j] == mask[i] {
			j++
		}
		part := string(r[i:j])
		if mask[i] {
			style := sHighlight.Bold(true)
			if selected {
				style = style.Background(sRowActive.GetBackground())
			}
			b.WriteString(style.Render(part))
		} else if selected {
			b.WriteString(sRowActive.Render(part))
		} else {
			b.WriteString(part)
		}
		i = j
	}
	return b.String()
}
func (m Model) removeRecentSelection() (tea.Model, tea.Cmd) {
	filtered := m.recentFiltered()
	if m.recentCursor >= len(filtered) {
		return m, nil
	}
	index := filtered[m.recentCursor]
	row := m.recentRows[index]
	if err := store.RemoveRecent(m.projectsDir, row.ref); err != nil {
		m.recentNotice = "Could not remove history: " + err.Error()
		return m, nil
	}
	m.recentUndo = &row
	m.recentRows = append(m.recentRows[:index:index], m.recentRows[index+1:]...)
	m.recentCursor = min(m.recentCursor, len(m.recentFiltered()))
	m.recentNotice = "Removed from history: " + row.ref.Command
	return m, nil
}
func (m Model) undoRecentRemoval() (tea.Model, tea.Cmd) {
	if m.recentUndo == nil {
		return m, nil
	}
	row := *m.recentUndo
	if err := store.RestoreRecent(m.projectsDir, row.ref); err != nil {
		m.recentNotice = "Could not restore history: " + err.Error()
		return m, nil
	}
	query := m.recentSearch
	m.gotoRecent()
	m.recentSearch = query
	m.focusRecentCommand()
	for cursor, index := range m.recentFiltered() {
		ref := m.recentRows[index].ref
		if ref.Project == row.ref.Project && ref.Command == row.ref.Command {
			m.recentCursor = cursor
			break
		}
	}
	m.recentUndo = nil
	m.recentNotice = "Restored to history: " + row.ref.Command
	return m, nil
}
