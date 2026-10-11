package tui

import (
	"strings"
	"testing"

	mdl "github.com/Taka-S-dev/baton/internal/model"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func TestShortCommandSplitAndQueueMouse(t *testing.T) {
	for _, h := range []int{7, 12, 19} {
		c := mdl.Command{Name: "build", Cmd: "echo ok"}
		m := Model{width: 120, height: h, msItems: []msItem{{cmd: &c}}, msSelected: []int{0}}
		got := stripStyle(m.viewMultiSelect(120))
		if !strings.Contains(got, "Run queue") || !strings.Contains(got, "1. $ echo ok") {
			t.Fatal(got)
		}
		if len(strings.Split(got, "\n")) > h {
			t.Fatal(got)
		}
		for _, line := range strings.Split(got, "\n") {
			if ansi.StringWidth(line) > 120 {
				t.Fatal(line)
			}
		}
		next, _ := m.updateRunSelectMouse(tea.MouseMsg{X: 65, Y: 1, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
		if !next.(Model).runQueueFocused {
			t.Fatal("queue click missed")
		}
		if strings.Contains(stripStyle(m.viewMultiSelect(80)), "Run queue") {
			t.Fatal("narrow split")
		}
	}
}
