package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Taka-S-dev/baton/internal/config"
	mdl "github.com/Taka-S-dev/baton/internal/model"
	"github.com/Taka-S-dev/baton/internal/slot"
)

// ── Manage menu (Create / Edit / Delete) ──────────────────────────────────────

func (m *Model) updateManageCommands(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "down":
		m.moveListCursor(msg.String(), 3)
	case "enter":
		switch m.listCursor {
		case 0: // Create
			if len(m.templateCandidates()) == 0 {
				// No templates — direct input is the only option.
				return m.openCommandForm(-1)
			}
			m.screen = ScreenCreateCommandKind
			m.listItems = []string{"Write directly", "From template"}
			m.listCursor = 0
		case 1: // Edit
			names, refs := m.editableCommands()
			if len(refs) == 0 {
				m.errMsg = "no editable commands (TSV rows and TUI-created commands)"
				break
			}
			m.screen = ScreenEditCommandPick
			m.listItems = names
			m.editRefs = refs
			m.listCursor = 0
		case 2: // Delete
			names, refs := m.editableCommands()
			if len(refs) == 0 {
				m.errMsg = "no editable commands (TSV rows and TUI-created commands)"
				break
			}
			m.screen = ScreenDeleteCommand
			m.listItems = names
			m.editRefs = refs
			m.listCursor = 0
			m.deleteSelected = nil
		}
	case "esc":
		m.gotoMainMenu()
	}
	return m, nil
}

// ── Create: choose how to create (direct input / from template) ──────────────

func (m *Model) updateCreateCommandKind(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "down":
		m.moveListCursor(msg.String(), len(m.listItems))
	case "enter":
		if m.listCursor == 0 {
			return m.openCommandForm(-1)
		}
		m.screen = ScreenCreateCommandTemplate
		m.sce = &commandEditState{
			mode:          0,
			currentValues: make(map[string]string),
		}
	case "esc":
		m.screen = ScreenManageCommands
		m.listCursor = 0
	}
	return m, nil
}

// ── Command form: write a concrete command directly ──────────────────────────

// openCommandForm opens the direct-input form. Pass editIdx -1 to create,
// or the index of a concrete command in config.Commands to edit it.
func (m *Model) openCommandForm(editIdx int) (tea.Model, tea.Cmd) {
	cf := &commandFormState{mode: 0, editIdx: -1}
	if editIdx >= 0 && editIdx < len(m.config.Commands) {
		cmd := m.config.Commands[editIdx]
		cf.mode = 1
		cf.editIdx = editIdx
		cf.fields = [5]string{cmd.Name, cmd.Cmd, cmd.Dir, cmd.Group, cmd.Shell}
	}
	m.cf = cf
	m.screen = ScreenCommandForm
	m.nameInput.Prompt = commandFormLabels[0] + " > "
	m.nameInput.SetValue(cf.fields[0])
	return m, m.nameInput.Focus()
}

// openCommandFormTSV opens the form on a TSV row (config.Base index).
// Saving rewrites exactly that row via UpdateCommandTSV.
func (m *Model) openCommandFormTSV(baseIdx int) (tea.Model, tea.Cmd) {
	cmd := m.config.Base[baseIdx]
	cf := &commandFormState{mode: 1, editIdx: baseIdx, tsvEdit: true, origName: cmd.Name}
	cf.fields = [5]string{cmd.Name, cmd.Cmd, cmd.Dir, cmd.Group, cmd.Shell}
	m.cf = cf
	m.screen = ScreenCommandForm
	m.nameInput.Prompt = commandFormLabels[0] + " > "
	m.nameInput.SetValue(cf.fields[0])
	return m, m.nameInput.Focus()
}

// formNameTaken applies the duplicate-name check for the form, honoring
// whichever row is being edited.
func (m *Model) formNameTaken(name string) bool {
	cf := m.cf
	if cf.tsvEdit {
		return name != cf.origName && m.commandNameTaken(name, -1)
	}
	excludeIdx := -1
	if cf.mode == 1 {
		excludeIdx = cf.editIdx
	}
	return m.commandNameTaken(name, excludeIdx)
}

func (m *Model) closeCommandForm() {
	m.cf = nil
	m.nameInput.Prompt = "Name > "
	m.screen = ScreenManageCommands
}

// slotInsertAvailable reports whether the slot-insert panel applies to the
// current form field (only cmd and workdir accept {slot} placeholders).
func (cf *commandFormState) slotInsertAvailable(listCount int) bool {
	return (cf.fieldIdx == 1 || cf.fieldIdx == 2) && listCount > 0
}

// cfGotoField focuses the given form field, loading its stored text.
func (m *Model) cfGotoField(idx int) tea.Cmd {
	m.cf.fieldIdx = idx
	m.cf.slotPickFocus = false
	m.nameInput.Prompt = commandFormLabels[idx] + " > "
	m.nameInput.SetValue(m.cf.fields[idx])
	m.nameInput.CursorEnd()
	return m.nameInput.Focus()
}

// saveCommandForm validates every form field and persists the command.
// On a validation failure it focuses the offending field instead.
func (m *Model) saveCommandForm() (tea.Model, tea.Cmd) {
	cf := m.cf
	name, cmdStr := cf.fields[0], cf.fields[1]
	if name == "" {
		m.errMsg = "name cannot be empty"
		return m, m.cfGotoField(0)
	}
	if cmdStr == "" {
		m.errMsg = "cmd cannot be empty"
		return m, m.cfGotoField(1)
	}
	shell := strings.ToLower(cf.fields[4])
	if shell != "" && shell != "ps" {
		m.errMsg = `shell must be empty or "ps"`
		return m, m.cfGotoField(4)
	}
	if m.formNameTaken(name) {
		m.errMsg = "name already in use: " + name
		return m, m.cfGotoField(0)
	}
	if cf.tsvEdit {
		newCmd := m.config.Base[cf.editIdx] // keeps Slots and Source
		newCmd.Name, newCmd.Cmd, newCmd.Dir, newCmd.Group, newCmd.Shell = name, cmdStr, cf.fields[2], cf.fields[3], shell
		if file, err := config.UpdateCommandTSV(m.projectDir, cf.origName, newCmd); err != nil {
			m.errMsg = "failed to save: " + err.Error()
		} else {
			m.config.Base[cf.editIdx] = newCmd
			m.successMsg = "updated command \"" + name + "\" in " + file
		}
		m.closeCommandForm()
		m.listCursor = 1
		return m, nil
	}
	if cf.mode == 1 {
		cmd := m.config.Commands[cf.editIdx]
		cmd.Name, cmd.Cmd, cmd.Dir, cmd.Group, cmd.Shell = name, cmdStr, cf.fields[2], cf.fields[3], shell
		m.config.Commands[cf.editIdx] = cmd
		if err := m.saveConfig(); err != nil {
			m.errMsg = "failed to save: " + err.Error()
		} else {
			m.successMsg = "updated command \"" + name + "\""
		}
		m.closeCommandForm()
		m.listCursor = 0
		return m, nil
	}
	// New commands written in the form are hand-editable definitions, so
	// they go into the project's TSV (append-only; existing rows are
	// never touched). Template-derived commands stay in the local layer.
	newCmd := mdl.Command{
		Name:  name,
		Cmd:   cmdStr,
		Dir:   cf.fields[2],
		Group: cf.fields[3],
		Shell: shell,
	}
	if file, err := config.AppendCommandTSV(m.projectDir, newCmd); err != nil {
		m.errMsg = "failed to save: " + err.Error()
	} else {
		m.config.Base = append(m.config.Base, newCmd)
		m.successMsg = "created command \"" + name + "\" in " + file
	}
	m.closeCommandForm()
	m.listCursor = 0
	return m, nil
}

func (m *Model) updateCommandForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.cf == nil {
		return m, nil
	}
	cf := m.cf

	// Placeholder picker window has key focus. Left pane inserts {name},
	// right pane inserts the selected concrete value, both at the cursor
	// position of the field being edited.
	if cf.slotPickFocus {
		names := m.sortedListNames()
		var entries []mdl.ListEntry
		if cf.slotPickCursor < len(names) {
			entries = m.lists[names[cf.slotPickCursor]]
		}
		insertAtCursor := func(text string) {
			v := []rune(m.nameInput.Value())
			pos := m.nameInput.Position()
			if pos > len(v) {
				pos = len(v)
			}
			m.nameInput.SetValue(string(v[:pos]) + text + string(v[pos:]))
			m.nameInput.SetCursor(pos + len([]rune(text)))
			cf.slotPickFocus = false
		}

		switch msg.String() {
		case "up":
			if cf.slotPickPane == 1 {
				if cf.slotPickValueCursor > 0 {
					cf.slotPickValueCursor--
				}
			} else if cf.slotPickCursor > 0 {
				cf.slotPickCursor--
				cf.slotPickValueCursor = 0
			}
		case "down":
			if cf.slotPickPane == 1 {
				if cf.slotPickValueCursor < len(entries)-1 {
					cf.slotPickValueCursor++
				}
			} else if cf.slotPickCursor < len(names)-1 {
				cf.slotPickCursor++
				cf.slotPickValueCursor = 0
			}
		case "right":
			if cf.slotPickPane == 0 && len(entries) > 0 {
				cf.slotPickPane = 1
				cf.slotPickValueCursor = 0
			}
		case "left":
			cf.slotPickPane = 0
		case "enter":
			if cf.slotPickPane == 1 && cf.slotPickValueCursor < len(entries) {
				insertAtCursor(entries[cf.slotPickValueCursor].Value)
			} else if cf.slotPickPane == 0 && cf.slotPickCursor < len(names) {
				insertAtCursor("{" + names[cf.slotPickCursor] + "}")
			}
		case "tab", "esc":
			cf.slotPickFocus = false
		}
		return m, nil
	}

	switch msg.String() {
	case "tab":
		if cf.slotInsertAvailable(len(m.lists)) {
			cf.slotPickFocus = true
			cf.slotPickCursor = 0
			cf.slotPickPane = 0
			cf.slotPickValueCursor = 0
		}
		return m, nil
	case "enter":
		cf.fields[cf.fieldIdx] = strings.TrimSpace(m.nameInput.Value())
		// Validate the name as soon as the field is confirmed, so a
		// duplicate surfaces immediately instead of after the last field.
		if cf.fieldIdx == 0 {
			name := cf.fields[0]
			if name == "" {
				m.errMsg = "name cannot be empty"
				return m, m.cfGotoField(0)
			}
			if m.formNameTaken(name) {
				m.errMsg = "name already in use: " + name
				return m, m.cfGotoField(0)
			}
		}
		if cf.fieldIdx < len(cf.fields)-1 {
			return m, m.cfGotoField(cf.fieldIdx + 1)
		}
		return m.saveCommandForm()
	case "ctrl+s":
		// Save from any field — no need to walk the rest of the form.
		cf.fields[cf.fieldIdx] = strings.TrimSpace(m.nameInput.Value())
		return m.saveCommandForm()
	case "up":
		if cf.fieldIdx > 0 {
			cf.fields[cf.fieldIdx] = strings.TrimSpace(m.nameInput.Value())
			return m, m.cfGotoField(cf.fieldIdx - 1)
		}
	case "down":
		if cf.fieldIdx < len(cf.fields)-1 {
			cf.fields[cf.fieldIdx] = strings.TrimSpace(m.nameInput.Value())
			return m, m.cfGotoField(cf.fieldIdx + 1)
		}
	case "esc":
		if cf.fieldIdx > 0 {
			cf.fields[cf.fieldIdx] = strings.TrimSpace(m.nameInput.Value())
			return m, m.cfGotoField(cf.fieldIdx - 1)
		}
		m.closeCommandForm()
		m.listCursor = 0
	default:
		ti, cmd := m.nameInput.Update(msg)
		m.nameInput = ti
		return m, cmd
	}
	return m, nil
}

// ── Edit: pick which command to edit ──────────────────────────────────────────

// editableCommands lists the commands the TUI can edit or delete —
// TSV rows and local template-derived commands — in display order.
func (m *Model) editableCommands() ([]string, []editRef) {
	var names []string
	var refs []editRef
	for i, c := range m.config.Base {
		if c.Source == "tsv" {
			names = append(names, c.Name)
			refs = append(refs, editRef{tsv: true, idx: i})
		}
	}
	for i, c := range m.config.Commands {
		names = append(names, c.Name)
		refs = append(refs, editRef{idx: i})
	}
	return names, refs
}

// editRefCommand resolves an editRef to the command it points at.
func (m *Model) editRefCommand(ref editRef) *mdl.Command {
	if ref.tsv {
		return &m.config.Base[ref.idx]
	}
	return &m.config.Commands[ref.idx]
}

// gotoEditCommandPick opens the edit picker with the cursor on ref.
func (m *Model) gotoEditCommandPick(at editRef) {
	names, refs := m.editableCommands()
	m.screen = ScreenEditCommandPick
	m.listItems = names
	m.editRefs = refs
	m.listCursor = 0
	for i, r := range refs {
		if r == at {
			m.listCursor = i
			break
		}
	}
}

func (m *Model) updateEditCommandPick(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "down":
		m.moveListCursor(msg.String(), len(m.listItems))
	case "enter":
		if m.listCursor >= len(m.editRefs) {
			break
		}
		ref := m.editRefs[m.listCursor]
		if ref.tsv {
			return m.openCommandFormTSV(ref.idx)
		}
		cmd := m.config.Commands[ref.idx]
		if cmd.Template == "" {
			return m.openCommandForm(ref.idx)
		}
		idx := m.getTemplateRefIdx(cmd.Template)
		if idx < 0 {
			m.errMsg = "template not found: " + cmd.Template
			break
		}
		m.sce = &commandEditState{
			mode:           1,
			editIdx:        ref.idx,
			name:           cmd.Name,
			templateRefIdx: idx,
			currentValues:  make(map[string]string),
		}
		m.gotoEditCommandMode()
	case "esc":
		m.screen = ScreenManageCommands
		m.listCursor = 1
	}
	return m, nil
}

// gotoEditCommandMode opens the edit submenu for the template-derived
// command held in m.sce.
func (m *Model) gotoEditCommandMode() {
	m.screen = ScreenEditCommandMode
	m.listItems = []string{"Rename", "Change values", "Change template"}
	m.listCursor = 0
}

func (m *Model) updateEditCommandMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "down":
		m.moveListCursor(msg.String(), len(m.listItems))
	case "enter":
		switch m.listItems[m.listCursor] {
		case "Rename":
			m.nameInput.SetValue(m.sce.name)
			m.nameInputMode = nameInputRenameCommand
			m.nameInputErr = ""
			m.screen = ScreenNameInput
			return m, m.nameInput.Focus()
		case "Change values":
			// Straight to the slot pickers on the current template.
			m.sce.pickedTemplate = false
			return m.startCommandSlots()
		case "Change template":
			m.sce.pickedTemplate = true
			m.screen = ScreenEditCommandTemplate
		}
	case "esc":
		editIdx := m.sce.editIdx
		m.sce = nil
		m.gotoEditCommandPick(editRef{idx: editIdx})
	}
	return m, nil
}

// renameCommand renames the command being edited without touching its
// template or values; vars.tsv rows follow the new name on save.
func (m Model) renameCommand(name string) (tea.Model, tea.Cmd) {
	sce := m.sce
	if sce == nil || sce.editIdx < 0 || sce.editIdx >= len(m.config.Commands) {
		m.sce = nil
		m.screen = ScreenManageCommands
		m.listCursor = 1
		return m, nil
	}
	if m.commandNameTaken(name, sce.editIdx) {
		m.nameInputErr = "name already in use: " + name
		return m, nil
	}
	m.config.Commands[sce.editIdx].Name = name
	if err := m.saveConfig(); err != nil {
		m.errMsg = "failed to save: " + err.Error()
	} else {
		m.successMsg = "renamed command to \"" + name + "\""
	}
	m.sce = nil
	m.screen = ScreenManageCommands
	m.listCursor = 1
	return m, nil
}

// ── Create flow: template pick → slot resolution → name input ────────────────

func (m *Model) updateCreateCommand(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.sce == nil {
		return m, nil
	}
	sce := m.sce

	if m.screen == ScreenCreateCommandTemplate {
		switch msg.String() {
		case "up":
			if sce.templateRefIdx > 0 {
				sce.templateRefIdx--
			}
		case "down":
			if sce.templateRefIdx < len(m.templateCandidates())-1 {
				sce.templateRefIdx++
			}
		case "enter":
			return m.startCommandSlots()
		case "esc":
			m.sce = nil
			m.screen = ScreenCreateCommandKind
			m.listItems = []string{"Write directly", "From template"}
			m.listCursor = 1
		}
		return m, nil
	}

	// ScreenCreateCommandName (name input after slots)
	switch msg.String() {
	case "enter":
		name := m.nameInput.Value()
		if name == "" {
			m.errMsg = "name cannot be empty"
			break
		}
		if m.commandNameTaken(name, -1) {
			m.errMsg = "name already in use: " + name
			break
		}
		candidates := m.templateCandidates()
		if sce.templateRefIdx < 0 || sce.templateRefIdx >= len(candidates) {
			m.errMsg = "invalid template"
			break
		}
		tpl := candidates[sce.templateRefIdx]
		entry := mdl.Command{
			Name:     name,
			Template: tpl.Name,
			Values:   sce.currentValues,
		}
		entry, _ = slot.MaterializeCommand(entry, m.config)
		m.config.Commands = append(m.config.Commands, entry)
		if err := m.saveConfig(); err != nil {
			m.errMsg = "failed to save: " + err.Error()
		} else {
			m.successMsg = "created command \"" + name + "\""
		}
		m.screen = ScreenManageCommands
		m.listCursor = 0
		m.sce = nil
	case "esc":
		return m.backFromCommandNameInput()
	default:
		ti, cmd := m.nameInput.Update(msg)
		m.nameInput = ti
		return m, cmd
	}
	return m, nil
}

// ── Edit flow: template pick → slot resolution → name input ──────────────────

func (m *Model) updateEditCommand(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.sce == nil || m.sce.editIdx < 0 || m.sce.editIdx >= len(m.config.Commands) {
		return m, nil
	}
	sce := m.sce

	if m.screen == ScreenEditCommandTemplate {
		switch msg.String() {
		case "up":
			if sce.templateRefIdx > 0 {
				sce.templateRefIdx--
			}
		case "down":
			if sce.templateRefIdx < len(m.templateCandidates())-1 {
				sce.templateRefIdx++
			}
		case "enter":
			return m.startCommandSlots()
		case "esc":
			m.gotoEditCommandMode()
			m.listCursor = 2 // back onto "Change template"
		}
		return m, nil
	}

	// ScreenEditCommandName (name input after slots)
	switch msg.String() {
	case "enter":
		name := m.nameInput.Value()
		if name == "" {
			m.errMsg = "name cannot be empty"
			break
		}
		if m.commandNameTaken(name, sce.editIdx) {
			m.errMsg = "name already in use: " + name
			break
		}
		candidates := m.templateCandidates()
		if sce.templateRefIdx < 0 || sce.templateRefIdx >= len(candidates) {
			m.errMsg = "invalid template"
			break
		}
		tpl := candidates[sce.templateRefIdx]
		entry := mdl.Command{
			Name:     name,
			Template: tpl.Name,
			Values:   sce.currentValues,
		}
		entry, _ = slot.MaterializeCommand(entry, m.config)
		m.config.Commands[sce.editIdx] = entry
		if err := m.saveConfig(); err != nil {
			m.errMsg = "failed to save: " + err.Error()
		} else {
			m.successMsg = "updated command \"" + name + "\""
		}
		m.screen = ScreenManageCommands
		m.listCursor = 1
		m.sce = nil
	case "esc":
		return m.backFromCommandNameInput()
	default:
		ti, cmd := m.nameInput.Update(msg)
		m.nameInput = ti
		return m, cmd
	}
	return m, nil
}

// ── Delete flow ───────────────────────────────────────────────────────────────

func (m *Model) updateDeleteCommand(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	return m.updateDeleteList(msg, len(m.editRefs), nil,
		func() {
			m.screen = ScreenManageCommands
			m.listCursor = 2
		},
		func(indices []int) {
			// indices are descending, so both slices shrink safely.
			var tsvNames []string
			for _, i := range indices {
				ref := m.editRefs[i]
				if ref.tsv {
					tsvNames = append(tsvNames, m.config.Base[ref.idx].Name)
					m.config.Base = append(m.config.Base[:ref.idx], m.config.Base[ref.idx+1:]...)
				} else {
					m.config.Commands = append(m.config.Commands[:ref.idx], m.config.Commands[ref.idx+1:]...)
				}
			}
			err := m.saveConfig()
			if err == nil && len(tsvNames) > 0 {
				_, err = config.DeleteCommandsTSV(m.projectDir, tsvNames)
			}
			if err != nil {
				m.errMsg = "failed to save: " + err.Error()
			} else {
				m.successMsg = fmt.Sprintf("deleted %d command(s)", len(indices))
			}
		})
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// commandNameTaken reports whether name collides with another user
// command or a hand-written command. excludeIdx skips one user command
// (the one being edited); pass -1 when creating.
func (m *Model) commandNameTaken(name string, excludeIdx int) bool {
	for i, cmd := range m.config.Commands {
		if i != excludeIdx && cmd.Name == name {
			return true
		}
	}
	for _, cmd := range m.config.Base {
		if cmd.Name == name {
			return true
		}
	}
	return false
}

// templateCandidates returns commands usable as templates: any command
// with {slot} placeholders that is not itself template-derived, regardless
// of which file it lives in.
func (m Model) templateCandidates() []mdl.Command {
	var out []mdl.Command
	for _, cmd := range m.config.AllCommands() {
		if cmd.Template == "" && slot.HasPlaceholders(cmd) {
			out = append(out, cmd)
		}
	}
	return out
}

// getTemplateRefIdx returns the index of templateRef in templateCandidates, or -1.
func (m *Model) getTemplateRefIdx(templateRef string) int {
	for i, cmd := range m.templateCandidates() {
		if cmd.Name == templateRef {
			return i
		}
	}
	return -1
}

// startCommandSlots begins slot resolution for the selected template,
// or jumps straight to name input when the template has no slots.
func (m *Model) startCommandSlots() (tea.Model, tea.Cmd) {
	sce := m.sce
	candidates := m.templateCandidates()
	if sce.templateRefIdx < 0 || sce.templateRefIdx >= len(candidates) {
		m.errMsg = "invalid template"
		return m, nil
	}
	templateCmd := candidates[sce.templateRefIdx]
	sce.currentValues = make(map[string]string)
	slots := slot.GetSlots(templateCmd)
	if len(slots) == 0 {
		return m.gotoCommandNameInput()
	}
	sce.currentSlots = slots
	sce.currentSlotIdx = 0
	return m.openSlotPickForCommandEdit(&templateCmd)
}

func (m *Model) gotoCommandNameInput() (tea.Model, tea.Cmd) {
	sce := m.sce
	if sce.mode == 1 {
		m.screen = ScreenEditCommandName
		m.nameInput.SetValue(sce.name)
	} else {
		m.screen = ScreenCreateCommandName
		m.nameInput.SetValue("")
	}
	m.sp = nil
	return m, m.nameInput.Focus()
}

func (m *Model) openSlotPickForCommandEdit(cmd *mdl.Command) (tea.Model, tea.Cmd) {
	sce := m.sce
	s := sce.currentSlots[sce.currentSlotIdx]
	sp := &slotPickState{
		slotName:      s.Name,
		listName:      s.ListName,
		entries:       m.lists[s.ListName],
		cursor:        0,
		canSkip:       true,
		contextNames:  []string{sce.name},
		contextNotes:  []string{},
		contextIdx:    0,
		currentCmd:    cmd,
		resolvedSoFar: sce.currentValues,
	}
	sp.applyFilter()
	// When editing, start the cursor on the command's current value.
	if sce.mode == 1 && sce.editIdx >= 0 && sce.editIdx < len(m.config.Commands) {
		if cur, ok := m.config.Commands[sce.editIdx].Values[s.Name]; ok {
			for i, e := range sp.filtered {
				if e.Value == cur {
					sp.cursor = i
					break
				}
			}
		}
	}
	m.sp = sp
	m.screen = ScreenSlotPick
	return *m, nil
}

func (m *Model) finishCommandSlotResolution() (tea.Model, tea.Cmd) {
	sce := m.sce
	if sce.currentSlotIdx >= len(sce.currentSlots) {
		return m.gotoCommandNameInput()
	}
	templateCmd := m.templateCandidates()[sce.templateRefIdx]
	return m.openSlotPickForCommandEdit(&templateCmd)
}

func (m *Model) acceptCommandEditSlotValue(value string) (tea.Model, tea.Cmd) {
	sce := m.sce
	sce.currentValues[sce.currentSlots[sce.currentSlotIdx].Name] = value
	sce.currentSlotIdx++

	m.sp = nil
	return m.finishCommandSlotResolution()
}

func (m *Model) skipCommandEditSlot() (tea.Model, tea.Cmd) {
	sce := m.sce
	sce.currentSlotIdx++

	m.sp = nil
	return m.finishCommandSlotResolution()
}

// goBackCommandEditSlot steps back one slot, or to the template picker
// from the first slot.
func (m *Model) goBackCommandEditSlot() (tea.Model, tea.Cmd) {
	sce := m.sce
	m.sp = nil
	if sce.currentSlotIdx > 0 {
		sce.currentSlotIdx--
		delete(sce.currentValues, sce.currentSlots[sce.currentSlotIdx].Name)
		templateCmd := m.templateCandidates()[sce.templateRefIdx]
		return m.openSlotPickForCommandEdit(&templateCmd)
	}
	return m.backToCommandTemplatePick()
}

// backFromCommandNameInput steps back from the name input to the last slot,
// or to the template picker when the template has no slots.
func (m *Model) backFromCommandNameInput() (tea.Model, tea.Cmd) {
	sce := m.sce
	if len(sce.currentSlots) > 0 {
		sce.currentSlotIdx = len(sce.currentSlots) - 1
		delete(sce.currentValues, sce.currentSlots[sce.currentSlotIdx].Name)
		templateCmd := m.templateCandidates()[sce.templateRefIdx]
		return m.openSlotPickForCommandEdit(&templateCmd)
	}
	return m.backToCommandTemplatePick()
}

func (m *Model) backToCommandTemplatePick() (tea.Model, tea.Cmd) {
	switch {
	case m.sce.mode == 1 && !m.sce.pickedTemplate:
		// Change values skipped the template pick — return to the menu.
		m.gotoEditCommandMode()
		m.listCursor = 1 // back onto "Change values"
	case m.sce.mode == 1:
		m.screen = ScreenEditCommandTemplate
	default:
		m.screen = ScreenCreateCommandTemplate
	}
	return m, nil
}
