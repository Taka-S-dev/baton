package tui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestCompactTabsAndFooterFitAndRemainClickable(t *testing.T) {
	for _, width := range []int{24, 32, 40, 48, 60, 80, 110} {
		for tab := 0; tab < tabCount; tab++ {
			m := exploreProject()
			m.width = width
			m.exploreTab = tab
			if lipgloss.Width(m.exploreTabLine()) > width {
				t.Fatalf("tabs overflow at %d", width)
			}
			if lipgloss.Width(m.compactFooter(width)) > width {
				t.Fatalf("footer overflow at %d", width)
			}
			l := m.exploreLayout(width)
			for target, span := range l.tabX {
				if span[1] > width {
					t.Fatalf("tab %d outside %d", target, width)
				}
				clicked := clickAt(m, span[0], l.tabY)
				if clicked.exploreTab != target {
					t.Fatalf("click at width %d selected %d not %d", width, clicked.exploreTab, target)
				}
			}
		}
	}
}
