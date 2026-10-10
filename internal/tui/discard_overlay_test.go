package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestDiscardOverlayKeepsListBehindIt(t *testing.T) {
	m := exploreProject()
	m.height = 32
	m.width = 90
	m.setupMultiSelect()
	m.msSearchTI.SetValue("build")
	m.msSelected = []int{0, 1}
	m.msEscArmed = true
	view := stripStyle(m.viewMultiSelect(m.width))
	for _, text := range []string{"Select commands", "Discard selection?", "Selected(2)"} {
		if !strings.Contains(view, text) {
			t.Fatalf("missing background or dialog: %s", text)
		}
	}
	if frameRows(view) > m.height {
		t.Fatal("overlay exceeds screen")
	}
	m = exploreModel(exploreFirst(m.updateMultiSelect(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})))
	if m.msEscArmed || len(m.msSelected) != 2 || m.msSearchTI.Value() != "build" {
		t.Fatal("keep changed selection or search")
	}
}
