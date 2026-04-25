package tui

import (
	"sort"

	tea "github.com/charmbracelet/bubbletea"

	mdl "github.com/Taka-S-dev/baton/internal/model"
	"github.com/Taka-S-dev/baton/internal/store"
)

// ── Alias management ──────────────────────────────────────────────────────────

func (m Model) updateAliasMgmt(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.listCursor > 0 {
			m.listCursor--
		}
	case "down", "j":
		if m.listCursor < len(m.listItems)-1 {
			m.listCursor++
		}
	case "enter":
		switch m.listItems[m.listCursor] {
		case "Create alias":
			m.screen = ScreenCreateAlias
			return m, m.setupMultiSelectCmdsOnly()
		case "Edit alias":
			m.screen = ScreenEditAlias
			names := make([]string, len(m.aliases))
			for i, a := range m.aliases {
				names[i] = a.Name
			}
			m.listItems = names
			m.listCursor = 0
		case "Delete alias":
			m.screen = ScreenDeleteAlias
			names := make([]string, len(m.aliases))
			for i, a := range m.aliases {
				names[i] = a.Name
			}
			m.listItems = names
			m.listCursor = 0
		}
	case "esc":
		m.gotoMainMenu()
	}
	return m, nil
}

func (m Model) updateEditAlias(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.listCursor > 0 {
			m.listCursor--
		}
	case "down", "j":
		if m.listCursor < len(m.listItems)-1 {
			m.listCursor++
		}
	case "enter":
		if len(m.aliases) > 0 {
			m.editTargetIdx = m.listCursor
			m.screen = ScreenEditAliasMode
			m.listItems = []string{"Rename", "Change commands"}
			m.listCursor = 0
		}
	case "esc":
		m.screen = ScreenAliasMgmt
		m.listItems = []string{"Create alias", "Edit alias", "Delete alias"}
		m.listCursor = 0
	}
	return m, nil
}

func (m Model) updateEditAliasMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.listCursor > 0 {
			m.listCursor--
		}
	case "down", "j":
		if m.listCursor < len(m.listItems)-1 {
			m.listCursor++
		}
	case "enter":
		switch m.listItems[m.listCursor] {
		case "Rename":
			m.nameInput.SetValue(m.aliases[m.editTargetIdx].Name)
			m.nameInputMode = nameInputEditAlias
			m.nameInputErr = ""
			m.screen = ScreenNameInput
			return m, m.nameInput.Focus()
		case "Change commands":
			m.screen = ScreenEditAliasCommands
			return m, m.setupMultiSelectWithPreSelected(m.aliases[m.editTargetIdx].Steps)
		}
	case "esc":
		m.screen = ScreenEditAlias
		names := make([]string, len(m.aliases))
		for i, a := range m.aliases {
			names[i] = a.Name
		}
		m.listItems = names
		m.listCursor = m.editTargetIdx
	}
	return m, nil
}

func (m Model) updateDeleteAlias(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	gotoAliasMgmt := func() {
		m.screen = ScreenAliasMgmt
		m.listItems = []string{"Create alias", "Edit alias", "Delete alias"}
		m.listCursor = 0
	}
	if m.deleteConfirm {
		switch msg.String() {
		case "tab", "left", "right", "h", "l":
			m.deleteBtn = 1 - m.deleteBtn
		case "enter":
			if m.deleteBtn == 1 {
				indices := m.deleteSelected
				sort.Sort(sort.Reverse(sort.IntSlice(indices)))
				for _, i := range indices {
					m.aliases = append(m.aliases[:i], m.aliases[i+1:]...)
				}
				if err := store.SaveAliases(m.projectDir, m.aliases); err != nil {
					m.errMsg = "failed to save aliases: " + err.Error()
				}
				m.deleteSelected = nil
				m.deleteConfirm = false
				m.deleteBtn = 0
				gotoAliasMgmt()
			} else {
				m.deleteConfirm = false
				m.deleteBtn = 0
			}
		case "esc":
			m.deleteConfirm = false
			m.deleteBtn = 0
		}
		return m, nil
	}
	switch msg.String() {
	case "up", "k":
		if m.listCursor > 0 {
			m.listCursor--
		}
	case "down", "j":
		if m.listCursor < len(m.listItems)-1 {
			m.listCursor++
		}
	case " ", "　":
		found := false
		for i, s := range m.deleteSelected {
			if s == m.listCursor {
				m.deleteSelected = append(m.deleteSelected[:i], m.deleteSelected[i+1:]...)
				found = true
				break
			}
		}
		if !found {
			m.deleteSelected = append(m.deleteSelected, m.listCursor)
		}
	case "enter":
		if len(m.aliases) == 0 {
			break
		}
		if len(m.deleteSelected) == 0 {
			m.deleteSelected = []int{m.listCursor}
		}
		m.deleteConfirm = true
		m.deleteBtn = 0
	case "esc":
		m.deleteSelected = nil
		gotoAliasMgmt()
	}
	return m, nil
}

// ── Save / rename alias ───────────────────────────────────────────────────────

func (m Model) saveAlias(name string) (tea.Model, tea.Cmd) {
	for _, a := range m.aliases {
		if a.Name == name {
			m.nameInputErr = "name already exists"
			return m, nil
		}
	}
	r := m.resolve
	var steps []string
	for _, item := range r.rawItems {
		steps = append(steps, item.name())
	}
	a := mdl.Alias{Name: name, Steps: steps}
	if len(r.workflowVars) > 0 {
		a.Vars = r.workflowVars
	}
	m.aliases = append(m.aliases, a)
	if err := store.SaveAliases(m.projectDir, m.aliases); err != nil {
		m.errMsg = "failed to save aliases: " + err.Error()
	}
	m.resolve = nil
	m.gotoMainMenu()
	return m, nil
}

func (m Model) renameAlias(idx int, name string) (tea.Model, tea.Cmd) {
	if idx < 0 || idx >= len(m.aliases) {
		m.screen = ScreenAliasMgmt
		m.listItems = []string{"Create alias", "Edit alias", "Delete alias"}
		m.listCursor = 0
		return m, nil
	}
	for i, a := range m.aliases {
		if i != idx && a.Name == name {
			m.nameInputErr = "name already exists"
			return m, nil
		}
	}
	m.aliases[idx].Name = name
	if err := store.SaveAliases(m.projectDir, m.aliases); err != nil {
		m.errMsg = "failed to save aliases: " + err.Error()
	}
	m.screen = ScreenAliasMgmt
	m.listItems = []string{"Create alias", "Edit alias", "Delete alias"}
	m.listCursor = 0
	return m, nil
}
