package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func TestActionPopupPreservesBackground(t *testing.T) {
	for _, width := range []int{40, 110} {
		m := exploreProject()
		m.width = width
		background := m.viewExplore(m.viewWidth())
		m.exploreMenu = true
		p := m.actionPopup(m.viewWidth(), len(m.exploreActions(m.exploreItems(tabCommands)[0])))
		rendered := m.overlayExploreActions(background, m.viewWidth())
		before, after := strings.Split(background, "\n"), strings.Split(rendered, "\n")
		if len(before) != len(after) {
			t.Fatal("overlay changed frame height")
		}
		for y := range before {
			if y < p.y || y >= p.y+p.h {
				if before[y] != after[y] {
					t.Fatal("overlay changed background outside popup")
				}
			} else {
				if ansi.Strip(ansi.Cut(before[y], 0, p.x)) != ansi.Strip(ansi.Cut(after[y], 0, p.x)) {
					t.Fatal("overlay erased left background")
				}
			}
			if y >= p.y && y < p.y+p.h && lipgloss.Width(after[y]) > m.viewWidth() {
				t.Fatal("popup exceeds terminal width")
			}
		}
	}
}

func TestActionPopupNavigationAndMouseIsolation(t *testing.T) {
	m := exploreProject()
	m = press(m, tea.KeyMsg{Type: tea.KeySpace})
	original := m.exploreCursor
	m = press(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.exploreMenuCursor != 1 {
		t.Fatal("j must move within menu")
	}
	m = press(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.exploreMenuCursor != 0 {
		t.Fatal("k must move within menu")
	}
	l := m.exploreLayout(m.viewWidth())
	m = mouseAt(m, 1, l.listY, tea.MouseActionPress, tea.MouseButtonWheelDown)
	if m.exploreCursor != original || !m.exploreMenu {
		t.Fatal("wheel reached background")
	}
	closed := clickAt(m, 1, l.listY+1)
	if closed.exploreMenu || closed.exploreCursor != original || closed.cf != nil {
		t.Fatal("outside click must only dismiss")
	}
	p := m.actionPopup(m.viewWidth(), len(m.exploreActions(m.exploreItems(tabCommands)[0])))
	deleted := clickAt(m, p.x+3, p.y+1+3)
	if deleted.screen != ScreenDeleteCommand || !deleted.deleteConfirm {
		t.Fatal("delete action must still require confirmation")
	}
}

func TestActionPopupShortTerminalScrollsSelection(t *testing.T) {
	m := exploreProject()
	m.width, m.height = 40, 12
	m.exploreMenu = true
	m.exploreMenuCursor = 3
	view := stripStyle(m.View())
	if !strings.Contains(view, "x  Delete") {
		t.Fatal("selected action is not visible")
	}
	if frameLines(m.View()) > m.height {
		t.Fatal("popup exceeds terminal height")
	}
	m = press(m, tea.KeyMsg{Type: tea.KeySpace})
	if m.exploreMenu {
		t.Fatal("Space must dismiss")
	}
}
