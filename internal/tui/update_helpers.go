package tui

import (
	"sort"
	"strings"

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

// ── Pick-screen filter ────────────────────────────────────────────────────────

// setPickBase installs the full item set for a filterable pick screen
// and resets the filter. texts are the search haystacks, parallel to
// names; they are lowercased here.
func (m *Model) setPickBase(names, texts []string) {
	m.pickBase = names
	m.pickTexts = make([]string, len(texts))
	for i, t := range texts {
		m.pickTexts[i] = strings.ToLower(t)
	}
	m.pickSearch = ""
	m.applyPickFilter()
}

// applyPickFilter recomputes the visible rows from the current filter.
func (m *Model) applyPickFilter() {
	if m.pickSearch == "" {
		m.listItems = append([]string(nil), m.pickBase...)
		m.pickMap = nil
		return
	}
	terms := strings.Fields(strings.ToLower(m.pickSearch))
	m.listItems = nil
	m.pickMap = make([]int, 0, len(m.pickBase))
	for i, txt := range m.pickTexts {
		if matchesAllTerms(txt, terms) {
			m.listItems = append(m.listItems, m.pickBase[i])
			m.pickMap = append(m.pickMap, i)
		}
	}
}

// pickOrig maps a visible row index to the original item index.
func (m *Model) pickOrig(i int) int {
	if m.pickMap == nil {
		return i
	}
	return m.pickMap[i]
}

// clearPickFilter resets the filter and shows every row again.
func (m *Model) clearPickFilter() {
	m.pickSearch = ""
	m.applyPickFilter()
	m.listCursor = 0
}

// handlePickTyping feeds printable keys and backspace into the pick
// filter. Returns true when the key was consumed; onChange (optional)
// runs after the visible rows changed.
func (m *Model) handlePickTyping(msg tea.KeyMsg, onChange func()) bool {
	switch {
	case msg.String() == "backspace":
		if m.pickSearch == "" {
			return false
		}
		r := []rune(m.pickSearch)
		m.pickSearch = string(r[:len(r)-1])
	case msg.String() == " ":
		if m.pickSearch == "" {
			return false
		}
		m.pickSearch += " "
	case msg.Type == tea.KeyRunes && len(msg.Runes) > 0 && msg.Runes[0] >= 32:
		m.pickSearch += string(msg.Runes)
	default:
		return false
	}
	m.applyPickFilter()
	m.listCursor = 0
	if onChange != nil {
		onChange()
	}
	return true
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

// updateDeleteList implements the shared delete screen: Tab toggles the
// hovered row (Enter with nothing selected deletes the cursor row), a
// No/Yes confirm dialog (Tab/←/→/h/l to switch), typing filters the list,
// and deletion runs in descending-index order so earlier removals don't
// shift later indices.
//
// count is the number of VISIBLE rows (len of m.listItems). onMove
// (optional) runs after the cursor moves or the filter changes.
// onDelete receives the selected ORIGINAL indices sorted descending.
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
	case "tab":
		if count > 0 && m.listCursor < count {
			m.toggleDeleteSelected(m.pickOrig(m.listCursor))
		}
	case "enter":
		if count == 0 || m.listCursor >= count {
			break
		}
		if len(m.deleteSelected) == 0 {
			m.deleteSelected = []int{m.pickOrig(m.listCursor)}
		}
		m.deleteConfirm = true
		m.deleteBtn = 0
	case "esc":
		if m.pickSearch != "" {
			m.clearPickFilter()
			if onMove != nil {
				onMove()
			}
			break
		}
		m.deleteSelected = nil
		onExit()
	default:
		m.handlePickTyping(msg, onMove)
	}
	return *m, nil
}
