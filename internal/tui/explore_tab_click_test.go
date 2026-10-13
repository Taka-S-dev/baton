package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTabClickLeavesCommandEditor(t *testing.T) {
	for _, dirty := range []bool{false, true} {
		for _, decision := range []string{"k", "d"} {
			m := exploreProject()
			m = createKey(m, tea.KeyMsg{Type: tea.KeyEnter})
			if dirty {
				m.nameInput.SetValue("echo unsaved")
			}
			l := m.exploreLayout(m.viewWidth())
			m = exploreModel(exploreFirst(m.Update(tea.MouseMsg{X: l.tabX[tabLists][0], Y: l.tabY, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})))
			if !dirty {
				if m.exploreTab != tabLists || m.cf != nil {
					t.Fatal("clean form did not switch directly")
				}
				continue
			}
			if m.leaveAsk != leaveForm || m.exploreTab != tabCommands {
				t.Fatal("missing unsaved confirmation")
			}
			m = createKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(decision)})
			if decision == "k" {
				if m.cf == nil || m.nameInput.Value() != "echo unsaved" || m.exploreTab != tabCommands || m.explorePendingTab != 0 {
					t.Fatal("keep editing lost state")
				}
			} else if m.cf != nil || m.exploreTab != tabLists {
				t.Fatal("discard did not complete requested switch")
			}
		}
	}
}
