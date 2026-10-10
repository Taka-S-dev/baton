package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestValueEditorsStayBesideList(t *testing.T) {
	for _, tab := range []int{tabLists, tabVars} {
		for _, create := range []bool{false, true} {
			m := exploreProject()
			m.exploreTab = tab
			key := tea.KeyMsg{Type: tea.KeyEnter}
			if create {
				key = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")}
			}
			m = createKey(m, key)
			if !m.exploreValueFlow() || m.exploreListFocused() {
				t.Fatalf("tab=%d create=%v: editor has no focus", tab, create)
			}
			view := stripStyle(m.View())
			if !strings.Contains(view, "[3 Lists]") || (!strings.Contains(view, "Preview") && !strings.Contains(view, "Value details")) {
				t.Fatalf("editor replaced workspace: %s", view)
			}
			if frameLines(m.View()) > m.height {
				t.Fatal("editor overflows terminal")
			}
			m = createKey(m, tea.KeyMsg{Type: tea.KeyEsc})
			if m.screen != ScreenExplore || !m.exploreListFocused() {
				t.Fatalf("tab=%d: Esc did not return to list", tab)
			}
		}
	}
}

func TestValueEditorResizePreservesDraft(t *testing.T) {
	m := exploreProject()
	m.exploreTab = tabVars
	m = createKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	m.nameInput.SetValue("draft-value")
	m = exploreModel(exploreFirst(m.Update(tea.WindowSizeMsg{Width: 60, Height: 22})))
	if strings.Contains(stripStyle(m.View()), "[3 Lists]") {
		t.Fatal("narrow editor must use full screen")
	}
	m = exploreModel(exploreFirst(m.Update(tea.WindowSizeMsg{Width: 110, Height: 30})))
	if !strings.Contains(stripStyle(m.View()), "[3 Lists]") || m.nameInput.Value() != "draft-value" {
		t.Fatal("resize lost the workspace or draft")
	}
}

func TestRightPaneClickOpensValueEditor(t *testing.T) {
	for _, tab := range []int{tabLists, tabVars} {
		m := exploreProject()
		m.exploreTab = tab
		l := m.exploreLayout(m.viewWidth())
		m = clickAt(m, l.rightX+3, l.fieldY)
		if !m.exploreValueFlow() {
			t.Fatalf("tab %d: click did not open editor", tab)
		}
	}
}

func TestListPaneMouseSelectsAndOpensEntry(t *testing.T) {
	m := exploreProject()
	m.exploreTab = tabLists
	m = createKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	l := m.exploreLayout(m.viewWidth())
	click := tea.MouseMsg{X: l.rightX + 3, Y: l.fieldY + 1, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress}
	m = exploreModel(exploreFirst(m.Update(click)))
	if m.le.cursor != 1 || m.le.editing {
		t.Fatal("click did not select entry")
	}
	m = exploreModel(exploreFirst(m.Update(click)))
	if !m.le.editing {
		t.Fatal("second click did not open entry")
	}
}

// TestAddValueRowLooksLikeAnAction: the editor's "+ Add value" row is an
// action, so it is drawn like the list pane's "+ Add list" — in the
// accent with its key beside it — and not like the entries above it.
func TestAddValueRowLooksLikeAnAction(t *testing.T) {
	for _, width := range []int{110, 60} { // beside the list, and full screen
		m := exploreProject()
		m.width, m.exploreTab = width, tabLists
		m = createKey(m, tea.KeyMsg{Type: tea.KeyEnter})
		if m.le == nil {
			t.Fatalf("width %d: Enter did not open the list editor", width)
		}
		view := m.View()
		if !strings.Contains(view, accent("+ Add value  a")) {
			t.Fatalf("width %d: the add row is not drawn as an action:\n%s", width, stripStyle(view))
		}
	}
}
