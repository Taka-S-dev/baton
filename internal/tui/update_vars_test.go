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
	want := []string{"Create variable", "Edit variable", "Delete variable"}
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

	nm, _ = m.updateVarRebase(tea.KeyMsg{Type: tea.KeyEnter}) // apply defaults
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
