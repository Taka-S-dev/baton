package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestProjectWheelDoesNotSelectPointerRow(t *testing.T) {
	m := exploreProject()
	m.screen = ScreenProjectSelect
	m.projects = []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	y, _ := m.projectRows(m.viewWidth())
	send := func(row int, button tea.MouseButton) {
		t.Helper()
		next, _ := m.Update(tea.MouseMsg{X: 5, Y: row, Button: button, Action: tea.MouseActionPress})
		m = next.(Model)
	}
	send(y+5, tea.MouseButtonWheelDown)
	if m.listCursor != 1 {
		t.Fatalf("wheel jumped to pointer: %d", m.listCursor)
	}
	send(y, tea.MouseButtonWheelDown)
	if m.listCursor != 2 {
		t.Fatal("wheel depends on pointer position")
	}
	send(y+5, tea.MouseButtonWheelUp)
	if m.listCursor != 1 {
		t.Fatal("wheel up failed")
	}
	m.listCursor = 0
	send(y+5, tea.MouseButtonWheelUp)
	if m.listCursor != 0 {
		t.Fatal("wrapped at start")
	}
	m.listCursor = 7
	send(y, tea.MouseButtonWheelDown)
	if m.listCursor != 7 {
		t.Fatal("wrapped at end")
	}
	send(y+3, tea.MouseButtonLeft)
	if m.listCursor != 3 {
		t.Fatal("click no longer selects row")
	}
	m.projectMode = "actions"
	send(y, tea.MouseButtonWheelDown)
	if m.listCursor != 3 {
		t.Fatal("wheel moved behind popup")
	}
	m.projectMode = "search"
	m.projectFilter = "a"
	m.listCursor = 0
	send(y+5, tea.MouseButtonWheelDown)
	if m.listCursor != 0 || m.projectFilter != "a" || m.projectMode != "search" {
		t.Fatal("search changed")
	}
	m.projects = nil
	send(y, tea.MouseButtonWheelDown)
	if m.listCursor != 0 {
		t.Fatal("empty list cursor invalid")
	}
}
