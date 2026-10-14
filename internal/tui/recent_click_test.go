package tui

import (
	"testing"

	"github.com/Taka-S-dev/baton/internal/store"
	tea "github.com/charmbracelet/bubbletea"
)

func TestRecentClickSelectsWithoutOpening(t *testing.T) {
	for _, height := range []int{8, 30} {
		m := Model{screen: ScreenRecent, width: 100, height: height, recentRows: []recentRow{
			{ref: store.RecentEntry{Project: "demo", Command: "build"}},
			{ref: store.RecentEntry{Project: "demo", Command: "test"}},
		}}
		p := m.recentGeometry(m.viewWidth())
		click := func(x int) {
			t.Helper()
			next, cmd := m.updateRecentMouse(tea.MouseMsg{X: x, Y: p.listY + 1, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
			m = next.(Model)
			if cmd != nil || m.screen != ScreenRecent || m.resolve != nil || m.errMsg != "" || m.recentCursor != 1 {
				t.Fatal("row click opened a command instead of focusing it")
			}
		}
		click(p.x + 9)
		click(p.x + 9)
		click(p.x)
		if len(m.recentQueue) != 0 {
			t.Fatal("row body or cursor marker toggled queue")
		}
		click(p.x + 3)
		if len(m.recentQueue) != 1 || m.recentQueue[0].Command != "test" {
			t.Fatal("checkbox did not queue clicked row")
		}
		click(p.x + 3)
		if len(m.recentQueue) != 0 {
			t.Fatal("checkbox did not remove row")
		}
	}
}
