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

// runModel starts a run over the given commands, resolving slots the
// way Run commands does, and returns the model sitting on the first
// slot picker.
func runModel(t *testing.T, lists map[string][]mdl.ListEntry, cmds ...*mdl.Command) Model {
	t.Helper()
	m := Model{lists: lists}
	items := make([]msItem, len(cmds))
	for i, c := range cmds {
		items[i] = msItem{cmd: c}
	}
	nm, _ := m.startResolveFlow(items)
	return nm.(Model)
}

// TestSlotReuse_SecondStepOpensOnTheFirstAnswer checks the everyday case
// of clean and build asking for the same directory: the second picker
// opens on the first answer and says where it came from, but still lets
// a different value be chosen — nothing is applied silently.
func TestSlotReuse_SecondStepOpensOnTheFirstAnswer(t *testing.T) {
	lists := map[string][]mdl.ListEntry{
		"dirs": {{Value: "./app"}, {Value: "./api"}, {Value: "./web"}},
	}
	clean := mdl.Command{Name: "clean", Cmd: "make clean", Dir: "{workdir}", Slots: map[string]string{"workdir": "dirs"}}
	build := mdl.Command{Name: "build", Cmd: "make build", Dir: "{workdir}", Slots: map[string]string{"workdir": "dirs"}}
	m := runModel(t, lists, &clean, &build)

	if m.sp == nil || m.sp.reuseFrom != "" {
		t.Fatalf("the first question has no earlier answer to reuse, sp=%+v", m.sp)
	}
	nm, _ := m.acceptSlotValue("./web")
	m = nm.(Model)

	if m.sp == nil {
		t.Fatalf("the second step must ask its own question, screen=%v", m.screen)
	}
	if m.sp.reuseFrom != "clean" || m.sp.reuseValue != "./web" {
		t.Fatalf("reuse hint = %q/%q, want it to name clean and ./web", m.sp.reuseFrom, m.sp.reuseValue)
	}
	if got := m.sp.filtered[m.sp.cursor].Value; got != "./web" {
		t.Fatalf("cursor sits on %q, want the earlier answer ./web", got)
	}
	// Nothing is forced: picking another value resolves to that value.
	nm, _ = m.acceptSlotValue("./api")
	m = nm.(Model)
	if len(m.confirmRunItems) != 2 {
		t.Fatalf("both commands must resolve, got %+v", m.confirmRunItems)
	}
	if dir := m.confirmRunItems[1].Cmd.Dir; dir != "./api" {
		t.Fatalf("second command dir = %q, want the newly picked ./api", dir)
	}
}

// TestSlotReuse_ScopeAndForms checks which questions count as the same
// one: a different list under the same slot name is a different
// question, a variadic answer comes back pre-toggled, and a hand-typed
// answer comes back as the custom-value row.
func TestSlotReuse_ScopeAndForms(t *testing.T) {
	lists := map[string][]mdl.ListEntry{
		"dirs":  {{Value: "./app"}, {Value: "./api"}},
		"other": {{Value: "./x"}},
		"svcs":  {{Value: "api"}, {Value: "web"}, {Value: "worker"}},
	}

	// Same slot name, different list: no reuse offered.
	a := mdl.Command{Name: "a", Cmd: "x {target}", Slots: map[string]string{"target": "dirs"}}
	bOther := mdl.Command{Name: "b", Cmd: "y {target}", Slots: map[string]string{"target": "other"}}
	m := runModel(t, lists, &a, &bOther)
	nm, _ := m.acceptSlotValue("./app")
	m = nm.(Model)
	if m.sp.reuseFrom != "" {
		t.Fatalf("a different list is a different question, got reuse from %q", m.sp.reuseFrom)
	}

	// Variadic: the earlier picks come back toggled on.
	up1 := mdl.Command{Name: "up1", Cmd: "up {svcs...}"}
	up2 := mdl.Command{Name: "up2", Cmd: "restart {svcs...}"}
	m = runModel(t, lists, &up1, &up2)
	nm, _ = m.acceptSlotValue("api worker")
	m = nm.(Model)
	if got := m.sp.picked; len(got) != 2 || got[0] != "api" || got[1] != "worker" {
		t.Fatalf("variadic picks = %v, want the earlier answer pre-toggled", got)
	}

	// Hand-typed value: not in the list, so it returns as the custom row.
	t1 := mdl.Command{Name: "t1", Cmd: "x {target}", Slots: map[string]string{"target": "dirs"}}
	t2 := mdl.Command{Name: "t2", Cmd: "y {target}", Slots: map[string]string{"target": "dirs"}}
	m = runModel(t, lists, &t1, &t2)
	nm, _ = m.acceptSlotValue("./typed-by-hand")
	m = nm.(Model)
	if m.sp.search != "./typed-by-hand" || m.sp.cursor != len(m.sp.filtered) {
		t.Fatalf("typed answer must return as the custom row, search=%q cursor=%d filtered=%d",
			m.sp.search, m.sp.cursor, len(m.sp.filtered))
	}
	nm, _ = m.updateSlotPick(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(Model)
	if len(m.confirmRunItems) != 2 || m.confirmRunItems[1].Cmd.Cmd != "y ./typed-by-hand" {
		t.Fatalf("Enter on the custom row must keep the typed value, got %+v", m.confirmRunItems)
	}
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

// TestSetupMultiSelectWithPreSelected_KeepsStepOrder guards workflow
// editing against silent reordering: the pre-selection must follow the
// stored step order, not the command-list order, because msSelected
// order is what gets saved on Enter.
func TestSetupMultiSelectWithPreSelected_KeepsStepOrder(t *testing.T) {
	m := &Model{}
	m.config.Commands = []mdl.Command{
		{Name: "auth", Cmd: "make auth"},
		{Name: "web", Cmd: "make web"},
	}
	m.setupMultiSelectWithPreSelected([]string{"web", "auth", "ghost"})

	var names []string
	for _, idx := range m.msSelected {
		names = append(names, m.msItems[idx].name())
	}
	if len(names) != 2 || names[0] != "web" || names[1] != "auth" {
		t.Fatalf("pre-selected order = %v, want [web auth] (stored step order)", names)
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

// TestUpdateSlotPick_BackspaceEditsFilter checks that Backspace removes
// one character from the slot-pick filter (rune-safe) and refilters,
// rather than being swallowed.
func TestUpdateSlotPick_BackspaceEditsFilter(t *testing.T) {
	cmd := mdl.Command{Name: "up", Cmd: "up {env}"}
	m := Model{}
	m.lists = map[string][]mdl.ListEntry{"env": {{Value: "prod"}, {Value: "dev"}}}
	nm, _ := m.startResolveFlow([]msItem{{cmd: &cmd}})
	m = nm.(Model)

	key := func(k tea.KeyMsg) {
		nm, _ = m.updateSlotPick(k)
		m = nm.(Model)
	}
	for _, r := range "prodx" {
		key(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	if len(m.sp.filtered) != 0 {
		t.Fatalf("filtered = %v, want no matches for %q", m.sp.filtered, m.sp.search)
	}

	key(tea.KeyMsg{Type: tea.KeyBackspace})
	if m.sp.search != "prod" {
		t.Fatalf("search = %q after Backspace, want %q", m.sp.search, "prod")
	}
	if len(m.sp.filtered) != 1 || m.sp.filtered[0].Value != "prod" {
		t.Fatalf("filtered = %v — Backspace must refilter", m.sp.filtered)
	}

	// Deleting on an empty filter must be a no-op, not a crash.
	for i := 0; i < 5; i++ {
		key(tea.KeyMsg{Type: tea.KeyBackspace})
	}
	if m.sp.search != "" {
		t.Fatalf("search = %q, want empty", m.sp.search)
	}
}

// TestUpdateMultiSelect_EnterActsOnHoveredRow checks the fzf-style
// fallback: with nothing toggled, Enter runs the hovered row alone —
// respecting the active filter — and with no matches it does nothing.
func TestUpdateMultiSelect_EnterActsOnHoveredRow(t *testing.T) {
	a := mdl.Command{Name: "auth", Cmd: "make auth"}
	b := mdl.Command{Name: "web", Cmd: "make web"}
	m := msModel([]msItem{{cmd: &a}, {cmd: &b}})
	m.screen = ScreenRunCommands
	m.msSearchTI.Focus()

	nm, _ := m.updateMultiSelect(tea.KeyMsg{Type: tea.KeyDown})
	m = nm.(Model)
	nm, _ = m.updateMultiSelect(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(Model)
	if m.screen != ScreenConfirmRun {
		t.Fatalf("screen = %v, want confirm run", m.screen)
	}
	if len(m.confirmRunItems) != 1 || m.confirmRunItems[0].Name != "web" {
		t.Fatalf("confirm items = %v, want just the hovered command", m.confirmRunItems)
	}

	// No matches: Enter must stay put.
	m2 := msModel([]msItem{{cmd: &a}})
	m2.screen = ScreenRunCommands
	m2.msSearchTI.SetValue("zzz")
	nm, _ = m2.updateMultiSelect(tea.KeyMsg{Type: tea.KeyEnter})
	m2 = nm.(Model)
	if m2.screen != ScreenRunCommands {
		t.Fatalf("screen = %v — Enter on an empty result list must do nothing", m2.screen)
	}
}

// TestUpdateMultiSelect_CreateWorkflowHoveredRow checks the same fallback
// applies to workflow creation: Enter with nothing toggled proceeds to the
// name input with the hovered command as the single step.
func TestUpdateMultiSelect_CreateWorkflowHoveredRow(t *testing.T) {
	a := mdl.Command{Name: "auth", Cmd: "make auth"}
	m := msModel([]msItem{{cmd: &a}})
	m.nameInput = textinput.New()
	m.screen = ScreenCreateWorkflow

	nm, _ := m.updateMultiSelect(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(Model)
	if m.screen != ScreenNameInput {
		t.Fatalf("screen = %v, want the name input", m.screen)
	}
	if len(m.pendingWorkflowCmds) != 1 || m.pendingWorkflowCmds[0] != "auth" {
		t.Fatalf("pending commands = %v, want the hovered command", m.pendingWorkflowCmds)
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
