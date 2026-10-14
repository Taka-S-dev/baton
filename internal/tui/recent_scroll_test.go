package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Taka-S-dev/baton/internal/store"
	tea "github.com/charmbracelet/bubbletea"
)

func TestRecentScrollbarAndWheel(t *testing.T) {
	for _, height := range []int{4, 10, 30} {
		m := Model{screen: ScreenRecent, width: 100, height: height}
		for i := 0; i < 20; i++ {
			m.recentRows = append(m.recentRows, recentRow{ref: store.RecentEntry{Command: fmt.Sprint(i)}})
		}
		p := m.recentGeometry(m.viewWidth())
		if !strings.Contains(stripStyle(m.viewRecent(m.viewWidth())), "┃") {
			t.Fatal("missing overflow scrollbar")
		}
		nm, _ := m.updateRecentMouse(tea.MouseMsg{X: p.x + 10, Y: p.listY, Button: tea.MouseButtonWheelDown})
		m = nm.(Model)
		if m.recentCursor != 0 || m.recentGeometry(m.viewWidth()).start != min(3, p.rows) {
			t.Fatal("wheel moved selection or jumped")
		}
		// A track click must never launch the row underneath it.
		nm, cmd := m.updateRecentMouse(tea.MouseMsg{X: p.x + p.width - 1, Y: p.listY + p.rows - 1, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
		m = nm.(Model)
		if cmd != nil || m.screen != ScreenRecent {
			t.Fatal("scrollbar click launched command")
		}
		m.recentRows = m.recentRows[:1]
		m.recentOffset = 0
		nm, _ = m.updateRecentMouse(tea.MouseMsg{X: p.x + 10, Y: p.listY, Button: tea.MouseButtonWheelDown})
		m = nm.(Model)
		if m.recentOffset != 0 || m.recentCursor != 0 || strings.Contains(stripStyle(m.viewRecent(m.viewWidth())), "┃") {
			t.Fatal("non-overflow list scrolled")
		}
	}
}
