package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	mdl "github.com/Taka-S-dev/baton/internal/model"
	tea "github.com/charmbracelet/bubbletea"
)

func TestCandidateScrollPreservesSelection(t *testing.T) {
	for _, height := range []int{16, 36} {
		m := Model{width: 120, height: height, screen: ScreenRunCommands}
		m.runHome.ready = true
		for i := 0; i < 60; i++ {
			c := mdl.Command{Name: fmt.Sprintf("cmd%02d", i), Cmd: "echo hi"}
			m.msItems = append(m.msItems, msItem{cmd: &c})
		}
		m.msSelected = []int{4, 2}
		send := func(x, y int, b tea.MouseButton, a tea.MouseAction) {
			next, _ := m.Update(tea.MouseMsg{X: x, Y: y + 1, Button: b, Action: a})
			m = next.(Model)
		}
		a := m.candidateScrollArea()
		send(5, a.y, tea.MouseButtonWheelDown, tea.MouseActionPress)
		if m.candidateScrollArea().offset != 3 || m.msCursor != 0 {
			t.Fatal("wheel moved selection or failed to scroll")
		}
		a = m.candidateScrollArea()
		top, _ := scrollbarThumb(a.total, a.rows, a.offset)
		send(a.width-1, a.y+top, tea.MouseButtonLeft, tea.MouseActionPress)
		send(a.width-1, a.y+a.rows+5, tea.MouseButtonLeft, tea.MouseActionMotion)
		send(a.width-1, a.y+a.rows+5, tea.MouseButtonLeft, tea.MouseActionRelease)
		a = m.candidateScrollArea()
		if a.offset != a.total-a.rows || m.msCursor != 0 || m.msSelected[0] != 4 || m.msSelected[1] != 2 {
			t.Fatal("drag changed selection")
		}
		send(5, a.y, tea.MouseButtonLeft, tea.MouseActionPress)
		if m.msCursor != a.offset {
			t.Fatal("clicked wrong scrolled row")
		}
		m = homeKey(m, tea.KeyMsg{Type: tea.KeyUp})
		if m.msScrollManual {
			t.Fatal("keyboard did not restore following")
		}
	}
}

func TestCandidatePaddingDoesNotShowEllipsis(t *testing.T) {
	for _, line := range []string{"build          ", sRowActive.Render("build          ")} {
		got := ansi.Strip(candidateRow(line, 12, 30, 5, 0, 0))
		if strings.Contains(got, "…") || ansi.StringWidth(got) != 12 {
			t.Fatalf("padding marked as truncated: %q", got)
		}
	}
	if got := ansi.Strip(candidateRow("a genuinely long command", 12, 30, 5, 0, 0)); !strings.Contains(got, "…") {
		t.Fatal("missing real truncation marker")
	}
}
