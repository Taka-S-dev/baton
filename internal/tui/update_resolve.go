package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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
	for i := range m.config.Base {
		m.msItems = append(m.msItems, msItem{cmd: &m.config.Base[i]})
	}
	for i := range m.config.Commands {
		m.msItems = append(m.msItems, msItem{cmd: &m.config.Commands[i]})
	}
	if includeAliases {
		for i := range m.aliases {
			m.msItems = append(m.msItems, msItem{alias: &m.aliases[i]})
		}
	}
	m.msSearchTI = newMSTI("/ ")
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
	terms := strings.Fields(strings.ToLower(m.msSearchTI.Value()))
	for i, item := range m.msItems {
		if matchesAllTerms(item.searchText(), terms) {
			out = append(out, i)
		}
	}
	return out
}

func (m Model) updateMultiSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	filtered := m.msFiltered()
	n := len(filtered)

	// While the discard window is up, Esc confirms and any other key
	// just closes it without acting on the list underneath.
	escArmed := m.msEscArmed
	m.msEscArmed = false
	if escArmed && msg.String() != "esc" {
		return m, nil
	}

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
		if m.msSearchTI.Value() != "" {
			m.msSearchTI.SetValue("")
			m.msCursor = 0
			m.msViewStart = 0
			return m, nil
		}
		if len(m.msSelected) > 0 && !escArmed {
			m.msEscArmed = true
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
		if m.screen == ScreenCreateWorkflow {
			m.gotoWorkflowMgmt()
			return m, nil
		}
		if m.screen == ScreenCreateAlias {
			m.gotoAliasMgmt()
			return m, nil
		}
		m.gotoMainMenu()
	default:
		prevSearch := m.msSearchTI.Value()
		var cmd tea.Cmd
		m.msSearchTI, cmd = m.msSearchTI.Update(msg)
		if m.msSearchTI.Value() != prevSearch {
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
	purpose := purposeRunCommands
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

		// Command (template-derived ones are baked at load time).
		// Remaining {slots} — including ones skipped at creation — are
		// resolved interactively.
		if r.currentValues == nil {
			r.currentValues = make(map[string]string)
			r.currentSlots = slot.GetSlots(*item.cmd)
			r.currentSlotIdx = 0
		}
		if r.currentSlotIdx >= len(r.currentSlots) {
			resolved := slot.Apply(*item.cmd, r.currentValues)
			if r.itemNotes[r.currentIdx] == "" {
				r.itemNotes[r.currentIdx] = m.cmdNote(resolved)
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
		cmd, ok := m.lookupStepCommand(stepName)
		if !ok {
			continue
		}
		for _, s := range slot.GetSlots(cmd) {
			if !seen[s.Name] {
				seen[s.Name] = true
				slots = append(slots, s)
			}
		}
	}
	return slots
}

// lookupStepCommand resolves a step name to a runnable command
// (template-derived commands are already baked at load time).
func (m Model) lookupStepCommand(name string) (mdl.Command, bool) {
	return m.config.FindCommand(name)
}

// cmdNote formats the "$ cmd (workdir: dir)" line shown in list contexts,
// with project variables resolved so notes show real paths.
func (m Model) cmdNote(cmd mdl.Command) string {
	cmd = slot.ApplyVarsToCommand(cmd, m.vars)
	dir := cmd.Dir
	if dir == "" {
		dir = "."
	}
	return "$ " + cmd.Cmd + "  (workdir: " + dir + ")"
}

// partialNote returns the display note for an item mid-resolution, or ""
// when the item has no directly displayable command (aliases).
func (m Model) partialNote(item msItem, values map[string]string) string {
	if item.cmd == nil {
		return ""
	}
	return m.cmdNote(slot.Apply(*item.cmd, values))
}

// itemNeedsSlots reports whether the item still has slots to resolve interactively.
func (m Model) itemNeedsSlots(item msItem) bool {
	if item.isAlias() {
		return item.alias.Vars == nil && len(m.collectAliasSlots(item.alias)) > 0
	}
	return len(slot.GetSlots(*item.cmd)) > 0
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
	case purposeRunCommands:
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
		} else {
			m.successMsg = "updated workflow \"" + m.workflows[m.editTargetIdx].Name + "\""
		}
		m.resolve = nil
		m.gotoWorkflowMgmt()
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
		} else {
			m.successMsg = "updated alias \"" + m.aliases[m.editTargetIdx].Name + "\""
		}
		m.resolve = nil
		m.gotoAliasMgmt()
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
			if m.sce != nil {
				return m.skipCommandEditSlot()
			}
			return m.skipSlot()
		}
		if sp.cursor == len(sp.filtered) {
			if sp.search != "" {
				if m.sce != nil {
					return m.acceptCommandEditSlotValue(sp.search)
				}
				return m.acceptSlotValue(sp.search)
			}
		} else {
			if m.sce != nil {
				return m.acceptCommandEditSlotValue(sp.filtered[sp.cursor].Value)
			}
			return m.acceptSlotValue(sp.filtered[sp.cursor].Value)
		}
	case "esc":
		if sp.search != "" {
			sp.search = ""
			sp.applyFilter()
			sp.cursor = 0
			return m, nil
		}
		if m.sce != nil {
			return m.goBackCommandEditSlot()
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
		if note := m.partialNote(item, r.currentValues); note != "" {
			r.itemNotes[r.currentIdx] = note
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
		if note := m.partialNote(item, r.currentValues); note != "" {
			r.itemNotes[r.currentIdx] = note
		}
	}

	m.sp = nil
	m.screen = ScreenRunCommands
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
			m.screen = ScreenRunCommands
		}
		return m, m.setupMultiSelect(r.purpose == purposeRunCommands)
	}

	r.currentIdx--
	if len(r.resolved) > 0 {
		r.resolved = r.resolved[:len(r.resolved)-1]
	}
	if r.purpose != purposeRunCommands {
		prevName := r.rawItems[r.currentIdx].name()
		delete(r.workflowVars, prevName)
	}
	r.itemNotes[r.currentIdx] = ""

	for r.currentIdx > 0 {
		if m.itemNeedsSlots(r.rawItems[r.currentIdx]) {
			break
		}
		r.currentIdx--
		if len(r.resolved) > 0 {
			r.resolved = r.resolved[:len(r.resolved)-1]
		}
		r.itemNotes[r.currentIdx] = ""
	}

	// If we landed on an item with no slots, no earlier slotted item exists — go back to selection.
	if !m.itemNeedsSlots(r.rawItems[r.currentIdx]) {
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
			m.screen = ScreenRunCommands
		}
		return m, m.setupMultiSelect(r.purpose == purposeRunCommands)
	}

	m.sp = nil
	return m.advanceResolve()
}

// ── Confirm vars ──────────────────────────────────────────────────────────────

func (m Model) openConfirmVars(vars map[string]map[string]string) (tea.Model, tea.Cmd) {
	var cmds []mdl.Command
	for _, item := range m.resolve.rawItems {
		if item.isAlias() || item.cmd == nil {
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
			mode := nameInputWorkflow
			if m.resolve.purpose == purposeCreateAlias {
				mode = nameInputAlias
			}
			return m.openNameInput(mode)
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

func newMSTI(prompt string) textinput.Model {
	ti := textinput.New()
	ti.Prompt = prompt
	ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("36"))
	ti.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("97"))
	ti.Width = 22
	ti.CharLimit = 64
	return ti
}
