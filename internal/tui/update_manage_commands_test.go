package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	mdl "github.com/Taka-S-dev/baton/internal/model"
	"github.com/Taka-S-dev/baton/internal/slot"
)

// TestCommandForm_ShellField walks the create form through all fields and
// checks the shell field: an unknown value is rejected with the cursor left
// on the field, and "ps" (any case) is normalized and saved.
func TestCommandForm_ShellField(t *testing.T) {
	m := &Model{}
	m.nameInput = textinput.New()
	m.projectDir = t.TempDir()
	m.openCommandForm(-1)

	enter := tea.KeyMsg{Type: tea.KeyEnter}
	typeAndEnter := func(v string) {
		m.nameInput.SetValue(v)
		m.updateCommandForm(enter)
	}

	typeAndEnter("list-src")          // name
	typeAndEnter("Get-ChildItem src") // cmd
	typeAndEnter("")                  // workdir
	typeAndEnter("")                  // group

	typeAndEnter("bash") // shell: invalid
	if m.errMsg == "" || m.cf == nil || m.cf.fieldIdx != 4 {
		t.Fatalf("invalid shell must be rejected on the field: errMsg=%q cf=%v", m.errMsg, m.cf)
	}

	typeAndEnter("PS") // shell: valid, normalized to lowercase
	if m.cf != nil {
		t.Fatalf("form must close after a valid save; errMsg=%q", m.errMsg)
	}
	// Form-created commands are hand-editable definitions: they land in
	// the TSV (Base layer), not in commands.local.json.
	if len(m.config.Base) != 1 || m.config.Base[0].Shell != "ps" {
		t.Fatalf("saved commands = %+v, want one Base command with Shell=ps", m.config.Base)
	}
	raw, err := os.ReadFile(filepath.Join(m.projectDir, "commands.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "list-src\t\t\tGet-ChildItem src\tps\t") {
		t.Fatalf("commands.tsv = %q, want the appended row", raw)
	}
}

// TestCommandForm_EditLoadsShell checks edit mode pre-fills the shell field
// from the existing command.
func TestCommandForm_EditLoadsShell(t *testing.T) {
	m := &Model{}
	m.nameInput = textinput.New()
	m.config.Commands = []mdl.Command{{Name: "x", Cmd: "Get-Date", Shell: "ps"}}
	m.openCommandForm(0)

	if m.cf == nil || m.cf.fields[4] != "ps" {
		t.Fatalf("edit form must pre-fill the shell field, got %+v", m.cf)
	}
}

// TestCommandForm_NameValidatedEarly checks a duplicate or empty name is
// rejected when the name field is confirmed, not after the last field —
// and that edit mode still accepts the command's own current name.
func TestCommandForm_NameValidatedEarly(t *testing.T) {
	m := &Model{}
	m.nameInput = textinput.New()
	m.config.Commands = []mdl.Command{{Name: "build", Cmd: "make"}}
	m.openCommandForm(-1)

	enter := tea.KeyMsg{Type: tea.KeyEnter}

	m.nameInput.SetValue("build")
	m.updateCommandForm(enter)
	if m.errMsg == "" || m.cf.fieldIdx != 0 {
		t.Fatalf("duplicate name must be rejected on the name field: errMsg=%q fieldIdx=%d", m.errMsg, m.cf.fieldIdx)
	}

	m.nameInput.SetValue("")
	m.updateCommandForm(enter)
	if m.errMsg == "" || m.cf.fieldIdx != 0 {
		t.Fatalf("empty name must be rejected on the name field: errMsg=%q fieldIdx=%d", m.errMsg, m.cf.fieldIdx)
	}

	m.nameInput.SetValue("build2")
	m.updateCommandForm(enter)
	if m.cf.fieldIdx != 1 {
		t.Fatalf("unique name must advance to the cmd field, fieldIdx=%d", m.cf.fieldIdx)
	}

	// Edit mode: keeping the command's own name is not a duplicate.
	m2 := &Model{}
	m2.nameInput = textinput.New()
	m2.config.Commands = []mdl.Command{{Name: "build", Cmd: "make"}}
	m2.openCommandForm(0)
	m2.updateCommandForm(enter)
	if m2.errMsg != "" || m2.cf.fieldIdx != 1 {
		t.Fatalf("edit mode must accept its own name: errMsg=%q fieldIdx=%d", m2.errMsg, m2.cf.fieldIdx)
	}
}

// TestEditCommand_RenameOnly checks the Rename shortcut: picking a
// template-derived command opens a Rename / Change values menu, Rename
// pre-fills the current name, and saving renames the command and its
// vars.tsv rows without walking the template or slot steps.
func TestEditCommand_RenameOnly(t *testing.T) {
	dir := t.TempDir()
	m := &Model{}
	m.projectDir = dir
	m.nameInput = textinput.New()
	m.config.Base = []mdl.Command{{Name: "build", Cmd: "make", Dir: "{workdir}"}}
	m.config.Commands = []mdl.Command{{Name: "as", Template: "build", Values: map[string]string{"workdir": "./src"}}}
	m.vars = map[string]string{"as.workdir": "./src"}
	m.screen = ScreenEditCommandPick
	names, refs := m.editableCommands()
	m.listItems, m.editRefs = names, refs
	m.listCursor = 0

	enter := tea.KeyMsg{Type: tea.KeyEnter}
	m.updateEditCommandPick(enter)
	if m.screen != ScreenEditCommandMode {
		t.Fatalf("picking a template command must open the mode menu, screen=%v", m.screen)
	}
	m.updateEditCommandMode(enter) // cursor 0 = Rename
	if m.screen != ScreenNameInput || m.nameInput.Value() != "as" {
		t.Fatalf("Rename must open a pre-filled name input: screen=%v value=%q", m.screen, m.nameInput.Value())
	}

	m.nameInput.SetValue("as2")
	nm, _ := m.updateNameInput(enter)
	got := nm.(Model)
	if got.config.Commands[0].Name != "as2" || got.screen != ScreenManageCommands {
		t.Fatalf("rename failed: %+v screen=%v", got.config.Commands[0], got.screen)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "vars.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "as2\tworkdir\t./src") {
		t.Fatalf("vars.tsv must follow the rename, got %q", raw)
	}
	if strings.Contains(string(raw), "as\tworkdir") {
		t.Fatalf("old vars.tsv rows must be pruned, got %q", raw)
	}
}

// TestEditCommandMode_ValuesVsTemplate checks the two edit paths:
// Change values goes straight to the slot picker (skipping the template
// screen) and Esc from the first slot returns to the menu; Change
// template opens the template picker with the current one selected.
func TestEditCommandMode_ValuesVsTemplate(t *testing.T) {
	newModel := func() *Model {
		m := &Model{}
		m.nameInput = textinput.New()
		m.lists = map[string][]mdl.ListEntry{"workdir": {{Value: "./a"}}}
		m.config.Base = []mdl.Command{{Name: "build", Cmd: "make", Dir: "{workdir}"}}
		m.config.Commands = []mdl.Command{{Name: "as", Template: "build", Values: map[string]string{"workdir": "./a"}}}
		m.screen = ScreenEditCommandPick
		names, refs := m.editableCommands()
		m.listItems, m.editRefs = names, refs
		m.listCursor = 0
		m.updateEditCommandPick(tea.KeyMsg{Type: tea.KeyEnter})
		return m
	}
	enter := tea.KeyMsg{Type: tea.KeyEnter}
	down := tea.KeyMsg{Type: tea.KeyDown}

	// Change values (cursor 1): straight to the slot picker.
	m := newModel()
	m.updateEditCommandMode(down)
	m.updateEditCommandMode(enter)
	if m.screen != ScreenSlotPick {
		t.Fatalf("Change values must skip the template screen, screen=%v", m.screen)
	}
	// Esc from the first slot returns to the menu, not the template pick.
	m.goBackCommandEditSlot()
	if m.screen != ScreenEditCommandMode {
		t.Fatalf("Esc from the first slot must return to the menu, screen=%v", m.screen)
	}

	// Change template (cursor 2): template picker on the current template.
	m = newModel()
	m.updateEditCommandMode(down)
	m.updateEditCommandMode(down)
	m.updateEditCommandMode(enter)
	if m.screen != ScreenEditCommandTemplate {
		t.Fatalf("Change template must open the template picker, screen=%v", m.screen)
	}
	if got := m.templateCandidates()[m.sce.templateRefIdx].Name; got != "build" {
		t.Fatalf("template cursor must be on the current template, got %q", got)
	}
}

// TestEditAndDeleteTSVCommands checks TSV rows are editable and
// deletable from the TUI: the form rewrites exactly the edited row,
// and deletion removes TSV rows and local commands in one pass.
func TestEditAndDeleteTSVCommands(t *testing.T) {
	dir := t.TempDir()
	tsv := "name\tgroup\tworkdir\tcmd\tshell\tslots\n" +
		"keep\t\t\techo keep\t\t\n" +
		"edit-me\tgrp\t\techo old\t\tx=xs\n" +
		"drop-me\t\t\techo drop\t\t\n"
	if err := os.WriteFile(filepath.Join(dir, "commands.tsv"), []byte(tsv), 0o644); err != nil {
		t.Fatal(err)
	}
	m := &Model{}
	m.nameInput = textinput.New()
	if err := m.loadProject(dir); err != nil {
		t.Fatal(err)
	}
	m.config.Commands = []mdl.Command{{Name: "local-cmd", Cmd: "echo local", Source: "local"}}

	// Edit "edit-me" (editRefs: keep=0, edit-me=1, drop-me=2, local-cmd=3).
	names, refs := m.editableCommands()
	m.listItems, m.editRefs = names, refs
	m.screen = ScreenEditCommandPick
	m.listCursor = 1
	m.updateEditCommandPick(tea.KeyMsg{Type: tea.KeyEnter})
	if m.cf == nil || !m.cf.tsvEdit {
		t.Fatalf("picking a TSV row must open the form in TSV mode, cf=%+v", m.cf)
	}
	m.nameInput.SetValue("edited")
	m.updateCommandForm(tea.KeyMsg{Type: tea.KeyCtrlS})
	raw, _ := os.ReadFile(filepath.Join(dir, "commands.tsv"))
	if !strings.Contains(string(raw), "edited\tgrp\t\techo old\t\tx=xs") {
		t.Fatalf("row not rewritten in place (slots must survive): %q", raw)
	}
	if !strings.Contains(string(raw), "keep\t") || !strings.Contains(string(raw), "drop-me\t") {
		t.Fatalf("other rows must be untouched: %q", raw)
	}
	if m.config.Base[1].Name != "edited" {
		t.Fatalf("in-memory Base not updated: %+v", m.config.Base[1])
	}

	// Delete "drop-me" (tsv) and "local-cmd" (local) together.
	names, refs = m.editableCommands()
	m.listItems, m.editRefs = names, refs
	m.screen = ScreenDeleteCommand
	m.deleteSelected = []int{2, 3}
	m.deleteConfirm = true
	m.deleteBtn = 1
	m.updateDeleteCommand(tea.KeyMsg{Type: tea.KeyEnter})
	raw, _ = os.ReadFile(filepath.Join(dir, "commands.tsv"))
	if strings.Contains(string(raw), "drop-me") {
		t.Fatalf("TSV row not deleted: %q", raw)
	}
	if len(m.config.Commands) != 0 {
		t.Fatalf("local command not deleted: %+v", m.config.Commands)
	}
	if strings.Contains(string(raw), "edited\t") == false || strings.Contains(string(raw), "keep\t") == false {
		t.Fatalf("unrelated rows must survive deletion: %q", raw)
	}
}

// TestOpenSlotPick_CursorOnCurrentValue checks that editing a command's
// values starts each slot picker with the cursor on the current value.
func TestOpenSlotPick_CursorOnCurrentValue(t *testing.T) {
	m := &Model{}
	m.lists = map[string][]mdl.ListEntry{
		"workdir": {{Value: "./a"}, {Value: "./b"}, {Value: "./c"}},
	}
	m.config.Commands = []mdl.Command{{Name: "as", Template: "build", Values: map[string]string{"workdir": "./c"}}}
	m.sce = &commandEditState{
		mode: 1, editIdx: 0,
		currentSlots:  []slot.Def{{Name: "workdir", ListName: "workdir"}},
		currentValues: map[string]string{},
	}
	tpl := mdl.Command{Name: "build", Cmd: "make", Dir: "{workdir}"}
	m.openSlotPickForCommandEdit(&tpl)
	if m.sp == nil || m.sp.cursor != 2 {
		t.Fatalf("cursor must start on the current value, sp=%+v", m.sp)
	}
}

// TestCommandForm_CtrlSSavesFromAnyField checks Ctrl+S saves the form
// without walking the remaining fields.
func TestCommandForm_CtrlSSavesFromAnyField(t *testing.T) {
	m := &Model{}
	m.projectDir = t.TempDir()
	m.nameInput = textinput.New()
	m.config.Commands = []mdl.Command{{Name: "old", Cmd: "echo hi"}}
	m.openCommandForm(0)

	m.nameInput.SetValue("new")
	m.updateCommandForm(tea.KeyMsg{Type: tea.KeyCtrlS})
	if m.cf != nil {
		t.Fatalf("Ctrl+S on the first field must save and close the form, errMsg=%q", m.errMsg)
	}
	if m.config.Commands[0].Name != "new" || m.config.Commands[0].Cmd != "echo hi" {
		t.Fatalf("saved command = %+v", m.config.Commands[0])
	}
}

// TestLoadProject_Vars checks vars.tsv is loaded, {$name} references
// resolve in the workflow step preview, and undefined references surface
// a load warning naming the variable.
func TestLoadProject_Vars(t *testing.T) {
	dir := t.TempDir()
	cfg := `{"commands": [{"name": "build", "workdir": "{$root}\\api", "cmd": "make {$phase}"}]}`
	if err := os.WriteFile(filepath.Join(dir, "commands.json"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "vars.tsv"), []byte("command\tname\tvalue\n*\troot\tC:\\work\\Phase2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := &Model{}
	if err := m.loadProject(dir); err != nil {
		t.Fatal(err)
	}
	if m.vars["root"] != `C:\work\Phase2` {
		t.Fatalf("vars = %v", m.vars)
	}
	if !strings.Contains(strings.Join(m.loadWarnings, "; "), "{$phase} is not defined") {
		t.Fatalf("loadWarnings = %q, want an undefined-var warning for phase", m.loadWarnings)
	}

	cmd, ok := m.workflowStepCommand("build")
	if !ok || cmd.Dir != `C:\work\Phase2\api` {
		t.Fatalf("workflowStepCommand dir = %q, want the var resolved", cmd.Dir)
	}
	if cmd.Cmd != "make {$phase}" {
		t.Fatalf("undefined var must stay literal, got %q", cmd.Cmd)
	}
}

// TestSaveConfig_ValuesLiveInVarsTsv checks the new layout: saving moves
// a template-derived command's fixed values into vars.tsv (command.slot
// names), strips them from commands.local.json, prunes entries of
// deleted commands, and a reload restores the values from vars.tsv.
func TestSaveConfig_ValuesLiveInVarsTsv(t *testing.T) {
	dir := t.TempDir()
	base := `{"commands": [{"name": "build", "workdir": "{workdir}", "cmd": "make build"}]}`
	if err := os.WriteFile(filepath.Join(dir, "commands.json"), []byte(base), 0o644); err != nil {
		t.Fatal(err)
	}

	m := &Model{}
	if err := m.loadProject(dir); err != nil {
		t.Fatal(err)
	}
	m.vars["stale.workdir"] = "./old" // entry of a command that no longer exists
	m.config.Commands = append(m.config.Commands, mdl.Command{
		Name: "as", Template: "build", Values: map[string]string{"workdir": "./src"},
	})
	if err := m.saveConfig(); err != nil {
		t.Fatal(err)
	}

	tsv, err := os.ReadFile(filepath.Join(dir, "vars.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(tsv), "as\tworkdir\t./src") {
		t.Fatalf("vars.tsv = %q, want the fixed value mirrored as a command/name/value row", tsv)
	}
	if strings.Contains(string(tsv), "stale") {
		t.Fatal("entries of deleted commands must be pruned")
	}
	local, err := os.ReadFile(filepath.Join(dir, "commands.local.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"values", "./src", "workdir", "cmd", "slots"} {
		if strings.Contains(string(local), field) {
			t.Fatalf("commands.local.json = %q, must not contain %q — template entries persist identity only", local, field)
		}
	}

	// Reload: values come back from vars.tsv and the command re-bakes.
	m2 := &Model{}
	if err := m2.loadProject(dir); err != nil {
		t.Fatal(err)
	}
	cmd, ok := m2.config.FindCommand("as")
	if !ok || cmd.Values["workdir"] != "./src" || cmd.Dir != "./src" {
		t.Fatalf("reloaded command = %+v, want values restored and baked", cmd)
	}
}

// TestLoadProject_LegacyValuesStillWork checks values still stored inside
// commands.local.json (pre-vars.tsv layout) keep working, and vars.tsv
// wins when both define the same slot.
func TestLoadProject_LegacyValuesStillWork(t *testing.T) {
	dir := t.TempDir()
	base := `{"commands": [{"name": "build", "workdir": "{workdir}", "cmd": "make build"}]}`
	local := `{"commands": [{"name": "as", "template": "build", "values": {"workdir": "./legacy"}}]}`
	if err := os.WriteFile(filepath.Join(dir, "commands.json"), []byte(base), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "commands.local.json"), []byte(local), 0o644); err != nil {
		t.Fatal(err)
	}

	m := &Model{}
	if err := m.loadProject(dir); err != nil {
		t.Fatal(err)
	}
	if cmd, _ := m.config.FindCommand("as"); cmd.Dir != "./legacy" {
		t.Fatalf("legacy values must still bake, got %+v", cmd)
	}

	if err := os.WriteFile(filepath.Join(dir, "vars.tsv"), []byte("as.workdir\t./from-vars\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m2 := &Model{}
	if err := m2.loadProject(dir); err != nil {
		t.Fatal(err)
	}
	if cmd, _ := m2.config.FindCommand("as"); cmd.Dir != "./from-vars" {
		t.Fatalf("vars.tsv must win over legacy values, got %+v", cmd)
	}
}

// TestLoadProject_WarnsOrphanedVars checks vars.tsv rows for a command
// that no longer exists surface a load warning, since the next save
// would drop them.
func TestLoadProject_WarnsOrphanedVars(t *testing.T) {
	dir := t.TempDir()
	cfg := `{"commands": [{"name": "build", "cmd": "make build"}]}`
	if err := os.WriteFile(filepath.Join(dir, "commands.json"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	tsv := "command\tname\tvalue\n*\troot\tC:\\x\nghost\tworkdir\t./src\n"
	if err := os.WriteFile(filepath.Join(dir, "vars.tsv"), []byte(tsv), 0o644); err != nil {
		t.Fatal(err)
	}

	m := &Model{}
	if err := m.loadProject(dir); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(m.loadWarnings, "; "), `unknown command "ghost"`) {
		t.Fatalf("loadWarnings = %q, want an orphaned-vars warning", m.loadWarnings)
	}
	if strings.Contains(strings.Join(m.loadWarnings, "; "), "root") {
		t.Fatalf("globals must not be treated as orphans: %q", m.loadWarnings)
	}
}

// TestLoadProject_WarnsUnknownShell checks a hand-written config with an
// unrecognized shell value surfaces a load warning instead of silently
// falling back to the platform default.
func TestLoadProject_WarnsUnknownShell(t *testing.T) {
	dir := t.TempDir()
	cfg := `{"commands": [{"name": "a", "cmd": "echo hi", "shell": "bash"}]}`
	if err := os.WriteFile(filepath.Join(dir, "commands.json"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}

	m := &Model{}
	if err := m.loadProject(dir); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(m.loadWarnings, "; "), "unknown shell") || !strings.Contains(strings.Join(m.loadWarnings, "; "), "bash") {
		t.Fatalf("loadWarnings = %q, want an unknown-shell warning naming the value", m.loadWarnings)
	}
}

// TestOpenSlotPickForCommandEdit_VariadicPrepopulatesPicked checks that
// editing a saved command's variadic slot restores the stored values as
// toggles, and confirming writes the joined string back.
func TestOpenSlotPickForCommandEdit_VariadicPrepopulatesPicked(t *testing.T) {
	m := &Model{}
	m.nameInput = textinput.New()
	m.lists = map[string][]mdl.ListEntry{"services": {
		{Value: "api"}, {Value: "web"}, {Value: "worker"},
	}}
	m.config.Commands = []mdl.Command{{
		Name: "up-all", Template: "up",
		Values: map[string]string{"services": "api worker"},
	}}
	m.sce = &commandEditState{
		mode: 1, editIdx: 0, name: "up-all",
		currentSlots:  []slot.Def{{Name: "services", ListName: "services", Variadic: true}},
		currentValues: map[string]string{},
	}
	tpl := mdl.Command{Name: "up", Cmd: "docker compose up {services...}"}
	m.openSlotPickForCommandEdit(&tpl)

	if m.sp == nil || !m.sp.variadic {
		t.Fatal("expected a variadic slot pick")
	}
	if got := m.sp.picked; len(got) != 2 || got[0] != "api" || got[1] != "worker" {
		t.Fatalf("picked = %v, want the stored values pre-toggled", got)
	}

	nm, _ := (*m).updateSlotPick(tea.KeyMsg{Type: tea.KeyEnter})
	got := nm.(*Model)
	if v := got.sce.currentValues["services"]; v != "api worker" {
		t.Fatalf("stored value = %q, want the joined picks", v)
	}
}
