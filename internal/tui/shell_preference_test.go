package tui

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestShellPreferencePersistsOnlyAfterSave(t *testing.T) {
	dir := t.TempDir()
	m := exploreProject()
	m.projectDir = dir
	m.openExploreCommandForm("")
	m.cf.fields[0], m.cf.fields[1], m.cf.fields[4] = "remember", "echo hello", "ps"
	m = exploreModel(exploreFirst(m.saveCommandForm()))
	data, err := os.ReadFile(filepath.Join(dir, ".last_shell"))
	if err != nil || string(data) != "ps\n" {
		t.Fatalf("preference not saved: %q %v", data, err)
	}
	fresh := exploreProject()
	fresh.projectDir = dir
	fresh.openExploreCommandForm("")
	if fresh.cf.fields[4] != "ps" || !fresh.cf.shellRemembered || fresh.formDirty() {
		t.Fatal("new form should open clean with remembered shell")
	}
	fresh.cfGotoField(4)
	if fresh.cf.shellCursor != 1 {
		t.Fatal("selector did not preselect PowerShell")
	}
	fresh.cf.shellCursor = 0
	fresh.acceptShellChoice()
	fresh = createKey(fresh, tea.KeyMsg{Type: tea.KeyEsc})
	fresh = createKey(fresh, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	fresh.openExploreCommandForm("")
	if fresh.cf.fields[4] != "ps" {
		t.Fatal("cancel changed preference")
	}
	fresh.cf.fields[0], fresh.cf.fields[1], fresh.cf.fields[4] = "default-command", "echo default", ""
	fresh = exploreModel(exploreFirst(fresh.saveCommandForm()))
	fresh.openExploreCommandForm("")
	if fresh.cf.fields[4] != "" || !fresh.cf.shellRemembered {
		t.Fatal("Default selection must also be remembered")
	}
}

func TestShellPreferenceDoesNotOverrideEditsOrOtherProjects(t *testing.T) {
	m := exploreProject()
	m.projectDir = t.TempDir()
	m.rememberShell("ps")
	m.openExploreCommandForm("build")
	if m.cf.fields[4] != "" || m.cf.shellRemembered {
		t.Fatal("edit must retain saved shell")
	}
	m.openCommandForm(-1)
	if m.cf.fields[4] != "ps" {
		t.Fatal("manage menu must reuse preference")
	}
	m.projectDir = t.TempDir()
	m.openExploreCommandForm("")
	if m.cf.fields[4] != "" || m.cf.shellRemembered {
		t.Fatal("preference leaked across projects")
	}
}

func TestShellPreferenceUnchangedOnSaveFailure(t *testing.T) {
	m := exploreProject()
	m.projectDir = t.TempDir()
	m.rememberShell("ps")
	// A directory at the output file path forces the command save to fail.
	if err := os.Mkdir(filepath.Join(m.projectDir, "commands.tsv"), 0700); err != nil {
		t.Fatal(err)
	}
	m.openExploreCommandForm("")
	m.cf.fields[0], m.cf.fields[1], m.cf.fields[4] = "cannot-save", "echo hi", ""
	m = exploreModel(exploreFirst(m.saveCommandForm()))
	data, err := os.ReadFile(filepath.Join(m.projectDir, ".last_shell"))
	if err != nil || string(data) != "ps\n" {
		t.Fatal("failed save changed shell preference")
	}
}
