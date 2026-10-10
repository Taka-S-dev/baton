package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestUnifiedEditorMenu(t *testing.T) {
	items := strings.Join(mainMenuItems(), "|")
	if strings.Contains(items, "Manage") || !strings.Contains(items, "Edit project") {
		t.Fatal(items)
	}
}

func TestNarrowWorkflowDetailsPreserveDraft(t *testing.T) {
	m := exploreProject()
	m.width = 60
	m.height = 24
	m.exploreTab = tabWorkflows
	m = createKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	m = createKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("J")})
	m = createKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	if !m.narrowDetails || !strings.Contains(stripStyle(m.View()), "Command details") {
		t.Fatal("missing narrow details")
	}
	m = createKey(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.narrowDetails || m.wfe == nil || !m.wfe.dirty {
		t.Fatal("lost draft")
	}
}

func TestCommandResizeKeepsDraft(t *testing.T) {
	m := exploreProject()
	m = createKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	m.nameInput.SetValue("echo draft")
	m = exploreModel(exploreFirst(m.Update(tea.WindowSizeMsg{Width: 50, Height: 24})))
	if m.screen != ScreenCommandForm || m.nameInput.Value() != "echo draft" {
		t.Fatal("narrow lost draft")
	}
	m = exploreModel(exploreFirst(m.Update(tea.WindowSizeMsg{Width: 110, Height: 30})))
	if m.screen != ScreenExplore || m.nameInput.Value() != "echo draft" {
		t.Fatal("wide lost draft")
	}
}
