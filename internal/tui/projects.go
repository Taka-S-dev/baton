package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/Taka-S-dev/baton/internal/config"
	"github.com/Taka-S-dev/baton/internal/projectfs"
	"github.com/Taka-S-dev/baton/internal/store"
	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) visibleProjects() []string {
	var names []string
	for _, name := range m.projects {
		if name != ".baton-trash" && strings.Contains(strings.ToLower(name), strings.ToLower(m.projectFilter)) {
			names = append(names, name)
		}
	}
	return names
}

func (m *Model) refreshProjects() {
	m.projects = config.ListProjects(m.projectsDir)
	store.SortProjectsByRecent(m.projectsDir, m.projects)
	m.projectCmdCounts = map[string]int{}
	for _, name := range m.projects {
		if name == ".baton-trash" {
			continue
		}
		if c, err := config.LoadConfig(filepath.Join(m.projectsDir, name)); err == nil {
			m.projectCmdCounts[name] = len(c.AllCommands())
		}
	}
	m.listCursor = min(m.listCursor, max(0, len(m.visibleProjects())-1))
}

func projectPath(base, name string) (string, error) {
	if name == "" || name == "." || name == ".." || strings.HasPrefix(name, ".") || strings.ContainsAny(name, "/\\<>:\"|?*\t\r\n") || strings.TrimSpace(name) != name || strings.HasSuffix(name, ".") {
		return "", fmt.Errorf("use a plain project name without path separators")
	}
	base, err := filepath.Abs(base)
	if err != nil {
		return "", err
	}
	path := filepath.Join(base, name)
	if filepath.Dir(path) != base {
		return "", fmt.Errorf("project must be inside projects directory")
	}
	return path, nil
}

func (m Model) updateProjects(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	names := m.visibleProjects()
	name := ""
	if m.listCursor >= 0 && m.listCursor < len(names) {
		name = names[m.listCursor]
	}
	key := msg.String()
	if m.projectMode == "search" {
		handled, changed, closed, cmd := updateSearchInput(&m.projectSearchTI, &m.projectFilter, msg, len(names) > 0)
		if changed {
			m.listCursor = 0
		}
		if closed {
			m.projectMode = ""
		}
		if handled {
			return m, cmd
		}
	}
	if m.projectMode == "new" {
		if key == "esc" {
			m.projectMode = ""
			return m, nil
		}
		if key == "enter" {
			path, err := projectPath(m.projectsDir, m.projectInput)
			var release func()
			if err == nil {
				release, err = projectfs.Lock(m.projectsDir)
			}
			if err == nil {
				err = os.Mkdir(path, 0755)
			}
			if err == nil {
				err = os.WriteFile(filepath.Join(path, "commands.tsv"), []byte("name\tgroup\tworkdir\tcmd\tshell\tslots\n"), 0644)
			}
			if release != nil {
				release()
			}
			if err != nil {
				m.errMsg = err.Error()
				return m, nil
			}
			m.projectMode = ""
			m.projectFilter = ""
			m.refreshProjects()
			for i, n := range m.visibleProjects() {
				if n == m.projectInput {
					m.listCursor = i
				}
			}
			m.successMsg = "Created project " + m.projectInput
			return m, nil
		}
		target := &m.projectInput
		if key == "backspace" {
			r := []rune(*target)
			if len(r) > 0 {
				*target = string(r[:len(r)-1])
			}
		} else if msg.Type == tea.KeySpace {
			*target += " "
		} else if msg.Type == tea.KeyRunes {
			*target += string(msg.Runes)
		}
		return m, nil
	}
	if m.projectMode == "delete" {
		if key == "esc" || key == "n" {
			m.projectMode = ""
			return m, nil
		}
		if key == "y" {
			path, err := projectPath(m.projectsDir, m.projectInput)
			var release func()
			if err == nil {
				release, err = projectfs.Lock(m.projectsDir)
			}
			if err == nil {
				var info os.FileInfo
				info, err = os.Lstat(path)
				if err == nil && (!info.IsDir() || info.Mode()&os.ModeSymlink != 0) {
					err = fmt.Errorf("only real project directories can be archived")
				}
			}
			trash := filepath.Join(m.projectsDir, ".baton-trash")
			if err == nil {
				err = os.MkdirAll(trash, 0755)
			}
			if err == nil {
				var info os.FileInfo
				info, err = os.Lstat(trash)
				if err == nil && info.Mode()&os.ModeSymlink != 0 {
					err = fmt.Errorf("trash must not be a symlink")
				}
			}
			if err == nil {
				err = os.Rename(path, filepath.Join(trash, m.projectInput+"-"+time.Now().Format("20060102-150405.000000000")))
			}
			if release != nil {
				release()
			}
			if err != nil {
				m.errMsg = err.Error()
				return m, nil
			}
			m.projectMode = ""
			m.refreshProjects()
			m.successMsg = "Moved settings to .baton-trash (recoverable)"
		}
		return m, nil
	}
	if m.projectMode == "actions" {
		m.projectMode = ""
		if key == "esc" {
			return m, nil
		}
		switch key {
		case "c", "o", "x":
		default:
			return m, nil
		}
	}
	switch key {
	case "up":
		m.listCursor = max(0, m.listCursor-1)
	case "down":
		m.listCursor = min(max(0, len(names)-1), m.listCursor+1)
	case "a", "n":
		m.projectMode = "new"
		m.projectInput = ""
	case "/":
		m.projectMode = "search"
		return m, startSearchInput(&m.projectSearchTI, m.projectFilter)
	case "r":
		m.refreshProjects()
	case " ":
		if name != "" {
			m.projectMode = "actions"
		}
	case "x", "delete":
		if name != "" {
			m.projectMode = "delete"
			m.projectInput = name
		}
	case "c":
		if name != "" {
			p := filepath.Join(m.projectsDir, name)
			return m, func() tea.Msg { return contextCopyMsg{clipboard.WriteAll(p)} }
		}
	case "o":
		if name != "" {
			p, err := filepath.Abs(filepath.Join(m.projectsDir, name))
			if err != nil {
				m.errMsg = err.Error()
				break
			}
			return m, func() tea.Msg {
				var cmd *exec.Cmd
				switch runtime.GOOS {
				case "windows":
					cmd = exec.Command("explorer.exe", p)
				case "darwin":
					cmd = exec.Command("open", p)
				default:
					cmd = exec.Command("xdg-open", p)
				}
				return projectFolderMsg{cmd.Run()}
			}
		}
	case "enter":
		if name == "" {
			break
		}
		if err := m.loadProject(filepath.Join(m.projectsDir, name)); err != nil {
			m.errMsg = err.Error()
		} else {
			m.gotoPostLoad()
		}
	case "esc":
		if m.projectFilter != "" {
			m.projectFilter = ""
			m.listCursor = 0
		} else {
			return m, tea.Quit
		}
	case "q":
		return m, tea.Quit
	}
	return m, nil
}

type projectFolderMsg struct{ err error }

func (m Model) projectRows(w int) (int, int) {
	if m.termHeight() < 18 {
		reserved := 3
		if m.showProjectAdd() {
			reserved++
		}
		return 2, min(max(1, len(m.visibleProjects())), max(1, m.termHeight()-reserved))
	}
	return 3, min(max(1, len(m.visibleProjects())), max(1, m.termHeight()-8))
}

func (m Model) showProjectAdd() bool {
	if m.termHeight() >= 18 {
		return true
	}
	for _, name := range m.projects {
		if name != ".baton-trash" && name != ".baton-recent" {
			return false
		}
	}
	return true
}

func (m Model) projectAddY(w int) int {
	if !m.showProjectAdd() {
		return -1
	}
	y, rows := m.projectRows(w)
	return y + rows + 1
}

func (m Model) viewProjects(w int) string {
	names := m.visibleProjects()
	y, rows := m.projectRows(w)
	_ = y
	var b strings.Builder
	b.WriteString(" " + accent(truncate("BATON · Projects", max(1, w-1))) + "\n")
	if m.termHeight() >= 18 {
		b.WriteString(" " + gray(truncate("Choose a project to run commands or edit settings.", max(1, w-1))) + "\n")
	}
	b.WriteString(" " + ansi.Truncate(searchInputView(m.projectSearchTI, m.projectFilter, m.projectMode == "search"), max(1, w-1), "…") + "\n")

	empty := "No projects · a creates one"
	if m.projectFilter != "" {
		empty = "No matches · Esc clears search"
	}
	nameW := 0
	for _, name := range names {
		nameW = max(nameW, lipgloss.Width(name))
	}
	nameW = min(nameW, max(1, w-18))
	writeWindowedList(&b, rows, len(names), m.listCursor, w, empty, func(i int) string {
		prefix := "   "
		if i == m.listCursor {
			prefix = " ▶ "
		}
		name := truncate(names[i], nameW)
		line := truncate(fmt.Sprintf("%s%s%s  %d commands", prefix, name, strings.Repeat(" ", max(0, nameW-lipgloss.Width(name))), m.projectCmdCounts[names[i]]), w)
		if i == m.listCursor {
			line += strings.Repeat(" ", max(0, w-lipgloss.Width(line)))
			line = sRowActive.Render(line)
		}
		return searchHighlight(line, []string{m.projectFilter}, i == m.listCursor, true)
	})
	for strings.Count(b.String(), "\n") < m.projectAddY(w) {
		b.WriteString("\n")
	}
	if m.showProjectAdd() {
		b.WriteString(accent("+ Add project  a") + "\n")
	}
	if m.termHeight() >= 18 {
		for strings.Count(b.String(), "\n") < m.termHeight()-2 {
			b.WriteString("\n")
		}
		if m.termHeight() >= 8 {
			b.WriteString(hline(w) + "\n")
		}
		keys := "↑↓ Select  Enter Open  / Search  Space Actions  a Add  r Refresh  Del Delete  q Quit  F1 Help"
		if m.projectMode == "search" {
			keys = "↓ Results  Enter Open  Esc Clear / Back  F1 Help"
		}
		b.WriteString(gray(truncate(keys, w)))
	} else {
		if strings.Count(b.String(), "\n") == m.termHeight() && !m.showProjectAdd() {
			content := strings.TrimSuffix(b.String(), "\n")
			b.Reset()
			b.WriteString(content)
			b.WriteString(" ")
		}
		if strings.Count(b.String(), "\n") <= m.termHeight()-1 {
			for strings.Count(b.String(), "\n") < m.termHeight()-1 {
				b.WriteString("\n")
			}
			b.WriteString(" ")
		}
	}
	if m.projectMode != "" && m.projectMode != "search" {
		title, body := "Project actions", "o  Open project folder\nc  Copy path\nx  Delete project\nEsc Close"
		switch m.projectMode {
		case "new":
			title = "New project"
			body = "Name > " + m.projectInput + "_\nEnter Create  Esc Cancel"
		case "delete":
			title = "Delete project?"
			body = filepath.Join(m.projectsDir, m.projectInput) + "\nMoves Baton settings to .baton-trash.\nCommand working directories are untouched.\ny Confirm  Esc Cancel"
		}
		pw := min(68, w)
		ph := min(8, m.termHeight())
		return overlayBox(b.String(), box(title, body, pw, ph, true), max(0, (w-pw)/2), max(0, (m.termHeight()-ph)/2), w)
	}
	return b.String()
}
