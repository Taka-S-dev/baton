package tui

import (
	"fmt"
	"strings"
	"testing"

	mdl "github.com/Taka-S-dev/baton/internal/model"
	tea "github.com/charmbracelet/bubbletea"
)

func TestWorkflowPaneScrolling(t *testing.T) {
	for _, width := range []int{60, 120} {
		m := Model{width: width, height: 30, screen: ScreenRunWorkflow}
		m.runHome.ready = true
		for i := 0; i < 60; i++ {
			m.workflows = append(m.workflows, mdl.Workflow{Name: fmt.Sprintf("workflow-%d", i), Commands: strings.Fields(strings.Repeat("build ", 70))})
		}
		send := func(x, y int, b tea.MouseButton, a tea.MouseAction) {
			next, _ := m.Update(tea.MouseMsg{X: x, Y: y + 1, Button: b, Action: a})
			m = next.(Model)
		}
		for _, area := range m.workflowScrollAreas() {
			send(area.x, area.y, tea.MouseButtonWheelDown, tea.MouseActionPress)
			areas := m.workflowScrollAreas()
			var a runScrollArea
			for _, v := range areas {
				if v.target == area.target {
					a = v
				}
			}
			if a.offset != 3 || m.listCursor != 0 {
				t.Fatal("wheel changed workflow or failed")
			}
			top, _ := scrollbarThumb(a.total, a.rows, a.offset)
			send(a.x+a.width-1, a.y+top, tea.MouseButtonLeft, tea.MouseActionPress)
			send(a.x+a.width-1, a.y+a.rows+10, tea.MouseButtonLeft, tea.MouseActionMotion)
			send(a.x+a.width-1, a.y+a.rows+10, tea.MouseButtonLeft, tea.MouseActionRelease)
			for _, v := range m.workflowScrollAreas() {
				if v.target == a.target && v.offset != v.total-v.rows {
					t.Fatal("drag failed")
				}
			}
			if m.listCursor != 0 || len(m.workflows[0].Commands) != 70 {
				t.Fatal("scroll mutated workflow")
			}
		}
		a := m.workflowScrollAreas()[0]
		send(a.x, a.y, tea.MouseButtonLeft, tea.MouseActionPress)
		if m.listCursor != a.offset {
			t.Fatal("clicked wrong scrolled workflow")
		}
		m.workflows = m.workflows[:1]
		m.listCursor = 0
		m.wfScrollManual = false
		m.workflows[0].Commands = nil
		for _, a := range m.workflowScrollAreas() {
			send(a.x, a.y, tea.MouseButtonWheelDown, tea.MouseActionPress)
		}
		for _, a := range m.workflowScrollAreas() {
			if a.offset != 0 {
				t.Fatal("fitting content moved")
			}
		}
	}
}
