package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestHelpPreservesDraftAndQuestionInput(t *testing.T) {
	m := exploreProject()
	m = createKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	m.nameInput.SetValue("echo")
	m = createKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	if m.helpOpen || !strings.Contains(m.nameInput.Value(), "?") {
		t.Fatal("question was not input")
	}
	value := m.nameInput.Value()
	m = createKey(m, tea.KeyMsg{Type: tea.KeyF1})
	if !m.helpOpen || !strings.Contains(stripStyle(m.View()), "Help") {
		t.Fatal("help missing")
	}
	m = createKey(m, tea.KeyMsg{Type: tea.KeyDown})
	m = createKey(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.helpOpen || m.nameInput.Value() != value || m.cf == nil {
		t.Fatal("help changed draft")
	}
}

func TestHelpUsesActionsAndDoesNotExecuteThem(t *testing.T) {
	m := exploreProject()
	m.exploreTab = tabLists
	m = createKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	if !m.helpOpen {
		t.Fatal("question did not open help")
	}
	m = createKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	if m.deleteConfirm || !m.helpOpen {
		t.Fatal("help executed action")
	}
	m.width = 40
	m.height = 18
	for i := 0; i < 100; i++ {
		m = createKey(m, tea.KeyMsg{Type: tea.KeyDown})
	}
	p := m.helpGeometry(m.viewWidth())
	if m.helpOffset < 1 || p.x+p.w > m.width || p.y+p.h > m.height {
		t.Fatal("help overflow")
	}
	m.width = 110
	m.height = 40
	m = publicClick(m, 0, 0)
	if m.helpOpen {
		t.Fatal("outside click did not close")
	}
}

func TestHelpHintConsistent(t *testing.T) {
	m := exploreProject()
	for _, searching := range []bool{false, true} {
		m.exploreSearching = searching
		if m.helpHint() != "F1 Help" {
			t.Fatal("inconsistent help hint")
		}
		for _, w := range []int{40, 60, 120} {
			footer := m.compactFooter(w)
			if strings.Contains(footer, "? Help") || !strings.Contains(footer, "F1 Help") {
				t.Fatalf("inconsistent footer: %q", footer)
			}
		}
	}
}
