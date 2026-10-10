package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestProjectSearchInlineAndNavigation(t *testing.T) {
	m := exploreProject()
	m.screen = ScreenProjectSelect
	m.projects = []string{"sample-a", "sample-b", "other"}
	m = createKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m = createKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("sample")})
	view := stripStyle(m.viewProjects(80))
	if strings.Contains(view, "Search projects") || !strings.Contains(view, "sample-a") || !strings.Contains(view, "sample-b") || !strings.Contains(view, "Enter Done") {
		t.Fatal("search obscured results")
	}
	m = createKey(m, tea.KeyMsg{Type: tea.KeyDown})
	if m.listCursor != 1 {
		t.Fatal("cannot navigate results")
	}
	m = createKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.projectMode != "" || m.projectFilter != "sample" || m.listCursor != 1 || m.screen != ScreenProjectSelect {
		t.Fatal("Enter did not preserve filtered selection")
	}
	m = createKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m = createKey(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.projectFilter != "" || m.projectMode != "" {
		t.Fatal("Esc did not clear search")
	}
}

func TestProjectSearchClickTargetsSearchLine(t *testing.T) {
	m := exploreProject()
	m.screen = ScreenProjectSelect
	m.projects = []string{"sample"}
	view := stripStyle(m.viewProjects(80))
	lines := strings.Split(view, "\n")
	if !strings.Contains(lines[0], "BATON · Projects") || !strings.Contains(lines[2], "/ Search…") {
		t.Fatal("title and search must be separate")
	}
	for _, y := range []int{0, 1} {
		next, _ := m.Update(tea.MouseMsg{X: 5, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
		m = next.(Model)
		if m.projectMode != "" {
			t.Fatal("heading activated search")
		}
	}
	next, _ := m.Update(tea.MouseMsg{X: 5, Y: 2, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	m = next.(Model)
	if m.projectMode != "search" {
		t.Fatal("search line not clickable")
	}
}
