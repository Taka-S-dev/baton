package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"

	mdl "github.com/Taka-S-dev/baton/internal/model"
	"github.com/Taka-S-dev/baton/internal/slot"
	"github.com/Taka-S-dev/baton/internal/store"
)

// ── Multi-select ──────────────────────────────────────────────────────────────

func (m *Model) setupMultiSelect(includeAliases bool) tea.Cmd {
	m.msItems = nil
	m.msCursor = 0
	m.msViewStart = 0
	m.msSelected = nil
	m.msActiveField = 0
	for i := range m.config.Commands {
		m.msItems = append(m.msItems, msItem{cmd: &m.config.Commands[i]})
	}
	if includeAliases {
		for i := range m.aliases {
			m.msItems = append(m.msItems, msItem{alias: &m.aliases[i]})
		}
	}
	m.msSearchTI = newMSTI("/ ", true)
	m.msGroupTI = newMSTI("Group / ", false)
	return m.msSearchTI.Focus()
}

func (m *Model) setupMultiSelectCmdsOnly() tea.Cmd {
	return m.setupMultiSelect(false)
}

func (m *Model) setupMultiSelectWithPreSelected(cmdNames []string) tea.Cmd {
	cmd := m.setupMultiSelectCmdsOnly()
	nameSet := make(map[string]bool, len(cmdNames))
	for _, n := range cmdNames {
		nameSet[n] = true
	}
	for i, item := range m.msItems {
		if nameSet[item.name()] {
			m.msSelected = append(m.msSelected, i)
		}
	}
	return cmd
}

func (m *Model) msFiltered() []int {
	var out []int
	qs := strings.ToLower(m.msSearchTI.Value())
	qg := strings.ToLower(m.msGroupTI.Value())
	for i, item := range m.msItems {
		if qs != "" {
			if !strings.Contains(strings.ToLower(item.name()), qs) &&
				!strings.Contains(strings.ToLower(item.group()), qs) {
				continue
			}
		}
		if qg != "" {
			if !strings.Contains(strings.ToLower(item.group()), qg) {
				continue
			}
		}
		out = append(out, i)
	}
	return out
}

func (m Model) updateMultiSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	filtered := m.msFiltered()
	n := len(filtered)

	switch msg.String() {
	case "up":
		if n > 0 {
			m.msCursor = (m.msCursor - 1 + n) % n
		}
	case "down":
		if n > 0 {
			m.msCursor = (m.msCursor + 1) % n
		}
	case "tab":
		m.msActiveField = 1 - m.msActiveField
		cyan := lipgloss.NewStyle().Foreground(lipgloss.Color("36"))
		dark := lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
		if m.msActiveField == 1 {
			m.msSearchTI.Blur()
			m.msSearchTI.PromptStyle = dark
			m.msGroupTI.PromptStyle = cyan
			return m, m.msGroupTI.Focus()
		}
		m.msGroupTI.Blur()
		m.msGroupTI.PromptStyle = dark
		m.msSearchTI.PromptStyle = cyan
		return m, m.msSearchTI.Focus()
	case " ", "　":
		if n > 0 && m.msCursor < n {
			origIdx := filtered[m.msCursor]
			found := false
			for i, s := range m.msSelected {
				if s == origIdx {
					m.msSelected = append(m.msSelected[:i], m.msSelected[i+1:]...)
					found = true
					break
				}
			}
			if !found {
				m.msSelected = append(m.msSelected, origIdx)
			}
		}
	case "enter":
		if len(m.msSelected) == 0 {
			break
		}
		selected := make([]msItem, len(m.msSelected))
		for i, idx := range m.msSelected {
			selected[i] = m.msItems[idx]
		}
		return m.startResolveFlow(selected)
	case "esc":
		if m.msSearchTI.Value() != "" || m.msGroupTI.Value() != "" {
			m.msSearchTI.SetValue("")
			m.msGroupTI.SetValue("")
			m.msCursor = 0
			m.msViewStart = 0
			return m, nil
		}
		if m.screen == ScreenEditWorkflowCommands {
			m.screen = ScreenEditWorkflowMode
			m.listItems = []string{"Rename", "Change commands"}
			m.listCursor = 0
			return m, nil
		}
		if m.screen == ScreenEditAliasCommands {
			m.screen = ScreenEditAliasMode
			m.listItems = []string{"Rename", "Change commands"}
			m.listCursor = 0
			return m, nil
		}
		m.gotoMainMenu()
	default:
		prevSearch := m.msSearchTI.Value()
		prevGroup := m.msGroupTI.Value()
		var cmd tea.Cmd
		if m.msActiveField == 0 {
			m.msSearchTI, cmd = m.msSearchTI.Update(msg)
		} else {
			m.msGroupTI, cmd = m.msGroupTI.Update(msg)
		}
		if m.msSearchTI.Value() != prevSearch || m.msGroupTI.Value() != prevGroup {
			m.msCursor = 0
			m.msViewStart = 0
		}
		return m, cmd
	}
	return m, nil
}

// ── Resolve flow ──────────────────────────────────────────────────────────────

func (m Model) startResolveFlow(items []msItem) (tea.Model, tea.Cmd) {
	names := make([]string, len(items))
	for i, it := range items {
		names[i] = it.name()
	}
	purpose := purposeRunManually
	if m.screen == ScreenCreateWorkflow {
		purpose = purposeCreateWorkflow
	} else if m.screen == ScreenCreateAlias {
		purpose = purposeCreateAlias
	} else if m.screen == ScreenEditWorkflowCommands {
		purpose = purposeEditWorkflow
	} else if m.screen == ScreenEditAliasCommands {
		purpose = purposeEditAlias
	}
	m.resolve = &resolveFlowState{
		purpose:      purpose,
		rawItems:     items,
		itemNames:    names,
		itemNotes:    make([]string, len(items)),
		workflowVars: make(map[string]map[string]string),
	}
	return m.advanceResolve()
}

func (m Model) advanceResolve() (tea.Model, tea.Cmd) {
	r := m.resolve
	if r.purpose == purposeEditWorkflow || r.purpose == purposeEditAlias {
		for r.currentIdx < len(r.rawItems) {
			r.currentIdx++
		}
		return m.finishResolveFlow()
	}
	for r.currentIdx < len(r.rawItems) {
		item := r.rawItems[r.currentIdx]

		if item.isAlias() && item.alias.Vars != nil {
			r.resolved = append(r.resolved, mdl.RunItem{
				Name:  item.alias.Name,
				Alias: item.alias,
			})
			r.itemNotes[r.currentIdx] = "(stored vars)"
			r.currentIdx++
			continue
		}

		if item.isAlias() {
			slots := m.collectAliasSlots(item.alias)
			if r.currentValues == nil {
				r.currentValues = make(map[string]string)
				r.currentSlots = slots
				r.currentSlotIdx = 0
			}
			if r.currentSlotIdx >= len(r.currentSlots) {
				r.resolved = append(r.resolved, mdl.RunItem{
					Name:   item.alias.Name,
					Alias:  item.alias,
					VarMap: r.currentValues,
				})
				r.itemNotes[r.currentIdx] = "(alias resolved)"
				r.currentIdx++
				r.currentValues = nil
				r.currentSlots = nil
				r.currentSlotIdx = 0
				continue
			}
			return m.openSlotPick(r.currentSlots[r.currentSlotIdx], item.cmd)
		}

		slots := slot.GetSlots(*item.cmd)
		if r.currentValues == nil {
			r.currentValues = make(map[string]string)
			r.currentSlots = slots
			r.currentSlotIdx = 0
		}
		if r.currentSlotIdx >= len(r.currentSlots) {
			resolved := slot.Apply(*item.cmd, r.currentValues)
			if r.itemNotes[r.currentIdx] == "" {
				dir := resolved.Dir
				if dir == "" {
					dir = "."
				}
				note := "$ " + resolved.Cmd + "  (workdir: " + dir + ")"
				r.itemNotes[r.currentIdx] = note
			}
			if (r.purpose == purposeCreateWorkflow || r.purpose == purposeCreateAlias) && len(r.currentValues) > 0 {
				r.workflowVars[item.cmd.Name] = copyMap(r.currentValues)
			}
			r.resolved = append(r.resolved, mdl.RunItem{Name: item.cmd.Name, Cmd: &resolved})
			r.currentIdx++
			r.currentValues = nil
			r.currentSlots = nil
			r.currentSlotIdx = 0
			continue
		}
		return m.openSlotPick(r.currentSlots[r.currentSlotIdx], item.cmd)
	}
	return m.finishResolveFlow()
}

func (m Model) collectAliasSlots(a *mdl.Alias) []slot.Def {
	seen := make(map[string]bool)
	var slots []slot.Def
	for _, stepName := range a.Steps {
		for _, cmd := range m.config.Commands {
			if cmd.Name == stepName {
				for _, s := range slot.GetSlots(cmd) {
					if !seen[s.Name] {
						seen[s.Name] = true
						slots = append(slots, s)
					}
				}
			}
		}
	}
	return slots
}

func (m Model) openSlotPick(s slot.Def, cmd *mdl.Command) (tea.Model, tea.Cmd) {
	entries := m.lists[s.ListName]
	r := m.resolve
	sp := &slotPickState{
		slotName:      s.Name,
		listName:      s.ListName,
		entries:       entries,
		cursor:        0,
		canSkip:       r.purpose == purposeCreateWorkflow || r.purpose == purposeCreateAlias,
		contextNames:  r.itemNames,
		contextNotes:  r.itemNotes,
		contextIdx:    r.currentIdx,
		currentCmd:    cmd,
		resolvedSoFar: copyMap(r.currentValues),
	}
	sp.applyFilter()
	m.sp = sp
	m.screen = ScreenSlotPick
	return m, nil
}

func (m Model) finishResolveFlow() (tea.Model, tea.Cmd) {
	r := m.resolve
	switch r.purpose {
	case purposeRunWorkflow:
		return m.startConfirmRun(r.resolved, r.workflowLabel)
	case purposeRunManually:
		return m.startConfirmRun(r.resolved, "manual")
	case purposeCreateWorkflow:
		if len(r.workflowVars) > 0 {
			return m.openConfirmVars(r.workflowVars)
		}
		return m.openNameInput(nameInputWorkflow)
	case purposeCreateAlias:
		if len(r.workflowVars) > 0 {
			return m.openConfirmVars(r.workflowVars)
		}
		return m.openNameInput(nameInputAlias)
	case purposeEditWorkflow:
		var cmdNames []string
		for _, item := range r.rawItems {
			cmdNames = append(cmdNames, item.name())
		}
		m.workflows[m.editTargetIdx].Commands = cmdNames
		m.workflows[m.editTargetIdx].Vars = nil
		if err := store.SaveWorkflows(m.projectDir, m.workflows); err != nil {
			m.errMsg = "failed to save workflows: " + err.Error()
		}
		m.resolve = nil
		m.gotoMainMenu()
		return m, nil
	case purposeEditAlias:
		var steps []string
		for _, item := range r.rawItems {
			steps = append(steps, item.name())
		}
		m.aliases[m.editTargetIdx].Steps = steps
		m.aliases[m.editTargetIdx].Vars = nil
		if err := store.SaveAliases(m.projectDir, m.aliases); err != nil {
			m.errMsg = "failed to save aliases: " + err.Error()
		}
		m.resolve = nil
		m.screen = ScreenAliasMgmt
		m.listItems = []string{"Create alias", "Edit alias", "Delete alias"}
		m.listCursor = 0
		return m, nil
	}
	m.gotoMainMenu()
	return m, nil
}

// ── Slot pick ─────────────────────────────────────────────────────────────────

func (m Model) updateSlotPick(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	sp := m.sp
	skipRow := len(sp.filtered) + 1
	total := skipRow
	if sp.canSkip {
		total++
	}

	switch msg.String() {
	case "up":
		sp.cursor = (sp.cursor - 1 + total) % total
	case "down", "tab":
		sp.cursor = (sp.cursor + 1) % total
	case "backspace":
		if len(sp.search) > 0 {
			sp.search = sp.search[:len(sp.search)-1]
			sp.applyFilter()
			sp.cursor = 0
		}
	case "enter":
		if sp.canSkip && sp.cursor == skipRow {
			return m.skipSlot()
		}
		if sp.cursor == len(sp.filtered) {
			if sp.search != "" {
				return m.acceptSlotValue(sp.search)
			}
		} else {
			return m.acceptSlotValue(sp.filtered[sp.cursor].Value)
		}
	case "esc":
		if sp.search != "" {
			sp.search = ""
			sp.applyFilter()
			sp.cursor = 0
			return m, nil
		}
		return m.goBackInResolve()
	default:
		if len(msg.Runes) == 1 && msg.Runes[0] >= 32 {
			sp.search += string(msg.Runes[0])
			sp.applyFilter()
			sp.cursor = 0
		}
	}
	return m, nil
}

func (m Model) skipSlot() (tea.Model, tea.Cmd) {
	r := m.resolve
	r.currentSlotIdx++

	if r.currentIdx < len(r.rawItems) {
		item := r.rawItems[r.currentIdx]
		if !item.isAlias() {
			partial := slot.Apply(*item.cmd, r.currentValues)
			dir := partial.Dir
			if dir == "" {
				dir = "."
			}
			r.itemNotes[r.currentIdx] = "$ " + partial.Cmd + "  (workdir: " + dir + ")"
		}
	}

	m.sp = nil
	m.screen = ScreenCreateWorkflow
	if r.purpose == purposeCreateAlias {
		m.screen = ScreenCreateAlias
	}
	return m.advanceResolve()
}

func (m Model) acceptSlotValue(value string) (tea.Model, tea.Cmd) {
	r := m.resolve
	r.currentValues[r.currentSlots[r.currentSlotIdx].Name] = value
	r.currentSlotIdx++

	if r.currentIdx < len(r.rawItems) {
		item := r.rawItems[r.currentIdx]
		if !item.isAlias() {
			partial := slot.Apply(*item.cmd, r.currentValues)
			partialDir := partial.Dir
			if partialDir == "" {
				partialDir = "."
			}
			note := "$ " + partial.Cmd + "  (workdir: " + partialDir + ")"
			r.itemNotes[r.currentIdx] = note
		}
	}

	m.sp = nil
	m.screen = ScreenRunManually
	if r.purpose == purposeCreateWorkflow {
		m.screen = ScreenCreateWorkflow
	} else if r.purpose == purposeCreateAlias {
		m.screen = ScreenCreateAlias
	} else if r.purpose == purposeRunWorkflow {
		m.screen = ScreenRunWorkflow
	}
	return m.advanceResolve()
}

func (m Model) goBackInResolve() (tea.Model, tea.Cmd) {
	r := m.resolve
	r.currentValues = nil
	r.currentSlots = nil
	r.currentSlotIdx = 0
	r.itemNotes[r.currentIdx] = ""

	if r.currentIdx == 0 {
		m.resolve = nil
		m.sp = nil
		switch r.purpose {
		case purposeCreateWorkflow:
			m.screen = ScreenCreateWorkflow
		case purposeCreateAlias:
			m.screen = ScreenCreateAlias
		case purposeRunWorkflow:
			m.screen = ScreenRunWorkflow
			return m, nil
		default:
			m.screen = ScreenRunManually
		}
		return m, m.setupMultiSelect(r.purpose == purposeRunManually)
	}

	r.currentIdx--
	if len(r.resolved) > 0 {
		r.resolved = r.resolved[:len(r.resolved)-1]
	}
	if r.purpose != purposeRunManually {
		prevName := r.rawItems[r.currentIdx].name()
		delete(r.workflowVars, prevName)
	}
	r.itemNotes[r.currentIdx] = ""

	for r.currentIdx > 0 {
		item := r.rawItems[r.currentIdx]
		if !item.isAlias() && len(slot.GetSlots(*item.cmd)) > 0 {
			break
		}
		if item.isAlias() && item.alias.Vars == nil {
			if len(m.collectAliasSlots(item.alias)) > 0 {
				break
			}
		}
		r.currentIdx--
		if len(r.resolved) > 0 {
			r.resolved = r.resolved[:len(r.resolved)-1]
		}
		r.itemNotes[r.currentIdx] = ""
	}

	// If we landed on an item with no slots, no earlier slotted item exists — go back to selection.
	landed := r.rawItems[r.currentIdx]
	hasSlots := (!landed.isAlias() && len(slot.GetSlots(*landed.cmd)) > 0) ||
		(landed.isAlias() && landed.alias.Vars == nil && len(m.collectAliasSlots(landed.alias)) > 0)
	if !hasSlots {
		m.resolve = nil
		m.sp = nil
		switch r.purpose {
		case purposeCreateWorkflow:
			m.screen = ScreenCreateWorkflow
		case purposeCreateAlias:
			m.screen = ScreenCreateAlias
		case purposeRunWorkflow:
			m.screen = ScreenRunWorkflow
			return m, nil
		default:
			m.screen = ScreenRunManually
		}
		return m, m.setupMultiSelect(r.purpose == purposeRunManually)
	}

	m.sp = nil
	return m.advanceResolve()
}

// ── Confirm vars ──────────────────────────────────────────────────────────────

func (m Model) openConfirmVars(vars map[string]map[string]string) (tea.Model, tea.Cmd) {
	var cmds []mdl.Command
	for _, item := range m.resolve.rawItems {
		if item.isAlias() {
			continue
		}
		if _, ok := vars[item.cmd.Name]; !ok {
			continue
		}
		cmds = append(cmds, *item.cmd)
	}
	m.cv = &confirmVarsState{
		cmds: cmds,
		vars: vars,
	}
	m.screen = ScreenConfirmVars
	return m, nil
}

func (m Model) updateConfirmVars(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cv := m.cv
	switch msg.String() {
	case "tab", "left", "right":
		cv.btn = 1 - cv.btn
	case "enter":
		if cv.btn == 0 {
			return m.openNameInput(nameInputMode(m.resolve.purpose - 1))
		}
		fallthrough
	case "esc":
		items := m.resolve.rawItems
		m.resolve = nil
		m.cv = nil
		return m.startResolveFlow(items)
	}
	return m, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func newMSTI(prompt string, active bool) textinput.Model {
	ti := textinput.New()
	ti.Prompt = prompt
	if active {
		ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("36"))
	} else {
		ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	}
	ti.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("97"))
	ti.Width = 22
	ti.CharLimit = 64
	return ti
}
