package tui

import (
	"strings"
	"testing"

	mdl "github.com/Taka-S-dev/baton/internal/model"
	tea "github.com/charmbracelet/bubbletea"
)

func TestRunMouseScrolling(t *testing.T) {
	for _, size := range [][2]int{{120, 36}, {60, 36}, {120, 16}} {
		m := Model{width: size[0], height: size[1], screen: ScreenRunCommands, runQueueFocused: true}
		m.runHome.ready = true
		c := mdl.Command{Name: "long", Cmd: strings.Repeat("abcdefgh ", 3000)}
		for i := 0; i < 40; i++ {
			m.msItems = append(m.msItems, msItem{cmd: &c})
			m.msSelected = append(m.msSelected, i)
		}
		send := func(x, y int, b tea.MouseButton, a tea.MouseAction) {
			t.Helper()
			next, _ := m.Update(tea.MouseMsg{X: x, Y: y + m.runHeaderRows(), Button: b, Action: a})
			m = next.(Model)
		}
		area := m.runScrollAreas()[0]
		send(area.x+1, area.y, tea.MouseButtonWheelDown, tea.MouseActionPress)
		if m.runQueueCursor != 3 {
			t.Fatalf("%v: queue wheel ignored", size)
		}
		send(area.x+1, area.y, tea.MouseButtonWheelUp, tea.MouseActionPress)
		if m.runQueueCursor != 0 {
			t.Fatal("wheel up ignored")
		}
		for _, expanded := range []bool{false, true} {
			m.runDetailOpen = expanded
			for _, area = range m.runScrollAreas() {
				if area.total <= area.rows {
					continue
				}
				x := area.x + area.width - 1
				top, _ := scrollbarThumb(area.total, area.rows, area.offset)
				send(x, area.y+top, tea.MouseButtonLeft, tea.MouseActionPress)
				if m.runScrollDrag != area.target {
					t.Fatal("thumb not captured", area)
				}
				send(x, area.y+area.rows+20, tea.MouseButtonLeft, tea.MouseActionMotion)
				current := m.runScrollAreas()
				var after runScrollArea
				for _, v := range current {
					if v.target == area.target {
						after = v
					}
				}
				if after.offset != after.total-after.rows {
					t.Fatalf("drag failed: %+v", after)
				}
				send(x, area.y+area.rows+20, tea.MouseButtonLeft, tea.MouseActionRelease)
				if m.runScrollDrag != 0 {
					t.Fatal("drag not released")
				}
				send(x, area.y, tea.MouseButtonLeft, tea.MouseActionPress)
				var paged runScrollArea
				for _, v := range m.runScrollAreas() {
					if v.target == area.target {
						paged = v
					}
				}
				if paged.offset >= after.offset {
					t.Fatal("track click did not page")
				}
				if area.target == 2 {
					before := m.runDetailOffset
					send(area.x+1, area.y, tea.MouseButtonWheelUp, tea.MouseActionPress)
					if m.runDetailOffset >= before {
						t.Fatal("detail wheel ignored")
					}
				}
			}
		}
	}
}
