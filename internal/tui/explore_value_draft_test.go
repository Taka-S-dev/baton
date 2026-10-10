package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewValueDraftFollowsInputAndRestoresSelection(t *testing.T) {
	for _, tab := range []int{tabLists, tabVars} {
		m := exploreProject()
		m.exploreTab = tab
		original := m.exploreCursor[tab]
		count := len(m.exploreItems(tab))
		m = createKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
		for _, name := range []string{"", "fresh_name"} {
			m.nameInput.SetValue(name)
			l := m.exploreLayout(m.viewWidth())
			if !l.items[l.cursor].draft || l.items[l.cursor].name != name {
				t.Fatal("draft not selected or updated")
			}
			if len(m.exploreItems(tab)) != count {
				t.Fatal("draft mutated saved items")
			}
			preview := stripStyle(m.explorePreviewBox(l))
			if strings.Contains(preview, "used by") || !strings.Contains(preview, "unsaved") {
				t.Fatal("stale preview")
			}
		}
		m.closeExploreValue()
		if m.exploreCursor[tab] != original || len(m.exploreVisible()) != count {
			t.Fatal("cancel failed to restore list")
		}
	}
}

func TestValueDraftVisibleThroughSearchAndBecomesSaved(t *testing.T) {
	for _, tab := range []int{tabLists, tabVars} {
		m := exploreProject()
		m.projectDir = t.TempDir()
		m.exploreTab = tab
		m.exploreSearch = "no-match"
		m = createKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
		m.nameInput.SetValue("fresh_name")
		l := m.exploreLayout(m.viewWidth())
		if len(l.items) != 1 || !l.items[0].draft {
			t.Fatal("search hid draft")
		}
		m = createKey(m, tea.KeyMsg{Type: tea.KeyEnter})
		if tab == tabVars {
			m.nameInput.SetValue("unique_value")
			l = m.exploreLayout(m.viewWidth())
			if l.items[l.cursor].name != "fresh_name" || l.items[l.cursor].note != "unique_value" {
				t.Fatal("value draft lost name")
			}
			m = createKey(m, tea.KeyMsg{Type: tea.KeyEnter})
		}
		if _, ok := m.newValueDraft(); ok {
			t.Fatal("saved draft remained")
		}
		it, ok := m.exploreCurrent()
		if !ok || it.name != "fresh_name" || it.draft {
			t.Fatal("saved row not selected")
		}
	}
}
