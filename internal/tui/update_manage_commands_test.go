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
	if len(m.config.Commands) != 1 || m.config.Commands[0].Shell != "ps" {
		t.Fatalf("saved commands = %+v, want one command with Shell=ps", m.config.Commands)
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
	if !strings.Contains(m.loadWarning, "undefined var {$phase}") {
		t.Fatalf("loadWarning = %q, want an undefined-var warning for phase", m.loadWarning)
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
	if !strings.Contains(m.loadWarning, `unknown command "ghost"`) {
		t.Fatalf("loadWarning = %q, want an orphaned-vars warning", m.loadWarning)
	}
	if strings.Contains(m.loadWarning, "root") {
		t.Fatalf("globals must not be treated as orphans: %q", m.loadWarning)
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
	if !strings.Contains(m.loadWarning, "unknown shell") || !strings.Contains(m.loadWarning, "bash") {
		t.Fatalf("loadWarning = %q, want an unknown-shell warning naming the value", m.loadWarning)
	}
}
