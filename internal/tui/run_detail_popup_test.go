package tui

import (
	"fmt"
	"strings"
	"testing"

	mdl "github.com/Taka-S-dev/baton/internal/model"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func TestRunDetailPopupClickAndDismiss(t *testing.T) {
	for _, height := range []int{16, 36} {
		m := homeTab(homeTestModel(t), '2')
		m.width = 120
		m.height = height
		m.msSelected = []int{2, 0}
		m.runQueueFocused = false
		a := m.runScrollAreas()[0]
		click := func(x, y int) {
			m = exploreModel(exploreFirst(m.Update(tea.MouseMsg{X: x, Y: y + 1, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})))
		}
		click(a.x+3, a.y)
		if !m.runQueueFocused || m.runDetailOpen {
			t.Fatal("first click must only select")
		}
		click(a.x+3, a.y)
		if !m.runDetailOpen {
			t.Fatal("second click did not open")
		}
		p := m.runDetailPopup()
		out := m.View()
		if !strings.Contains(ansi.Strip(out), "shell:") {
			t.Fatal("details missing")
		}
		for _, line := range strings.Split(out, "\n") {
			if ansi.StringWidth(line) > 120 {
				t.Fatal("popup overflow")
			}
		}
		click(p.x-1, p.y)
		if m.runDetailOpen || m.runQueueCursor != 0 || len(m.msSelected) != 2 || m.msSelected[0] != 2 {
			t.Fatal("dismiss changed queue")
		}
		m = homeKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
		if !m.runDetailOpen {
			t.Fatal("d did not open")
		}
		m = homeKey(m, tea.KeyMsg{Type: tea.KeyEsc})
		if m.runDetailOpen {
			t.Fatal("Esc did not close")
		}
	}
}

func TestDetailsAvoidSelectedRow(t *testing.T) {
	for _, width := range []int{60, 120} {
		for _, height := range []int{12, 20, 36} {
			for _, queue := range []bool{false, true} {
				for _, cursor := range []int{0, 5, 19} {
					m := Model{width: width, height: height, screen: ScreenRunCommands, runQueueFocused: queue, msCursor: cursor, runQueueCursor: cursor}
					for i := 0; i < 20; i++ {
						c := mdl.Command{Name: fmt.Sprint(i), Cmd: strings.Repeat("echo long ", 60)}
						m.msItems = append(m.msItems, msItem{cmd: &c})
						m.msSelected = append(m.msSelected, i)
					}
					target := 3
					if queue {
						target = 1
					}
					var a runScrollArea
					for _, v := range m.runScrollAreas() {
						if v.target == target {
							a = v
						}
					}
					row := a.y + cursor - a.offset
					m.runDetailOpen = true
					p := m.runDetailPopup()
					if p.x < 0 || p.y < 0 || p.x+p.w > width || p.y+p.h > m.termHeight() {
						t.Fatal("popup outside screen")
					}
					if row >= p.y && row < p.y+p.h && p.x < a.x+a.width && p.x+p.w > a.x {
						t.Fatalf("selected row covered: %dx%d queue=%v cursor=%d", width, height, queue, cursor)
					}
					before := p
					m.scrollRunDetail(5)
					if m.runDetailPopup() != before {
						t.Fatal("scroll moved popup")
					}
				}
			}
		}
	}
}
