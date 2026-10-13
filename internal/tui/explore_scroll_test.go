package tui

import (
	"fmt"
	"testing"

	mdl "github.com/Taka-S-dev/baton/internal/model"
	tea "github.com/charmbracelet/bubbletea"
)

func TestEditorScrollAllTabs(t *testing.T) {
	for tab := 0; tab < tabCount; tab++ {
		m := exploreProject()
		m.exploreTab = tab
		for i := 0; i < 70; i++ {
			name := fmt.Sprintf("item%02d", i)
			m.config.Base = append(m.config.Base, mdl.Command{Name: name, Cmd: "echo hi"})
			m.workflows = append(m.workflows, mdl.Workflow{Name: name})
			m.lists[name] = nil
			m.vars[name] = "value"
		}
		send := func(x, y int, b tea.MouseButton, a tea.MouseAction) {
			m = exploreModel(exploreFirst(m.Update(tea.MouseMsg{X: x, Y: y, Button: b, Action: a})))
		}
		l := m.exploreLayout(m.viewWidth())
		selected := m.exploreCursor
		send(2, l.listY, tea.MouseButtonWheelDown, tea.MouseActionPress)
		l = m.exploreLayout(m.viewWidth())
		if l.start != 3 || m.exploreCursor != selected {
			t.Fatal("wheel changed selection")
		}
		a := m.runScrollAreas()[0]
		top, _ := scrollbarThumb(a.total, a.rows, a.offset)
		send(a.x+a.width-1, a.y+top, tea.MouseButtonLeft, tea.MouseActionPress)
		send(a.x+a.width-1, a.y+a.rows+20, tea.MouseButtonLeft, tea.MouseActionMotion)
		send(a.x+a.width-1, a.y+a.rows+20, tea.MouseButtonLeft, tea.MouseActionRelease)
		l = m.exploreLayout(m.viewWidth())
		if l.start != len(l.items)-l.listH || m.exploreCursor != selected {
			t.Fatal("drag changed selection or missed end")
		}
		send(3, l.listY, tea.MouseButtonLeft, tea.MouseActionPress)
		if m.exploreCursor[tab] != l.start {
			t.Fatal("clicked wrong scrolled item")
		}
	}
}
func TestEditorScrollKeepsUnsavedForm(t *testing.T) {
	m := exploreProject()
	for i := 0; i < 70; i++ {
		m.config.Base = append(m.config.Base, mdl.Command{Name: fmt.Sprint(i), Cmd: "echo hi"})
	}
	m = createKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	m.nameInput.SetValue("echo unsaved")
	form := m.cf
	cursor := m.exploreCursor
	l := m.exploreLayout(m.viewWidth())
	m = exploreModel(exploreFirst(m.Update(tea.MouseMsg{X: 2, Y: l.listY, Button: tea.MouseButtonWheelDown, Action: tea.MouseActionPress})))
	if m.cf != form || m.nameInput.Value() != "echo unsaved" || m.exploreCursor != cursor || m.leaveAsk != leaveNone || m.exploreLayout(m.viewWidth()).start == 0 {
		t.Fatal("scroll disturbed editor")
	}
}
