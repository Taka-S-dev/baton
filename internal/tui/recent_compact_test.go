package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Taka-S-dev/baton/internal/store"
	"github.com/charmbracelet/x/ansi"
)

func TestRecentCompactFitsAndKeepsFocusedRowVisible(t *testing.T) {
	for _, width := range []int{24, 40, 80, 120} {
		for height := 5; height < 18; height++ {
			for cursor := 0; cursor <= 20; cursor++ {
				m := Model{width: width, height: height, recentCursor: cursor}
				for i := 0; i < 20; i++ {
					m.recentRows = append(m.recentRows, recentRow{ref: store.RecentEntry{Command: fmt.Sprintf("command-%02d", i)}})
				}
				p := m.recentGeometry(width)
				view := stripStyle(m.viewRecent(width))
				lines := strings.Split(view, "\n")
				if len(lines) > height {
					t.Fatalf("%dx%d: got %d lines", width, height, len(lines))
				}
				if strings.Contains(view, "Enter Select") || (!splitPanels(width) && strings.Contains(view, "│")) {
					t.Fatal(view)
				}
				if !strings.Contains(lines[p.browseY], "Browse") {
					t.Fatal(view)
				}
				if cursor < 20 && !strings.Contains(view, fmt.Sprintf("▶ command-%02d", cursor)) {
					t.Fatal(view)
				}
				if cursor == 20 && !strings.Contains(lines[p.browseY], "▶") {
					t.Fatal(view)
				}
				for _, line := range lines {
					if ansi.StringWidth(line) > width {
						t.Fatal(view)
					}
				}
			}
		}
	}
}
func TestRecentCompactEmptyAndTiny(t *testing.T) {
	for height := 1; height < 18; height++ {
		m := Model{width: 40, height: height, recentSearch: "missing"}
		lines := strings.Split(stripStyle(m.viewRecent(40)), "\n")
		if len(lines) > height {
			t.Fatal(lines)
		}
	}
}
