package tui

import (
	"fmt"
	"path/filepath"

	"github.com/Taka-S-dev/baton/internal/config"
	mdl "github.com/Taka-S-dev/baton/internal/model"
	"github.com/Taka-S-dev/baton/internal/slot"
	"github.com/Taka-S-dev/baton/internal/store"
	tea "github.com/charmbracelet/bubbletea"
)

func sameRecent(a, b store.RecentEntry) bool {
	return a.Project == b.Project && a.Command == b.Command
}

func (m Model) recentMarker(ref store.RecentEntry) string {
	for i, picked := range m.recentQueue {
		if sameRecent(ref, picked) {
			return fmt.Sprintf("[%d] ", i+1)
		}
	}
	return "[ ] "
}

func (m *Model) toggleRecent(row recentRow) {
	for i, ref := range m.recentQueue {
		if sameRecent(ref, row.ref) {
			m.recentQueue = append(m.recentQueue[:i:i], m.recentQueue[i+1:]...)
			return
		}
	}
	if row.problem != "" {
		m.errMsg = row.problem
		return
	}
	m.recentQueue = append(m.recentQueue, row.ref)
}

func (m Model) resolveRecentQueue(fallback store.RecentEntry) (tea.Model, tea.Cmd) {
	refs := m.recentQueue
	if len(refs) == 0 {
		refs = []store.RecentEntry{fallback}
	}
	items := make([]msItem, 0, len(refs))
	loaded := map[string]*config.Project{}
	for _, ref := range refs {
		p := loaded[ref.Project]
		if p == nil {
			project, err := config.LoadProject(ref.Project)
			if err != nil {
				m.errMsg = fmt.Sprintf("%s: %v", ref.Project, err)
				return m, nil
			}
			p = &project
			loaded[ref.Project] = p
		}
		cmd, ok := p.Config.FindCommand(ref.Command)
		if !ok {
			m.errMsg = fmt.Sprintf("%s: command %q removed or renamed", filepath.Base(ref.Project), ref.Command)
			return m, nil
		}
		items = append(items, msItem{cmd: &cmd, project: p, projectDir: ref.Project})
	}
	m.recentActive = true
	return m.startResolveFlow(items)
}

func (m Model) resolvedRunItem(item msItem, cmd mdl.Command) mdl.RunItem {
	r := mdl.RunItem{Name: cmd.Name, Cmd: &cmd}
	if item.project != nil {
		r.Project = item.projectDir
		r.Vars = item.project.Vars
		// A project with no variables must not fall back to another project's.
		if r.Vars == nil {
			r.Vars = map[string]string{}
		}
		r.Name += " · " + filepath.Base(item.projectDir)
	}
	return r
}

func (m Model) runItemCommand(item mdl.RunItem) mdl.Command {
	vars := m.vars
	if item.Vars != nil {
		vars = item.Vars
	}
	return slot.ApplyVarsToCommand(*item.Cmd, vars)
}

func (m Model) runItemProject(item mdl.RunItem) string {
	if item.Project != "" {
		return item.Project
	}
	return m.projectDir
}
