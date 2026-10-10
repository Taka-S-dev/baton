package tui

import (
	mdl "github.com/Taka-S-dev/baton/internal/model"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"reflect"
	"strings"
	"testing"
)

func TestRunSelectClickFocusAndDetails(t *testing.T) {
	a := mdl.Command{Name: "one", Cmd: "echo first"}
	b := mdl.Command{Name: "two", Cmd: "echo second"}
	m := msModel([]msItem{{cmd: &a}, {cmd: &b}})
	m.width = 120
	m.height = 30
	m.screen = ScreenRunCommands
	m.msSelected = []int{1, 0}
	click := func(x, y int) {
		next, _ := m.Update(tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
		m = next.(Model)
	}
	click(70, 1)
	if !m.runQueueFocused || m.runQueueCursor != 0 {
		t.Fatal("queue focus")
	}
	if !strings.Contains(m.viewRunQueue(59), "echo second") {
		t.Fatal("details")
	}
	y := -1
	m.viewMultiSelectCandidates(60, false, func(i, row int) {
		if i == 1 {
			y = row
		}
	})
	click(10, y)
	if m.runQueueFocused || m.msCursor != 1 {
		t.Fatal("candidate focus")
	}
	if !reflect.DeepEqual(m.msSelected, []int{1, 0}) || m.screen != ScreenRunCommands {
		t.Fatal("click changed run selection")
	}
	m.msEscArmed = true
	click(70, 2)
	if m.runQueueFocused {
		t.Fatal("click leaked through modal")
	}
}

func TestHeaderClickCyclesBackToDefinition(t *testing.T) {
	a := mdl.Command{Name: "zeta"}
	b := mdl.Command{Name: "alpha"}
	m := msModel([]msItem{{cmd: &a}, {cmd: &b}})
	m.width = 120
	m.height = 30
	m.screen = ScreenRunCommands
	m.commandSort = "definition"
	m.msSelected = []int{0, 1}
	for _, want := range []string{"name", "name-desc", "definition"} {
		y := -1
		view := m.viewMultiSelectCandidates(60, false, func(i, row int) {
			if i == -1 {
				y = row
			}
		})
		line := strings.Split(ansi.Strip(view), "\n")[y]
		at := strings.Index(line, "Name")
		x := lipgloss.Width(line[:at])
		next, _ := m.Update(tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
		m = next.(Model)
		if m.commandSort != want {
			t.Fatalf("got %s want %s", m.commandSort, want)
		}
		if m.msFiltered()[m.msCursor] != 0 || !reflect.DeepEqual(m.msSelected, []int{0, 1}) {
			t.Fatal("identity or queue changed")
		}
	}
	if strings.Contains(m.commandSortLabel(), "↑") || strings.Contains(m.commandSortLabel(), "↓") {
		t.Fatal("sort arrow remains")
	}
}
