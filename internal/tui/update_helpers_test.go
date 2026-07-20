package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestUpdateDeleteList_TabAndSpaceToggle checks both Tab (consistent with
// the command selector) and Space toggle the row on delete screens.
func TestUpdateDeleteList_TabAndSpaceToggle(t *testing.T) {
	m := &Model{}
	noop := func() {}
	noopDel := func([]int) {}

	m.updateDeleteList(tea.KeyMsg{Type: tea.KeyTab}, 3, nil, noop, noopDel)
	if len(m.deleteSelected) != 1 || m.deleteSelected[0] != 0 {
		t.Fatalf("Tab: selected = %v, want [0]", m.deleteSelected)
	}
	m.updateDeleteList(tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}, 3, nil, noop, noopDel)
	if len(m.deleteSelected) != 0 {
		t.Fatalf("Space: selected = %v, want empty (toggled off)", m.deleteSelected)
	}
}
