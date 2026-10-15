package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func TestWorkspaceHeaderNavigation(t *testing.T) {
	for _, height := range []int{10, 19, 20, 36} {
		for _, width := range []int{40, 80, 140} {
			m := homeTab(homeTestModel(t), '2')
			m.width, m.height = width, height
			tabRow := m.workspaceTabRow()
			run := strings.Split(ansi.Strip(m.View()), "\n")
			x := strings.Index(run[0], runProjectEditLabel)
			if x < 0 || !strings.HasPrefix(run[0], "  Run") || !strings.Contains(run[tabRow], "[2 Commands]") {
				t.Fatalf("invalid run header: %q", run[:3])
			}
			m = exploreModel(exploreFirst(m.Update(tea.MouseMsg{X: x + 1, Y: 0, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})))
			if m.screen != ScreenExplore {
				t.Fatal("header edit click failed")
			}
			edit := strings.Split(ansi.Strip(m.View()), "\n")
			if !strings.HasPrefix(edit[0], "  Edit project") || (tabRow == 2 && edit[1] != run[1]) || (!strings.Contains(edit[tabRow], "Commands") && !strings.Contains(edit[tabRow], "Cmds")) {
				t.Fatalf("inconsistent editor header: %q", edit[:3])
			}
			for _, line := range edit[:3] {
				if ansi.StringWidth(line) > width {
					t.Fatal("header overflows")
				}
			}
			m = clickExploreBack(t, m)
			if m.screen != ScreenRunCommands {
				t.Fatal("header back click failed")
			}
		}
	}
}

func TestShortWorkspaceReclaimsHintRows(t *testing.T) {
	m := homeTab(homeTestModel(t), '1')
	m.width, m.height = 120, 15
	if strings.Contains(ansi.Strip(m.View()), "Enter Review") {
		t.Fatal("short workflow still shows navigation hints")
	}
	rows := m.workflowScrollAreas()[0].rows
	m.runHome.searching = true
	if m.workflowScrollAreas()[0].rows != rows-1 {
		t.Fatal("hidden hints did not reclaim a workflow row")
	}
	m.runHome.searching = false
	m = exploreModel(exploreFirst(m.editFromRunHome()))
	l := m.exploreLayout(m.width)
	if strings.Contains(ansi.Strip(m.View()), "Enter Open") {
		t.Fatal("short editor still shows navigation hints")
	}
	m.exploreSearching = true
	if m.hideWorkspaceHints() || m.exploreLayout(m.width).bodyH != l.bodyH-2 {
		t.Fatal("search hints must reserve their own rows")
	}
}
