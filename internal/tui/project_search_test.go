package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"strings"
	"testing"
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
