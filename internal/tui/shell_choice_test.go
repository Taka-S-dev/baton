package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"strings"
	"testing"
)

func TestShellChoicesShowAndPreserveNavigation(t *testing.T) {
	for _, width := range []int{60, 110} {
		m := exploreProject()
		m.width = width
		m = createKey(m, tea.KeyMsg{Type: tea.KeyEnter})
		m.cfGotoField(4)
		view := stripStyle(m.View())
		if !strings.Contains(view, shellChoiceLabel("")) || !strings.Contains(view, "PowerShell (ps)") {
			t.Fatal("shell choices are not visible")
		}
		m = createKey(m, tea.KeyMsg{Type: tea.KeyDown})
		m = createKey(m, tea.KeyMsg{Type: tea.KeyEsc})
		if m.cf == nil || m.cf.fields[4] != "" || m.cf.shellOpen {
			t.Fatal("Esc must cancel pending choice")
		}
		m = createKey(m, tea.KeyMsg{Type: tea.KeyEnter})
		m = createKey(m, tea.KeyMsg{Type: tea.KeyDown})
		m = createKey(m, tea.KeyMsg{Type: tea.KeyEnter})
		if m.cf.fields[4] != "ps" || m.cf.shellOpen {
			t.Fatal("PowerShell choice was not confirmed")
		}
		m = createKey(m, tea.KeyMsg{Type: tea.KeyShiftTab})
		if m.cf.fieldIdx != 3 {
			t.Fatal("Shift+Tab must go to Group")
		}
		m = createKey(m, tea.KeyMsg{Type: tea.KeyTab})
		if !m.cf.shellOpen || m.cf.shellCursor != 1 {
			t.Fatal("reopened selector must retain PowerShell")
		}
	}
}
