package tui

import (
	"os"
	"path/filepath"
	"testing"

	mdl "github.com/Taka-S-dev/baton/internal/model"
	tea "github.com/charmbracelet/bubbletea"
)

func createKey(m Model, key tea.KeyMsg) Model { return exploreModel(exploreFirst(m.Update(key))) }

func TestExploreCreateListFromEmpty(t *testing.T) {
	m := exploreProject()
	m.projectDir = t.TempDir()
	m.lists = nil
	m.exploreTab = tabLists
	m = createKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	if m.screen != ScreenNameInput {
		t.Fatal("n did not open list creation")
	}
	m.nameInput.SetValue("env")
	m = createKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.le == nil || !m.le.adding {
		t.Fatal("first entry must be ready to type")
	}
	for _, r := range "dev" {
		m = createKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	m = createKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	m = createKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	m = createKey(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.screen != ScreenExplore || m.exploreTab != tabLists {
		t.Fatal("did not return to Lists")
	}
	it, ok := m.exploreCurrent()
	if !ok || it.name != "env" {
		t.Fatal("new list is not selected")
	}
	data, err := os.ReadFile(filepath.Join(m.projectDir, "lists", "env.tsv"))
	if err != nil || string(data) != "dev" {
		t.Fatalf("list not persisted: %q %v", data, err)
	}
}

func TestExploreCreateVariableFromEmpty(t *testing.T) {
	m := exploreProject()
	m.projectDir = t.TempDir()
	m.vars = nil
	m.lists = map[string][]mdl.ListEntry{}
	m.exploreTab = tabVars
	m.exploreSearch = "hidden"
	m = createKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	if m.screen != ScreenVarForm {
		t.Fatal("n did not open variable creation")
	}
	m.nameInput.SetValue("root")
	m = createKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	m.nameInput.SetValue("C:/work")
	m = createKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.screen != ScreenExplore || m.vars["root"] != "C:/work" {
		t.Fatal("variable was not saved and returned")
	}
	it, ok := m.exploreCurrent()
	if !ok || it.name != "root" {
		t.Fatal("new variable hidden by old search")
	}
	if _, err := os.Stat(filepath.Join(m.projectDir, "vars.tsv")); err != nil {
		t.Fatal(err)
	}
}

func TestExploreCancelEmptyCreation(t *testing.T) {
	for _, tab := range []int{tabLists, tabVars, tabWorkflows} {
		m := exploreProject()
		m.exploreTab = tab
		m = createKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
		m = createKey(m, tea.KeyMsg{Type: tea.KeyEsc})
		if m.screen != ScreenExplore || m.exploreTab != tab {
			t.Fatalf("tab %d: cancel did not return", tab)
		}
	}
}

func TestNewListFailureKeepsNameForm(t *testing.T) {
	for _, invalid := range []bool{false, true} {
		m := exploreProject()
		m.projectDir = t.TempDir()
		m.screen = ScreenNameInput
		if err := os.WriteFile(filepath.Join(m.projectDir, "lists"), []byte("blocked"), 0600); err != nil {
			t.Fatal(err)
		}
		name := "new-list"
		if invalid {
			name = "../outside"
		}
		m = exploreModel(exploreFirst(m.saveNewList(name)))
		if m.screen != ScreenNameInput || m.le != nil {
			t.Fatal("failed creation entered editor")
		}
		if _, ok := m.lists[name]; ok {
			t.Fatal("failed creation added list to memory")
		}
	}
}
