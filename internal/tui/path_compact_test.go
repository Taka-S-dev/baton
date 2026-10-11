package tui

import (
	"sort"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNaturalPathOrder(t *testing.T) {
	names := []string{"v29.exe", "v3.exe", "v100.exe", "v4.exe"}
	sort.Slice(names, func(i, j int) bool { return naturalPathLess(names[i], names[j]) })
	if strings.Join(names, ",") != "v3.exe,v4.exe,v29.exe,v100.exe" {
		t.Fatal(names)
	}
}

func TestCompletionFitsShortScreen(t *testing.T) {
	for _, height := range []int{7, 10, 14, 17} {
		m := multiCompletion(t)
		m.width = 80
		m.height = height
		m, read := completionKey(m, tea.KeyMsg{Type: tea.KeyTab})
		m = completionResult(m, read)
		view := stripStyle(m.View())
		if frameRows(view) > height {
			t.Fatalf("height %d: %s", height, view)
		}
		_, anchor := m.pathCompletionAnchor(80)
		p := m.pathCompletionLayout(80)
		if p.y < 0 || p.y+p.h > height || (anchor-1 >= p.y && anchor-1 < p.y+p.h) {
			t.Fatalf("input covered: %+v anchor %d", p, anchor)
		}
	}
}
