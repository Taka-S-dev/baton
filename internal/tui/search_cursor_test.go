package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func TestSearchInputsShareCursorAndEditing(t *testing.T) {
	for _, name := range []string{"projects", "recent", "editor", "commands", "workflows"} {
		t.Run(name, func(t *testing.T) {
			var m Model
			switch name {
			case "projects":
				m = Model{screen: ScreenProjectSelect}
			case "recent":
				m = Model{screen: ScreenRecent}
			case "editor":
				m = exploreProject()
			case "commands":
				m = homeTab(homeTestModel(t), '2')
			case "workflows":
				m = homeTab(homeTestModel(t), '1')
			}
			input := func() *textinput.Model {
				switch name {
				case "projects":
					return &m.projectSearchTI
				case "recent":
					return &m.recentSearchTI
				case "editor":
					return &m.exploreSearchTI
				default:
					return m.runSearchInput()
				}
			}
			next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
			m = next.(Model)
			if cmd == nil || !input().Focused() || input().Cursor.Mode() != cursor.CursorBlink {
				t.Fatal("search must start the common blinking cursor")
			}
			key := func(k tea.KeyType) { m = homeKey(m, tea.KeyMsg{Type: k}) }
			typeText := func(s string) { m = homeKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}) }
			typeText("ac")
			key(tea.KeyLeft)
			typeText("b")
			if input().Value() != "abc" {
				t.Fatal("middle insertion failed")
			}
			key(tea.KeyHome)
			key(tea.KeyDelete)
			key(tea.KeyEnd)
			key(tea.KeyBackspace)
			if input().Value() != "b" {
				t.Fatalf("editing mismatch: %q", input().Value())
			}
			key(tea.KeyEsc)
			if input().Value() != "" || !input().Focused() {
				t.Fatal("clear must keep input focus")
			}
			key(tea.KeyEsc)
			if input().Focused() {
				t.Fatal("search exit must blur cursor")
			}
		})
	}
}
