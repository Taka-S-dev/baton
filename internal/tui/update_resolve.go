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

func (m *Model) setupMultiSelect() tea.Cmd {
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
	m.msSearchTI = newMSTI("/ ")
	return m.msSearchTI.Focus()
}

func (m *Model) setupMultiSelectWithPreSelected(cmdNames []string) tea.Cmd {
	cmd := m.setupMultiSelect()
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
		sel := m.msSelected
		if len(sel) == 0 {
			// Nothing toggled: Enter acts on the hovered row (fzf-style),
			// same as the slot picker's variadic fallback.
			if n == 0 || m.msCursor >= n {
				break
			}
			sel = []int{filtered[m.msCursor]}
		}
		selected := make([]msItem, len(sel))
		names := make([]string, len(sel))
		for i, idx := range sel {
			selected[i] = m.msItems[idx]
			names[i] = m.msItems[idx].name()
		}
		switch m.screen {
		case ScreenCreateWorkflow:
			// Workflows store no values: creation is pick and name.
			m.pendingWorkflowCmds = names
			return m.openNameInput(nameInputWorkflow)
		case ScreenEditWorkflowCommands:
			m.workflows[m.editTargetIdx].Commands = names
			if err := store.SaveWorkflows(m.projectDir, m.workflows); err != nil {
				m.errMsg = "failed to save workflows: " + err.Error()
			} else {
				m.successMsg = "updated workflow \"" + m.workflows[m.editTargetIdx].Name + "\""
			}
			m.gotoWorkflowMgmt()
			return m, nil
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
		if m.screen == ScreenCreateWorkflow {
			m.gotoWorkflowMgmt()
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
	m.resolve = &resolveFlowState{
		purpose:   purposeRunCommands,
		rawItems:  items,
		itemNames: names,
		itemNotes: make([]string, len(items)),
	}
	return m.advanceResolve()
}

func (m Model) advanceResolve() (tea.Model, tea.Cmd) {
	r := m.resolve
	for r.currentIdx < len(r.rawItems) {
		item := r.rawItems[r.currentIdx]

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

// partialNote returns the display note for an item mid-resolution.
func (m Model) partialNote(item msItem, values map[string]string) string {
	if item.cmd == nil {
		return ""
	}
	return m.cmdNote(slot.Apply(*item.cmd, values))
}

// itemNeedsSlots reports whether the item still has slots to resolve interactively.
func (m Model) itemNeedsSlots(item msItem) bool {
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
		variadic:      s.Variadic,
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
	if r.purpose == purposeRunWorkflow {
		return m.startConfirmRun(r.resolved, r.workflowLabel)
	}
	return m.startConfirmRun(r.resolved, "manual")
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
	case "down":
		sp.cursor = (sp.cursor + 1) % total
	case "tab":
		if !sp.variadic {
			sp.cursor = (sp.cursor + 1) % total
			break
		}
		// Variadic slot: Tab toggles the hovered entry, or adds the typed
		// custom value and clears the input for the next one.
		if sp.cursor < len(sp.filtered) {
			sp.togglePicked(sp.filtered[sp.cursor].Value)
		} else if sp.cursor == len(sp.filtered) && sp.search != "" {
			sp.togglePicked(sp.search)
			sp.search = ""
			sp.applyFilter()
			sp.cursor = 0
		}
	case "enter":
		if sp.canSkip && sp.cursor == skipRow && m.sce != nil {
			return m.skipCommandEditSlot()
		}
		if sp.variadic && len(sp.picked) > 0 {
			joined := strings.Join(sp.picked, " ")
			if m.sce != nil {
				return m.acceptCommandEditSlotValue(joined)
			}
			return m.acceptSlotValue(joined)
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
	if r.purpose == purposeRunWorkflow {
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
		if r.purpose == purposeRunWorkflow {
			m.screen = ScreenRunWorkflow
			return m, nil
		}
		m.screen = ScreenRunCommands
		return m, m.setupMultiSelect()
	}

	r.currentIdx--
	if len(r.resolved) > 0 {
		r.resolved = r.resolved[:len(r.resolved)-1]
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
		if r.purpose == purposeRunWorkflow {
			m.screen = ScreenRunWorkflow
			return m, nil
		}
		m.screen = ScreenRunCommands
		return m, m.setupMultiSelect()
	}

	m.sp = nil
	return m.advanceResolve()
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
