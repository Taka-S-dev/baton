package tui

import (
	"fmt"
	"github.com/charmbracelet/bubbles/textinput"
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Taka-S-dev/baton/internal/config"
	mdl "github.com/Taka-S-dev/baton/internal/model"
	"github.com/Taka-S-dev/baton/internal/slot"
	"github.com/Taka-S-dev/baton/internal/store"
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
			m.editRefs = refs
			m.setPickBase(names, m.commandPickTexts(refs))
			m.listCursor = 0
		case 2: // Delete
			names, refs := m.editableCommands()
			if len(refs) == 0 {
				m.errMsg = "no editable commands (TSV rows and TUI-created commands)"
				break
			}
			m.screen = ScreenDeleteCommand
			m.editRefs = refs
			m.setPickBase(names, m.commandPickTexts(refs))
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
		m.gotoManageCommands()
		m.listCursor = 0
	}
	return m, nil
}

// ── Command form: write a concrete command directly ──────────────────────────

// openCommandForm opens the direct-input form. Pass editIdx -1 to create,
// or the index of a concrete command in config.Commands to edit it.
func (m *Model) openCommandForm(editIdx int) (tea.Model, tea.Cmd) {
	cf := &commandFormState{mode: 0, editIdx: -1, returnTo: ScreenManageCommands}
	if editIdx < 0 {
		m.applyShellPreference(cf)
	}
	if editIdx >= 0 && editIdx < len(m.config.Commands) {
		cmd := m.config.Commands[editIdx]
		cf.mode = 1
		cf.editIdx = editIdx
		cf.fields = [5]string{cmd.Name, cmd.Cmd, cmd.Dir, cmd.Group, cmd.Shell}
	}
	cf.original = cf.fields
	m.cf = cf
	m.screen = ScreenCommandForm
	return m, m.cfGotoField(formFieldAt(0))
}

// openCommandFormTSV opens the form on a TSV row (config.Base index).
// Saving rewrites exactly that row via UpdateCommandTSV.
func (m *Model) openCommandFormTSV(baseIdx int) (tea.Model, tea.Cmd) {
	cmd := m.config.Base[baseIdx]
	cf := &commandFormState{mode: 1, editIdx: baseIdx, tsvEdit: true, origName: cmd.Name, returnTo: ScreenManageCommands}
	cf.fields = [5]string{cmd.Name, cmd.Cmd, cmd.Dir, cmd.Group, cmd.Shell}
	cf.original = cf.fields
	m.cf = cf
	m.screen = ScreenCommandForm
	return m, m.cfGotoField(formFieldAt(0))
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
	back := m.cf.returnTo
	m.cf = nil
	m.nameInput.Blur()
	m.nameInput.Prompt = "Name > "
	m.screen = back
}

// cfGotoField focuses the given form field, loading its stored text.
func (m *Model) cfGotoField(idx int) tea.Cmd {
	m.cf.fieldIdx = idx
	m.cf.pickOpen = false
	m.cf.shellOpen = idx == 4
	m.cf.shellCursor = 0
	if strings.EqualFold(m.cf.fields[4], "ps") {
		m.cf.shellCursor = 1
	}
	m.resizeNameInput()
	// An empty name field shows the name the cmd suggests, dimmed in
	// place, and → takes it. Elsewhere the placeholder is nothing.
	m.nameInput.Placeholder = ""
	if idx == 0 && m.cf.mode == 0 && m.cf.fields[0] == "" {
		m.nameInput.Placeholder = m.suggestFormName()
	}
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
		return m, m.formError(0, "a name is needed")
	}
	if cmdStr == "" {
		return m, m.formError(1, "a command line is needed")
	}
	shell := strings.ToLower(cf.fields[4])
	if shell != "" && shell != "ps" {
		return m, m.formError(4, `empty, or "ps" for PowerShell`)
	}
	if m.formNameTaken(name) {
		return m, m.formError(0, "already in use: "+name)
	}
	if cf.tsvEdit {
		newCmd := m.config.Base[cf.editIdx] // keeps Slots and Source
		newCmd.Name, newCmd.Cmd, newCmd.Dir, newCmd.Group, newCmd.Shell = name, cmdStr, cf.fields[2], cf.fields[3], shell
		if file, err := config.UpdateCommandTSV(m.projectDir, cf.origName, newCmd); err != nil {
			m.errMsg = "failed to save: " + err.Error()
			return m, nil
		} else {
			m.rememberShell(shell)
			m.config.Base[cf.editIdx] = newCmd
			m.successMsg = "updated command \"" + name + "\" in " + file
			if cf.origName != name {
				if err := m.persistRetargetedRefs(cf.origName, name); err != nil {
					m.errMsg = "failed to update references: " + err.Error()
				}
			}
		}
		m.closeCommandForm()
		m.listCursor = 1
		return m, nil
	}
	if cf.mode == 1 {
		cmd := m.config.Commands[cf.editIdx]
		oldName := cmd.Name
		cmd.Name, cmd.Cmd, cmd.Dir, cmd.Group, cmd.Shell = name, cmdStr, cf.fields[2], cf.fields[3], shell
		m.config.Commands[cf.editIdx] = cmd
		var wfSteps, tplRefs int
		if oldName != name {
			wfSteps, tplRefs, _ = m.retargetCommandRefs(oldName, name)
		}
		err := m.saveConfig()
		if err == nil && wfSteps > 0 {
			err = store.SaveWorkflows(m.projectDir, m.workflows)
		}
		if err != nil {
			m.errMsg = "failed to save: " + err.Error()
			return m, nil
		} else {
			m.rememberShell(shell)
			m.successMsg = "updated command \"" + name + "\"" + refSummary(wfSteps, tplRefs)
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
		// The pickers list the hand-written layer by source, so without
		// one this row would be written and run yet stay uneditable until
		// the next load set it.
		Source: "tsv",
	}
	if file, err := config.AppendCommandTSV(m.projectDir, newCmd); err != nil {
		m.errMsg = "failed to save: " + err.Error()
		return m, nil
	} else {
		m.rememberShell(shell)
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
	if m.leaveAsk != leaveNone {
		return m, m.updateLeave(msg)
	}
	cf := m.cf
	if cf.fieldIdx == 4 {
		if handled, cmd := m.updateShellChoice(msg); handled {
			return m, cmd
		}
	}
	pos := formPos(cf.fieldIdx)
	key := msg.String()
	commit := func() { cf.fields[cf.fieldIdx] = strings.TrimSpace(m.nameInput.Value()) }

	// The candidate popup takes the keys that pick from it. Typing keeps
	// narrowing it, since what follows the brace is the filter; any key
	// that is none of these closes it and is handled as if it were not
	// there, so the popup never traps a keystroke meant for the form.
	if cf.pickOpen {
		cands := m.pickCandidates(cf.pickTyped(m.nameInput))
		switch key {
		case "up":
			cf.pickCursor = max(0, cf.pickCursor-1)
			return m, nil
		case "down":
			cf.pickCursor = min(max(0, len(cands)-1), cf.pickCursor+1)
			return m, nil
		case "enter":
			if cf.pickCursor < len(cands) {
				m.insertPick(cands[cf.pickCursor].text)
			}
			cf.pickOpen = false
			return m, nil
		case "esc":
			cf.pickOpen = false
			return m, nil
		case "backspace":
			ti, cmd := m.nameInput.Update(msg)
			m.nameInput = ti
			// Deleting the brace itself takes the popup with it.
			if m.nameInput.Position() <= cf.pickFrom {
				cf.pickOpen = false
			}
			cf.pickCursor = 0
			return m, cmd
		}
		if msg.Type == tea.KeyRunes {
			ti, cmd := m.nameInput.Update(msg)
			m.nameInput = ti
			// A closing brace typed by hand means the placeholder was
			// finished without the popup.
			if strings.ContainsRune(key, '}') {
				cf.pickOpen = false
			}
			cf.pickCursor = 0
			return m, cmd
		}
		cf.pickOpen = false
	}

	switch key {
	case "tab", "shift+tab":
		// Tab is the next field and nothing else, on every field: a key
		// that meant three things depending on where the caret was is
		// what made the form hard to learn.
		commit()
		step := 1
		if key == "shift+tab" {
			step = -1
		}
		next := pos + step
		if next < 0 || next >= len(formOrder) {
			return m, nil
		}
		return m, m.cfGotoField(formFieldAt(next))
	case "right":
		// On an empty name field the suggestion is shown in place, and
		// → takes it — the shell's idiom for accepting what is shown
		// dimmed ahead of the caret.
		if cf.fieldIdx == 0 && m.nameInput.Value() == "" && m.nameInput.Placeholder != "" {
			m.nameInput.SetValue(m.nameInput.Placeholder)
			m.nameInput.CursorEnd()
			return m, nil
		}
	case "enter":
		commit()
		// Validate the name as soon as the field is confirmed, so a
		// duplicate surfaces immediately instead of after the last field.
		if cf.fieldIdx == 0 {
			name := cf.fields[0]
			if name == "" {
				return m, m.formError(0, "a name is needed")
			}
			if m.formNameTaken(name) {
				return m, m.formError(0, "already in use: "+name)
			}
		}
		if pos < len(formOrder)-1 {
			return m, m.cfGotoField(formFieldAt(pos + 1))
		}
		return m.saveCommandForm()
	case "ctrl+s":
		// Save from any field — no need to walk the rest of the form.
		commit()
		return m.saveCommandForm()
	case "up":
		if pos > 0 {
			commit()
			return m, m.cfGotoField(formFieldAt(pos - 1))
		}
		return m, nil
	case "down":
		if pos < len(formOrder)-1 {
			commit()
			return m, m.cfGotoField(formFieldAt(pos + 1))
		}
		return m, nil
	case "esc":
		// Escape leaves the editor from every field; Shift+Tab moves back.
		if m.askToLeave(leaveForm) {
			return m, nil
		}
		m.closeCommandForm()
		m.listCursor = 0
		return m, nil
	}

	ti, cmd := m.nameInput.Update(msg)
	m.nameInput = ti
	// Typing in the field the error is about is taken as dealing with it.
	if cf.fieldErr != "" && cf.errField == cf.fieldIdx {
		cf.fieldErr = ""
	}
	// A brace typed into the cmd or workdir opens the candidates for
	// what can go there, the way an editor's completion opens on the
	// character that starts a name.
	if key == "{" && (cf.fieldIdx == 1 || cf.fieldIdx == 2) && len(m.pickCandidates("")) > 0 {
		cf.pickOpen = true
		cf.pickCursor = 0
		cf.pickFrom = m.nameInput.Position() - 1
	}
	return m, cmd
}

// pickCandidate is one thing that can be put where a brace was typed.
type pickCandidate struct {
	text string // what is inserted: {env}, {env...} or {$root}
	what string // what it means, beside it
}

// pickCandidates lists what a brace can become, narrowed by what has
// been typed after it: every list as a {slot} and as a variadic
// {slot...}, then every global as a {$var}, each with what choosing it
// would mean. The description is the point — a newcomer who has not
// met a slot or a variable is choosing between two things the braces
// alone do not tell apart.
func (m Model) pickCandidates(typed string) []pickCandidate {
	var out []pickCandidate
	add := func(text, what string) {
		if typed == "" || strings.Contains(strings.ToLower(text), strings.ToLower("{"+typed)) {
			out = append(out, pickCandidate{text, what})
		}
	}
	for _, name := range m.sortedListNames() {
		n := len(m.lists[name])
		add("{"+name+"}", fmt.Sprintf("pick at run time · %s", countLabel(n, "choice")))
		add("{"+name+"...}", "pick several at run time")
	}
	for _, k := range sortedKeys(m.vars) {
		if !strings.Contains(k, ".") {
			add("{$"+k+"}", "fixed · "+m.vars[k])
		}
	}
	return out
}

// pickTyped is what has been typed after the brace that opened the
// popup, up to the caret — the filter the candidates are narrowed by.
func (cf *commandFormState) pickTyped(ti textinput.Model) string {
	v := []rune(ti.Value())
	from, to := cf.pickFrom+1, ti.Position()
	if from < 0 || from > len(v) || to < from || to > len(v) {
		return ""
	}
	return string(v[from:to])
}

// insertPick replaces the brace and what was typed after it with the
// chosen placeholder, and leaves the caret after it.
func (m *Model) insertPick(text string) {
	v := []rune(m.nameInput.Value())
	from, to := m.cf.pickFrom, m.nameInput.Position()
	if from < 0 || from > len(v) || to < from || to > len(v) {
		return
	}
	m.nameInput.SetValue(string(v[:from]) + text + string(v[to:]))
	m.nameInput.SetCursor(from + len([]rune(text)))
}

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
// commandPickTexts builds the search haystacks for a set of editRefs,
// matching the same fields as the Run commands search.
func (m *Model) commandPickTexts(refs []editRef) []string {
	texts := make([]string, len(refs))
	for i, r := range refs {
		texts[i] = msItem{cmd: m.editRefCommand(r)}.searchText()
	}
	return texts
}

func (m *Model) gotoEditCommandPick(at editRef) {
	names, refs := m.editableCommands()
	m.screen = ScreenEditCommandPick
	m.editRefs = refs
	m.setPickBase(names, m.commandPickTexts(refs))
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
		if m.listCursor >= len(m.listItems) {
			break
		}
		ref := m.editRefs[m.pickOrig(m.listCursor)]
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
		if m.pickSearch != "" {
			m.clearPickFilter()
			break
		}
		m.gotoManageCommands()
		m.listCursor = 1
	default:
		m.handlePickTyping(msg, nil)
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
			m.nameInput.Prompt = "Name > "
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
		if m.returnFromFlow() {
			return m, nil
		}
		m.gotoEditCommandPick(editRef{idx: editIdx})
	}
	return m, nil
}

// retargetCommandRefs rewrites every reference to a renamed command from
// oldName to newName: workflow steps, template references of derived
// commands, and scoped vars.tsv keys. It only mutates memory — callers
// persist the files whose returned count is non-zero.
func (m *Model) retargetCommandRefs(oldName, newName string) (wfSteps, tplRefs, varKeys int) {
	for wi := range m.workflows {
		for si, step := range m.workflows[wi].Commands {
			if step == oldName {
				m.workflows[wi].Commands[si] = newName
				wfSteps++
			}
		}
	}
	for i := range m.config.Commands {
		if m.config.Commands[i].Template == oldName {
			m.config.Commands[i].Template = newName
			tplRefs++
		}
	}
	// Scoped keys are "command.slot" with the separator on the last dot;
	// a key re-keys only when the suffix after oldName is a single segment.
	prefix := oldName + "."
	var moved []string
	for k := range m.vars {
		if strings.HasPrefix(k, prefix) && !strings.Contains(k[len(prefix):], ".") {
			moved = append(moved, k)
		}
	}
	for _, k := range moved {
		m.vars[newName+"."+k[len(prefix):]] = m.vars[k]
		delete(m.vars, k)
		varKeys++
	}
	return wfSteps, tplRefs, varKeys
}

// persistRetargetedRefs retargets references after a rename outside the
// local layer (a TSV row) and writes only the files that changed —
// commands.local.json is untouched unless a template reference moved.
// The reference summary is appended to the current success message.
func (m *Model) persistRetargetedRefs(oldName, newName string) error {
	wfSteps, tplRefs, varKeys := m.retargetCommandRefs(oldName, newName)
	if wfSteps > 0 {
		if err := store.SaveWorkflows(m.projectDir, m.workflows); err != nil {
			return err
		}
	}
	if tplRefs > 0 {
		// saveConfig persists the moved template references and, via its
		// vars mirror, the re-keyed scoped values in one pass.
		if err := m.saveConfig(); err != nil {
			return err
		}
	} else if varKeys > 0 {
		if err := slot.SaveVars(m.projectDir, m.vars); err != nil {
			return err
		}
	}
	m.successMsg += refSummary(wfSteps, tplRefs)
	return nil
}

// refSummary phrases retargeted reference counts for a success message,
// or "" when nothing referenced the old name.
func refSummary(wfSteps, tplRefs int) string {
	var parts []string
	if wfSteps > 0 {
		parts = append(parts, fmt.Sprintf("%d workflow step(s)", wfSteps))
	}
	if tplRefs > 0 {
		parts = append(parts, fmt.Sprintf("%d derived command(s)", tplRefs))
	}
	if len(parts) == 0 {
		return ""
	}
	return " (updated " + strings.Join(parts, ", ") + ")"
}

// renameCommand renames the command being edited without touching its
// template or values; vars.tsv rows follow the new name on save.
func (m Model) renameCommand(name string) (tea.Model, tea.Cmd) {
	sce := m.sce
	if sce == nil || sce.editIdx < 0 || sce.editIdx >= len(m.config.Commands) {
		m.sce = nil
		m.gotoManageCommands()
		m.listCursor = 1
		return m, nil
	}
	if m.commandNameTaken(name, sce.editIdx) {
		m.nameInputErr = "name already in use: " + name
		return m, nil
	}
	oldName := m.config.Commands[sce.editIdx].Name
	m.config.Commands[sce.editIdx].Name = name
	wfSteps, tplRefs, _ := m.retargetCommandRefs(oldName, name)
	err := m.saveConfig()
	if err == nil && wfSteps > 0 {
		err = store.SaveWorkflows(m.projectDir, m.workflows)
	}
	if err != nil {
		m.errMsg = "failed to save: " + err.Error()
	} else {
		m.successMsg = "renamed command to \"" + name + "\"" + refSummary(wfSteps, tplRefs)
	}
	m.sce = nil
	m.gotoManageCommands()
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
	case "tab":
		if s := m.suggestCommandName(); s != "" {
			m.nameInput.SetValue(s)
			m.nameInput.CursorEnd()
		}
		return m, nil
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
		m.selectExploreCreated(tabCommands, name)
		m.gotoManageCommands()
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
	case "tab":
		if s := m.suggestCommandName(); s != "" {
			m.nameInput.SetValue(s)
			m.nameInput.CursorEnd()
		}
		return m, nil
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
		oldName := m.config.Commands[sce.editIdx].Name
		m.config.Commands[sce.editIdx] = entry
		var wfSteps, tplRefs int
		if oldName != name {
			wfSteps, tplRefs, _ = m.retargetCommandRefs(oldName, name)
		}
		err := m.saveConfig()
		if err == nil && wfSteps > 0 {
			err = store.SaveWorkflows(m.projectDir, m.workflows)
		}
		if err != nil {
			m.errMsg = "failed to save: " + err.Error()
		} else {
			m.successMsg = "updated command \"" + name + "\"" + refSummary(wfSteps, tplRefs)
		}
		m.selectExploreCreated(tabCommands, name)
		m.gotoManageCommands()
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
	return m.updateDeleteList(msg, len(m.listItems), nil,
		func() {
			m.gotoManageCommands()
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
		// Pre-fill a suggested name so saved commands follow one naming
		// rule by default — Enter accepts it, editing replaces it.
		m.screen = ScreenCreateCommandName
		m.nameInput.SetValue(m.suggestCommandName())
	}
	m.sp = nil
	return m, m.nameInput.Focus()
}

// suggestName proposes "<template>-<value>-..." from slot values (in
// slot order, slots without a value omitted), so saved commands get
// distinguishable names without inventing a scheme each time.
//
// On a collision with an existing name the values are rebuilt with their
// parent path segment first — two workdirs like app1/src and app2/src
// usually differ there, not in the basename — and only when that still
// collides does a numeric suffix step in. excludeIdx is the command the
// name is being chosen for, which does not collide with itself.
func (m *Model) suggestName(tplName string, slots []slot.Def, values map[string]string, excludeIdx int) string {
	build := func(segments int) string {
		parts := []string{tplName}
		for _, s := range slots {
			if v, ok := values[s.Name]; ok {
				if p := sanitizeNamePart(v, segments); p != "" {
					parts = append(parts, p)
				}
			}
		}
		base := strings.Join(parts, "-")
		if r := []rune(base); len(r) > 48 {
			base = strings.TrimRight(string(r[:48]), "-.")
		}
		return base
	}
	name := build(1)
	if !m.commandNameTaken(name, excludeIdx) {
		return name
	}
	if deep := build(2); deep != name {
		name = deep
		if !m.commandNameTaken(name, excludeIdx) {
			return name
		}
	}
	base := name
	for i := 2; m.commandNameTaken(name, excludeIdx); i++ {
		name = fmt.Sprintf("%s-%d", base, i)
	}
	return name
}

// suggestCommandName names the command being created or edited from the
// slot values picked during the flow.
func (m *Model) suggestCommandName() string {
	sce := m.sce
	cands := m.templateCandidates()
	if sce == nil || sce.templateRefIdx < 0 || sce.templateRefIdx >= len(cands) {
		return ""
	}
	exclude := -1
	if sce.mode == 1 {
		exclude = sce.editIdx
	}
	return m.suggestName(cands[sce.templateRefIdx].Name, sce.currentSlots, sce.currentValues, exclude)
}

// suggestRenameForCommand names the command being renamed. Rename skips
// the slot pickers, so the values come from the command itself rather
// than from the flow — which is the point: it re-derives the canonical
// name for an entry that was named before the rule existed.
func (m *Model) suggestRenameForCommand() string {
	sce := m.sce
	cands := m.templateCandidates()
	if sce == nil || sce.templateRefIdx < 0 || sce.templateRefIdx >= len(cands) {
		return ""
	}
	if sce.editIdx < 0 || sce.editIdx >= len(m.config.Commands) {
		return ""
	}
	tpl := cands[sce.templateRefIdx]
	return m.suggestName(tpl.Name, slot.GetSlots(tpl), m.config.Commands[sce.editIdx].Values, sce.editIdx)
}

// sanitizeNamePart reduces a slot value to a short name fragment: its
// last `segments` path segments, lowercased, with everything but
// letters, digits and dots collapsed into single hyphens.
func sanitizeNamePart(v string, segments int) string {
	fields := strings.FieldsFunc(v, func(r rune) bool { return r == '/' || r == '\\' })
	if len(fields) > segments {
		fields = fields[len(fields)-segments:]
	}
	v = strings.ToLower(strings.Join(fields, "-"))
	var sb strings.Builder
	pendingHyphen := false
	for _, r := range v {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '.' {
			if pendingHyphen && sb.Len() > 0 {
				sb.WriteByte('-')
			}
			sb.WriteRune(r)
			pendingHyphen = false
		} else {
			pendingHyphen = true
		}
	}
	out := strings.Trim(sb.String(), "-.")
	if r := []rune(out); len(r) > 20 {
		out = strings.TrimRight(string(r[:20]), "-.")
	}
	return out
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
		variadic:      s.Variadic,
		contextNames:  []string{sce.name},
		contextNotes:  []string{},
		contextIdx:    0,
		currentCmd:    cmd,
		resolvedSoFar: sce.currentValues,
	}
	sp.applyFilter()
	// When editing, start from the command's current value: pre-toggle a
	// variadic slot's stored values, or park the cursor on a single one.
	if sce.mode == 1 && sce.editIdx >= 0 && sce.editIdx < len(m.config.Commands) {
		if cur, ok := m.config.Commands[sce.editIdx].Values[s.Name]; ok {
			if s.Variadic {
				sp.picked = strings.Fields(cur)
			} else {
				for i, e := range sp.filtered {
					if e.Value == cur {
						sp.cursor = i
						break
					}
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
	if m.exploreLaunched && m.sce.mode == 0 {
		m.sce, m.sp = nil, nil
		m.returnFromFlow()
		return m, nil
	}
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

// formOrder is the order the form asks for its fields, which is not the
// order they are stored in: the TSV keeps the name first because that is
// the column order, but the form asks for the cmd first. The name is the
// only part of a command that takes thought, and making it the first
// gate — rejected on the spot when it collides — is what made
// registering one feel expensive.
var formOrder = [5]int{1, 0, 2, 3, 4}

// formPos is where a stored field sits in the order it is asked.
func formPos(idx int) int {
	for p, f := range formOrder {
		if f == idx {
			return p
		}
	}
	return 0
}

// formFieldAt is the stored field asked at position p.
func formFieldAt(p int) int { return formOrder[min(max(p, 0), len(formOrder)-1)] }

// suggestNameFromCmd derives a name from what the command actually runs,
// so the field can be filled with Tab instead of invented. It keeps the
// first couple of words that are neither a flag nor a placeholder, and
// reduces a path to its last segment — `go test ./internal/{pkg}/`
// becomes `go-test`.
func suggestNameFromCmd(cmd string) string {
	var parts []string
	for _, tok := range strings.Fields(cmd) {
		if strings.HasPrefix(tok, "-") || strings.Contains(tok, "{") {
			continue
		}
		tok = strings.Trim(tok, `"'`)
		if i := strings.LastIndexAny(tok, `/\`); i >= 0 {
			tok = tok[i+1:]
		}
		var clean strings.Builder
		for _, r := range strings.ToLower(tok) {
			switch {
			case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
				clean.WriteRune(r)
			case r == '-' || r == '_' || r == '.':
				clean.WriteRune('-')
			}
		}
		if p := strings.Trim(clean.String(), "-"); p != "" {
			parts = append(parts, p)
		}
		if len(parts) == 2 {
			break
		}
	}
	return strings.Join(parts, "-")
}

// suggestFormName is suggestNameFromCmd with a numeric suffix when the
// obvious name is already taken, the same way saved commands avoid
// colliding.
func (m *Model) suggestFormName() string {
	base := suggestNameFromCmd(m.cf.fields[1])
	if base == "" || !m.formNameTaken(base) {
		return base
	}
	for i := 2; i < 100; i++ {
		candidate := fmt.Sprintf("%s-%d", base, i)
		if !m.formNameTaken(candidate) {
			return candidate
		}
	}
	return base
}

// resizeNameInput sets how wide the text field may draw, which depends
// on where the form is: beside the tree it has the pane, on a screen of
// its own it has the terminal.
//
// It has to be set before the value is. The textinput works out which
// part of a value too long to fit it should show when the value or the
// cursor moves, not when it draws, so a width handed to it at draw time
// arrives after the decision it should have informed — which is how a
// workdir came to be shown with its drive letter cut off the left.
func (m *Model) resizeNameInput() {
	w := m.viewWidth()
	if m.screen == ScreenExplore {
		if l := m.exploreLayout(w); !l.single {
			w = l.rightW - 2 // inside the editor box's edges
		}
	}
	m.nameInput.Width = max(12, w-14)
}

// gotoManageCommands opens the Manage commands submenu — or, for a flow
// launched from the overview, goes back to the overview instead.
func (m *Model) gotoManageCommands() {
	if m.returnFromFlow() {
		return
	}
	m.screen = ScreenManageCommands
}

// formError puts a validation failure under the field it is about and
// moves the caret there. It stays until that field is typed in, where
// the banner would have vanished on the next key.
func (m *Model) formError(field int, msg string) tea.Cmd {
	m.cf.fieldErr, m.cf.errField = msg, field
	return m.cfGotoField(field)
}
