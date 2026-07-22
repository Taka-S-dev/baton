package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	mdl "github.com/Taka-S-dev/baton/internal/model"
)

func msModel(items []msItem) Model {
	m := Model{}
	m.msItems = items
	m.msSearchTI = textinput.New()
	return m
}

// TestMsFiltered_MatchesEmbeddedText checks the search hits text beyond
// name/group: the command body and template values.
func TestMsFiltered_MatchesEmbeddedText(t *testing.T) {
	build := mdl.Command{Name: "build", Group: "dev", Cmd: "make auth-service"}
	deploy := mdl.Command{Name: "deploy", Template: "deploy-tpl", Values: map[string]string{"env": "prod"}}
	m := msModel([]msItem{{cmd: &build}, {cmd: &deploy}})

	cases := []struct {
		query string
		want  []int
	}{
		{"auth", []int{0}},         // command body
		{"prod", []int{1}},         // template value
		{"deploy-tpl", []int{1}},   // template name
		{"nothing-matches-x", nil}, // no hit
		{"", []int{0, 1}},          // empty query keeps everything
		{"  ", []int{0, 1}},        // whitespace-only query keeps everything
		{"MAKE AUTH", []int{0}},    // case-insensitive
	}
	for _, c := range cases {
		m.msSearchTI.SetValue(c.query)
		got := m.msFiltered()
		if !equalInts(got, c.want) {
			t.Errorf("query %q: got %v, want %v", c.query, got, c.want)
		}
	}
}

// TestMsFiltered_AndSearch checks that space-separated terms are ANDed:
// every term must hit somewhere in the item, in any field and any order.
func TestMsFiltered_AndSearch(t *testing.T) {
	makeAuth := mdl.Command{Name: "auth", Cmd: "make auth-service"}
	makeWeb := mdl.Command{Name: "web", Cmd: "make web-app"}
	npmAuth := mdl.Command{Name: "auth-ui", Cmd: "npm run auth"}
	m := msModel([]msItem{{cmd: &makeAuth}, {cmd: &makeWeb}, {cmd: &npmAuth}})

	cases := []struct {
		query string
		want  []int
	}{
		{"make auth", []int{0}},
		{"auth make", []int{0}}, // order-independent
		{"make", []int{0, 1}},
		{"auth", []int{0, 2}},
		{"make npm", nil}, // both terms required
	}
	for _, c := range cases {
		m.msSearchTI.SetValue(c.query)
		got := m.msFiltered()
		if !equalInts(got, c.want) {
			t.Errorf("query %q: got %v, want %v", c.query, got, c.want)
		}
	}
}

// TestMsFiltered_GroupMatchesInSearch checks a group name is reachable
// from the single search field, alone and combined with other terms.
func TestMsFiltered_GroupMatchesInSearch(t *testing.T) {
	a := mdl.Command{Name: "build", Group: "dev", Cmd: "make all"}
	b := mdl.Command{Name: "push", Group: "release", Cmd: "make push"}
	m := msModel([]msItem{{cmd: &a}, {cmd: &b}})

	m.msSearchTI.SetValue("dev")
	if got := m.msFiltered(); !equalInts(got, []int{0}) {
		t.Errorf("group term: got %v, want [0]", got)
	}
	m.msSearchTI.SetValue("release make")
	if got := m.msFiltered(); !equalInts(got, []int{1}) {
		t.Errorf("group+cmd terms: got %v, want [1]", got)
	}
}

// TestSlotPickFilter_AndSearch checks the slot-picker filter uses the same
// AND semantics across value and label.
func TestSlotPickFilter_AndSearch(t *testing.T) {
	s := &slotPickState{entries: []mdl.ListEntry{
		{Value: "prod-eu", Label: "Production Europe"},
		{Value: "prod-us", Label: "Production US"},
		{Value: "dev", Label: "Development"},
	}}

	s.search = "prod eu"
	s.applyFilter()
	if len(s.filtered) != 1 || s.filtered[0].Value != "prod-eu" {
		t.Errorf("query %q: got %v", s.search, s.filtered)
	}

	s.search = "europe prod"
	s.applyFilter()
	if len(s.filtered) != 1 || s.filtered[0].Value != "prod-eu" {
		t.Errorf("query %q: got %v", s.search, s.filtered)
	}

	s.search = ""
	s.applyFilter()
	if len(s.filtered) != 3 {
		t.Errorf("empty query: got %d entries, want 3", len(s.filtered))
	}
}

// TestUpdateMultiSelect_SpaceTypesTabToggles checks the fzf-style keymap:
// Space reaches the search input (needed for AND search) and Tab toggles
// the hovered item.
func TestUpdateMultiSelect_SpaceTypesTabToggles(t *testing.T) {
	a := mdl.Command{Name: "auth", Cmd: "make auth-service"}
	b := mdl.Command{Name: "web", Cmd: "make web-app"}
	m := msModel([]msItem{{cmd: &a}, {cmd: &b}})
	m.msSearchTI.Focus()

	for _, r := range "make auth" {
		key := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
		if r == ' ' {
			key.Type = tea.KeySpace
		}
		nm, _ := m.updateMultiSelect(key)
		m = nm.(Model)
	}
	if got := m.msSearchTI.Value(); got != "make auth" {
		t.Fatalf("search value = %q, want %q — space must reach the search input", got, "make auth")
	}
	if got := m.msFiltered(); !equalInts(got, []int{0}) {
		t.Fatalf("filtered = %v, want [0]", got)
	}

	nm, _ := m.updateMultiSelect(tea.KeyMsg{Type: tea.KeyTab})
	m = nm.(Model)
	if !equalInts(m.msSelected, []int{0}) {
		t.Fatalf("selected = %v, want [0] — Tab must toggle selection", m.msSelected)
	}
	nm, _ = m.updateMultiSelect(tea.KeyMsg{Type: tea.KeyTab})
	m = nm.(Model)
	if len(m.msSelected) != 0 {
		t.Fatalf("selected = %v, want empty after second Tab", m.msSelected)
	}
}

// TestUpdateMultiSelect_EscDiscardGuard checks the double-Esc guard: with
// items selected the first Esc only warns, any other key disarms it, and
// a second consecutive Esc discards the selection and leaves the screen.
// Without selections Esc leaves immediately as before.
func TestUpdateMultiSelect_EscDiscardGuard(t *testing.T) {
	a := mdl.Command{Name: "auth", Cmd: "make auth"}
	m := msModel([]msItem{{cmd: &a}})
	m.screen = ScreenRunCommands
	m.msSearchTI.Focus()

	nm, _ := m.updateMultiSelect(tea.KeyMsg{Type: tea.KeyTab})
	m = nm.(Model)

	nm, _ = m.updateMultiSelect(tea.KeyMsg{Type: tea.KeyEscape})
	m = nm.(Model)
	if m.screen != ScreenRunCommands || !m.msEscArmed || len(m.msSelected) != 1 {
		t.Fatalf("first Esc: screen=%v armed=%v selected=%v — must warn and stay", m.screen, m.msEscArmed, m.msSelected)
	}

	nm, _ = m.updateMultiSelect(tea.KeyMsg{Type: tea.KeyTab})
	m = nm.(Model)
	if m.msEscArmed {
		t.Fatal("any key other than Esc must close the discard window")
	}
	if len(m.msSelected) != 1 {
		t.Fatalf("selected = %v — a key that closes the window must not act on the list", m.msSelected)
	}

	nm, _ = m.updateMultiSelect(tea.KeyMsg{Type: tea.KeyEscape})
	m = nm.(Model)
	nm, _ = m.updateMultiSelect(tea.KeyMsg{Type: tea.KeyEscape})
	m = nm.(Model)
	if m.screen != ScreenMainMenu {
		t.Fatalf("second Esc: screen=%v, want main menu", m.screen)
	}

	m2 := msModel([]msItem{{cmd: &a}})
	m2.screen = ScreenRunCommands
	nm, _ = m2.updateMultiSelect(tea.KeyMsg{Type: tea.KeyEscape})
	m2 = nm.(Model)
	if m2.screen != ScreenMainMenu {
		t.Fatalf("no selection: screen=%v, want main menu after one Esc", m2.screen)
	}
}

// TestUpdateSlotPick_VariadicToggleAndJoin checks a {name...} slot:
// Tab toggles entries on and off, and Enter joins the picked values
// with spaces into the resolved command.
func TestUpdateSlotPick_VariadicToggleAndJoin(t *testing.T) {
	cmd := mdl.Command{Name: "up", Cmd: "docker compose up {services...}"}
	m := Model{}
	m.lists = map[string][]mdl.ListEntry{"services": {
		{Value: "api"}, {Value: "web"}, {Value: "worker"},
	}}
	nm, _ := m.startResolveFlow([]msItem{{cmd: &cmd}})
	m = nm.(Model)
	if m.screen != ScreenSlotPick || m.sp == nil || !m.sp.variadic {
		t.Fatalf("expected a variadic slot pick, got screen=%v", m.screen)
	}

	key := func(k tea.KeyMsg) {
		nm, _ = m.updateSlotPick(k)
		m = nm.(Model)
	}
	tab := tea.KeyMsg{Type: tea.KeyTab}
	key(tab)                           // toggle api
	key(tea.KeyMsg{Type: tea.KeyDown}) // → web
	key(tab)                           // toggle web
	if got := m.sp.picked; len(got) != 2 || got[0] != "api" || got[1] != "web" {
		t.Fatalf("picked = %v, want [api web]", got)
	}
	key(tab) // toggle web off again
	if got := m.sp.picked; len(got) != 1 || got[0] != "api" {
		t.Fatalf("picked = %v after un-toggle, want [api]", got)
	}
	key(tab) // and back on

	key(tea.KeyMsg{Type: tea.KeyEnter})
	if m.screen != ScreenConfirmRun {
		t.Fatalf("screen = %v, want confirm run", m.screen)
	}
	if got := m.confirmRunItems[0].Cmd.Cmd; got != "docker compose up api web" {
		t.Fatalf("resolved cmd = %q", got)
	}
}

// TestUpdateSlotPick_VariadicCustomToggle checks Tab on the custom row:
// the typed value joins the picks and the input clears for the next one.
func TestUpdateSlotPick_VariadicCustomToggle(t *testing.T) {
	cmd := mdl.Command{Name: "up", Cmd: "up {services...}"}
	m := Model{}
	m.lists = map[string][]mdl.ListEntry{"services": {{Value: "api"}}}
	nm, _ := m.startResolveFlow([]msItem{{cmd: &cmd}})
	m = nm.(Model)

	key := func(k tea.KeyMsg) {
		nm, _ = m.updateSlotPick(k)
		m = nm.(Model)
	}
	for _, r := range "db" {
		key(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	// "db" matches nothing, so the cursor sits on the custom row.
	key(tea.KeyMsg{Type: tea.KeyTab})
	if got := m.sp.picked; len(got) != 1 || got[0] != "db" {
		t.Fatalf("picked = %v, want [db]", got)
	}
	if m.sp.search != "" {
		t.Fatalf("search = %q — must clear after toggling a custom value", m.sp.search)
	}

	key(tea.KeyMsg{Type: tea.KeyTab}) // cursor is back on "api"
	key(tea.KeyMsg{Type: tea.KeyEnter})
	if got := m.confirmRunItems[0].Cmd.Cmd; got != "up db api" {
		t.Fatalf("resolved cmd = %q, want picks in toggle order", got)
	}
}

// TestUpdateSlotPick_VariadicEnterFallsBackToSingle checks that Enter with
// nothing toggled behaves exactly like a normal slot: the hovered entry wins.
func TestUpdateSlotPick_VariadicEnterFallsBackToSingle(t *testing.T) {
	cmd := mdl.Command{Name: "up", Cmd: "up {services...}"}
	m := Model{}
	m.lists = map[string][]mdl.ListEntry{"services": {{Value: "api"}, {Value: "web"}}}
	nm, _ := m.startResolveFlow([]msItem{{cmd: &cmd}})
	m = nm.(Model)

	nm, _ = m.updateSlotPick(tea.KeyMsg{Type: tea.KeyDown})
	m = nm.(Model)
	nm, _ = m.updateSlotPick(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(Model)
	if got := m.confirmRunItems[0].Cmd.Cmd; got != "up web" {
		t.Fatalf("resolved cmd = %q, want the hovered entry alone", got)
	}
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
