package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSearchResultActions(t *testing.T) {
	m := homeTab(homeTestModel(t), '2')
	key := func(s string) { m = homeKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}) }
	key("/")
	key("no-match-xyz")
	m = homeKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	if !m.runHome.searching || !m.msSearchTI.Focused() {
		t.Fatal("no match must keep input active")
	}
	m = homeKey(m, tea.KeyMsg{Type: tea.KeyEsc})
	key("build")
	m = homeKey(m, tea.KeyMsg{Type: tea.KeyTab})
	if len(m.msSelected) != 1 || !m.runHome.searching {
		t.Fatal("Tab must select while searching")
	}
	m = homeKey(m, tea.KeyMsg{Type: tea.KeyDown})
	if m.runHome.searching || m.msSearchTI.Focused() || m.msSearchTI.Value() != "build" || m.msCursor != 0 {
		t.Fatal("Down must focus current result without losing query")
	}
	key("e")
	if m.screen != ScreenExplore || m.runHome.commandSearch != "build" {
		t.Fatal("edit lost search context")
	}
}

func TestRunSearchResultFocusAndResume(t *testing.T) {
	for _, tab := range []rune{'1', '2'} {
		m := homeTab(homeTestModel(t), tab)
		m = homeKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
		m = homeKey(m, tea.KeyMsg{Type: tea.KeyDown})
		if m.runHome.searching || m.runSearchInput().Focused() {
			t.Fatal("results must own focus")
		}
		m = homeKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
		if !m.runHome.searching || !m.runSearchInput().Focused() {
			t.Fatal("slash must restore input focus")
		}
		m = homeKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("no-result-xyz")})
		m = homeKey(m, tea.KeyMsg{Type: tea.KeyDown})
		if !m.runHome.searching {
			t.Fatal("empty results must keep input focus")
		}
		m = homeKey(m, tea.KeyMsg{Type: tea.KeyEsc})
		if m.runSearchInput().Value() != "" || !m.runHome.searching {
			t.Fatal("Esc must clear before exiting")
		}
	}
}

func TestRunSortKeyDoesNotStealSearchText(t *testing.T) {
	m := homeTab(homeTestModel(t), '2')
	before := m.commandSort
	m = homeKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	if m.commandSort == before {
		t.Fatal("s must sort the result list")
	}
	before = m.commandSort
	m = homeKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m = homeKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	if m.commandSort != before || m.msSearchTI.Value() != "s" {
		t.Fatal("search must own s")
	}
}
