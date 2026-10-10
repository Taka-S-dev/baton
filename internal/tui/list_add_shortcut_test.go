package tui

import (
	"testing"

	mdl "github.com/Taka-S-dev/baton/internal/model"
	tea "github.com/charmbracelet/bubbletea"
)

func TestListAddShortcutFromAnyRow(t *testing.T) {
	for _, width := range []int{60, 110} {
		m := exploreProject()
		m.width = width
		m.projectDir = t.TempDir()
		m.exploreTab = tabLists
		m.lists["env"] = make([]mdl.ListEntry, 100)
		for i := range m.lists["env"] {
			m.lists["env"][i].Value = "existing"
		}
		m = createKey(m, tea.KeyMsg{Type: tea.KeyEnter})
		m.le.cursor = 3
		m = createKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
		if !m.le.adding || m.le.cursor != 3 {
			t.Fatal("a must add without moving to the last row")
		}
		m = createKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
		if m.le.addVal != "a" {
			t.Fatal("a must be literal text while adding")
		}
		m = createKey(m, tea.KeyMsg{Type: tea.KeyEnter})
		m = createKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
		m = createKey(m, tea.KeyMsg{Type: tea.KeyEsc})
		if m.le.cursor != 3 || len(m.le.entries) != 100 {
			t.Fatal("cancel must preserve selection and entries")
		}
		m = createKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
		if m.le.addVal != "" || m.le.addLbl != "" || m.le.addFld != 0 {
			t.Fatal("new addition reused cancelled draft")
		}
		m = createKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")})
		m = createKey(m, tea.KeyMsg{Type: tea.KeyEnter})
		m = createKey(m, tea.KeyMsg{Type: tea.KeyEnter})
		if len(m.le.entries) != 101 || m.le.cursor != 100 || m.le.entries[100].Value != "b" {
			t.Fatal("save must select the appended entry")
		}
	}
}
