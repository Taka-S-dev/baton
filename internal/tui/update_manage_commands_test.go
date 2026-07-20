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
