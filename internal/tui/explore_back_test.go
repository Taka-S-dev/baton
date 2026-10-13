package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func clickExploreBack(t *testing.T, m Model) Model {
	t.Helper()
	lines := strings.Split(ansi.Strip(m.View()), "\n")
	for y, line := range lines {
		if x := strings.Index(line, exploreBackLabel); x >= 0 {
			next, _ := m.Update(tea.MouseMsg{X: ansi.StringWidth(line[:x]) + 2, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
			return exploreModel(next)
		}
	}
	t.Fatal("back button missing")
	return m
}

func TestExploreBackReturnsToRun(t *testing.T) {
	for _, height := range []int{10, 36} {
		m := homeTab(homeTestModel(t), '2')
		m.width = 120
		m.height = height
		m.msSelected = []int{2, 0}
		m.msSearchTI.SetValue("build")
		m = homeKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
		m = clickExploreBack(t, m)
		if m.screen != ScreenRunCommands || m.msSearchTI.Value() != "build" || len(m.msSelected) != 2 {
			t.Fatal("run state lost")
		}
	}
}

func TestExploreBackConfirmsUnsavedForm(t *testing.T) {
	m := exploreProject()
	m.width = 120
	m = createKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	m.nameInput.SetValue("echo unsaved")
	m = clickExploreBack(t, m)
	if m.leaveAsk != leaveForm || m.cf == nil || m.nameInput.Value() != "echo unsaved" {
		t.Fatal("unsaved edits bypassed")
	}
}
