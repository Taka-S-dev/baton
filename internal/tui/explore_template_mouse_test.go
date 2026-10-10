package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func templateMenuForMouse() Model {
	m := exploreProject()
	m.config.Base[0].Cmd = "make {target}"
	for i, it := range m.exploreItems(tabCommands) {
		if it.name == "build-api" {
			m.exploreCursor[tabCommands] = i
		}
	}
	return createKey(m, tea.KeyMsg{Type: tea.KeyEnter})
}

func publicClick(m Model, x, y int) Model {
	return exploreModel(exploreFirst(m.Update(tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})))
}

func TestTemplateMenuLeftClickReturnsWithoutReopening(t *testing.T) {
	for _, target := range []string{"row", "header", "blank", "back"} {
		m := templateMenuForMouse()
		original := m.exploreCursor[tabCommands]
		l := m.exploreLayout(m.viewWidth())
		x, y := 4, l.listY
		want := 0
		switch target {
		case "header":
			y = l.bodyY
			want = original
		case "blank":
			y = l.bodyY + l.bodyH - 3
			want = original
		case "back":
			x, y = l.rightX+4, l.editY
			want = original
		}
		m = publicClick(m, x, y)
		if m.screen != ScreenExplore || m.sce != nil || m.cf != nil || !m.exploreListFocused() || m.exploreCursor[tabCommands] != want {
			t.Fatalf("%s: click did not return/select without opening editor", target)
		}
	}
}

func TestTemplateDraftLeftClickAsksBeforeLeaving(t *testing.T) {
	m := exploreProject()
	m = createKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
	m = createKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	l := m.exploreLayout(m.viewWidth())
	m = publicClick(m, 4, l.listY+1)
	if m.leaveAsk != leaveTemplate || m.sce == nil {
		t.Fatal("draft was discarded without asking")
	}
	m = createKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.leaveAsk != leaveNone || m.sce.currentValues["env"] != "dev" {
		t.Fatal("default must keep the draft")
	}
	m = publicClick(m, 4, l.listY+1)
	m = createKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	if m.screen != ScreenExplore || m.sce != nil || m.sp != nil {
		t.Fatal("discard did not return to list")
	}
}

func TestTemplateMenuRightClickOpensChoice(t *testing.T) {
	m := templateMenuForMouse()
	l := m.exploreLayout(m.viewWidth())
	m = publicClick(m, l.rightX+4, l.fieldY+1)
	if m.screen != ScreenSlotPick || m.sp == nil {
		t.Fatal("Change values was not clickable")
	}
}
