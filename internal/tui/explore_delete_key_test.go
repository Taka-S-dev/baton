package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestValueDeleteKeysOpenConfirmation(t *testing.T) {
	for _, tab := range []int{tabLists, tabVars} {
		for _, key := range []tea.KeyMsg{{Type: tea.KeyDelete}, {Type: tea.KeyRunes, Runes: []rune("x")}} {
			m := exploreProject()
			m.exploreTab = tab
			before := len(m.exploreItems(tab))
			if !strings.Contains(m.exploreKeys(), "Del/x Delete") {
				t.Fatal("missing delete hint")
			}
			m = createKey(m, key)
			if !m.deleteConfirm || len(m.deleteSelected) != 1 {
				t.Fatal("missing confirmation")
			}
			if tab == tabLists && m.screen != ScreenDeleteList {
				t.Fatal("wrong list delete screen")
			}
			if tab == tabVars && m.screen != ScreenDeleteVar {
				t.Fatal("wrong variable delete screen")
			}
			if len(m.exploreItems(tab)) != before {
				t.Fatal("deleted before confirmation")
			}
			m = createKey(m, tea.KeyMsg{Type: tea.KeyEsc})
			if len(m.exploreItems(tab)) != before {
				t.Fatal("cancel deleted item")
			}
		}
	}
}

func TestValueDeleteDoesNotActWhileSearchingOrEditing(t *testing.T) {
	for _, tab := range []int{tabLists, tabVars} {
		m := exploreProject()
		m.exploreTab = tab
		m.exploreSearching = true
		m = createKey(m, tea.KeyMsg{Type: tea.KeyDelete})
		if m.deleteConfirm {
			t.Fatal("search triggered delete")
		}
	}
	m := exploreProject()
	m.exploreTab = tabVars
	m = createKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	m.nameInput.SetValue("abc")
	m.nameInput.CursorStart()
	m = createKey(m, tea.KeyMsg{Type: tea.KeyDelete})
	if m.deleteConfirm || m.nameInput.Value() != "bc" {
		t.Fatal("Delete did not edit text")
	}
}
