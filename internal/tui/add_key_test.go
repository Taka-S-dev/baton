package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"strings"
	"testing"
)

func TestAddKeyAndLegacyAlias(t *testing.T) {
	for tab := 0; tab < tabCount; tab++ {
		for _, key := range []string{"a", "n"} {
			m := exploreProject()
			m.exploreTab = tab
			m = createKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
			switch tab {
			case tabCommands:
				if m.cf == nil {
					t.Fatal("command not opened")
				}
			case tabWorkflows:
				if m.screen != ScreenCreateWorkflow {
					t.Fatal("workflow not opened")
				}
			case tabLists:
				if m.screen != ScreenNameInput {
					t.Fatal("list not opened")
				}
			case tabVars:
				if m.screen != ScreenVarForm {
					t.Fatal("variable not opened")
				}
			}
		}
	}
}

func TestAddKeyBelongsToFocusedPane(t *testing.T) {
	m := exploreProject()
	m.exploreTab = tabWorkflows
	m.wfe = &workflowEditState{idx: 0, steps: []string{"build"}}
	m = createKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	if !m.wfe.adding || m.screen != ScreenExplore {
		t.Fatal("wrong add target")
	}
	m = createKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	if m.wfe.addSearch != "a" {
		t.Fatal("search swallowed a")
	}
	m = exploreProject()
	m.exploreTab = tabVars
	m = createKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	m = createKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	if m.nameInput.Value() != "a" {
		t.Fatal("input swallowed a")
	}
}

func TestInactiveAddButtonHidesShortcut(t *testing.T) {
	m := exploreProject()
	l := m.exploreLayout(m.viewWidth())
	line := strings.Split(stripStyle(m.exploreListPane(l)), "\n")[l.bodyH-2]
	if !strings.Contains(line, "command  a") {
		t.Fatal("active button missing a")
	}
	m = createKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	l = m.exploreLayout(m.viewWidth())
	line = strings.Split(stripStyle(m.exploreListPane(l)), "\n")[l.bodyH-2]
	if !strings.Contains(line, "+ Add command") || strings.Contains(line, "command  a") {
		t.Fatal("inactive button advertises key")
	}
}
