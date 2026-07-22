package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	mdl "github.com/Taka-S-dev/baton/internal/model"
)

func listsModel() Model {
	m := Model{}
	m.nameInput = textinput.New()
	m.lists = map[string][]mdl.ListEntry{
		"project": {{Value: "Z:\\api", Label: "api"}},
		"env":     {{Value: "staging"}, {Value: "production"}},
	}
	m.gotoManageLists()
	return m
}

// TestManageLists_Submenu checks Manage lists follows the same shape as
// the other Manage screens: a Create/Edit/Delete submenu where every item
// responds to Enter, and no letter-key shortcuts.
func TestManageLists_Submenu(t *testing.T) {
	want := []string{"Create list", "Edit list", "Delete list"}
	for i, item := range want {
		m := listsModel()
		if len(m.listItems) != 3 || m.listItems[i] != item {
			t.Fatalf("submenu = %v, want %v", m.listItems, want)
		}
		m.listCursor = i
		nm, _ := m.updateManageLists(tea.KeyMsg{Type: tea.KeyEnter})
		if got := nm.(Model); got.screen == ScreenManageLists {
			t.Errorf("item %q: Enter did not leave the submenu", item)
		}
	}

	// The old letter shortcuts must be dead keys now.
	m := listsModel()
	for _, k := range []rune{'n', 'a', 'd'} {
		nm, _ := m.updateManageLists(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{k}})
		m = nm.(Model)
		if m.screen != ScreenManageLists {
			t.Fatalf("letter key %q must not act on the submenu", k)
		}
	}
}

// TestEditListPick_SortedAndReturns checks the pick screen lists names in
// stable sorted order, Enter opens the entry editor on a copy, and Esc
// from the editor returns to the pick screen with the cursor kept.
func TestEditListPick_SortedAndReturns(t *testing.T) {
	m := listsModel()
	m.listCursor = 1 // Edit list
	nm, _ := m.updateManageLists(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(Model)
	if m.screen != ScreenEditListPick {
		t.Fatalf("screen = %v, want edit-list pick", m.screen)
	}
	if len(m.listItems) != 2 || m.listItems[0] != "env" || m.listItems[1] != "project" {
		t.Fatalf("listItems = %v, want sorted [env project]", m.listItems)
	}

	m.listCursor = 1 // project
	nm, _ = m.updateEditListPick(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(Model)
	if m.screen != ScreenEditList || m.le == nil || m.le.name != "project" {
		t.Fatalf("Enter must open the entry editor for the hovered list")
	}
	if !m.le.fromPick {
		t.Fatal("editor entered from the pick screen must record its origin")
	}

	nm, _ = m.updateEditList(tea.KeyMsg{Type: tea.KeyEscape})
	m = nm.(Model)
	if m.screen != ScreenEditListPick {
		t.Fatalf("screen = %v, want back on the pick screen", m.screen)
	}
	if m.listItems[m.listCursor] != "project" {
		t.Fatalf("cursor on %q, want to stay on the edited list", m.listItems[m.listCursor])
	}
}

// TestDeleteList_Flow checks the delete screen uses the shared confirm
// flow: Enter arms the No/Yes window, confirming removes the file and the
// in-memory list, and control returns to the submenu with a notice.
func TestDeleteList_Flow(t *testing.T) {
	dir := t.TempDir()
	listsDir := filepath.Join(dir, "lists")
	if err := os.MkdirAll(listsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(listsDir, "env.tsv"), []byte("staging\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := listsModel()
	m.projectDir = dir
	m.listCursor = 2 // Delete list
	nm, _ := m.updateManageLists(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(Model)
	if m.screen != ScreenDeleteList {
		t.Fatalf("screen = %v, want delete list", m.screen)
	}

	// Cursor starts on "env" (sorted): Enter arms, Tab → Yes, Enter confirms.
	nm, _ = m.updateDeleteLists(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(Model)
	if !m.deleteConfirm {
		t.Fatal("Enter must open the confirm window")
	}
	nm, _ = m.updateDeleteLists(tea.KeyMsg{Type: tea.KeyTab})
	m = nm.(Model)
	nm, _ = m.updateDeleteLists(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(Model)

	if _, err := os.Stat(filepath.Join(listsDir, "env.tsv")); !os.IsNotExist(err) {
		t.Fatal("confirmed delete must remove the .tsv file")
	}
	if _, ok := m.lists["env"]; ok {
		t.Fatal("confirmed delete must remove the in-memory list")
	}
	if m.screen != ScreenManageLists {
		t.Fatalf("screen = %v, want back on the submenu", m.screen)
	}
	if m.successMsg == "" {
		t.Fatal("a successful delete must set a notice")
	}
}

// TestCreateList_EscReturnsToSubmenu checks Create list opens the name
// input and Esc lands back on the submenu, not the main menu.
func TestCreateList_EscReturnsToSubmenu(t *testing.T) {
	m := listsModel()
	m.listCursor = 0 // Create list
	nm, _ := m.updateManageLists(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(Model)
	if m.screen != ScreenNameInput || m.nameInputMode != nameInputNewList {
		t.Fatalf("screen = %v mode = %v, want the new-list name input", m.screen, m.nameInputMode)
	}
	nm, _ = m.updateNameInput(tea.KeyMsg{Type: tea.KeyEscape})
	m = nm.(Model)
	if m.screen != ScreenManageLists {
		t.Fatalf("screen = %v, want back on the submenu", m.screen)
	}
}
