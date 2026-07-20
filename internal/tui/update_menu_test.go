package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestMainMenu_EveryItemResponds guards against the menu definition and
// the Enter dispatch drifting apart: every rendered item must either
// leave the main menu or (for Exit) return a quit command, and every
// item must have a right-pane description.
func TestMainMenu_EveryItemResponds(t *testing.T) {
	for i, item := range mainMenuItems() {
		if _, ok := menuItemInfos[item]; !ok {
			t.Errorf("item %q has no menuItemInfos entry", item)
		}

		m := &Model{}
		m.gotoMainMenu()
		m.listCursor = i
		nm, cmd := m.updateMainMenu(tea.KeyMsg{Type: tea.KeyEnter})
		got := nm.(Model)

		if item == "Exit" {
			if cmd == nil {
				t.Errorf("item %q: Enter must return a quit command", item)
			}
			continue
		}
		if got.screen == ScreenMainMenu {
			t.Errorf("item %q: Enter did not leave the main menu", item)
		}
	}
}
