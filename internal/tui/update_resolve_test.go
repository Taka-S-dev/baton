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
// name/group: the command body, template values, and alias steps/vars.
func TestMsFiltered_MatchesEmbeddedText(t *testing.T) {
	build := mdl.Command{Name: "build", Group: "dev", Cmd: "make auth-service"}
	deploy := mdl.Command{Name: "deploy", Template: "deploy-tpl", Values: map[string]string{"env": "prod"}}
	al := mdl.Alias{Name: "release", Steps: []string{"build", "deploy"},
		Vars: map[string]map[string]string{"deploy": {"env": "staging"}}}
	m := msModel([]msItem{{cmd: &build}, {cmd: &deploy}, {alias: &al}})

	cases := []struct {
		query string
		want  []int
	}{
		{"auth", []int{0}},         // command body
		{"prod", []int{1}},         // template value
		{"staging", []int{2}},      // alias var value
		{"deploy", []int{1, 2}},    // name + alias step
		{"deploy-tpl", []int{1}},   // template name
		{"nothing-matches-x", nil}, // no hit
		{"", []int{0, 1, 2}},       // empty query keeps everything
		{"  ", []int{0, 1, 2}},     // whitespace-only query keeps everything
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
