package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"testing"
)

// Exercise the public event dispatcher, including the full-screen fallback.
func TestEscapeReturnsToListFromEveryField(t *testing.T) {
	for _, width := range []int{60, 110} {
		for field := 0; field < 5; field++ {
			for _, dirty := range []bool{false, true} {
				m := exploreProject()
				m.width = width
				m.exploreCursor[tabCommands] = 1
				m = exploreModel(exploreFirst(m.Update(tea.KeyMsg{Type: tea.KeyEnter})))
				m.cfGotoField(field)
				if field == 4 {
					m = exploreModel(exploreFirst(m.Update(tea.KeyMsg{Type: tea.KeyEsc})))
					if m.cf == nil || m.cf.shellOpen {
						t.Fatal("first Esc must only close shell choices")
					}
				}
				original := m.nameInput.Value()
				if dirty {
					m.nameInput.SetValue(original + " changed")
				}
				m = exploreModel(exploreFirst(m.Update(tea.KeyMsg{Type: tea.KeyEsc})))
				if dirty {
					if m.cf == nil || m.cf.fieldIdx != field || m.leaveAsk != leaveForm {
						t.Fatalf("width=%d field=%d: Esc must immediately ask about changes", width, field)
					}
					m = exploreModel(exploreFirst(m.Update(tea.KeyMsg{Type: tea.KeyEsc})))
					if m.cf == nil || m.leaveAsk != leaveNone || m.nameInput.Value() != original+" changed" {
						t.Fatal("Esc on confirmation must preserve the draft")
					}
					m = exploreModel(exploreFirst(m.Update(tea.KeyMsg{Type: tea.KeyEsc})))
					m = exploreModel(exploreFirst(m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})))
				}
				if m.cf != nil || m.screen != ScreenExplore || m.exploreCursor[tabCommands] != 1 || !m.exploreListFocused() || m.nameInput.Focused() {
					t.Fatalf("width=%d field=%d dirty=%v: did not return focus to the selected row", width, field, dirty)
				}
			}
		}
	}
}

func TestEscapeClosesCompletionBeforeLeavingEditor(t *testing.T) {
	m := exploreProject()
	m = exploreModel(exploreFirst(m.Update(tea.KeyMsg{Type: tea.KeyEnter})))
	m = exploreModel(exploreFirst(m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("{")})))
	if !m.cf.pickOpen {
		t.Fatal("completion did not open")
	}
	value := m.nameInput.Value()
	m = exploreModel(exploreFirst(m.Update(tea.KeyMsg{Type: tea.KeyEsc})))
	if m.cf == nil || m.cf.pickOpen || m.leaveAsk != leaveNone || m.nameInput.Value() != value {
		t.Fatal("first Esc must only close completion")
	}
	m = exploreModel(exploreFirst(m.Update(tea.KeyMsg{Type: tea.KeyEsc})))
	if m.leaveAsk != leaveForm {
		t.Fatal("second Esc must ask about the unsaved draft")
	}
}
