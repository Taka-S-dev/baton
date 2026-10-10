package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestFixedCreateButton(t *testing.T) {
	for tab := 0; tab < tabCount; tab++ {
		for _, search := range []string{"", "no-such-item-123"} {
			m := exploreProject()
			m.exploreTab = tab
			m.exploreSearch = search
			m.exploreSearching = search != ""
			l := m.exploreLayout(m.viewWidth())
			lines := strings.Split(stripStyle(m.viewExplore(m.viewWidth())), "\n")
			if l.addY < 0 || !strings.Contains(lines[l.addY], "+ Add ") {
				t.Fatalf("tab %d missing fixed button", tab)
			}
			m = publicClick(m, 3, l.addY)
			switch tab {
			case tabCommands:
				if m.cf == nil {
					t.Fatal("command button failed")
				}
			case tabWorkflows:
				if m.screen != ScreenCreateWorkflow {
					t.Fatal("workflow button failed")
				}
			case tabLists:
				if m.screen != ScreenNameInput {
					t.Fatal("list button failed")
				}
			case tabVars:
				if m.screen != ScreenVarForm {
					t.Fatal("variable button failed")
				}
			}
		}
	}
}

func TestCreateButtonStaysFixedAndProtectsDraft(t *testing.T) {
	m := exploreProject()
	l := m.exploreLayout(m.viewWidth())
	m.exploreCursor[tabCommands] = len(l.items) - 1
	if m.exploreLayout(m.viewWidth()).addY != l.addY {
		t.Fatal("button moved")
	}
	m = createKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	m.nameInput.SetValue("draft")
	l = m.exploreLayout(m.viewWidth())
	m = publicClick(m, 3, l.addY)
	if m.leaveAsk != leaveForm || m.nameInput.Value() != "draft" {
		t.Fatal("button lost draft")
	}
}
