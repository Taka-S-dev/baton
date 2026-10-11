package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/Taka-S-dev/baton/internal/config"
	mdl "github.com/Taka-S-dev/baton/internal/model"
	"github.com/Taka-S-dev/baton/internal/slot"
	"github.com/Taka-S-dev/baton/internal/store"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

type recentRow struct {
	ref     store.RecentEntry
	command mdl.Command
	problem string
}

func (m *Model) gotoRecent() {
	m.screen = ScreenRecent
	m.resolve = nil
	m.sp = nil
	m.exploreLaunched = false
	m.recentRows = nil
	m.recentCursor = 0
	m.recentSearch = ""
	m.loadWarnings = nil
	refs, err := store.LoadRecent(m.projectsDir, m.launchDir)
	if err != nil {
		m.errMsg = err.Error()
	}
	// Load each project once for display. Enter reloads it before resolving.
	loaded := map[string]*config.Project{}
	problems := map[string]string{}
	for _, ref := range refs {
		row := recentRow{ref: ref}
		p, seen := loaded[ref.Project]
		if !seen {
			value, err := config.LoadProject(ref.Project)
			if err != nil {
				problems[ref.Project] = err.Error()
			} else {
				p = &value
			}
			loaded[ref.Project] = p
		}
		if p == nil {
			row.problem = problems[ref.Project]
		} else if cmd, ok := p.Config.FindCommand(ref.Command); ok {
			row.command = slot.ApplyVarsToCommand(cmd, p.Vars)
		} else {
			row.problem = "Command removed or renamed"
		}
		m.recentRows = append(m.recentRows, row)
	}
	m.focusRecentCommand()
}

// Prefer the newest available reference; otherwise focus Browse projects.
func (m *Model) focusRecentCommand() {
	filtered := m.recentFiltered()
	m.recentCursor = len(filtered)
	for cursor, index := range filtered {
		if m.recentRows[index].problem == "" {
			m.recentCursor = cursor
			return
		}
	}
}
func (m Model) recentFiltered() []int {
	var result []int
	terms := strings.Fields(strings.ToLower(m.recentSearch))
	for i, row := range m.recentRows {
		if matchesAllTerms(strings.ToLower(row.ref.Command+" "+filepath.Base(row.ref.Project)+" "+row.command.Cmd), terms) {
			result = append(result, i)
		}
	}
	return result
}
func (m Model) updateRecent(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	filtered := m.recentFiltered()
	n := len(filtered) + 1 // browse is always available, including an empty history
	switch msg.String() {
	case "up":
		m.recentCursor = (m.recentCursor + n - 1) % n
	case "down":
		m.recentCursor = (m.recentCursor + 1) % n
	case "ctrl+r":
		m.gotoRecent()
	case "esc":
		if m.recentSearch != "" {
			m.recentSearch = ""
			m.focusRecentCommand()
		} else {
			return m, tea.Quit
		}
	case "backspace":
		r := []rune(m.recentSearch)
		if len(r) > 0 {
			m.recentSearch = string(r[:len(r)-1])
			m.focusRecentCommand()
		}
	case "enter":
		if m.recentCursor >= len(filtered) {
			m.recentActive = false
			m.screen = ScreenProjectSelect
			m.projectFilter = ""
			m.projectMode = ""
			m.refreshProjects()
			m.listCursor = 0
			return m, nil
		}
		row := m.recentRows[filtered[m.recentCursor]]
		if err := m.loadProject(row.ref.Project); err != nil {
			m.errMsg = err.Error()
			return m, nil
		}
		cmd, ok := m.config.FindCommand(row.ref.Command)
		if !ok {
			m.errMsg = "Command removed or renamed; choose another command or browse projects"
			return m, nil
		}
		m.recentActive = true
		return m.startResolveFlow([]msItem{{cmd: &cmd}})
	default:
		if msg.Type == tea.KeySpace {
			m.recentSearch += " "
			m.focusRecentCommand()
		}
		if msg.Type == tea.KeyRunes {
			for _, r := range msg.Runes {
				if !unicode.IsControl(r) {
					m.recentSearch += string(r)
				}
			}
			m.focusRecentCommand()
		}
	}
	return m, nil
}

// Geometry is shared with hit testing so separators never select commands.
type recentLayout struct{ x, width, rows, start, listY, browseY int }

func (m Model) recentGeometry(w int) recentLayout {
	x := 2
	if w < 20 {
		x = 0
	}
	n := len(m.recentFiltered())
	rows := min(max(1, n), min(8, max(1, m.termHeight()-11)))
	return recentLayout{x: x, width: max(1, min(96, w-2*x)), rows: rows, start: windowStart(rows, n, min(m.recentCursor, max(0, n-1))), listY: 5, browseY: 9 + rows}
}
func (m Model) viewRecent(w int) string {
	p := m.recentGeometry(w)
	filtered := m.recentFiltered()
	rule := gray(strings.Repeat("─", p.width))
	search := gray("/ Search recent commands…")
	if m.recentSearch != "" {
		search = "/ " + m.recentSearch + "_"
	}
	lines := []string{accentBold("[ Recent commands ]"), gray("In: " + m.launchDir), "", search, rule}
	rowLine := func(label string, selected bool) string {
		if selected {
			return sRowActive.Width(p.width).Render(ansi.Truncate("▶ "+ansi.Strip(label), p.width, "…"))
		}
		return "  " + label
	}
	for row := 0; row < p.rows; row++ {
		i := p.start + row
		if i >= len(filtered) {
			hint := "No matching commands"
			if len(m.recentRows) == 0 {
				hint = "No commands run here yet"
			}
			lines = append(lines, gray("  "+hint))
			continue
		}
		entry := m.recentRows[filtered[i]]
		label := entry.ref.Command + gray("  · "+filepath.Base(entry.ref.Project))
		if entry.problem != "" {
			label += gray("  [unavailable]")
		}
		lines = append(lines, rowLine(label, i == m.recentCursor))
	}
	preview := "Preview"
	if len(filtered) > p.rows {
		preview += fmt.Sprintf("  · %d–%d / %d", p.start+1, min(p.start+p.rows, len(filtered)), len(filtered))
	}
	lines = append(lines, gray("── "+preview+" "+strings.Repeat("─", max(0, p.width-ansi.StringWidth(preview)-4))))
	cmdLine, dirLine := gray("Choose a project to find more commands."), ""
	if m.recentCursor < len(filtered) {
		entry := m.recentRows[filtered[m.recentCursor]]
		if entry.problem != "" {
			cmdLine = warn(entry.problem)
		} else {
			cmdLine = "$ " + entry.command.Cmd
			dir := entry.command.Dir
			if dir == "" {
				dir = m.launchDir
			}
			dirLine = gray("workdir: " + dir)
		}
	}
	lines = append(lines, cmdLine, dirLine, rule, rowLine("Browse projects…", m.recentCursor == len(filtered)))
	footer := "↑↓ Move  Enter Select  Esc Quit"
	if m.recentSearch != "" {
		footer = "↑↓ Move  Enter Select  Esc Clear"
	}
	if p.width >= 58 {
		footer += "  Ctrl+R Refresh"
	}
	lines = append(lines, gray(footer))
	for i := range lines {
		lines[i] = strings.Repeat(" ", p.x) + ansi.Truncate(lines[i], p.width, "…")
	}
	if len(lines) > m.termHeight() {
		lines = lines[:m.termHeight()]
	}
	return strings.Join(lines, "\n")
}
func (m Model) updateRecentMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		return m.updateRecent(tea.KeyMsg{Type: tea.KeyUp})
	case tea.MouseButtonWheelDown:
		return m.updateRecent(tea.KeyMsg{Type: tea.KeyDown})
	case tea.MouseButtonLeft:
		if msg.Action != tea.MouseActionPress {
			return m, nil
		}
		p := m.recentGeometry(m.viewWidth())
		if msg.X < p.x || msg.X >= p.x+p.width {
			return m, nil
		}
		n := len(m.recentFiltered())
		if msg.Y == p.browseY {
			m.recentCursor = n
			return m.updateRecent(tea.KeyMsg{Type: tea.KeyEnter})
		}
		row := msg.Y - p.listY
		if row >= 0 && row < p.rows && p.start+row < n {
			m.recentCursor = p.start + row
			return m.updateRecent(tea.KeyMsg{Type: tea.KeyEnter})
		}
	}
	return m, nil
}
