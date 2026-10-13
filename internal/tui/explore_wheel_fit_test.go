package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestExploreWheelDoesNotMoveFittingList(t *testing.T) {
	for tab := 0; tab < 4; tab++ {
		m := exploreProject()
		m.width = 120
		m.height = 60
		m.exploreTab = tab
		l := m.exploreLayout(m.viewWidth())
		if len(l.items) > l.listH {
			t.Fatal("fixture overflows")
		}
		for _, button := range []tea.MouseButton{tea.MouseButtonWheelDown, tea.MouseButtonWheelUp} {
			for _, x := range []int{2, l.rightX + 2} {
				got := mouseAt(m, x, l.listY, tea.MouseActionPress, button)
				if got.exploreCursor != m.exploreCursor {
					t.Fatal("fitting list jumped")
				}
			}
		}
	}
}
