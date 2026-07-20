package tui

import (
	"sort"

	tea "github.com/charmbracelet/bubbletea"
)

// moveListCursor moves listCursor within a list of n items.
// Returns true when the key was up/down (i.e. it was handled).
func (m *Model) moveListCursor(key string, n int) bool {
	switch key {
	case "up":
		if m.listCursor > 0 {
			m.listCursor--
		}
		return true
	case "down":
		if m.listCursor < n-1 {
			m.listCursor++
		}
		return true
	}
	return false
}

// sortedListNames returns the selection-list names in stable order.
func (m Model) sortedListNames() []string {
	names := make([]string, 0, len(m.lists))
	for k := range m.lists {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// toggleDeleteSelected toggles idx in the delete-selection set.
func (m *Model) toggleDeleteSelected(idx int) {
	for i, s := range m.deleteSelected {
		if s == idx {
			m.deleteSelected = append(m.deleteSelected[:i], m.deleteSelected[i+1:]...)
			return
		}
	}
	m.deleteSelected = append(m.deleteSelected, idx)
}

// updateDeleteList implements the shared delete screen: multi-select with
// Tab or Space (Enter with nothing selected deletes the cursor row), a
// No/Yes confirm dialog (Tab/←/→/h/l to switch), and descending-index
// deletion so earlier removals don't shift later indices.
//
// count is the number of deletable items. onMove (optional) runs after the
// cursor moves. onDelete receives the selected indices sorted descending.
// onExit leaves the screen (Esc, or after a confirmed delete).
func (m *Model) updateDeleteList(msg tea.KeyMsg, count int, onMove func(), onExit func(), onDelete func(indices []int)) (tea.Model, tea.Cmd) {
	if m.deleteConfirm {
		switch msg.String() {
		case "tab", "left", "right", "h", "l":
			m.deleteBtn = 1 - m.deleteBtn
		case "enter":
			confirmed := m.deleteBtn == 1
			m.deleteConfirm = false
			m.deleteBtn = 0
			if confirmed {
				indices := append([]int(nil), m.deleteSelected...)
				sort.Sort(sort.Reverse(sort.IntSlice(indices)))
				m.deleteSelected = nil
				onDelete(indices)
				onExit()
			}
		case "esc":
			m.deleteConfirm = false
			m.deleteBtn = 0
		}
		return *m, nil
	}

	switch msg.String() {
	case "up", "down":
		if m.moveListCursor(msg.String(), count) && onMove != nil {
			onMove()
		}
	case "tab", " ", "　":
		if count > 0 {
			m.toggleDeleteSelected(m.listCursor)
		}
	case "enter":
		if count == 0 {
			break
		}
		if len(m.deleteSelected) == 0 {
			m.deleteSelected = []int{m.listCursor}
		}
		m.deleteConfirm = true
		m.deleteBtn = 0
	case "esc":
		m.deleteSelected = nil
		onExit()
	}
	return *m, nil
}
