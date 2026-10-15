package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func TestRunDividerBothTabs(t *testing.T) {
	for _, tab := range []rune{'1', '2'} {
		for _, height := range []int{16, 36} {
			m := homeTab(homeTestModel(t), tab)
			m.width = 140
			m.height = height
			m.msSelected = []int{2, 0}
			initial := m.runLeftWidth(140)
			send := func(x, y int, b tea.MouseButton, a tea.MouseAction) {
				m = exploreModel(exploreFirst(m.Update(tea.MouseMsg{X: x, Y: y + m.runHeaderRows(), Button: b, Action: a})))
			}
			send(initial, 3, tea.MouseButtonLeft, tea.MouseActionPress)
			send(80, 3, tea.MouseButtonLeft, tea.MouseActionMotion)
			send(80, 3, tea.MouseButtonLeft, tea.MouseActionRelease)
			if m.runLeftWidth(140) != 80 || m.runPaneDragging {
				t.Fatal("resize failed")
			}
			for _, line := range strings.Split(m.View(), "\n") {
				if ansi.StringWidth(line) > 140 {
					t.Fatal("overflow")
				}
			}
			if tab == '2' {
				a := m.candidateScrollArea()
				if a.width != 80 {
					t.Fatal("candidate hit area stale")
				}
				if len(m.msSelected) != 2 || m.msSelected[0] != 2 {
					t.Fatal("selection changed")
				}
			}
			send(80, 3, tea.MouseButtonRight, tea.MouseActionPress)
			if m.runLeftWidth(140) != initial {
				t.Fatal("reset failed")
			}
		}
	}
}
