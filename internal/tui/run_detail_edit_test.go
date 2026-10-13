package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func TestRunDetailEditButton(t *testing.T) {
	for _, width := range []int{60, 120} {
		for _, expanded := range []bool{false} {
			for _, queue := range []bool{false, true} {
				m := homeTab(homeTestModel(t), '2')
				m.width = width
				m.height = 36
				m.msSelected = []int{2}
				m.msCursor = 0
				m.runQueueFocused = queue
				m.runDetailOpen = expanded
				if width == 60 && queue {
					continue
				}
				want := m.runDetailCommand().Name
				lines := strings.Split(ansi.Strip(m.View()), "\n")
				x, y := -1, -1
				for row, line := range lines {
					if pos := strings.Index(line, runProjectEditLabel); pos >= 0 {
						x = ansi.StringWidth(line[:pos])
						y = row
					}
				}
				if x < 0 {
					t.Fatal("button not rendered")
				}
				next, _ := m.Update(tea.MouseMsg{X: x + 2, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
				got := next.(Model)
				if got.screen != ScreenExplore || got.runDetailOpen {
					t.Fatal("did not enter editor")
				}
				items := got.exploreItems(tabCommands)
				if items[got.exploreCursor[tabCommands]].name != want {
					t.Fatal("edited wrong item", want)
				}
				key := homeKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
				if key.exploreCursor[tabCommands] != got.exploreCursor[tabCommands] || key.screen != got.screen {
					t.Fatal("key and click differ")
				}
				got.gotoRunHome()
				if len(got.msSelected) != 1 || got.msItems[got.msSelected[0]].name() != m.msItems[2].name() {
					t.Fatal("queue not preserved")
				}
			}
		}
	}
}
