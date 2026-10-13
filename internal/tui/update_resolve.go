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
	m.loadCommandSort()
	m.msItems = nil
	m.msCursor = 0
	m.msSelected = nil
	m.runQueueFocused = false
	m.runQueueCursor = 0
	for i := range m.config.Base {
		m.msItems = append(m.msItems, msItem{cmd: &m.config.Base[i]})
	}
	for i := range m.config.Commands {
		m.msItems = append(m.msItems, msItem{cmd: &m.config.Commands[i]})
	}
	m.msSearchTI = newMSTI("/ ")
	m.msSearchTI.Blur()
	return nil
}

func (m *Model) setupMultiSelectWithPreSelected(cmdNames []string) tea.Cmd {
	cmd := m.setupMultiSelect()
	// Pre-select in cmdNames order — msSelected order IS the step order,
	// so walking msItems here would silently rewrite the workflow to
	// list order on save.
	idxByName := make(map[string]int, len(m.msItems))
	for i, item := range m.msItems {
		idxByName[item.name()] = i
	}
	for _, n := range cmdNames {
		if i, ok := idxByName[n]; ok {
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
	m.sortCommandIndices(out)
	return out
}

func (m Model) updateMultiSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.msScrollManual = false
	filtered := m.msFiltered()
	n := len(filtered)

	escArmed := false
	if m.msEscArmed {
		switch msg.String() {
		case "up", "down", "tab":
			m.discardCursor = 1 - m.discardCursor
			return m, nil
		case "esc", "k":
			m.msEscArmed = false
			return m, nil
		case "d":
			escArmed = true
		case "enter":
			if m.discardCursor == 0 {
				m.msEscArmed = false
				return m, nil
			}
			escArmed = true
		default:
			return m, nil
		}
		m.msEscArmed = false
		m.msSelected = nil
		m.msSearchTI.SetValue("")
		msg = tea.KeyMsg{Type: tea.KeyEsc}
	}
	if !m.runHomeVisible() && !m.msEscArmed && !escArmed {
		if msg.String() == "f2" || msg.String() == "ctrl+q" {
			m.msSearchTI.Blur()
			m.runQueueFocused = !m.runQueueFocused
			return m, nil
		}
		if m.msSearchTI.Focused() && msg.String() != "tab" {
			query := m.msSearchTI.Value()
			handled, changed, _, cmd := updateSearchInput(&m.msSearchTI, &query, msg, n > 0 || (msg.String() == "enter" && len(m.msSelected) > 0))
			if changed {
				m.msCursor = 0
			}
			if handled {
				return m, cmd
			}
		} else if msg.String() == "/" {
			m.runQueueFocused = false
			return m, startSearchInput(&m.msSearchTI, m.msSearchTI.Value())
		} else if msg.String() == "s" && !m.runQueueFocused {
			m.cycleCommandSort()
			return m, nil
		}
	}
	if !escArmed && !m.msSearchTI.Focused() {
		switch msg.String() {
		case "d":
			m.runDetailOpen = true
			m.scrollRunDetail(0)
			return m, nil
		case "pgup":
			m.scrollRunDetail(-5)
			return m, nil
		case "pgdown":
			m.scrollRunDetail(5)
			return m, nil
		}
	}
	if msg.String() == "ctrl+q" || msg.String() == "f2" {
		m.runQueueFocused = !m.runQueueFocused
		return m, nil
	}
	if m.runQueueFocused && !escArmed {
		n := len(m.msSelected)
		m.runQueueCursor = min(m.runQueueCursor, max(0, n-1))
		i := m.runQueueCursor
		switch msg.String() {
		case "left":
			m.runQueueFocused = false
			return m, nil
		case "esc":
			m.runQueueFocused = false
			return m, nil
		case "up":
			m.runQueueCursor = max(0, i-1)
			return m, nil
		case "down":
			m.runQueueCursor = min(max(0, n-1), i+1)
			return m, nil
		case "K", "shift+up":
			if i > 0 {
				m.msSelected[i-1], m.msSelected[i] = m.msSelected[i], m.msSelected[i-1]
				m.runQueueCursor--
			}
			return m, nil
		case "J", "shift+down":
			if i < n-1 {
				m.msSelected[i+1], m.msSelected[i] = m.msSelected[i], m.msSelected[i+1]
				m.runQueueCursor++
			}
			return m, nil
		case "x", "delete":
			if n > 0 {
				m.msSelected = append(m.msSelected[:i], m.msSelected[i+1:]...)
				m.runQueueCursor = min(i, max(0, n-2))
			}
			return m, nil
		case "enter":
			if n == 0 {
				return m, nil
			}
		default:
			return m, nil
		}
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
			return m, nil
		}
		if len(m.msSelected) > 0 && !escArmed {
			m.msEscArmed = true
			m.discardCursor = 0
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
		m.openRunProjects()
	default:
		return m, nil
	}
	return m, nil
}

// ── Resolve flow ──────────────────────────────────────────────────────────────

func (m Model) startResolveFlow(items []msItem) (tea.Model, tea.Cmd) {
	m.confirmBack = nil
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

// slotAnswerKey identifies a question within one run. The same slot name
// drawing from the same list is the same question; the same name mapped
// to a different list (via the slots column) is a different one, so the
// two never share an answer.
func slotAnswerKey(s slot.Def) string { return s.Name + "\x00" + s.ListName }

func (m Model) openSlotPick(s slot.Def, cmd *mdl.Command) (tea.Model, tea.Cmd) {
	entries := m.lists[s.ListName]
	r := m.resolve
	sp := &slotPickState{
		pathDef:       s,
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
	if s.PathKind != "" {
		sp.entries = nil
	}
	sp.applyFilter()
	// An earlier step already answered this question — clean and build
	// asking for the same directory is the everyday case. Open on that
	// answer so repeating it costs one Enter, and never apply it on the
	// user's behalf: steps that need different values still just pick
	// them, the way the same command twice in a workflow expects.
	if prev, ok := r.answered[slotAnswerKey(s)]; ok {
		sp.reuseFrom, sp.reuseValue = prev.by, prev.value
		if s.Variadic {
			sp.picked = strings.Fields(prev.value)
		} else if i := indexOfEntry(sp.filtered, prev.value); i >= 0 {
			sp.cursor = i
		} else {
			// Typed by hand rather than picked: offer it back as the
			// custom-value row, which sits just past the entries.
			sp.search = prev.value
			sp.applyFilter()
			sp.cursor = len(sp.filtered)
		}
	}
	m.sp = sp
	m.screen = ScreenSlotPick
	if s.PathKind != "" {
		return m, m.startPathCompletion()
	}
	return m, startSearchInput(&sp.input, sp.search)
}

// indexOfEntry returns the position of value in entries, or -1.
func indexOfEntry(entries []mdl.ListEntry, value string) int {
	for i, e := range entries {
		if e.Value == value {
			return i
		}
	}
	return -1
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
	if sp.pathDef.PathKind != "" && msg.String() == "ctrl+t" {
		return m, nil
	}
	if sp.pathDef.PathKind != "" && sp.canSkip && m.sce != nil && msg.String() == "ctrl+k" {
		return m.skipCommandEditSlot()
	}
	if sp.completion != nil {
		return m.updatePathCompletion(msg)
	}
	if sp.pathDef.PathKind != "" && (msg.String() == "ctrl+@" || msg.String() == "ctrl+space" || msg.String() == "tab" || (msg.String() == "enter" && sp.completedPath == "" && len(sp.picked) == 0)) {
		return m, m.startPathCompletion()
	}
	skipRow := sp.valueSkipRow()
	total := skipRow
	if sp.canSkip {
		total++
	}

	sp.cursor = min(max(0, sp.cursor), max(0, total-1))
	switch msg.String() {
	case "up":
		if total == 0 {
			return m, nil
		}
		sp.cursor = (sp.cursor - 1 + total) % total
	case "down":
		if total == 0 {
			return m, nil
		}
		sp.cursor = (sp.cursor + 1) % total
	case "tab", "ctrl+t":
		if total == 0 {
			return m, nil
		}
		if !sp.variadic {
			sp.cursor = (sp.cursor + 1) % total
			break
		}
		// Variadic slot: Ctrl+T toggles the hovered entry, or adds the typed
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
	case "backspace":
		if sp.search != "" {
			r := []rune(sp.search)
			sp.search = string(r[:len(r)-1])
			sp.applyFilter()
			sp.cursor = 0
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
	m.rememberConfirmBack()
	if m.sp != nil && m.sp.pathDef.PathKind != "" && m.projectsDir != "" && m.sp.currentCmd != nil {
		focus := m.sp.completedPath
		if m.sp.variadic && len(m.sp.picked) > 0 {
			focus = m.sp.picked[len(m.sp.picked)-1]
		}
		if base, err := m.completionBase(); err == nil && focus != "" {
			if err := store.SavePathFocus(m.projectsDir, base, m.projectDir, m.sp.currentCmd.Name, m.sp.placeholder(), focus); err != nil {
				m.errMsg = "Could not remember file focus: " + err.Error()
			}
		}
	}
	r := m.resolve
	def := r.currentSlots[r.currentSlotIdx]
	r.currentValues[def.Name] = value
	if r.answered == nil {
		r.answered = make(map[string]answeredSlot)
	}
	r.answered[slotAnswerKey(def)] = answeredSlot{value: value, by: r.itemNames[r.currentIdx]}
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

	if r.currentIdx == 0 && m.recentActive {
		m.resolve = nil
		m.sp = nil
		m.gotoRecent()
		return m, nil
	}
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
