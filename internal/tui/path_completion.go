package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"unicode"

	"github.com/Taka-S-dev/baton/internal/slot"
	"github.com/Taka-S-dev/baton/internal/store"
	tea "github.com/charmbracelet/bubbletea"
)

type pathCandidate struct {
	value     string
	score     int
	directory bool
	parent    bool
}

type pathCompletion struct {
	base, query                       string
	focusValue                        string
	entries                           []pathCandidate
	cursor                            int
	loading                           bool
	err                               error
	autoInsert                        bool
	draft                             []string
	touched                           bool
	originalSearch, originalCompleted string
	originalCursor                    int
}

type pathCompletionResult struct {
	owner   *slotPickState
	request *pathCompletion
	entries []pathCandidate
	err     error
}

// Resolve the actual execution directory, without changing the process cwd.
// A slot that chooses workdir itself is completed from the launch directory.
func (m Model) completionBase() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if m.sp.currentCmd == nil || m.sp.currentCmd.Dir == "" {
		return cwd, nil
	}
	for _, def := range slot.GetSlots(*m.sp.currentCmd) {
		if def.Name == m.sp.slotName && strings.Contains(m.sp.currentCmd.Dir, def.Placeholder()) {
			return cwd, nil
		}
	}
	cmd := slot.Apply(*m.sp.currentCmd, m.sp.resolvedSoFar)
	dir := slot.ApplyVars(cmd.Dir, m.vars)
	if slot.Pattern.MatchString(dir) || slot.VarPattern.MatchString(dir) {
		return "", fmt.Errorf("choose workdir inputs before completing a relative path")
	}
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(cwd, dir)
	}
	return filepath.Clean(dir), nil
}

func (m Model) startPathCompletion() tea.Cmd {
	base, err := m.completionBase()
	if problems := slot.PathErrors(m.sp.placeholder()); len(problems) > 0 {
		err = fmt.Errorf("%s", problems[0])
	}
	p := &pathCompletion{base: base, query: m.sp.search, err: err, autoInsert: false, draft: append([]string(nil), m.sp.picked...), originalSearch: m.sp.search, originalCompleted: m.sp.completedPath, originalCursor: m.sp.cursor}
	if err == nil && m.sp.search == "" && len(m.sp.picked) == 0 && m.sce == nil && m.projectsDir != "" && m.sp.currentCmd != nil {
		value := store.LoadPathFocus(m.projectsDir, base, m.projectDir, m.sp.currentCmd.Name, m.sp.placeholder())
		if value != "" {
			prefix, _ := filepath.Split(value)
			entries, readErr := completePathsFor(base, prefix, m.sp.pathDef.PathKind, m.sp.pathDef.Extensions)
			if readErr == nil {
				for _, entry := range entries {
					if entry.value == value && m.pathCandidateSelectable(entry) {
						p.focusValue = value
						p.query = prefix
						m.sp.search = prefix
						break
					}
				}
			}
		}
	}
	m.sp.completion = p
	if err != nil {
		return nil
	}
	return m.requestPathCompletion(p)
}

func (m Model) requestPathCompletion(p *pathCompletion) tea.Cmd {
	p.loading = true
	owner := m.sp
	base, query := p.base, p.query
	def := owner.pathDef
	return func() tea.Msg {
		entries, err := completePathsFor(base, query, def.PathKind, def.Extensions)
		if parent, _, ok := completionParent(base, query); ok {
			entries = append([]pathCandidate{{value: parent, directory: true, parent: true}}, entries...)
		}
		return pathCompletionResult{owner: owner, request: p, entries: entries, err: err}
	}
}

func (m Model) receivePathCompletion(msg pathCompletionResult) (tea.Model, tea.Cmd) {
	if m.screen != ScreenSlotPick || m.sp != msg.owner || m.sp.completion != msg.request {
		return m, nil
	}
	p := m.sp.completion
	p.entries, p.err, p.loading = msg.entries, msg.err, false
	if len(p.entries) > 1 && p.entries[0].parent {
		p.cursor = 1
	}
	for i, entry := range p.entries {
		if p.focusValue != "" && filepath.Clean(entry.value) == filepath.Clean(p.focusValue) {
			p.cursor = i
			break
		}
	}
	p.focusValue = ""
	if p.autoInsert && p.err == nil && len(p.entries) == 1 {
		m.insertPathCandidate(p.entries[0].value)
	}
	return m, nil
}

func (m Model) insertPathCandidate(value string) {
	m.sp.search = value
	m.sp.completedPath = value
	m.sp.applyFilter()
	m.sp.cursor = len(m.sp.filtered) // Enter must use the inserted path, not a list entry.
	m.sp.completion = nil
}

func (m Model) updatePathCompletion(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	p := m.sp.completion
	if msg.String() == "ctrl+k" && m.sp.canSkip && m.sce != nil {
		return m.skipCommandEditSlot()
	}
	switch msg.String() {
	case "esc":
		if m.sp.search != "" {
			m.sp.search = ""
			return m, m.refilterPathCompletion()
		}
		m.closePathCompletion()
	case "up", "shift+tab":
		if len(p.entries) > 0 {
			p.cursor = (p.cursor + len(p.entries) - 1) % len(p.entries)
		}
	case "tab":
		if m.sp.variadic {
			if !p.loading && p.err == nil && len(p.entries) > 0 && m.pathCandidateSelectable(p.entries[p.cursor]) {
				p.toggle(p.entries[p.cursor].value)
			}
			return m, nil
		}
		if !p.loading && p.err == nil && len(p.entries) > 0 {
			if m.pathCandidateSelectable(p.entries[p.cursor]) {
				m.insertPathCandidate(p.entries[p.cursor].value)
			} else {
				return m.updatePathCompletion(tea.KeyMsg{Type: tea.KeyEnter})
			}
		}
		return m, nil
	case "down":
		if len(p.entries) > 0 {
			p.cursor = (p.cursor + 1) % len(p.entries)
		}
	case "left":
		return m, m.parentPathCompletion()
	case "right":
		if len(p.entries) > 0 && p.entries[p.cursor].parent {
			return m, m.parentPathCompletion()
		}
		if !p.loading && p.err == nil && len(p.entries) > 0 {
			value := p.entries[p.cursor].value
			if strings.HasSuffix(value, "/") || strings.HasSuffix(value, "\\") {
				m.sp.search = value
				return m, m.refilterPathCompletion()
			}
		}
	case "enter":
		if len(p.entries) > 0 && p.entries[p.cursor].parent {
			return m, m.parentPathCompletion()
		}
		if !p.loading && p.err == nil && len(p.entries) > 0 && !m.pathCandidateSelectable(p.entries[p.cursor]) {
			m.sp.search = p.entries[p.cursor].value
			return m, m.refilterPathCompletion()
		}
		if m.sp.variadic {
			if len(p.draft) == 0 && !p.touched {
				if p.loading || p.err != nil || len(p.entries) == 0 || !m.pathCandidateSelectable(p.entries[p.cursor]) {
					return m, nil
				}
				p.toggle(p.entries[p.cursor].value)
			}
			if len(p.draft) == 0 {
				m.sp.picked = nil
				return m, nil
			}
			m.sp.picked = append([]string(nil), p.draft...)
			m.sp.search = ""
			m.sp.applyFilter()
			m.sp.cursor = len(m.sp.filtered)
			m.sp.completion = nil
			return m.confirmCompletedPath()
		}
		if !p.loading && p.err == nil && len(p.entries) > 0 && m.pathCandidateSelectable(p.entries[p.cursor]) {
			m.insertPathCandidate(p.entries[p.cursor].value)
			return m.confirmCompletedPath()
		}
	case "backspace", "ctrl+u":
		if msg.String() == "ctrl+u" {
			m.sp.search = ""
		} else if r := []rune(m.sp.search); len(r) > 0 {
			m.sp.search = string(r[:len(r)-1])
		}
		return m, m.refilterPathCompletion()
	default:
		if msg.Type == tea.KeyRunes {
			for _, r := range msg.Runes {
				if !unicode.IsControl(r) {
					m.sp.search += string(r)
				}
			}
			return m, m.refilterPathCompletion()
		}
	}
	return m, nil
}

func (m Model) refilterPathCompletion() tea.Cmd {
	old := m.sp.completion
	m.sp.applyFilter()
	m.sp.cursor = len(m.sp.filtered)
	// A new request object invalidates pending reads, even when the user returns
	// to the same query. Background reads never mutate UI state.
	next := *old
	next.query = m.sp.search
	next.entries = nil
	next.cursor = 0
	next.err = nil
	next.autoInsert = false
	p := &next
	m.sp.completion = p
	if p.base == "" {
		p.err = old.err
		return nil
	}
	return m.requestPathCompletion(p)
}

func completePaths(base, query string) ([]pathCandidate, error) {
	return completePathsFor(base, query, "", nil)
}

func completePathsFor(base, query, kind string, extensions []string) ([]pathCandidate, error) {
	// The whole slot is a path, so spaces are part of the filename, not token separators.
	prefix, needle := filepath.Split(query)
	dir := prefix
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(base, dir)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", dir, err)
	}
	var out []pathCandidate
	separator := string(os.PathSeparator)
	if strings.Contains(query, "/") {
		separator = "/"
	}
	for _, entry := range entries {
		name := entry.Name()
		if strings.IndexFunc(name, unicode.IsControl) >= 0 {
			continue
		}
		if strings.HasPrefix(name, ".") && !strings.HasPrefix(needle, ".") {
			continue
		}
		score, ok := pathMatchScore(name, needle)
		if !ok {
			continue
		}
		value := prefix + name
		isDir := entry.IsDir()
		if entry.Type()&os.ModeSymlink != 0 {
			if info, e := os.Stat(filepath.Join(dir, name)); e == nil {
				isDir = info.IsDir()
			}
		}
		if !isDir && (kind == "dir" || !matchesPathExtensions(name, extensions)) {
			continue
		}
		if isDir {
			value += separator
		}
		out = append(out, pathCandidate{value: value, score: score, directory: isDir})
	}
	sort.Slice(out, func(i, j int) bool {
		if kind == "file" && out[i].directory != out[j].directory {
			return !out[i].directory
		}
		if out[i].score != out[j].score {
			return out[i].score < out[j].score
		}
		return naturalPathLess(out[i].value, out[j].value)
	})
	return out, nil
}

// Prefix matches come first, then contiguous matches, then a rune-safe
// subsequence match. No shell or external fuzzy-finder process is invoked.
func pathMatchScore(name, query string) (int, bool) {
	if runtime.GOOS == "windows" {
		name = strings.ToLower(name)
		query = strings.ToLower(query)
	}
	if strings.HasPrefix(name, query) {
		return 0, true
	}
	if strings.Contains(name, query) {
		return 1, true
	}
	remaining := []rune(query)
	for _, r := range name {
		if len(remaining) > 0 && r == remaining[0] {
			remaining = remaining[1:]
		}
	}
	return 2, len(remaining) == 0
}

func (p *pathCompletion) selected(value string) bool {
	for _, v := range p.draft {
		if v == value {
			return true
		}
	}
	return false
}

func (p *pathCompletion) toggle(value string) {
	p.touched = true
	for i, v := range p.draft {
		if v == value {
			p.draft = append(p.draft[:i], p.draft[i+1:]...)
			return
		}
	}
	p.draft = append(p.draft, value)
}

func matchesPathExtensions(name string, extensions []string) bool {
	if len(extensions) == 0 {
		return true
	}
	for _, ext := range extensions {
		if strings.HasSuffix(strings.ToLower(name), strings.ToLower(ext)) {
			return true
		}
	}
	return false
}

func (m Model) pathCandidateSelectable(c pathCandidate) bool {
	return !c.parent && (m.sp.pathDef.PathKind != "file" || !c.directory)
}

// Closing suggestions keeps explicit selections without choosing the highlighted row.
func (m Model) closePathCompletion() {
	if m.sp.variadic {
		m.sp.picked = append([]string(nil), m.sp.completion.draft...)
		m.sp.search = ""
		m.sp.applyFilter()
		m.sp.cursor = len(m.sp.filtered)
	}
	m.sp.completion = nil
}

// Return a normalized parent query while preserving the user's slash style.
func completionParent(base, query string) (string, string, bool) {
	prefix, _ := filepath.Split(query)
	if prefix == "" {
		return "", "", false
	}
	current := filepath.Clean(prefix)
	absolute := current
	if !filepath.IsAbs(absolute) {
		absolute = filepath.Join(base, absolute)
	}
	if filepath.Dir(absolute) == absolute {
		return "", "", false
	}
	parent := filepath.Clean(filepath.Join(current, ".."))
	sep := string(os.PathSeparator)
	if strings.Contains(query, "/") {
		sep = "/"
	}
	format := func(p string) string {
		if sep == "/" {
			p = filepath.ToSlash(p)
		}
		if p == "." {
			return ""
		}
		return strings.TrimRight(p, "/\\") + sep
	}
	return format(parent), format(current), true
}

func (m Model) parentPathCompletion() tea.Cmd {
	parent, focus, ok := completionParent(m.sp.completion.base, m.sp.search)
	if !ok {
		return nil
	}
	m.sp.search = parent
	cmd := m.refilterPathCompletion()
	m.sp.completion.focusValue = focus
	return cmd
}

func (m Model) confirmCompletedPath() (tea.Model, tea.Cmd) {
	if m.resolve == nil && m.sce == nil {
		return m, nil
	}
	return m.updateSlotPick(tea.KeyMsg{Type: tea.KeyEnter})
}
