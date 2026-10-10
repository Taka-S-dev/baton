package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"strings"
	"testing"
)

func TestExploreDerivedCommandMouseUsesTemplateEditor(t *testing.T) {
	for _, fieldClick := range []bool{false, true} {
		m := exploreProject()
		// Use an actual slotted template so the derived editor can resolve it.
		m.config.Base[0].Cmd = "make {target}"
		for i, it := range m.exploreVisible() {
			if it.name == "build-api" {
				m.exploreCursor[tabCommands] = i
			}
		}
		keyboard := press(m, tea.KeyMsg{Type: tea.KeyEnter})
		l := m.exploreLayout(m.viewWidth())
		x, y := 4, l.listY+m.exploreCursor[tabCommands]-l.start
		if fieldClick {
			x, y = l.rightX+4, l.fieldY
		}
		mouse := clickAt(m, x, y)
		if keyboard.sce == nil || mouse.sce == nil || mouse.cf != nil || mouse.screen != keyboard.screen {
			t.Fatalf("field=%v: mouse and Enter must open the template editor; mouse=%v keyboard=%v", fieldClick, mouse.screen, keyboard.screen)
		}
	}
}

func TestExploreNewHintOnlyOnSupportedTab(t *testing.T) {
	for tab := 0; tab < tabCount; tab++ {
		m := exploreProject()
		m.exploreTab = tab
		advertised := strings.Contains(m.exploreKeys(), "a Add")
		got := typeRunes(m, "n")
		if advertised != (got.cf != nil || got.screen != ScreenExplore) {
			t.Errorf("tab %d advertises an unavailable action", tab)
		}
	}
}

func TestExploreFocusFollowsActiveInteraction(t *testing.T) {
	m := exploreProject()
	if !m.exploreListFocused() {
		t.Fatal("list should initially have focus")
	}
	m = press(m, tea.KeyMsg{Type: tea.KeySpace})
	if m.exploreListFocused() {
		t.Error("action menu must take focus from the list")
	}
	m = openReleaseEditor(t)
	if m.exploreListFocused() {
		t.Error("workflow editor must take focus from the list")
	}
	m.wfe.dirty = true
	l := m.exploreLayout(m.viewWidth())
	got := clickAt(m, 4, l.listY+1)
	if got.leaveAsk != leaveWorkflow || got.exploreCursor[tabWorkflows] != m.exploreCursor[tabWorkflows] {
		t.Fatal("unsaved confirmation must block list navigation")
	}
	if !strings.Contains(got.exploreKeys(), "Discard") || strings.Contains(got.exploreKeys(), "J/K") {
		t.Fatal("confirmation must show its own key hints")
	}
}
