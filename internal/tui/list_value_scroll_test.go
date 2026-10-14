package tui

import (
	"strings"
	"testing"

	mdl "github.com/Taka-S-dev/baton/internal/model"
	tea "github.com/charmbracelet/bubbletea"
)

func TestListValueScrollbar(t *testing.T) {
	m := exploreProject()
	m.exploreTab = tabLists
	m.lists["env"] = make([]mdl.ListEntry, 30)
	m = createKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	send := func(x, y int, b tea.MouseButton, a tea.MouseAction) {
		m = exploreModel(exploreFirst(m.Update(tea.MouseMsg{X: x, Y: y, Button: b, Action: a})))
	}
	a := m.listValueScrollArea()
	if !strings.Contains(stripStyle(m.exploreValueBox(m.exploreLayout(m.viewWidth()))), "┃") {
		t.Fatal("missing scrollbar")
	}
	send(a.x+2, a.y, tea.MouseButtonWheelDown, tea.MouseActionPress)
	if m.le.scrollOffset != 3 || m.le.cursor != 0 {
		t.Fatal("wheel moved selection")
	}
	a = m.listValueScrollArea()
	top, _ := scrollbarThumb(a.total, a.rows, a.offset)
	send(a.x+a.width-1, a.y+top, tea.MouseButtonLeft, tea.MouseActionPress)
	send(a.x+a.width-1, a.y+a.rows-1, tea.MouseButtonLeft, tea.MouseActionMotion)
	send(a.x+a.width-1, a.y+a.rows-1, tea.MouseButtonLeft, tea.MouseActionRelease)
	if m.le.scrollOffset != a.total-a.rows || m.le.cursor != 0 {
		t.Fatal("drag failed")
	}
	a = m.listValueScrollArea()
	send(a.x+2, a.y, tea.MouseButtonLeft, tea.MouseActionPress)
	if m.le.cursor != a.offset || m.le.scrollOffset != a.offset {
		t.Fatal("click jumped")
	}
}
