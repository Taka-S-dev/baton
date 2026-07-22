package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestUpdateDeleteList_TabTogglesSpaceTypes checks the delete screens use
// the app-wide keymap: Tab toggles the hovered row, Space goes to the
// filter (needed for AND search) instead of toggling.
func TestUpdateDeleteList_TabTogglesSpaceTypes(t *testing.T) {
	m := &Model{}
	m.setPickBase([]string{"a", "b", "c"}, []string{"a", "b", "c"})
	noop := func() {}
	noopDel := func([]int) {}

	m.updateDeleteList(tea.KeyMsg{Type: tea.KeyTab}, 3, nil, noop, noopDel)
	if len(m.deleteSelected) != 1 || m.deleteSelected[0] != 0 {
		t.Fatalf("Tab: selected = %v, want [0]", m.deleteSelected)
	}
	m.updateDeleteList(tea.KeyMsg{Type: tea.KeyTab}, 3, nil, noop, noopDel)
	if len(m.deleteSelected) != 0 {
		t.Fatalf("second Tab: selected = %v, want empty (toggled off)", m.deleteSelected)
	}

	m.updateDeleteList(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}}, 3, nil, noop, noopDel)
	if m.pickSearch != "b" {
		t.Fatalf("typing must reach the filter, got %q", m.pickSearch)
	}
	if len(m.deleteSelected) != 0 {
		t.Fatalf("typing must not toggle, selected = %v", m.deleteSelected)
	}
	m.updateDeleteList(tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}, len(m.listItems), nil, noop, noopDel)
	if m.pickSearch != "b " {
		t.Fatalf("Space must reach the filter, got %q", m.pickSearch)
	}
}

// TestUpdateDeleteList_FilterMapsToOriginalIndices checks toggling and
// deleting through an active filter still target the original items.
func TestUpdateDeleteList_FilterMapsToOriginalIndices(t *testing.T) {
	m := &Model{}
	m.setPickBase([]string{"alpha", "beta", "gamma"}, []string{"alpha", "beta", "gamma"})
	noop := func() {}
	var deleted []int
	onDel := func(idx []int) { deleted = idx }

	// Filter down to "gamma" (original index 2), toggle it, confirm.
	for _, r := range "gam" {
		m.updateDeleteList(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}, len(m.listItems), nil, noop, onDel)
	}
	if len(m.listItems) != 1 || m.listItems[0] != "gamma" {
		t.Fatalf("filtered items = %v, want [gamma]", m.listItems)
	}
	m.updateDeleteList(tea.KeyMsg{Type: tea.KeyTab}, len(m.listItems), nil, noop, onDel)
	if len(m.deleteSelected) != 1 || m.deleteSelected[0] != 2 {
		t.Fatalf("selected = %v, want the ORIGINAL index [2]", m.deleteSelected)
	}

	m.updateDeleteList(tea.KeyMsg{Type: tea.KeyEnter}, len(m.listItems), nil, noop, onDel)
	m.updateDeleteList(tea.KeyMsg{Type: tea.KeyTab}, len(m.listItems), nil, noop, onDel) // → Yes
	m.updateDeleteList(tea.KeyMsg{Type: tea.KeyEnter}, len(m.listItems), nil, noop, onDel)
	if len(deleted) != 1 || deleted[0] != 2 {
		t.Fatalf("deleted = %v, want [2]", deleted)
	}
}

// TestUpdateDeleteList_EscClearsFilterFirst checks the staged Esc: an
// active filter is cleared first, only the second Esc leaves the screen.
func TestUpdateDeleteList_EscClearsFilterFirst(t *testing.T) {
	m := &Model{}
	m.setPickBase([]string{"a", "b"}, []string{"a", "b"})
	exited := false
	onExit := func() { exited = true }
	noopDel := func([]int) {}

	m.updateDeleteList(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}, len(m.listItems), nil, onExit, noopDel)
	m.updateDeleteList(tea.KeyMsg{Type: tea.KeyEscape}, len(m.listItems), nil, onExit, noopDel)
	if exited {
		t.Fatal("first Esc with a filter must only clear the filter")
	}
	if m.pickSearch != "" || len(m.listItems) != 2 {
		t.Fatalf("filter not cleared: search=%q items=%v", m.pickSearch, m.listItems)
	}
	m.updateDeleteList(tea.KeyMsg{Type: tea.KeyEscape}, len(m.listItems), nil, onExit, noopDel)
	if !exited {
		t.Fatal("second Esc must leave the screen")
	}
}
