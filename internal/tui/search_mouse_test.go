package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func TestRunSearchClickAcrossLayouts(t *testing.T) {
	for _, tab := range []rune{'1', '2'} {
		for _, size := range [][2]int{{60, 12}, {80, 30}, {140, 12}, {140, 36}} {
			m := homeTab(homeTestModel(t), tab)
			m.width, m.height = size[0], size[1]
			row := -1
			for y, line := range strings.Split(ansi.Strip(m.View()), "\n") {
				if strings.Contains(line, "/ Search") {
					row = y
					break
				}
			}
			if row < 0 {
				t.Fatalf("search missing: %v tab %c", size, tab)
			}
			next, cmd := m.Update(tea.MouseMsg{X: 4, Y: row, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
			m = next.(Model)
			if !m.runHome.searching || !m.runSearchInput().Focused() || cmd == nil {
				t.Fatalf("click did not focus: %v tab %c row %d", size, tab, row)
			}
			m = homeKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("abc")})
			if m.runSearchInput().Value() != "abc" {
				t.Fatal("click did not enable typing")
			}
			m = homeTab(homeTestModel(t), tab)
			m.width, m.height = size[0], size[1]
			if size[0] >= 100 {
				next, _ = m.Update(tea.MouseMsg{X: size[0] - 2, Y: row, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
				if next.(Model).runHome.searching {
					t.Fatal("detail pane activated search")
				}
			}
		}
	}
}
