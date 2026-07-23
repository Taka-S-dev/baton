package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	mdl "github.com/Taka-S-dev/baton/internal/model"
)

func varsModel(t *testing.T) Model {
	t.Helper()
	m := Model{}
	m.nameInput = textinput.New()
	m.projectDir = t.TempDir()
	m.vars = map[string]string{}
	m.gotoVarsMgmt()
	return m
}

// TestVarsMgmt_Submenu checks Manage vars follows the standard submenu
// shape and every item responds to Enter.
func TestVarsMgmt_Submenu(t *testing.T) {
	want := []string{"Create variable (global)", "Edit variable", "Delete variable"}
	for i, item := range want {
		m := varsModel(t)
		if len(m.listItems) != 3 || m.listItems[i] != item {
			t.Fatalf("submenu = %v, want %v", m.listItems, want)
		}
		m.listCursor = i
		nm, _ := m.updateVarsMgmt(tea.KeyMsg{Type: tea.KeyEnter})
		if got := nm.(Model); got.screen == ScreenManageVars {
			t.Errorf("item %q: Enter did not leave the submenu", item)
		}
	}
}

// TestCreateVariable_Flow checks name validation and that saving writes
// a global "*" row to vars.tsv.
func TestCreateVariable_Flow(t *testing.T) {
	m := varsModel(t)
	m.vars["taken"] = "x"
	nm, _ := m.updateVarsMgmt(tea.KeyMsg{Type: tea.KeyEnter}) // Create
	m = nm.(Model)
	if m.screen != ScreenVarForm {
		t.Fatalf("screen = %v, want the var form", m.screen)
	}

	enter := tea.KeyMsg{Type: tea.KeyEnter}
	step := func(v string) {
		m.nameInput.SetValue(v)
		nm, _ = m.updateVarForm(enter)
		m = nm.(Model)
	}

	step("bad name") // space → invalid
	if m.errMsg == "" || m.ve.fieldIdx != 0 {
		t.Fatalf("invalid name must be rejected, errMsg=%q", m.errMsg)
	}
	step("taken")
	if m.errMsg == "" || m.ve.fieldIdx != 0 {
		t.Fatalf("duplicate name must be rejected, errMsg=%q", m.errMsg)
	}
	step("root")
	if m.ve.fieldIdx != 1 {
		t.Fatal("valid name must advance to the value field")
	}
	step(`C:\demo\phase1`)
	if m.screen != ScreenManageVars || m.successMsg == "" {
		t.Fatalf("save must return to the submenu with a notice, screen=%v", m.screen)
	}

	raw, err := os.ReadFile(filepath.Join(m.projectDir, "vars.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "*\troot\tC:\\demo\\phase1") {
		t.Fatalf("vars.tsv = %q, want the global row", raw)
	}
}

// TestEditVariable_RebaseOffer checks the core safety design: after a
// value change, literals are matched by PREFIX (never substring), a
// suspicious boundary defaults to off, and applying rewrites the checked
// values into {$name} references in vars.tsv and the list files.
func TestEditVariable_RebaseOffer(t *testing.T) {
	m := varsModel(t)
	if err := os.MkdirAll(filepath.Join(m.projectDir, "lists"), 0o755); err != nil {
		t.Fatal(err)
	}
	m.vars = map[string]string{
		"root":       `C:\old`,
		"as.workdir": `C:\old\api`,  // clean boundary → default on
		"b.x":        `C:\oldish`,   // prefix but no separator → default off
		"c.y":        `dir C:\old2`, // contains but not prefix → not a candidate
	}
	m.lists = map[string][]mdl.ListEntry{
		"project": {{Value: `C:\old\web`, Label: "web"}, {Value: "other"}},
	}

	// Edit root: pick it, change the value.
	nm, _ := m.updateVarsMgmt(tea.KeyMsg{Type: tea.KeyDown})
	m = nm.(Model)
	nm, _ = m.updateVarsMgmt(tea.KeyMsg{Type: tea.KeyEnter}) // Edit variable
	m = nm.(Model)
	if m.screen != ScreenEditVarPick {
		t.Fatalf("screen = %v, want the edit pick", m.screen)
	}
	nm, _ = m.updateEditVarPick(tea.KeyMsg{Type: tea.KeyEnter}) // only global: root
	m = nm.(Model)
	m.nameInput.SetValue(`C:\new`)
	nm, _ = m.updateVarForm(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(Model)

	if m.screen != ScreenVarRebase || m.vr == nil {
		t.Fatalf("changed value with literals must open the rebase offer, screen=%v", m.screen)
	}
	if len(m.vr.items) != 3 {
		t.Fatalf("items = %+v, want 3 candidates (as.workdir, b.x, list entry)", m.vr.items)
	}
	byOld := map[string]varRebaseItem{}
	for _, it := range m.vr.items {
		byOld[it.oldValue] = it
	}
	if it := byOld[`C:\old\api`]; !it.on || it.newValue != `{$root}\api` {
		t.Fatalf("clean boundary must default on with a reference rewrite: %+v", it)
	}
	if it := byOld[`C:\oldish`]; it.on {
		t.Fatalf("suspicious boundary must default off: %+v", it)
	}
	if it := byOld[`C:\old\web`]; !it.on || it.newValue != `{$root}\web` {
		t.Fatalf("list entry must be a default-on candidate: %+v", it)
	}

	nm, _ = m.updateVarRebase(tea.KeyMsg{Type: tea.KeyEnter}) // → confirm window
	m = nm.(Model)
	if !m.vr.confirm {
		t.Fatal("Enter with checked items must open the No/Yes window")
	}
	nm, _ = m.updateVarRebase(tea.KeyMsg{Type: tea.KeyTab}) // → Yes
	m = nm.(Model)
	nm, _ = m.updateVarRebase(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(Model)
	if m.screen != ScreenManageVars || !strings.Contains(m.successMsg, "rebased 2") {
		t.Fatalf("apply must return with a rebase notice, got %q", m.successMsg)
	}
	if m.vars["as.workdir"] != `{$root}\api` || m.vars["b.x"] != `C:\oldish` {
		t.Fatalf("vars after apply = %v", m.vars)
	}
	if m.lists["project"][0].Value != `{$root}\web` {
		t.Fatalf("list entry not rebased: %+v", m.lists["project"])
	}

	rawVars, _ := os.ReadFile(filepath.Join(m.projectDir, "vars.tsv"))
	if !strings.Contains(string(rawVars), "as\tworkdir\t{$root}\\api") {
		t.Fatalf("vars.tsv = %q, want the rebased scoped row", rawVars)
	}
	rawList, _ := os.ReadFile(filepath.Join(m.projectDir, "lists", "project.tsv"))
	if !strings.Contains(string(rawList), `{$root}\web`) || !strings.Contains(string(rawList), "web") {
		t.Fatalf("project.tsv = %q, want the rebased entry with its label", rawList)
	}
}

// TestEditVariable_EscKeepsLiterals checks skipping the offer changes
// nothing beyond the variable itself.
func TestEditVariable_EscKeepsLiterals(t *testing.T) {
	m := varsModel(t)
	m.vars = map[string]string{"root": `C:\old`, "as.workdir": `C:\old\api`}

	m.ve = &varEditState{mode: 1, name: "root", oldValue: `C:\old`}
	nm, _ := m.saveVar(`C:\new`)
	m = nm.(Model)
	if m.screen != ScreenVarRebase {
		t.Fatalf("screen = %v, want the rebase offer", m.screen)
	}
	nm, _ = m.updateVarRebase(tea.KeyMsg{Type: tea.KeyEscape})
	m = nm.(Model)
	if m.vars["as.workdir"] != `C:\old\api` {
		t.Fatalf("Esc must keep literals, got %q", m.vars["as.workdir"])
	}
	if m.vars["root"] != `C:\new` {
		t.Fatalf("the variable itself must stay changed, got %q", m.vars["root"])
	}
}

// TestEditVariable_NoLiterals checks a change with no matching literals
// skips the offer entirely.
func TestEditVariable_NoLiterals(t *testing.T) {
	m := varsModel(t)
	m.vars = map[string]string{"root": `C:\old`, "as.workdir": `{$root}\api`}

	m.ve = &varEditState{mode: 1, name: "root", oldValue: `C:\old`}
	nm, _ := m.saveVar(`C:\new`)
	m = nm.(Model)
	if m.screen != ScreenManageVars {
		t.Fatalf("no literals must go straight back, screen=%v", m.screen)
	}
	if m.vars["as.workdir"] != `{$root}\api` {
		t.Fatalf("reference values must never be touched, got %q", m.vars["as.workdir"])
	}
}

// TestDeleteVariable_Flow checks deletion through the shared confirm flow
// removes the row from vars.tsv.
func TestDeleteVariable_Flow(t *testing.T) {
	m := varsModel(t)
	m.vars = map[string]string{"root": `C:\x`, "host": "example"}

	m.listCursor = 2 // Delete variable
	nm, _ := m.updateVarsMgmt(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(Model)
	if m.screen != ScreenDeleteVar || len(m.listItems) != 2 {
		t.Fatalf("screen=%v items=%v", m.screen, m.listItems)
	}

	// Cursor starts on "host" (sorted): confirm its deletion.
	nm, _ = m.updateDeleteVars(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(Model)
	nm, _ = m.updateDeleteVars(tea.KeyMsg{Type: tea.KeyTab}) // → Yes
	m = nm.(Model)
	nm, _ = m.updateDeleteVars(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(Model)

	if _, ok := m.vars["host"]; ok {
		t.Fatal("confirmed delete must remove the variable")
	}
	raw, _ := os.ReadFile(filepath.Join(m.projectDir, "vars.tsv"))
	if strings.Contains(string(raw), "host") || !strings.Contains(string(raw), "root") {
		t.Fatalf("vars.tsv = %q, want host gone and root kept", raw)
	}
	if m.screen != ScreenManageVars {
		t.Fatalf("screen = %v, want back on the submenu", m.screen)
	}
}

// TestVarPick_ShowsWholeFile checks the pick list shows every vars.tsv
// row — globals first, then saved commands' fixed values — so the
// screen matches what the file actually contains.
func TestVarPick_ShowsWholeFile(t *testing.T) {
	m := varsModel(t)
	m.vars = map[string]string{
		"root":       `C:\x`,
		"as.workdir": `{$root}\api`,
		"b.env":      "staging",
	}
	m.setVarPickBase()
	if len(m.listItems) != 3 {
		t.Fatalf("listItems = %v, want all 3 rows", m.listItems)
	}
	if m.varPickNames[0] != "root" || m.varPickNames[1] != "as.workdir" || m.varPickNames[2] != "b.env" {
		t.Fatalf("order = %v, want globals first then scoped, sorted", m.varPickNames)
	}
	if !strings.Contains(m.listItems[0], "{$root}") || !strings.Contains(m.listItems[1], "as.workdir") {
		t.Fatalf("labels = %v", m.listItems)
	}
}

// TestEditScopedValue checks editing a saved fixed value from Manage
// vars: vars.tsv is written and the owning command re-bakes so the
// change is live without a reload.
func TestEditScopedValue(t *testing.T) {
	m := varsModel(t)
	m.config.Base = []mdl.Command{{Name: "build", Cmd: "make", Dir: "{workdir}", Source: "tsv"}}
	m.config.Commands = []mdl.Command{{Name: "as", Template: "build", Values: map[string]string{"workdir": "./old"}, Dir: "./old", Cmd: "make", Source: "local"}}
	m.vars = map[string]string{"as.workdir": "./old"}

	m.screen = ScreenEditVarPick
	m.setVarPickBase()
	nm, _ := m.updateEditVarPick(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(Model)
	if m.screen != ScreenVarForm || m.nameInput.Value() != "./old" {
		t.Fatalf("form must open pre-filled, screen=%v value=%q", m.screen, m.nameInput.Value())
	}

	m.nameInput.SetValue("./new")
	nm, _ = m.updateVarForm(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(Model)
	if m.screen != ScreenManageVars {
		t.Fatalf("scoped edit must return straight to the submenu (no rebase offer), screen=%v", m.screen)
	}
	if m.vars["as.workdir"] != "./new" {
		t.Fatalf("vars = %v", m.vars)
	}
	cmd, _ := m.config.FindCommand("as")
	if cmd.Values["workdir"] != "./new" || cmd.Dir != "./new" {
		t.Fatalf("command must be synced and re-baked, got %+v", cmd)
	}
	raw, _ := os.ReadFile(filepath.Join(m.projectDir, "vars.tsv"))
	if !strings.Contains(string(raw), "as\tworkdir\t./new") {
		t.Fatalf("vars.tsv = %q", raw)
	}
}

// TestDeleteScopedValue checks deleting a fixed value un-fixes the
// slot: the command re-bakes with the {placeholder} restored.
func TestDeleteScopedValue(t *testing.T) {
	m := varsModel(t)
	m.config.Base = []mdl.Command{{Name: "build", Cmd: "make", Dir: "{workdir}", Source: "tsv"}}
	m.config.Commands = []mdl.Command{{Name: "as", Template: "build", Values: map[string]string{"workdir": "./old"}, Dir: "./old", Cmd: "make", Source: "local"}}
	m.vars = map[string]string{"as.workdir": "./old"}

	m.screen = ScreenDeleteVar
	m.setVarPickBase()
	nm, _ := m.updateDeleteVars(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(Model)
	nm, _ = m.updateDeleteVars(tea.KeyMsg{Type: tea.KeyTab}) // → Yes
	m = nm.(Model)
	nm, _ = m.updateDeleteVars(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(Model)

	if _, ok := m.vars["as.workdir"]; ok {
		t.Fatal("row must be deleted")
	}
	cmd, _ := m.config.FindCommand("as")
	if _, ok := cmd.Values["workdir"]; ok {
		t.Fatalf("fixed value must be removed from the command, got %+v", cmd.Values)
	}
	if cmd.Dir != "{workdir}" {
		t.Fatalf("slot must be un-fixed (re-baked to the placeholder), got %q", cmd.Dir)
	}
	if !strings.Contains(m.successMsg, "prompted at run time") {
		t.Fatalf("notice must explain the un-fix, got %q", m.successMsg)
	}
}

// TestEditScopedValue_PropagateOffer checks editing a fixed value whose
// old value is shared by OTHER rows opens an explicit propagate offer:
// the edited row is excluded, non-matching rows are not candidates, and
// applying changes the checked rows and syncs their commands.
func TestEditScopedValue_PropagateOffer(t *testing.T) {
	m := varsModel(t)
	if err := os.MkdirAll(filepath.Join(m.projectDir, "lists"), 0o755); err != nil {
		t.Fatal(err)
	}
	m.config.Base = []mdl.Command{{Name: "build", Cmd: "make", Dir: "{workdir}", Source: "tsv"}}
	m.config.Commands = []mdl.Command{
		{Name: "ddd", Template: "build", Values: map[string]string{"workdir": "./src"}, Dir: "./src", Cmd: "make", Source: "local"},
		{Name: "fffff", Template: "build", Values: map[string]string{"workdir": "./src"}, Dir: "./src", Cmd: "make", Source: "local"},
	}
	m.vars = map[string]string{
		"ddd.workdir":   "./src",
		"fffff.workdir": "./src",
		"other.workdir": "./elsewhere",
	}
	m.lists = map[string][]mdl.ListEntry{"paths": {{Value: "./src"}, {Value: "./elsewhere"}}}

	m.ve = &varEditState{mode: 1, name: "ddd.workdir", oldValue: "./src"}
	nm, _ := m.saveVar("./step")
	m = nm.(Model)
	if m.screen != ScreenVarRebase || m.vr == nil || !m.vr.propagate {
		t.Fatalf("shared old value must open the propagate offer, screen=%v", m.screen)
	}
	if len(m.vr.items) != 2 {
		t.Fatalf("items = %+v, want fffff.workdir and the list entry only", m.vr.items)
	}
	for _, it := range m.vr.items {
		if it.key == "ddd.workdir" {
			t.Fatal("the edited row must not be offered to itself")
		}
		if !it.on || it.newValue != "./step" {
			t.Fatalf("exact match must default on with the new value: %+v", it)
		}
	}

	// Enter opens the No/Yes window; No returns to the checklist untouched.
	nm, _ = m.updateVarRebase(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(Model)
	nm, _ = m.updateVarRebase(tea.KeyMsg{Type: tea.KeyEnter}) // No (default)
	m = nm.(Model)
	if m.vr == nil || m.vr.confirm || m.vars["fffff.workdir"] != "./src" {
		t.Fatalf("No must return to the checklist without applying, vars=%v", m.vars)
	}

	nm, _ = m.updateVarRebase(tea.KeyMsg{Type: tea.KeyEnter}) // → confirm
	m = nm.(Model)
	nm, _ = m.updateVarRebase(tea.KeyMsg{Type: tea.KeyTab}) // → Yes
	m = nm.(Model)
	nm, _ = m.updateVarRebase(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(Model)
	if m.vars["fffff.workdir"] != "./step" || m.vars["other.workdir"] != "./elsewhere" {
		t.Fatalf("vars after propagate = %v", m.vars)
	}
	if m.lists["paths"][0].Value != "./step" || m.lists["paths"][1].Value != "./elsewhere" {
		t.Fatalf("lists after propagate = %+v", m.lists["paths"])
	}
	cmd, _ := m.config.FindCommand("fffff")
	if cmd.Dir != "./step" {
		t.Fatalf("command must re-bake with the propagated value, got %+v", cmd)
	}
	if !strings.Contains(m.successMsg, "changed 2 matching value(s)") {
		t.Fatalf("notice = %q", m.successMsg)
	}
}

// TestEditScopedValue_PropagateEscKeepsOthers checks Esc applies nothing
// beyond the edited row itself.
func TestEditScopedValue_PropagateEscKeepsOthers(t *testing.T) {
	m := varsModel(t)
	m.vars = map[string]string{"ddd.workdir": "./src", "fffff.workdir": "./src"}

	m.ve = &varEditState{mode: 1, name: "ddd.workdir", oldValue: "./src"}
	nm, _ := m.saveVar("./step")
	m = nm.(Model)
	nm, _ = m.updateVarRebase(tea.KeyMsg{Type: tea.KeyEscape})
	m = nm.(Model)
	if m.vars["ddd.workdir"] != "./step" || m.vars["fffff.workdir"] != "./src" {
		t.Fatalf("Esc must keep the others, got %v", m.vars)
	}
}

// TestEditListEntry_PropagateOffer checks the symmetric direction:
// editing a list entry whose old value other places share (fixed
// values, other lists) opens the same offer, applying syncs
// everything, and the flow returns to the entry editor with a fresh
// working copy.
func TestEditListEntry_PropagateOffer(t *testing.T) {
	m := varsModel(t)
	if err := os.MkdirAll(filepath.Join(m.projectDir, "lists"), 0o755); err != nil {
		t.Fatal(err)
	}
	m.config.Base = []mdl.Command{{Name: "build", Cmd: "make", Dir: "{workdir}", Source: "tsv"}}
	m.config.Commands = []mdl.Command{{Name: "bbb", Template: "build", Values: map[string]string{"workdir": "./src"}, Dir: "./src", Cmd: "make", Source: "local"}}
	m.vars = map[string]string{"bbb.workdir": "./src"}
	m.lists = map[string][]mdl.ListEntry{
		"paths":  {{Value: "./src", Label: "src"}, {Value: "./other"}},
		"backup": {{Value: "./src"}},
	}
	m.le = &listEditState{name: "paths", entries: append([]mdl.ListEntry{}, m.lists["paths"]...), fromPick: true}
	m.le.editing = true
	m.le.editValTI = newListTextinput("Value > ", "./step")
	m.le.editLblTI = newListTextinput("Label > ", "src")
	m.le.editFld = 1
	m.screen = ScreenEditList

	nm, _ := m.updateEditList(tea.KeyMsg{Type: tea.KeyEnter}) // save entry
	m = nm.(Model)
	if m.screen != ScreenVarRebase || m.vr == nil || !m.vr.propagate || !m.vr.returnToList {
		t.Fatalf("shared old value must open the propagate offer, screen=%v", m.screen)
	}
	// Candidates: the fixed value and backup's entry — NOT the edited
	// entry itself, and not ./other.
	if len(m.vr.items) != 2 {
		t.Fatalf("items = %+v, want the fixed value and the backup entry", m.vr.items)
	}

	nm, _ = m.updateVarRebase(tea.KeyMsg{Type: tea.KeyEnter}) // → confirm
	m = nm.(Model)
	nm, _ = m.updateVarRebase(tea.KeyMsg{Type: tea.KeyTab}) // → Yes
	m = nm.(Model)
	nm, _ = m.updateVarRebase(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(Model)

	if m.screen != ScreenEditList {
		t.Fatalf("flow must return to the entry editor, screen=%v", m.screen)
	}
	if m.vars["bbb.workdir"] != "./step" || m.lists["backup"][0].Value != "./step" {
		t.Fatalf("propagate missed a target: vars=%v backup=%v", m.vars, m.lists["backup"])
	}
	cmd, _ := m.config.FindCommand("bbb")
	if cmd.Dir != "./step" {
		t.Fatalf("command must re-bake, got %+v", cmd)
	}
	if m.le.entries[0].Value != "./step" {
		t.Fatalf("editor working copy must be refreshed, got %+v", m.le.entries)
	}
	raw, _ := os.ReadFile(filepath.Join(m.projectDir, "lists", "backup.tsv"))
	if !strings.Contains(string(raw), "./step") {
		t.Fatalf("backup.tsv = %q", raw)
	}
}

// TestCreateVariable_ExtractsLiterals checks the create-time offer:
// creating a global whose value matches existing literals proposes
// rewriting them to {$name}, so values that should move together end
// up sharing the variable — the safe alternative to propagating a
// scoped edit to other rows that merely hold the same string.
func TestCreateVariable_ExtractsLiterals(t *testing.T) {
	m := varsModel(t)
	m.config.Base = []mdl.Command{{Name: "say", Cmd: "echo {message}", Source: "tsv"}}
	m.config.Commands = []mdl.Command{
		{Name: "aaaa", Template: "say", Values: map[string]string{"message": "hello"}, Cmd: "echo hello", Source: "local"},
		{Name: "ddd", Template: "say", Values: map[string]string{"message": "hello"}, Cmd: "echo hello", Source: "local"},
	}
	m.vars = map[string]string{"aaaa.message": "hello", "ddd.message": "hello"}

	m.ve = &varEditState{mode: 0, name: "msg"}
	nm, _ := m.saveVar("hello")
	m = nm.(Model)
	if m.screen != ScreenVarRebase || m.vr == nil || !m.vr.created {
		t.Fatalf("create with matching literals must open the extraction offer, screen=%v", m.screen)
	}
	if len(m.vr.items) != 2 {
		t.Fatalf("items = %+v, want both scoped values", m.vr.items)
	}
	for _, it := range m.vr.items {
		if !it.on || it.newValue != "{$msg}" {
			t.Fatalf("exact matches must default on and become plain references: %+v", it)
		}
	}

	nm, _ = m.updateVarRebase(tea.KeyMsg{Type: tea.KeyEnter}) // → confirm
	m = nm.(Model)
	nm, _ = m.updateVarRebase(tea.KeyMsg{Type: tea.KeyTab}) // → Yes
	m = nm.(Model)
	nm, _ = m.updateVarRebase(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(Model)
	if m.vars["aaaa.message"] != "{$msg}" || m.vars["ddd.message"] != "{$msg}" {
		t.Fatalf("vars after extraction = %v", m.vars)
	}
	if !strings.Contains(m.successMsg, "created") || !strings.Contains(m.successMsg, "rebased 2") {
		t.Fatalf("notice = %q", m.successMsg)
	}
	// Both commands re-baked; {$msg} resolves at run time.
	cmd, _ := m.config.FindCommand("aaaa")
	if cmd.Values["message"] != "{$msg}" {
		t.Fatalf("command not synced: %+v", cmd)
	}
}

// TestVarForm_PromptNotDoubled guards the shared textinput's prompt:
// the var form sets its own field prompts, and the plain name inputs
// must reset the prompt afterwards instead of showing a leftover.
func TestVarForm_PromptNotDoubled(t *testing.T) {
	m := varsModel(t)
	nm, _ := m.updateVarsMgmt(tea.KeyMsg{Type: tea.KeyEnter}) // Create
	m = nm.(Model)
	if m.nameInput.Prompt != "name  > " {
		t.Fatalf("create form prompt = %q, want the form's own field prompt", m.nameInput.Prompt)
	}
	view := m.viewVarForm(80)
	if strings.Contains(view, "Name >") {
		t.Fatalf("default prompt leaked into the var form:\n%s", view)
	}

	m.nameInput.SetValue("root")
	nm, _ = m.updateVarForm(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(Model)
	if m.nameInput.Prompt != "value > " {
		t.Fatalf("value field prompt = %q", m.nameInput.Prompt)
	}

	// A later plain name input must get its default prompt back.
	nm, _ = m.openNameInput(nameInputWorkflow)
	m = nm.(Model)
	if m.nameInput.Prompt != "Name > " {
		t.Fatalf("openNameInput prompt = %q, want the default restored", m.nameInput.Prompt)
	}
}

// TestVarRefLocations counts references across commands, lists and
// saved values.
func TestVarRefLocations(t *testing.T) {
	m := varsModel(t)
	m.config.Base = []mdl.Command{{Name: "build", Cmd: `make -C {$root}`}}
	m.config.Commands = []mdl.Command{{Name: "x", Dir: `{$root}\api`, Cmd: "make"}}
	m.lists = map[string][]mdl.ListEntry{"project": {{Value: `{$root}\web`}}}
	m.vars = map[string]string{"root": `C:\x`, "as.workdir": `{$root}\src`}

	refs := m.varRefLocations("root")
	if len(refs) != 4 {
		t.Fatalf("refs = %v, want 4 (2 commands, 1 list, 1 saved value)", refs)
	}
}
