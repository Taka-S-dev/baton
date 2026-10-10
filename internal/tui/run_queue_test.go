package tui

import (
	"reflect"
	"testing"

	mdl "github.com/Taka-S-dev/baton/internal/model"
	tea "github.com/charmbracelet/bubbletea"
)

func TestRunQueueOrderReachesReview(t *testing.T) {
	a := mdl.Command{Name: "a", Cmd: "echo a"}
	b := mdl.Command{Name: "b", Cmd: "echo b"}
	c := mdl.Command{Name: "c", Cmd: "echo c"}
	m := msModel([]msItem{{cmd: &a}, {cmd: &b}, {cmd: &c}})
	m.screen = ScreenRunCommands
	m.msSelected = []int{0, 1, 2}
	m.runQueueFocused = true
	next, _ := m.updateMultiSelect(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("J")})
	m = next.(Model)
	if !reflect.DeepEqual(m.msSelected, []int{1, 0, 2}) {
		t.Fatal(m.msSelected)
	}
	next, _ = m.updateMultiSelect(tea.KeyMsg{Type: tea.KeyDelete})
	m = next.(Model)
	if !reflect.DeepEqual(m.msSelected, []int{1, 2}) || len(m.msItems) != 3 {
		t.Fatal(m.msSelected)
	}
	next, _ = m.updateMultiSelect(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(Model)
	if len(m.confirmRunItems) != 2 || m.confirmRunItems[0].Cmd.Name != "b" || m.confirmRunItems[1].Cmd.Name != "c" {
		t.Fatalf("review order: %+v", m.confirmRunItems)
	}
}
