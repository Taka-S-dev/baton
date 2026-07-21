package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	mdl "github.com/Taka-S-dev/baton/internal/model"
)

// TestMainMenu_WarningsRecomputed guards against stale diagnostics:
// warnings must describe the current in-memory state, so fixing the
// problem inside the TUI clears them on the next return to the menu.
func TestMainMenu_WarningsRecomputed(t *testing.T) {
	m := &Model{}
	m.projectDir = "x" // any non-empty project marks one as loaded
	m.workflows = []mdl.Workflow{{Name: "wf", Commands: []string{"ghost"}}}

	m.gotoMainMenu()
	if len(m.loadWarnings) == 0 {
		t.Fatal("broken workflow must produce a warning")
	}

	m.workflows = nil // the workflow was deleted in the TUI
	m.gotoMainMenu()
	if len(m.loadWarnings) != 0 {
		t.Fatalf("warnings must clear once the cause is gone, got %v", m.loadWarnings)
	}
}

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
