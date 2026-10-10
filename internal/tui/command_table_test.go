package tui

import (
	"github.com/charmbracelet/lipgloss"
	"strings"
	"testing"
)

func TestCommandTableExplainsTypes(t *testing.T) {
	m := exploreProject()
	m.exploreSplit = 90
	m.width = 200
	view := stripStyle(m.View())
	for _, label := range []string{"Name", "Group", "Type", "Command", "From: build", "Template", "Direct"} {
		if !strings.Contains(view, label) {
			t.Fatalf("missing %s", label)
		}
	}
	for _, old := range []string{"(build)", "[make]", "{...}"} {
		if strings.Contains(view, old) {
			t.Fatalf("ambiguous label remains: %s", old)
		}
	}
	for i, it := range m.exploreItems(tabCommands) {
		if it.name == "build-api" {
			m.exploreCursor[tabCommands] = i
		}
	}
	m.config.Commands[0].Values = map[string]string{"workdir": "./src333444"}
	view = stripStyle(m.View())
	for _, label := range []string{"Type > From template", "Template > build", "Fixed > workdir = ./src333444"} {
		if !strings.Contains(view, label) {
			t.Fatalf("missing detail %s", label)
		}
	}
}

func TestCommandColumnsFitAndSearchRemainsAvailable(t *testing.T) {
	it := wsItem{name: "長い名前のコマンド", group: "group", commandType: "From: long-template", note: "$ echo hello"}
	for w := 10; w < 100; w++ {
		if lipgloss.Width(commandTableLine(it, w)) > w {
			t.Fatalf("row exceeds %d", w)
		}
	}
	m := exploreProject()
	l := m.exploreLayout(m.viewWidth())
	m = clickAt(m, 4, l.bodyY+1)
	if !m.exploreSearching {
		t.Fatal("search hit area moved onto column header")
	}
	m.exploreSearch = "make"
	if len(m.exploreVisible()) != 3 {
		t.Fatal("hidden group no longer searchable")
	}
}
