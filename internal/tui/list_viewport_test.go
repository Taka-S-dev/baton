package tui

import (
	"fmt"
	"testing"

	mdl "github.com/Taka-S-dev/baton/internal/model"
	tea "github.com/charmbracelet/bubbletea"
)

func TestEditorClickKeepsViewport(t *testing.T) {
	m := exploreProject()
	for i := 0; i < 70; i++ {
		m.config.Base = append(m.config.Base, mdl.Command{Name: fmt.Sprint(i), Cmd: "echo hi"})
	}
	m.exploreScrollManual[tabCommands] = true
	m.exploreScroll[tabCommands] = 10
	l := m.exploreLayout(m.viewWidth())
	index := l.start + l.listH - 2
	m = exploreModel(exploreFirst(m.Update(tea.MouseMsg{X: 3, Y: l.listY + l.listH - 2, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})))
	if m.exploreCursor[tabCommands] != index || m.exploreLayout(m.viewWidth()).start != 10 {
		t.Fatal("click recentered")
	}
	m = homeKey(m, tea.KeyMsg{Type: tea.KeyDown})
	if m.exploreLayout(m.viewWidth()).start != 10 {
		t.Fatal("visible cursor scrolled")
	}
	m = homeKey(m, tea.KeyMsg{Type: tea.KeyDown})
	if m.exploreLayout(m.viewWidth()).start != 11 {
		t.Fatal("edge did not scroll by one")
	}
}

func TestRevealRow(t *testing.T) {
	for _, v := range [][2]int{{10, 10}, {14, 10}, {15, 11}, {9, 9}} {
		if got := revealRow(10, 5, 50, v[0]); got != v[1] {
			t.Fatalf("cursor %d: %d", v[0], got)
		}
	}
}
