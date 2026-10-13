package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestDividerResizeAndResetKeepsForm(t *testing.T) {
	m := exploreProject()
	m = createKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	m.nameInput.SetValue("echo unsaved")
	before := m.exploreCursor
	l := m.exploreLayout(m.viewWidth())
	initial := l.leftW
	send := func(x, y int, b tea.MouseButton, a tea.MouseAction) {
		m = exploreModel(exploreFirst(m.Update(tea.MouseMsg{X: x, Y: y, Button: b, Action: a})))
	}
	if !strings.Contains(stripStyle(m.exploreDivider(l)), "↔") {
		t.Fatal("missing resize affordance")
	}
	send(l.leftW, l.bodyY+2, tea.MouseButtonLeft, tea.MouseActionPress)
	send(55, l.bodyY+2, tea.MouseButtonLeft, tea.MouseActionMotion)
	if !m.exploreDragging || m.exploreLayout(m.viewWidth()).leftW != 55 {
		t.Fatal("drag failed")
	}
	send(55, l.bodyY+2, tea.MouseButtonLeft, tea.MouseActionRelease)
	if m.exploreDragging {
		t.Fatal("release failed")
	}
	send(55, l.bodyY+2, tea.MouseButtonRight, tea.MouseActionPress)
	if m.exploreLayout(m.viewWidth()).leftW != initial || m.exploreSplit != 0 {
		t.Fatal("reset failed")
	}
	if m.cf == nil || m.nameInput.Value() != "echo unsaved" || m.exploreCursor != before || m.leaveAsk != leaveNone {
		t.Fatal("resize disturbed editor")
	}
	send(initial, 0, tea.MouseButtonLeft, tea.MouseActionPress)
	if m.exploreDragging {
		t.Fatal("heading started drag")
	}
}
