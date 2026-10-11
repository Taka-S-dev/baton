package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestWorkflowCommandPickerExplicitSearch(t *testing.T) {
	for _, screen := range []Screen{ScreenCreateWorkflow, ScreenEditWorkflowCommands} {
		m := homeTestModel(t)
		m.screen = screen
		m.setupMultiSelect()
		if m.msSearchTI.Focused() {
			t.Fatal("picker must start in list mode")
		}
		before := m.commandSort
		m = homeKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
		if m.commandSort == before {
			t.Fatal("s must sort")
		}
		before = m.commandSort
		m = homeKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
		m = homeKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
		if m.msSearchTI.Value() != "s" || m.commandSort != before {
			t.Fatal("search must own s")
		}
		m = homeKey(m, tea.KeyMsg{Type: tea.KeyEsc})
		if !m.msSearchTI.Focused() || m.msSearchTI.Value() != "" {
			t.Fatal("Esc must clear search")
		}
		m = homeKey(m, tea.KeyMsg{Type: tea.KeyDown})
		if m.msSearchTI.Focused() || m.msCursor != 0 {
			t.Fatal("Down must focus current result")
		}
		m = homeKey(m, tea.KeyMsg{Type: tea.KeyTab})
		if len(m.msSelected) != 1 {
			t.Fatal("Tab must toggle selection")
		}
		m = homeKey(m, tea.KeyMsg{Type: tea.KeyF2})
		if !m.runQueueFocused {
			t.Fatal("F2 must open queue")
		}
	}
}
