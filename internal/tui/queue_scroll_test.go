package tui

import (
	"strings"
	"testing"

	mdl "github.com/Taka-S-dev/baton/internal/model"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func TestDetailsPreserveFullTextAndScroll(t *testing.T) {
	c := mdl.Command{Name: "deploy", Cmd: strings.Repeat("echo 日本語  ", 30), Dir: "very/long/directory"}
	m := Model{width: 100, height: 30, msItems: []msItem{{cmd: &c}}, msSelected: []int{0}, runQueueFocused: true}
	lines, _ := m.runDetailLines(18)
	if !strings.Contains(strings.Join(lines, ""), c.Cmd) {
		t.Fatal("wrapped command lost content")
	}
	for _, line := range lines {
		if ansi.StringWidth(line) > 18 {
			t.Fatal("overflow", line)
		}
	}
	m.scrollRunDetail(9999)
	if m.runDetailOffset == 0 {
		t.Fatal("did not scroll")
	}
	before := m.runDetailOffset
	m.scrollRunDetail(9999)
	if m.runDetailOffset != before {
		t.Fatal("did not clamp")
	}
	m.scrollRunDetail(-9999)
	if m.runDetailOffset != 0 {
		t.Fatal("did not return to start")
	}
	m.runHome.ready = true
	m.screen = ScreenRunCommands
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	m = next.(Model)
	if !m.runDetailOpen {
		t.Fatal("details did not open")
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(Model)
	if m.runDetailOpen || len(m.msSelected) != 1 {
		t.Fatal("closing details changed selection")
	}
}

func TestScrollbarOnlyForOverflow(t *testing.T) {
	if strings.Contains(stripStyle(scrollRows([]string{"abc"}, 0, 3, 12)), "┃") {
		t.Fatal("unnecessary bar")
	}
	s := stripStyle(scrollRows([]string{"a", "b", "c", "d"}, 1, 2, 12))
	if !strings.Contains(s, "┃") {
		t.Fatal("missing overflow bar")
	}
	for _, line := range strings.Split(s, "\n") {
		if ansi.StringWidth(line) != 12 {
			t.Fatal("unstable width")
		}
	}
}
