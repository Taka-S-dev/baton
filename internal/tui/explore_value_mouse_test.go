package tui

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestValuePaneClickBack(t *testing.T) {
	for _, tab := range []int{tabLists, tabVars} {
		for _, back := range []bool{false, true} {
			m := exploreProject()
			m.exploreTab = tab
			m = createKey(m, tea.KeyMsg{Type: tea.KeyEnter})
			l := m.exploreLayout(m.viewWidth())
			x, y := 2, l.bodyY+l.bodyH-3
			if back {
				x, y = l.rightX+4, l.editY
			}
			m = publicClick(m, x, y)
			if m.screen != ScreenExplore || !m.exploreListFocused() {
				t.Fatalf("tab %d back %v: did not return", tab, back)
			}
		}
	}
}

func TestValuePaneProtectsDraft(t *testing.T) {
	for _, tab := range []int{tabLists, tabVars} {
		m := exploreProject()
		m.exploreTab = tab
		m = createKey(m, tea.KeyMsg{Type: tea.KeyEnter})
		if tab == tabLists {
			m.le.startAdding()
			m.le.addVal = "new-value"
		} else {
			m.nameInput.SetValue("new-value")
		}
		l := m.exploreLayout(m.viewWidth())
		m = publicClick(m, 2, l.bodyY)
		if m.leaveAsk != leaveValue {
			t.Fatal("draft was not protected")
		}
		m = createKey(m, tea.KeyMsg{Type: tea.KeyEsc})
		if !m.valueDraftDirty() || !m.exploreValueFlow() {
			t.Fatal("cancel lost draft")
		}
		m = publicClick(m, 2, l.bodyY)
		m = createKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
		if m.screen != ScreenExplore {
			t.Fatal("discard did not return")
		}
	}
}

func TestListSaveAndLeave(t *testing.T) {
	for _, fail := range []bool{false, true} {
		for _, editing := range []bool{false, true} {
			m := exploreProject()
			m.projectDir = t.TempDir()
			if fail {
				if err := os.WriteFile(filepath.Join(m.projectDir, "lists"), []byte("blocked"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			m.exploreTab = tabLists
			m = createKey(m, tea.KeyMsg{Type: tea.KeyEnter})
			name := m.le.name
			old := append([]string{}, m.le.entries[0].Value)
			if editing {
				m = createKey(m, tea.KeyMsg{Type: tea.KeyEnter})
				m.le.editValTI.SetValue("new-value")
			} else {
				m.le.startAdding()
				m.le.addVal = "new-value"
			}
			l := m.exploreLayout(m.viewWidth())
			m = publicClick(m, 2, l.bodyY)
			m = createKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
			if fail {
				if !m.valueDraftDirty() || m.errMsg == "" || m.lists[name][0].Value != old[0] {
					t.Fatal("failed save lost draft or changed saved data")
				}
			} else {
				if m.screen != ScreenExplore {
					t.Fatal("save did not return")
				}
				if _, err := os.Stat(filepath.Join(m.projectDir, "lists", name+".tsv")); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
}

func TestVariableMouseSaveAndLeave(t *testing.T) {
	for _, create := range []bool{false, true} {
		for _, fail := range []bool{false, true} {
			m := exploreProject()
			m.projectDir = t.TempDir()
			m.vars = map[string]string{"sample": "unique-original"}
			if fail {
				if err := os.Mkdir(filepath.Join(m.projectDir, "vars.tsv"), 0700); err != nil {
					t.Fatal(err)
				}
			}
			m.exploreTab = tabVars
			key := tea.KeyMsg{Type: tea.KeyEnter}
			if create {
				key = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")}
			}
			m = createKey(m, key)
			name := "sample"
			if create {
				name = "newvar"
				m.nameInput.SetValue(name)
				m = createKey(m, tea.KeyMsg{Type: tea.KeyEnter})
			}
			m.nameInput.SetValue("unique-new-value")
			l := m.exploreLayout(m.viewWidth())
			m = publicClick(m, l.rightX+4, l.editY)
			if m.leaveAsk != leaveValue {
				t.Fatal("missing confirmation")
			}
			m = createKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
			if fail {
				if m.screen != ScreenVarForm || m.nameInput.Value() != "unique-new-value" || m.errMsg == "" {
					t.Fatal("failed save lost draft")
				}
				if m.vars[name] == "unique-new-value" {
					t.Fatal("failed save changed memory")
				}
			} else {
				if m.screen != ScreenExplore || m.vars[name] != "unique-new-value" {
					t.Fatal("variable did not save and return")
				}
			}
		}
	}
}

func TestNewVariableNameClickBackKeepsDraft(t *testing.T) {
	m := exploreProject()
	m.exploreTab = tabVars
	m = createKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	m.nameInput.SetValue("newvar")
	l := m.exploreLayout(m.viewWidth())
	m = publicClick(m, 2, l.bodyY)
	if m.leaveAsk != leaveValue {
		t.Fatal("name draft not protected")
	}
	m = createKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	if m.screen != ScreenVarForm || m.nameInput.Value() != "newvar" || m.errMsg == "" {
		t.Fatal("incomplete variable draft lost")
	}
	m = publicClick(m, 2, l.bodyY)
	m = createKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	if m.screen != ScreenExplore {
		t.Fatal("discard did not return")
	}
}
