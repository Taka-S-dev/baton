package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	mdl "github.com/Taka-S-dev/baton/internal/model"
)

// exploreProject is a project with one of everything the workspace has
// to show, including a workflow step whose command is gone.
func exploreProject() Model {
	m := Model{exploreTab: tabCommands, width: 110, height: 30, screen: ScreenExplore, projectDir: "demo", configFile: "commands.tsv"}
	m.config = mdl.Config{Base: []mdl.Command{
		{Name: "build", Group: "make", Cmd: "make build", Dir: `{$root}\api`, Source: "tsv"},
		{Name: "test", Group: "make", Cmd: "go test ./...", Source: "tsv"},
		{Name: "deploy", Group: "deploy", Cmd: "deploy --env {env}", Source: "tsv"},
		{Name: "hello", Cmd: "echo hello", Source: "tsv"},
	}}
	m.config.Commands = []mdl.Command{{Name: "build-api", Template: "build", Cmd: "make build", Source: "local"}}
	m.workflows = []mdl.Workflow{
		{Name: "release", Commands: []string{"build", "test", "gone"}},
		{Name: "smoke", Commands: []string{"hello"}},
	}
	m.lists = map[string][]mdl.ListEntry{"env": {{Value: "dev"}, {Value: "prod", Label: "production"}}}
	m.vars = map[string]string{"root": `C:\work\Phase1`, "build-api.projDir": `Z:\api`}
	m.nameInput = textinput.New()
	return m
}

func exploreModel(tm tea.Model) Model {
	if p, ok := tm.(*Model); ok {
		return *p
	}
	return tm.(Model)
}

func exploreFirst(tm tea.Model, _ tea.Cmd) tea.Model { return tm }

func press(m Model, k tea.KeyMsg) Model {
	return exploreModel(exploreFirst(m.updateExplore(k)))
}

func typeRunes(m Model, s string) Model {
	for _, r := range s {
		m = press(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	return m
}

func clickAt(m Model, x, y int) Model {
	return exploreModel(exploreFirst(m.updateExploreMouse(tea.MouseMsg{
		X: x, Y: y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft,
	})))
}

func mouseAt(m Model, x, y int, a tea.MouseAction, b tea.MouseButton) Model {
	return exploreModel(exploreFirst(m.updateExploreMouse(tea.MouseMsg{X: x, Y: y, Action: a, Button: b})))
}

// stripStyle removes the ANSI escapes so a test can assert on layout.
func stripStyle(s string) string {
	var b strings.Builder
	in := false
	for _, r := range s {
		switch {
		case r == 0x1b:
			in = true
		case in && r == 'm':
			in = false
		case !in:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// rowOf is the screen row whose text contains label, or -1. Reading the
// row out of the rendered view, rather than repeating the arithmetic the
// view used, is the only way a test can catch the two drifting apart.
func rowOf(view, label string) int {
	for i, ln := range strings.Split(stripStyle(view), "\n") {
		if strings.Contains(ln, label) {
			return i
		}
	}
	return -1
}

// TestExploreTabsHoldOneKindEach is the shape of the workspace: a tab
// per kind of thing, so a row in the list is always something you can
// act on, and never a label standing among things.
func TestExploreTabsHoldOneKindEach(t *testing.T) {
	m := exploreProject()
	want := map[int][]string{
		tabCommands:  {"deploy", "build", "test", "build-api", "hello"},
		tabWorkflows: {"release", "smoke"},
		tabLists:     {"env"},
		tabVars:      {"root"},
	}
	for tab, names := range want {
		var got []string
		for _, it := range m.exploreItems(tab) {
			got = append(got, it.name)
		}
		if strings.Join(got, ",") != strings.Join(names, ",") {
			t.Errorf("tab %s lists %v, want %v", tabNames[tab], got, names)
		}
	}
	// A group is a tag on the command, and the order keeps a group's
	// commands together, which is what a level for it would have done.
	items := m.exploreItems(tabCommands)
	if !strings.Contains(items[1].label, "[make]") || !strings.Contains(items[2].label, "[make]") {
		t.Errorf("a group's commands are not adjacent and tagged: %+v", items)
	}
}

// TestExploreTabSwitching checks the ways between tabs: the number
// keys, the arrows, and a click on the label — and that each tab keeps
// its own place, so coming back lands where you were.
func TestExploreTabSwitching(t *testing.T) {
	m := exploreProject()
	m = press(m, tea.KeyMsg{Type: tea.KeyDown})
	if m.exploreCursor[tabCommands] != 1 {
		t.Fatalf("the cursor did not move on the commands tab")
	}
	m = press(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	if m.exploreTab != tabWorkflows {
		t.Fatalf("1 did not open the workflows tab")
	}
	m = press(m, tea.KeyMsg{Type: tea.KeyRight})
	m = press(m, tea.KeyMsg{Type: tea.KeyRight})
	if m.exploreTab != tabLists {
		t.Errorf("→ did not move to the next tab")
	}
	m = press(m, tea.KeyMsg{Type: tea.KeyLeft})
	if m.exploreTab != tabCommands || m.exploreCursor[tabCommands] != 1 {
		t.Errorf("coming back did not land where the tab was left: tab=%d cursor=%d", m.exploreTab, m.exploreCursor[tabCommands])
	}

	l := m.exploreLayout(m.viewWidth())
	y := rowOf(m.View(), "[2 Commands]")
	if y != l.tabY {
		t.Fatalf("the tab line is drawn on row %d but the layout says %d", y, l.tabY)
	}
	clicked := clickAt(m, l.tabX[tabVars][0]+2, y)
	if clicked.exploreTab != tabVars {
		t.Errorf("clicking the Variables label opened tab %d", clicked.exploreTab)
	}
	if beside := clickAt(m, l.tabX[tabCommands][1]+1, y); beside.exploreTab != tabCommands {
		t.Errorf("a click between labels changed the tab")
	}
}

// TestExploreSearchNarrowsTheList checks / and what it does: the list
// shrinks to what matches, the arrows still move through it, Esc
// clears it, and switching tabs drops it.
func TestExploreSearchNarrowsTheList(t *testing.T) {
	m := exploreProject()
	m = press(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	if !m.exploreSearching {
		t.Fatal("/ did not open the search")
	}
	m = typeRunes(m, "make")
	items := m.exploreVisible()
	if len(items) != 3 { // build and test by group, build-api by its "$ make build"
		t.Fatalf("searching make left %d rows, want 3: %+v", len(items), items)
	}
	if !strings.Contains(stripStyle(m.View()), "Commands (3/5)") {
		t.Errorf("the box title does not say how many of how many matched\n%s", m.View())
	}
	m = press(m, tea.KeyMsg{Type: tea.KeyDown})
	if m.exploreSearching || m.exploreCursor[tabCommands] != 0 {
		t.Fatal("Down must focus current result")
	}
	m = press(m, tea.KeyMsg{Type: tea.KeyDown})
	if m.exploreCursor[tabCommands] != 1 {
		t.Error("the arrows do not move through the matches while searching")
	}
	m = press(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.exploreSearching || m.exploreSearch != "make" {
		t.Error("Enter should close the search and keep the filter")
	}
	m = press(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.exploreSearch != "" || len(m.exploreVisible()) != 5 {
		t.Error("Esc did not clear the filter")
	}

	// While the search is open a digit is search text, so the search is
	// closed first and the filter left standing; then the tab changes.
	m = typeRunes(press(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")}), "env")
	m = press(m, tea.KeyMsg{Type: tea.KeyEnter})
	m = press(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	if m.exploreTab != tabLists || m.exploreSearch != "" {
		t.Errorf("a search typed over one tab followed to another: tab=%d search=%q", m.exploreTab, m.exploreSearch)
	}
}

// TestExploreEditsInTheBox checks the edit cycle on a terminal with room
// for it: Enter puts the form in the editor box, the row being written
// shows in the list as it is typed, nothing is saved until Ctrl+S, and
// Esc comes back to the list with the form gone.
func TestExploreEditsInTheBox(t *testing.T) {
	m := exploreProject()
	m = press(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	if m.cf == nil || m.screen != ScreenExplore {
		t.Fatalf("n did not open the form in place: cf=%v screen=%v", m.cf, m.screen)
	}
	if m.cf.fieldIdx != 1 {
		t.Fatalf("the form opened on field %d, want the cmd", m.cf.fieldIdx)
	}
	key := func(k tea.KeyMsg) { m = exploreModel(exploreFirst(m.updateCommandForm(k))) }
	for _, r := range "go test ./{env}/" {
		key(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	key(tea.KeyMsg{Type: tea.KeyTab}) // name, shown with its suggestion in place
	if got := m.nameInput.Placeholder; got != "go-test" {
		t.Fatalf("the empty name field suggests %q, want a name derived from the cmd", got)
	}
	key(tea.KeyMsg{Type: tea.KeyRight}) // takes it
	if got := m.nameInput.Value(); got != "go-test" {
		t.Fatalf("→ took %q, want the suggestion", got)
	}
	key(tea.KeyMsg{Type: tea.KeyDown}) // workdir
	key(tea.KeyMsg{Type: tea.KeyDown}) // group
	for _, r := range "make" {
		key(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	view := stripStyle(m.View())
	for _, want := range []string{"New command", "go-test", "Commands (6)", "{env} → env"} {
		if !strings.Contains(view, want) {
			t.Errorf("while writing, the screen does not show %q\n%s", want, view)
		}
	}
	var draft *wsItem
	for _, it := range m.exploreVisible() {
		if it.name == "go-test" {
			d := it
			draft = &d
		}
	}
	if draft == nil || !draft.draft {
		t.Fatalf("the row being written is not in the list as a draft: %+v", draft)
	}
	if _, ok := m.config.FindCommand("go-test"); ok {
		t.Error("the draft was saved before Ctrl+S")
	}

	// Esc asks about unsaved changes immediately, from any field.
	key(tea.KeyMsg{Type: tea.KeyEsc})
	if m.leaveAsk != leaveForm {
		t.Fatalf("Esc out of a form with changes did not ask what to do with them")
	}
	key(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	if m.cf != nil || m.screen != ScreenExplore {
		t.Errorf("discarding landed on %v with cf=%v", m.screen, m.cf)
	}
}

// TestExploreEditsFullScreenWhenNarrow checks the other arrangement:
// without room for the right side, Enter gives the form the screen,
// and closing it comes back here.
func TestExploreEditsFullScreenWhenNarrow(t *testing.T) {
	m := exploreProject()
	m.width, m.height = 60, 20
	if !m.exploreLayout(m.viewWidth()).single {
		t.Fatal("60 columns should be too narrow for the right side")
	}
	m = press(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.cf == nil || m.screen != ScreenCommandForm {
		t.Fatalf("Enter on a narrow terminal did not open the full-screen form: screen=%v", m.screen)
	}
	if m.cf.returnTo != ScreenExplore {
		t.Error("the full-screen form would not come back to the overview")
	}
}

// TestExploreFieldsSitWhereTheyLook is what makes a click on a field
// mean "edit this one": browsing and editing draw the fields on the
// same rows, and a click on a row — found by reading the view — puts
// the caret in that field.
func TestExploreFieldsSitWhereTheyLook(t *testing.T) {
	m := exploreProject()
	browsing := m.View()
	editing := press(m, tea.KeyMsg{Type: tea.KeyEnter})
	if editing.cf == nil {
		t.Fatal("Enter did not open the form")
	}
	for _, label := range []string{"Name >", "Workdir >", "Group >", "Shell >"} {
		if a, b := rowOf(browsing, label), rowOf(editing.View(), label); a != b || a < 0 {
			t.Errorf("%q is on row %d browsing and %d editing", label, a, b)
		}
	}

	l := m.exploreLayout(m.viewWidth())
	for _, i := range formOrder {
		label := commandFormPaneLabels[i]
		y := rowOf(m.View(), label+" >")
		if y < 0 {
			t.Fatalf("%q is not drawn\n%s", label, m.View())
		}
		got := clickAt(m, l.rightX+4, y)
		if got.cf == nil || got.cf.fieldIdx != i {
			t.Errorf("clicking the %q row focused %+v, want field %d", label, got.cf, i)
		}
	}
}

// TestExploreClickHitsTheRowTheViewDrew walks every list row the view
// drew and clicks it, proving the handler and the view agree on where
// rows are. A second click on the selected row opens it, and a click on
// the list while the form is open leaves the form the way Esc does.
func TestExploreClickHitsTheRowTheViewDrew(t *testing.T) {
	m := exploreProject()
	l := m.exploreLayout(m.viewWidth())
	for i, it := range l.items {
		y := -1
		for lineNo, line := range strings.Split(stripStyle(m.View()), "\n") {
			if strings.Contains(truncate(line, l.leftW), it.name+"  ") {
				y = lineNo
				break
			}
		}
		if y < 0 {
			t.Fatalf("row %q is not drawn", it.label)
		}
		if y != l.listY+i-l.start {
			t.Errorf("%q is drawn on row %d but the layout puts it on %d", it.label, y, l.listY+i-l.start)
		}
		if got := clickAt(m, 4, y); got.exploreCursor[tabCommands] != i {
			t.Errorf("clicking %q selected %d", it.label, got.exploreCursor[tabCommands])
		}
	}
	// Above the list and below it, a click changes nothing.
	for _, y := range []int{0, l.listY - 2, l.listY + l.listH + 1} {
		if got := clickAt(m, 4, y); got.exploreCursor != m.exploreCursor {
			t.Errorf("a click at y=%d outside the rows moved the cursor", y)
		}
	}

	m = clickAt(m, 4, l.listY+1)
	m = clickAt(m, 4, l.listY+1)
	if m.cf == nil || m.cf.origName != l.items[1].name {
		t.Fatalf("a second click on the selected row did not open it: %+v", m.cf)
	}
	away := clickAt(m, 4, l.listY)
	if away.cf != nil || away.exploreCursor[tabCommands] != 0 {
		t.Errorf("a click on the list while editing did not leave the form and select the row: cf=%v cursor=%d", away.cf, away.exploreCursor[tabCommands])
	}
}

// TestExploreWheelAndDrag checks the wheel moves the cursor, stays off
// the list while the form has the keys, and that the split follows a
// drag of the column between the boxes without letting go until the
// button comes up — and cannot be dragged to where a box has no width.
func TestExploreWheelAndDrag(t *testing.T) {
	m := exploreProject()
	m.exploreCursor[tabCommands] = 4
	up := mouseAt(m, 0, 0, tea.MouseActionPress, tea.MouseButtonWheelUp)
	if up.exploreCursor[tabCommands] != 4-wheelRows {
		t.Errorf("wheel up moved the cursor to %d", up.exploreCursor[tabCommands])
	}
	editing := press(m, tea.KeyMsg{Type: tea.KeyEnter})
	if got := mouseAt(editing, 0, 0, tea.MouseActionPress, tea.MouseButtonWheelUp); got.exploreCursor != editing.exploreCursor {
		t.Error("the wheel moved the list while the form was open")
	}

	l := m.exploreLayout(m.viewWidth())
	m = mouseAt(m, l.leftW, l.listY, tea.MouseActionPress, tea.MouseButtonLeft)
	if !m.exploreDragging {
		t.Fatal("pressing the column between the boxes did not start a drag")
	}
	m = mouseAt(m, 50, l.listY+5, tea.MouseActionMotion, tea.MouseButtonLeft)
	if got := m.exploreLayout(m.viewWidth()).leftW; got != 50 {
		t.Errorf("the list is %d wide after dragging to 50", got)
	}
	m = mouseAt(m, 50, l.listY+5, tea.MouseActionRelease, tea.MouseButtonLeft)
	if m.exploreDragging {
		t.Error("releasing did not end the drag")
	}
	for _, x := range []int{0, m.width + 10} {
		m.exploreSplit = x
		got := m.exploreLayout(m.viewWidth())
		if got.leftW < exploreMinList || got.rightW < exploreMinRight {
			t.Errorf("a split at %d left boxes of %d and %d", x, got.leftW, got.rightW)
		}
	}
}

// TestExplorePreviewSaysWhatPointsAtIt is the reference walk, read in
// the preview rather than by seeing two lists at once.
func TestExplorePreviewSaysWhatPointsAtIt(t *testing.T) {
	m := exploreProject()
	m.exploreCursor[tabCommands] = 1 // build
	view := stripStyle(m.View())
	for _, want := range []string{`workflow "release" step 1`, `derived command "build-api"`, "commands.tsv", `C:\work\Phase1\api`} {
		if !strings.Contains(view, want) {
			t.Errorf("the preview for build does not show %q\n%s", want, view)
		}
	}
	m.exploreTab = tabLists
	if view := stripStyle(m.View()); !strings.Contains(view, `command "deploy", {env}`) {
		t.Errorf("the preview for a list does not name the command drawing on it\n%s", view)
	}
	m.exploreTab = tabVars
	if view := stripStyle(m.View()); !strings.Contains(view, `command "build"`) {
		t.Errorf("the preview for a global does not name what resolves through it\n%s", view)
	}
	m.exploreTab = tabWorkflows
	if view := stripStyle(m.View()); !strings.Contains(view, "(not found)") || !strings.Contains(view, "$ make build") {
		t.Errorf("a workflow does not show its steps and what they run\n%s", view)
	}
}

// TestExploreEmptyTabExplainsItself checks a tab with nothing in it
// says what the kind of thing is and how to get one, which is exactly
// when a newcomer needs to be told.
func TestExploreEmptyTabExplainsItself(t *testing.T) {
	m := exploreProject()
	m.lists = map[string][]mdl.ListEntry{}
	m.exploreTab = tabLists
	view := stripStyle(m.View())
	if !strings.Contains(view, "{slot}") || !strings.Contains(view, "none yet") {
		t.Errorf("an empty tab does not explain itself\n%s", view)
	}
}

// TestExploreBoxesKeepTheirSize is the invariant behind drawing boxes
// at all: each is exactly the size the layout gave it whatever is in
// it, so the frame never outgrows the terminal.
func TestExploreBoxesKeepTheirSize(t *testing.T) {
	for _, h := range []int{3, 5, 12} {
		for _, content := range []string{"", "one line", strings.Repeat("a line\n", 40), strings.Repeat("x", 200)} {
			got := box("T", content, 30, h, true)
			lines := strings.Split(got, "\n")
			if len(lines) != h {
				t.Errorf("box of height %d rendered %d lines", h, len(lines))
			}
			for _, ln := range lines {
				if w := lipgloss.Width(ln); w != 30 {
					t.Errorf("a box line is %d wide, want 30: %q", w, stripStyle(ln))
				}
			}
		}
	}
	if box("x", "", 30, 1, false) != "" {
		t.Error("a box too short to have an inside should render nothing")
	}
}

// TestExploreSelectedRowIsABarAcrossTheBox: a marker says which row, a
// bar says which row and how far the box reaches, and it dims when the
// editor beside it has the keys.
func TestExploreSelectedRowIsABarAcrossTheBox(t *testing.T) {
	m := exploreProject()
	l := m.exploreLayout(m.viewWidth())
	inner := l.leftW - 2
	row := exploreRow(l.items[0], true, true, inner)
	if got := lipgloss.Width(row); got != inner {
		t.Errorf("the bar is %d wide, want the box's %d", got, inner)
	}
	if lipgloss.Width(exploreRow(l.items[0], false, true, inner)) >= inner {
		t.Error("an unselected row was padded to the full width")
	}
	if sRowActive.Render("x") == sRowInactive.Render("x") {
		t.Skip("no colour profile here: the two bars render identically")
	}
	if exploreRow(l.items[0], true, false, inner) == row {
		t.Error("the bar looks the same whether or not the list has the keys")
	}
}

func formKey(m Model, k tea.KeyMsg) Model {
	return exploreModel(exploreFirst(m.updateCommandForm(k)))
}

// TestFormTabIsTheNextField holds the one meaning Tab has on every
// field: the next one, with Shift+Tab the previous. It used to insert a
// placeholder on two fields and suggest a name on a third, which is
// three things to learn for one key.
func TestFormTabIsTheNextField(t *testing.T) {
	m := press(exploreProject(), tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	want := []int{1, 0, 2, 3, 4, 4} // cmd, name, workdir, group, shell, and it stops there
	for i, field := range want {
		if m.cf.fieldIdx != field {
			t.Fatalf("after %d Tabs the caret is on field %d, want %d", i, m.cf.fieldIdx, field)
		}
		m = formKey(m, tea.KeyMsg{Type: tea.KeyTab})
	}
	m = formKey(m, tea.KeyMsg{Type: tea.KeyShiftTab})
	if m.cf.fieldIdx != 3 {
		t.Errorf("Shift+Tab from the shell went to field %d, want the group", m.cf.fieldIdx)
	}
	// A value typed before Tab is kept.
	m = formKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	m = formKey(m, tea.KeyMsg{Type: tea.KeyTab})
	if m.cf.fields[3] != "x" {
		t.Errorf("Tab dropped what was typed in the field: %q", m.cf.fields[3])
	}
}

// TestFormBraceOpensTheCandidates is the placeholder picker as a
// completion: { opens it under the field, what is typed after the brace
// narrows it, Enter puts the chosen placeholder in place of the brace,
// and Esc leaves what was typed alone.
func TestFormBraceOpensTheCandidates(t *testing.T) {
	m := press(exploreProject(), tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	typ := func(s string) {
		for _, r := range s {
			m = formKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		}
	}
	typ("deploy --env ")
	if m.cf.pickOpen {
		t.Fatal("the popup opened without a brace")
	}
	typ("{")
	if !m.cf.pickOpen {
		t.Fatal("{ did not open the candidates")
	}
	cands := m.pickCandidates(m.cf.pickTyped(m.nameInput))
	if len(cands) != 8 { // list, variadic list, global, and five reserved path forms
		t.Fatalf("candidates = %+v, want list, variadic list, global and reserved path forms", cands)
	}
	view := stripStyle(m.View())
	for _, want := range []string{"{env}", "pick at run time · 2 choices", "{env...}", "pick several", "{$root}", `fixed · C:\work\Phase1`} {
		if !strings.Contains(view, want) {
			t.Errorf("the popup does not show %q\n%s", want, view)
		}
	}

	typ("$")
	if got := m.pickCandidates(m.cf.pickTyped(m.nameInput)); len(got) != 1 || got[0].text != "{$root}" {
		t.Errorf("typing $ after the brace left %+v, want only the global", got)
	}
	m = formKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.cf.pickOpen {
		t.Error("Enter did not close the popup")
	}
	if got := m.nameInput.Value(); got != "deploy --env {$root}" {
		t.Errorf("Enter wrote %q, want the placeholder in place of the brace and what followed it", got)
	}
	if m.nameInput.Position() != len([]rune("deploy --env {$root}")) {
		t.Errorf("the caret is at %d, want after the placeholder", m.nameInput.Position())
	}

	// Esc keeps what was typed, and deleting the brace closes the popup.
	typ(" {en")
	m = formKey(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.cf.pickOpen || !strings.HasSuffix(m.nameInput.Value(), " {en") {
		t.Errorf("Esc did not leave the text alone: open=%v value=%q", m.cf.pickOpen, m.nameInput.Value())
	}
	typ("{")
	for i := 0; i < 2; i++ {
		m = formKey(m, tea.KeyMsg{Type: tea.KeyBackspace})
	}
	if m.cf.pickOpen {
		t.Error("deleting the brace left the popup open")
	}

	// On the name field a brace is only a brace.
	m = formKey(m, tea.KeyMsg{Type: tea.KeyTab})
	typ("{")
	if m.cf.pickOpen {
		t.Error("the popup opened on the name field")
	}
}

// TestFormCandidatesAlsoShowFullScreen checks the popup draws under the
// fields on the form's own screen too, which is where a narrow terminal
// edits.
func TestFormCandidatesAlsoShowFullScreen(t *testing.T) {
	m := exploreProject()
	m.width, m.height = 60, 24
	m = press(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.screen != ScreenCommandForm {
		t.Fatalf("a narrow terminal should edit full screen, got %v", m.screen)
	}
	m = formKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("{")})
	view := stripStyle(m.View())
	if !strings.Contains(view, "{env}") || !strings.Contains(view, "pick at run time") {
		t.Errorf("the full-screen form does not show the candidates\n%s", view)
	}
	if !strings.Contains(view, "Enter Insert") {
		t.Errorf("the key line does not say what the keys do while the popup is open\n%s", view)
	}
}

// TestFormPopupStaysInsideTheTerminal holds the frame invariant with the
// candidates open: they take rows under the fields, and on both the box
// and the full-screen form those rows come out of what the terminal has,
// never out of the bottom of it. The project here has enough lists and
// globals that the popup has more to show than it ever has room for.
func TestFormPopupStaysInsideTheTerminal(t *testing.T) {
	build := func(w, h int) Model {
		m := exploreProject()
		for _, n := range []string{"a", "b", "c", "d", "e", "f"} {
			m.lists[n] = []mdl.ListEntry{{Value: "x"}}
			m.vars[n] = "v"
		}
		m.width, m.height = w, h
		m = press(m, tea.KeyMsg{Type: tea.KeyEnter})
		return formKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("{")})
	}
	for h := 13; h <= 40; h++ {
		for _, w := range []int{60, 110} {
			m := build(w, h)
			if !m.cf.pickOpen {
				t.Fatalf("%dx%d: the popup did not open", w, h)
			}
			if got := frameLines(m.View()); got > h {
				t.Errorf("%dx%d (%v): the frame is %d lines with the popup open", w, h, m.screen, got)
			}
			// Where there is room for the window, the count of what is
			// off it is drawn; where there is not, nothing is lost silently.
			if h >= 24 && !strings.Contains(stripStyle(m.View()), "more") {
				t.Errorf("%dx%d: more candidates than fit, but no count of what is off the window\n%s", w, h, m.View())
			}
		}
	}
}

func space(m Model) Model { return press(m, tea.KeyMsg{Type: tea.KeySpace, Runes: []rune(" ")}) }

// TestExploreSpaceShowsWhatCanBeDone is the menu as a way to learn the
// keys: it lists the actions for the kind of thing under the cursor with
// the key beside each, and either the arrows and Enter or the key itself
// runs one. One table drives both, so they cannot disagree.
func TestExploreSpaceShowsWhatCanBeDone(t *testing.T) {
	m := space(exploreProject())
	if !m.exploreMenu {
		t.Fatal("Space did not open the menu")
	}
	view := stripStyle(m.View())
	for _, want := range []string{"Actions · deploy", "e  Edit", "d  Duplicate", "r  Rename", "x  Delete"} {
		if !strings.Contains(view, want) {
			t.Errorf("the menu does not show %q\n%s", want, view)
		}
	}
	if !strings.Contains(view, "Enter Do it") {
		t.Errorf("the key line does not say the menu has the keys\n%s", view)
	}

	// Each kind has its own actions, and a tab's menu reads the kind.
	for tab, want := range map[int]string{tabWorkflows: "Edit steps", tabLists: "Edit entries", tabVars: "Edit value"} {
		n := exploreProject()
		n.exploreTab = tab
		if got := stripStyle(space(n).View()); !strings.Contains(got, want) {
			t.Errorf("the %s menu does not offer %q\n%s", tabNames[tab], want, got)
		}
	}

	// Esc and Space close it; anything that is not an action closes it
	// too rather than being swallowed.
	if got := press(m, tea.KeyMsg{Type: tea.KeyEsc}); got.exploreMenu {
		t.Error("Esc did not close the menu")
	}
	if got := press(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}); got.exploreMenu {
		t.Error("a key that is not an action left the menu open")
	}

	// The arrows and Enter run the chosen action; the key runs it directly.
	byArrows := press(press(m, tea.KeyMsg{Type: tea.KeyDown}), tea.KeyMsg{Type: tea.KeyEnter})
	if byArrows.cf == nil || byArrows.cf.mode != 0 || byArrows.cf.fields[1] != "deploy --env {env}" {
		t.Errorf("↓ Enter did not duplicate: cf=%+v", byArrows.cf)
	}
	byKey := press(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	if byKey.cf == nil || byKey.cf.fields[1] != "deploy --env {env}" {
		t.Errorf("d in the menu did not duplicate: cf=%+v", byKey.cf)
	}
	if byKey.exploreMenu {
		t.Error("running an action left the menu open")
	}
	// The copy opens on the name, left to give, with a free one shown.
	if byKey.cf.fieldIdx != 0 || byKey.nameInput.Placeholder != "deploy-2" {
		t.Errorf("the copy opened on field %d with %q suggested, want the name field and deploy-2", byKey.cf.fieldIdx, byKey.nameInput.Placeholder)
	}
}

// TestExploreActionKeysWorkWithoutTheMenu checks the keys the menu
// teaches work on the list directly, and land each flow on the step
// that matters with the item already chosen.
func TestExploreActionKeysWorkWithoutTheMenu(t *testing.T) {
	m := exploreProject()
	if got := press(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")}); got.screen != ScreenDeleteCommand || !got.deleteConfirm || len(got.deleteSelected) != 1 {
		t.Errorf("x did not open the delete confirmation on the command: screen=%v confirm=%v selected=%v", got.screen, got.deleteConfirm, got.deleteSelected)
	}
	// A TSV row is renamed through the form, opened on its name field:
	// saving is the one path that rewrites the row and moves every
	// reference with it.
	if got := press(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")}); got.cf == nil || !got.cf.tsvEdit || got.cf.origName != "deploy" || got.cf.fieldIdx != 0 {
		t.Errorf("r did not open the form on the command's name: cf=%+v", got.cf)
	}
	// A command of the app-managed layer has a rename of its own.
	d := exploreProject()
	d.exploreCursor[tabCommands] = 3 // build-api, derived
	if got := press(d, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")}); got.screen != ScreenNameInput || got.nameInputMode != nameInputRenameCommand || got.nameInput.Value() != "build-api" {
		t.Errorf("r on a derived command did not open its rename: screen=%v mode=%v value=%q", got.screen, got.nameInputMode, got.nameInput.Value())
	}

	w := exploreProject()
	w.exploreTab = tabWorkflows
	if got := press(w, tea.KeyMsg{Type: tea.KeyEnter}); got.screen != ScreenExplore || got.wfe == nil || len(got.wfe.steps) != 3 {
		t.Errorf("Enter on a workflow did not open its steps in the box: screen=%v wfe=%+v", got.screen, got.wfe)
	}
	if got := press(w, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")}); got.screen != ScreenDeleteWorkflow || !got.deleteConfirm || got.deleteSelected[0] != 0 {
		t.Errorf("x on a workflow did not open its delete confirmation: screen=%v selected=%v", got.screen, got.deleteSelected)
	}

	l := exploreProject()
	l.exploreTab = tabLists
	if got := press(l, tea.KeyMsg{Type: tea.KeyEnter}); got.screen != ScreenEditList || got.le == nil || got.le.name != "env" {
		t.Errorf("Enter on a list did not open its entries: screen=%v le=%+v", got.screen, got.le)
	}

	v := exploreProject()
	v.exploreTab = tabVars
	v.exploreCursor[tabVars] = 0 // root
	if got := press(v, tea.KeyMsg{Type: tea.KeyEnter}); got.screen != ScreenVarForm || got.ve == nil || got.ve.name != "root" {
		t.Errorf("Enter on a variable did not open its value: screen=%v ve=%+v", got.screen, got.ve)
	}
	if got := press(v, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")}); got.screen != ScreenDeleteVar || !got.deleteConfirm {
		t.Errorf("x on a variable did not open its delete confirmation: screen=%v", got.screen)
	}
}

// TestFlowsLaunchedHereComeBackHere checks the way back: a flow that
// would return to its own menu returns to the overview when it was
// started from it, and only then — reached from the menu it still goes
// back to the menu.
func TestFlowsLaunchedHereComeBackHere(t *testing.T) {
	// Backing out of the delete confirmation, then out of the screen.
	m := press(exploreProject(), tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	m = exploreModel(exploreFirst(m.updateDeleteCommand(tea.KeyMsg{Type: tea.KeyEsc})))
	m = exploreModel(exploreFirst(m.updateDeleteCommand(tea.KeyMsg{Type: tea.KeyEsc})))
	if m.screen != ScreenExplore || m.exploreLaunched {
		t.Errorf("leaving the delete screen landed on %v (launched=%v), want the overview", m.screen, m.exploreLaunched)
	}

	// Finishing a rename.
	r := press(exploreProject(), tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	// The rename rewrites the row in the project's TSV, so the project
	// has to be on disk for it to take.
	r.projectDir = t.TempDir()
	tsv := "name\tgroup\tworkdir\tcmd\tshell\tslots\n" +
		"build\tmake\t{$root}\\api\tmake build\t\t\n" +
		"test\tmake\t\tgo test ./...\t\t\n" +
		"deploy\tdeploy\t\tdeploy --env {env}\t\t\n" +
		"hello\t\t\techo hello\t\t\n"
	if err := os.WriteFile(filepath.Join(r.projectDir, "commands.tsv"), []byte(tsv), 0o644); err != nil {
		t.Fatal(err)
	}
	r.nameInput.SetValue("deploy2")
	r = formKey(r, tea.KeyMsg{Type: tea.KeyCtrlS})
	if r.screen != ScreenExplore || r.cf != nil {
		t.Errorf("finishing a rename landed on %v (cf=%v, err=%q), want the overview", r.screen, r.cf, r.errMsg)
	}
	if _, ok := r.config.FindCommand("deploy2"); !ok {
		t.Error("the rename did not take")
	}

	// The same flow reached from the manage menu still returns there.
	n := exploreProject()
	n.screen = ScreenManageCommands
	n.listItems = []string{"Create command", "Edit command", "Delete command"}
	n.listCursor = 2
	n = exploreModel(exploreFirst(n.updateManageCommands(tea.KeyMsg{Type: tea.KeyEnter})))
	n = exploreModel(exploreFirst(n.updateDeleteCommand(tea.KeyMsg{Type: tea.KeyEsc})))
	if n.screen != ScreenManageCommands {
		t.Errorf("from the manage menu, leaving the delete screen landed on %v", n.screen)
	}
}

// TestExploreMenuRowsAreClickable: with the menu in the editor box, a
// click on a row — found by reading the view — runs that action, and a
// click anywhere else closes the menu.
func TestExploreMenuRowsAreClickable(t *testing.T) {
	m := space(exploreProject())
	l := m.exploreLayout(m.viewWidth())
	y := rowOf(m.View(), "x  Delete")
	if y < 0 {
		t.Fatalf("the delete row is not drawn\n%s", m.View())
	}
	got := clickAt(m, l.rightX+4, y)
	if got.screen != ScreenDeleteCommand || !got.deleteConfirm {
		t.Errorf("clicking the delete row did not run it: screen=%v", got.screen)
	}
	if closed := clickAt(m, 4, l.listY+1); closed.exploreMenu {
		t.Error("a click outside the menu left it open")
	}
}

func wfKey(m Model, k tea.KeyMsg) Model {
	return exploreModel(exploreFirst(m.updateWorkflowEditor(k)))
}

func openReleaseEditor(t *testing.T) Model {
	t.Helper()
	m := exploreProject()
	m.exploreTab = tabWorkflows
	m = press(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.wfe == nil || m.workflows[m.wfe.idx].Name != "release" {
		t.Fatalf("Enter on a workflow did not open its steps in the box: wfe=%+v", m.wfe)
	}
	return m
}

// TestWorkflowStepsEditInTheBox is the editor as a working copy: steps
// move, come out and go in without touching the workflow until Ctrl+S,
// and Esc leaves it as it was.
func TestWorkflowStepsEditInTheBox(t *testing.T) {
	m := openReleaseEditor(t)
	m.projectDir = t.TempDir()
	steps := func() string { return strings.Join(m.wfe.steps, ",") }

	m = wfKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("J")})
	if steps() != "test,build,gone" || m.wfe.cursor != 1 || !m.wfe.dirty {
		t.Fatalf("J did not move the step down with the cursor: %s cursor=%d dirty=%v", steps(), m.wfe.cursor, m.wfe.dirty)
	}
	m = wfKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("K")})
	if steps() != "build,test,gone" || m.wfe.cursor != 0 {
		t.Fatalf("K did not move it back up: %s", steps())
	}
	m = wfKey(m, tea.KeyMsg{Type: tea.KeyDown})
	m = wfKey(m, tea.KeyMsg{Type: tea.KeyDown})
	m = wfKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	if steps() != "build,test" || m.wfe.cursor != 1 {
		t.Fatalf("x did not take the step out: %s cursor=%d", steps(), m.wfe.cursor)
	}
	// Taking a step out does not touch the command it named.
	if _, ok := m.config.FindCommand("test"); !ok {
		t.Error("removing a step deleted its command")
	}
	if strings.Contains(stripStyle(m.View()), "gone") {
		t.Error("a removed step is still drawn")
	}
	if !strings.Contains(stripStyle(m.View()), "unsaved") {
		t.Error("the box does not say the steps are unsaved")
	}
	// Nothing has reached the workflow.
	if got := strings.Join(m.workflows[0].Commands, ","); got != "build,test,gone" {
		t.Fatalf("the workflow changed before Ctrl+S: %s", got)
	}

	// Add: a picker of the commands, narrowed by typing, goes in after
	// the cursor.
	m = wfKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	if !m.wfe.adding {
		t.Fatal("a did not open the picker")
	}
	for _, r := range "hel" {
		m = wfKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	if got := m.addCandidates(); len(got) != 1 || got[0] != "hello" {
		t.Fatalf("typing hel left %v, want hello", got)
	}
	m = wfKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	if steps() != "build,test,hello" || m.wfe.adding || m.wfe.cursor != 2 {
		t.Fatalf("Enter did not add the command after the cursor: %s adding=%v cursor=%d", steps(), m.wfe.adding, m.wfe.cursor)
	}
	// The preview reads the working copy, not the saved workflow.
	if view := stripStyle(m.View()); !strings.Contains(view, "$ echo hello") || strings.Contains(view, "(not found)") {
		t.Errorf("the preview does not follow the working copy\n%s", view)
	}

	saved := wfKey(m, tea.KeyMsg{Type: tea.KeyCtrlS})
	if saved.wfe != nil || strings.Join(saved.workflows[0].Commands, ",") != "build,test,hello" {
		t.Fatalf("Ctrl+S did not write the steps: wfe=%v steps=%v err=%q", saved.wfe, saved.workflows[0].Commands, saved.errMsg)
	}
	if _, err := os.Stat(filepath.Join(saved.projectDir, "workflows.json")); err != nil {
		t.Errorf("Ctrl+S did not save workflows.json: %v", err)
	}

	// Esc throws the working copy away.
	e := openReleaseEditor(t)
	e = wfKey(e, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	e = wfKey(e, tea.KeyMsg{Type: tea.KeyEsc})
	if e.leaveAsk != leaveWorkflow {
		t.Fatal("Esc with a step taken out did not ask what to do")
	}
	e = wfKey(e, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	if e.wfe != nil || len(e.workflows[0].Commands) != 3 {
		t.Errorf("discarding did not leave the workflow as it was: wfe=%v steps=%v", e.wfe, e.workflows[0].Commands)
	}
}

// TestWorkflowStepOpensItsCommand checks Enter on a step: it goes to
// the command the step names, on the commands tab, open for editing —
// and refuses while there are unsaved steps, which would otherwise be
// lost on the way.
func TestWorkflowStepOpensItsCommand(t *testing.T) {
	m := openReleaseEditor(t)
	m = wfKey(m, tea.KeyMsg{Type: tea.KeyDown}) // test
	m = wfKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.wfe != nil || m.exploreTab != tabCommands || m.cf != nil || len(m.exploreHistory) != 1 {
		t.Errorf("Enter on a step did not open its command: tab=%d wfe=%v cf=%+v", m.exploreTab, m.wfe, m.cf)
	}

	d := openReleaseEditor(t)
	d = wfKey(d, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("J")})
	d = wfKey(d, tea.KeyMsg{Type: tea.KeyEnter})
	d = press(d, tea.KeyMsg{Type: tea.KeyEsc})
	if d.wfe == nil || !d.wfe.dirty {
		t.Errorf("Enter with unsaved steps left the editor without a word: wfe=%v err=%q", d.wfe, d.errMsg)
	}
}

// TestWorkflowStepsAreClickable: a click on a step row — found by
// reading the view — puts the cursor there, the wheel moves it, and a
// click on the list leaves the editor the way Esc does.
func TestWorkflowStepsAreClickable(t *testing.T) {
	m := openReleaseEditor(t)
	l := m.exploreLayout(m.viewWidth())
	y := rowOf(m.View(), "3  gone")
	if y < 0 {
		t.Fatalf("the third step is not drawn\n%s", m.View())
	}
	if got := clickAt(m, l.rightX+4, y); got.wfe == nil || got.wfe.cursor != 2 {
		t.Errorf("clicking the third step put the cursor on %+v", got.wfe)
	}
	// The editor state is shared by pointer between copies of the model,
	// so each gesture starts from a fresh editor.
	m = openReleaseEditor(t)
	if got := mouseAt(m, l.rightX+4, y, tea.MouseActionPress, tea.MouseButtonWheelDown); got.wfe.cursor != 1 {
		t.Errorf("the wheel moved the step cursor to %d", got.wfe.cursor)
	}
	m = openReleaseEditor(t)
	if got := clickAt(m, 4, l.listY); got.wfe != nil {
		t.Error("a click on the list left the editor open")
	}
}

// TestWorkflowEditsFullScreenWhenNarrow: without room for the box, the
// multi-select screen does the same job and comes back here.
func TestWorkflowEditsFullScreenWhenNarrow(t *testing.T) {
	m := exploreProject()
	m.width, m.height = 60, 20
	m.exploreTab = tabWorkflows
	m = press(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.screen != ScreenExplore || m.wfe == nil {
		t.Errorf("a narrow terminal should edit on its own screen: screen=%v wfe=%v", m.screen, m.wfe)
	}
}

// TestFormSaysWhatIsUnsaved checks the three marks the form carries:
// "unsaved" in the title once anything differs from what was opened,
// a mark on each field that differs, and the error under the field it
// is about — staying there until that field is typed in, where the
// banner used to vanish on the next key.
func TestFormSaysWhatIsUnsaved(t *testing.T) {
	m := exploreProject()
	m = press(m, tea.KeyMsg{Type: tea.KeyEnter}) // edit deploy
	if m.formDirty() || strings.Contains(stripStyle(m.View()), "unsaved") {
		t.Fatal("a form just opened on a command is already unsaved")
	}
	m = formKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	m = formKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	view := stripStyle(m.View())
	if !m.formDirty() || !strings.Contains(view, "● unsaved") {
		t.Errorf("typing did not mark the form unsaved\n%s", view)
	}
	// The changed field carries the mark and the others do not: the
	// mark sits where the margin is, right before the label.
	if !strings.Contains(view, "●Cmd >") && !strings.Contains(view, "● Cmd >") {
		t.Errorf("the changed field is not marked\n%s", view)
	}
	if strings.Contains(view, "●Name >") || strings.Contains(view, "● Name >") {
		t.Errorf("an unchanged field is marked\n%s", view)
	}
	// Put the value back and the marks go.
	m = formKey(m, tea.KeyMsg{Type: tea.KeyBackspace})
	m = formKey(m, tea.KeyMsg{Type: tea.KeyBackspace})
	if m.formDirty() {
		t.Error("restoring the value left the form unsaved")
	}

	// A name already in use is said under the name field, and stays
	// until the field is typed in.
	n := press(exploreProject(), tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	n = formKey(n, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")}) // some cmd
	n = formKey(n, tea.KeyMsg{Type: tea.KeyTab})                       // name
	for _, r := range "build" {
		n = formKey(n, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	n = formKey(n, tea.KeyMsg{Type: tea.KeyEnter})
	if n.cf.fieldIdx != 0 || n.cf.fieldErr == "" {
		t.Fatalf("a duplicate name was not refused under the name field: %+v", n.cf)
	}
	if got := stripStyle(n.View()); !strings.Contains(got, "⚠ already in use: build") {
		t.Errorf("the error is not drawn under the field\n%s", got)
	}
	if n.errMsg != "" {
		t.Error("the error also went to the banner")
	}
	n = formKey(n, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	if n.cf.fieldErr != "" {
		t.Error("typing in the field did not clear its error")
	}
}

// TestLeavingAsksOnlyWhenItWouldLoseSomething is the question on the
// way out: asked with changes, not asked without, and answered three
// ways — save, discard, keep editing — by key or by the arrows.
func TestLeavingAsksOnlyWhenItWouldLoseSomething(t *testing.T) {
	clean := press(exploreProject(), tea.KeyMsg{Type: tea.KeyEnter})
	clean = formKey(clean, tea.KeyMsg{Type: tea.KeyEsc})
	if clean.cf != nil || clean.leaveAsk != leaveNone {
		t.Fatalf("Esc on an unchanged form did not just leave: cf=%v ask=%d", clean.cf, clean.leaveAsk)
	}

	dirty := func() Model {
		m := press(exploreProject(), tea.KeyMsg{Type: tea.KeyEnter})
		m.projectDir = t.TempDir()
		tsv := "name\tgroup\tworkdir\tcmd\tshell\tslots\n" +
			"build\tmake\t{$root}\\api\tmake build\t\t\n" +
			"test\tmake\t\tgo test ./...\t\t\n" +
			"deploy\tdeploy\t\tdeploy --env {env}\t\t\n" +
			"hello\t\t\techo hello\t\t\n"
		if err := os.WriteFile(filepath.Join(m.projectDir, "commands.tsv"), []byte(tsv), 0o644); err != nil {
			t.Fatal(err)
		}
		m = formKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("!")})
		return formKey(m, tea.KeyMsg{Type: tea.KeyEsc})
	}

	m := dirty()
	if m.leaveAsk != leaveForm {
		t.Fatal("Esc on a changed form did not ask")
	}
	view := stripStyle(m.View())
	for _, want := range []string{"Unsaved changes", "s  Save and leave", "d  Discard changes", "k  Keep editing"} {
		if !strings.Contains(view, want) {
			t.Errorf("the question does not show %q\n%s", want, view)
		}
	}

	// The change is still in the field under the caret, uncommitted:
	// keeping editing keeps exactly that.
	kept := formKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if kept.cf == nil || kept.leaveAsk != leaveNone || kept.nameInput.Value() != "deploy --env {env}!" {
		t.Errorf("k did not keep editing with the change intact: cf=%+v value=%q", kept.cf, kept.nameInput.Value())
	}
	discarded := formKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	if discarded.cf != nil || discarded.leaveAsk != leaveNone {
		t.Errorf("d did not discard: cf=%v", discarded.cf)
	}
	if c, _ := discarded.config.FindCommand("deploy"); c.Cmd != "deploy --env {env}" {
		t.Errorf("discarding changed the command: %q", c.Cmd)
	}
	saved := formKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	if saved.cf != nil || saved.leaveAsk != leaveNone {
		t.Errorf("s did not save and leave: cf=%v err=%q", saved.cf, saved.errMsg)
	}
	if c, _ := saved.config.FindCommand("deploy"); c.Cmd != "deploy --env {env}!" {
		t.Errorf("s did not write the change: %q", c.Cmd)
	}
	// Enter defaults to keeping the draft.
	byArrows := formKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	if byArrows.cf == nil || byArrows.leaveAsk != leaveNone {
		t.Error("Enter did not keep editing")
	}

	// A click on the list while the form has changes asks too, rather
	// than throwing the changes away.
	c := press(exploreProject(), tea.KeyMsg{Type: tea.KeyEnter})
	c = formKey(c, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("!")})
	l := c.exploreLayout(c.viewWidth())
	if got := clickAt(c, 4, l.listY+1); got.leaveAsk != leaveForm || got.cf == nil {
		t.Errorf("a click away from a changed form did not ask: ask=%d cf=%v", got.leaveAsk, got.cf)
	}
}

// TestLeaveQuestionShowsFullScreenToo: on the form's own screen the
// question is drawn under the fields, inside the terminal.
func TestLeaveQuestionShowsFullScreenToo(t *testing.T) {
	m := exploreProject()
	m.width, m.height = 60, 22
	m = press(m, tea.KeyMsg{Type: tea.KeyEnter})
	m = formKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("!")})
	m = formKey(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.screen != ScreenCommandForm || m.leaveAsk != leaveForm {
		t.Fatalf("screen=%v ask=%d", m.screen, m.leaveAsk)
	}
	view := stripStyle(m.View())
	if !strings.Contains(view, "Unsaved changes") || !strings.Contains(view, "● unsaved") {
		t.Errorf("the full-screen form does not show the question and the mark\n%s", view)
	}
	if got := frameLines(m.View()); got > m.height {
		t.Errorf("the question pushed the frame to %d lines in a %d-line terminal", got, m.height)
	}
}

// TestDraftIsMarkedByItselfNotByName: giving the draft a name already in
// use — the case the form is about to refuse — must not mark the saved
// command of that name as the draft too. There is one draft row.
func TestDraftIsMarkedByItselfNotByName(t *testing.T) {
	m := press(exploreProject(), tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	m = formKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	m = formKey(m, tea.KeyMsg{Type: tea.KeyTab})
	for _, r := range "build" {
		m = formKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	drafts := 0
	for _, it := range m.exploreVisible() {
		if it.draft {
			drafts++
			if it.note != "$ x" {
				t.Errorf("the saved command was marked as the draft: %+v", it)
			}
		}
	}
	if drafts != 1 {
		t.Errorf("%d rows are marked as the draft, want 1", drafts)
	}
}
