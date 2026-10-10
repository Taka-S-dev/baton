package tui

import "strings"

// Drafts exist only in the view: saved data and the previous cursor stay intact.
func (m Model) newValueDraft() (wsItem, bool) {
	if !m.exploreLaunched {
		return wsItem{}, false
	}
	name, value := "", ""
	kind := exList
	switch {
	case m.exploreTab == tabLists && m.screen == ScreenNameInput && m.nameInputMode == nameInputNewList:
		name = strings.TrimSpace(m.nameInput.Value())
	case m.exploreTab == tabVars && m.screen == ScreenVarForm && m.ve != nil && m.ve.mode == 0:
		kind = exVar
		name = m.ve.name
		if m.ve.fieldIdx == 0 {
			name = strings.TrimSpace(m.nameInput.Value())
		} else {
			value = m.nameInput.Value()
		}
	default:
		return wsItem{}, false
	}
	label := name
	if label == "" {
		label = draftPlaceholderName
	}
	return wsItem{kind: kind, name: name, label: "+ " + label + " · unsaved", note: value, draft: true}, true
}

func (m Model) withValueDraft(items []wsItem) []wsItem {
	if draft, ok := m.newValueDraft(); ok {
		return append(items, draft)
	}
	return items
}
