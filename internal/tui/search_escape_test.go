package tui

import (
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestRunSearchEscapeClearsThenExits(t *testing.T) {
	for _, tab := range []rune{'1', '2'} {
		m := homeTab(homeTestModel(t), tab)
		m.msSelected = []int{2, 0}
		screen := m.screen
		m = homeKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
		m = homeKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("test")})
		m = homeKey(m, tea.KeyMsg{Type: tea.KeyEsc})
		if m.screen != screen || !m.runHome.searching || !m.runSearchInput().Focused() || m.runSearchInput().Value() != "" {
			t.Fatal("first Esc must clear and keep search focused")
		}
		if !strings.Contains(m.runHomeKeys(), "Esc Done") {
			t.Fatal("empty search hint")
		}
		// Navigation letters remain text after clearing, so q cannot quit.
		m = homeKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q12")})
		if m.runSearchInput().Value() != "q12" {
			t.Fatal("could not search again")
		}
		m = homeKey(m, tea.KeyMsg{Type: tea.KeyEsc})
		m = homeKey(m, tea.KeyMsg{Type: tea.KeyEsc})
		if m.screen != screen || m.runHome.searching || m.runSearchInput().Focused() || m.msEscArmed {
			t.Fatal("second Esc must only leave search")
		}
		if !reflect.DeepEqual(m.msSelected, []int{2, 0}) {
			t.Fatal("execution order lost")
		}
	}
}

func TestEditorSearchEscapeClearsThenExits(t *testing.T) {
	for tab := 0; tab < tabCount; tab++ {
		m := exploreProject()
		m.exploreTab = tab
		m.msSelected = []int{2, 0}
		m = homeKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
		m = homeKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("test")})
		m = homeKey(m, tea.KeyMsg{Type: tea.KeyEsc})
		if m.screen != ScreenExplore || !m.exploreSearching || m.exploreSearch != "" {
			t.Fatal("first Esc left editor search")
		}
		if !strings.Contains(m.exploreKeys(), "Esc Done") {
			t.Fatal("empty editor search hint")
		}
		m = homeKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q12")})
		if m.exploreSearch != "q12" || m.exploreTab != tab {
			t.Fatal("input treated as navigation")
		}
		m = homeKey(m, tea.KeyMsg{Type: tea.KeyEnter})
		if !m.exploreSearching || m.exploreSearch != "q12" {
			t.Fatal("Enter without results must keep searching")
		}
		m = homeKey(m, tea.KeyMsg{Type: tea.KeyEsc})
		m = homeKey(m, tea.KeyMsg{Type: tea.KeyEsc})
		if m.screen != ScreenExplore || m.exploreSearching || !reflect.DeepEqual(m.msSelected, []int{2, 0}) {
			t.Fatal("leaving search affected screen or selection")
		}
	}
}
