package tui

import (
	"strings"
	"testing"

	mdl "github.com/Taka-S-dev/baton/internal/model"
	"github.com/charmbracelet/bubbles/cursor"
	tea "github.com/charmbracelet/bubbletea"
)

func slotValuesFixture(t *testing.T) Model {
	t.Helper()
	cmd := mdl.Command{Name: "deploy", Cmd: "deploy --env {env}"}
	m := Model{width: 100, height: 24}
	m.lists = map[string][]mdl.ListEntry{"env": {{Value: "dev"}, {Value: "prod", Label: "production"}}}
	next, _ := m.startResolveFlow([]msItem{{cmd: &cmd}})
	m = next.(Model)
	if m.screen != ScreenSlotPick || m.sp == nil {
		t.Fatalf("expected the slot picker, got screen %v", m.screen)
	}
	return m
}

// TestSlotValuesShareThePickerBox: a list slot is offered in the same
// box as a path slot — title with the count, a search line, rows with
// the cursor bar, a footer — over the step context and the preview.
func TestSlotValuesShareThePickerBox(t *testing.T) {
	m := slotValuesFixture(t)
	view := stripStyle(m.View())
	for _, want := range []string{"{env} · 2 candidates", "Search / Filter values…", "▶ dev", "prod  ·  production", "Type a value above", "Enter Select", "$ deploy --env dev"} {
		if !strings.Contains(view, want) {
			t.Errorf("picker is missing %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "─ Select value ─") || strings.Contains(view, "filter list or type a value") {
		t.Errorf("the old inline list is still drawn:\n%s", view)
	}
	if frameRows(m.View()) > m.height {
		t.Errorf("picker overflows the terminal: %d rows", frameRows(m.View()))
	}
	typed := m
	typed.sp.search = "pr"
	typed.sp.applyFilter()
	view = stripStyle(typed.View())
	if !strings.Contains(view, "{env} · 1 candidates") || !strings.Contains(view, "Search / pr") || !strings.Contains(view, "Esc Clear filter") {
		t.Errorf("typing does not narrow the box:\n%s", view)
	}
}

// TestSlotValuesSearchCaretBlinks: the box's search line is a real text
// input, focused on open, so its caret blinks like the file picker's.
func TestSlotValuesSearchCaretBlinks(t *testing.T) {
	cmd := mdl.Command{Name: "deploy", Cmd: "deploy --env {env}"}
	m := Model{width: 100, height: 24}
	m.lists = map[string][]mdl.ListEntry{"env": {{Value: "dev"}}}
	next, focus := m.startResolveFlow([]msItem{{cmd: &cmd}})
	m = next.(Model)
	if focus == nil || !m.sp.input.Focused() {
		t.Fatal("the search field is not focused when the picker opens")
	}
	// The focus command delivers the first blink; answering it must
	// schedule the next, or the caret would stop after one toggle.
	first, ok := focus().(cursor.BlinkMsg)
	if !ok {
		t.Fatalf("focus did not produce a blink message: %T", focus())
	}
	if _, tick := m.Update(first); tick == nil {
		t.Fatal("a blink tick is not answered with the next one")
	}
}

// TestSlotValuesClickSelects: clicking a row moves the cursor there, and
// clicking the cursor row chooses it, as in the file picker.
func TestSlotValuesClickSelects(t *testing.T) {
	m := slotValuesFixture(t)
	y := rowOf(m.View(), "prod  ·  production")
	if y < 0 {
		t.Fatal("prod row not drawn")
	}
	next, _ := m.Update(tea.MouseMsg{X: 5, Y: y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	m = next.(Model)
	if m.sp == nil || m.sp.cursor != 1 {
		t.Fatalf("click did not move the cursor to prod: %+v", m.sp)
	}
	next, _ = m.Update(tea.MouseMsg{X: 5, Y: y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	m = next.(Model)
	if m.screen == ScreenSlotPick {
		t.Fatal("second click on the cursor row did not choose the value")
	}
}

// TestSlotValuesFitAShortTerminal: with no room for the box the picker
// falls back to the inline list, and never runs past the last row.
func TestSlotValuesFitAShortTerminal(t *testing.T) {
	for _, h := range []int{13, 15, 17} {
		m := slotValuesFixture(t)
		m.height = h
		if rows := frameRows(m.View()); rows > h {
			t.Errorf("height %d: picker draws %d rows", h, rows)
		}
		if !strings.Contains(stripStyle(m.View()), "dev") {
			t.Errorf("height %d: the values are not shown", h)
		}
	}
}
