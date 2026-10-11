package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	mdl "github.com/Taka-S-dev/baton/internal/model"
	"github.com/Taka-S-dev/baton/internal/runner"
)

func wfModel() Model {
	m := Model{}
	m.wfSearchTI = textinput.New()
	m.config = mdl.Config{Base: []mdl.Command{
		{Name: "build", Cmd: "make build"},
		{Name: "deploy", Cmd: "kubectl apply -n {env}"},
		{Name: "test", Cmd: "go test ./..."},
	}}
	m.workflows = []mdl.Workflow{
		{Name: "release", Commands: []string{"build", "deploy"}},
		{Name: "ci", Commands: []string{"test"}},
	}
	return m
}

// TestStepHeader_ShowsResolvedCommand checks that the header printed before
// each step echoes the resolved command line, so users can see (and re-run)
// exactly what executed.
func TestStepHeader_ShowsResolvedCommand(t *testing.T) {
	cases := []struct {
		name string
		cmd  mdl.Command
		want string
	}{
		{
			name: "with workdir",
			cmd:  mdl.Command{Name: "deploy", Cmd: "kubectl apply -n prod", Dir: "infra"},
			want: "\n── [2/3] deploy   workdir: infra\n   $ kubectl apply -n prod",
		},
		{
			name: "without workdir",
			cmd:  mdl.Command{Name: "build", Cmd: "make build"},
			want: "\n── [2/3] build\n   $ make build",
		},
	}
	for _, c := range cases {
		if got := stepHeader(2, 3, c.cmd.Name, c.cmd); got != c.want {
			t.Errorf("%s: stepHeader = %q, want %q", c.name, got, c.want)
		}
	}
}

// TestWfFiltered_AndSearch checks the Run workflow list search: AND terms
// matching the workflow name, its command names, and the command bodies.
func TestWfFiltered_AndSearch(t *testing.T) {
	m := wfModel()

	cases := []struct {
		query string
		want  []int
	}{
		{"", []int{0, 1}},
		{"release", []int{0}},         // name
		{"test", []int{1}},            // command name
		{"make", []int{0}},            // command body (as shown in the steps preview)
		{"deploy kubectl", []int{0}},  // AND across fields
		{"release test", nil},         // AND requires both in one workflow
		{"RELEASE KUBECTL", []int{0}}, // case-insensitive
	}
	for _, c := range cases {
		m.wfSearchTI.SetValue(c.query)
		got := m.wfFiltered()
		if !equalInts(got, c.want) {
			t.Errorf("query %q: got %v, want %v", c.query, got, c.want)
		}
	}
}

// TestUpdateRunWorkflow_SearchKeys checks that typed keys (including space)
// reach the search input, and that Esc clears the search before going back.
func TestUpdateRunWorkflow_SearchKeys(t *testing.T) {
	m := wfModel()
	m.screen = ScreenRunWorkflow
	m.wfSearchTI.Focus()

	for _, r := range "deploy kubectl" {
		key := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
		if r == ' ' {
			key.Type = tea.KeySpace
		}
		nm, _ := m.updateRunWorkflow(key)
		m = nm.(Model)
	}
	if got := m.wfSearchTI.Value(); got != "deploy kubectl" {
		t.Fatalf("search value = %q, want %q", got, "deploy kubectl")
	}
	if got := m.wfFiltered(); !equalInts(got, []int{0}) {
		t.Fatalf("filtered = %v, want [0]", got)
	}

	nm, _ := m.updateRunWorkflow(tea.KeyMsg{Type: tea.KeyEscape})
	m = nm.(Model)
	if m.screen != ScreenRunWorkflow || m.wfSearchTI.Value() != "" {
		t.Fatalf("first Esc must clear the search and stay: screen=%v search=%q", m.screen, m.wfSearchTI.Value())
	}
	nm, _ = m.updateRunWorkflow(tea.KeyMsg{Type: tea.KeyEscape})
	m = nm.(Model)
	if m.screen != ScreenProjectSelect {
		t.Fatalf("second Esc: screen=%v, want projects", m.screen)
	}
}

// TestUpdateRunWorkflow_RefusesMissingSteps checks a workflow whose step
// no longer resolves refuses to run entirely — a partial run would look
// like success while skipping real work — and names the missing steps.
func TestUpdateRunWorkflow_RefusesMissingSteps(t *testing.T) {
	m := wfModel()
	m.projectDir = t.TempDir()
	m.screen = ScreenRunWorkflow
	m.workflows[0].Commands = []string{"build", "ghost", "deploy"}

	nm, _ := m.updateRunWorkflow(tea.KeyMsg{Type: tea.KeyEnter}) // cursor 0 = release
	m = nm.(Model)
	if m.screen != ScreenRunWorkflow || m.resolve != nil {
		t.Fatalf("a broken workflow must not start resolving: screen=%v resolve=%v", m.screen, m.resolve)
	}
	if !strings.Contains(m.errMsg, "ghost") {
		t.Fatalf("errMsg = %q, want it to name the missing step", m.errMsg)
	}
	if m.lastWorkflow != "" {
		t.Fatalf("a refused run must not become the last workflow, got %q", m.lastWorkflow)
	}

	// An intact workflow still runs.
	m.errMsg = ""
	nm, _ = m.updateRunWorkflow(tea.KeyMsg{Type: tea.KeyDown})
	m = nm.(Model)
	nm, _ = m.updateRunWorkflow(tea.KeyMsg{Type: tea.KeyEnter}) // ci
	m = nm.(Model)
	if m.errMsg != "" || m.lastWorkflow != "ci" {
		t.Fatalf("intact workflow must run: errMsg=%q lastWorkflow=%q", m.errMsg, m.lastWorkflow)
	}
}

// TestWorkflowStepPick_RunsSubsetInWorkflowOrder checks the partial-run
// path: → opens the step picker, toggling picks steps, and Enter runs
// them in the workflow's own order regardless of the toggle order.
func TestWorkflowStepPick_RunsSubsetInWorkflowOrder(t *testing.T) {
	m := wfModel()
	m.projectDir = t.TempDir()
	m.screen = ScreenRunWorkflow
	m.workflows[0].Commands = []string{"build", "test", "deploy"}

	nm, _ := m.updateRunWorkflow(tea.KeyMsg{Type: tea.KeyRight})
	m = nm.(Model)
	if m.screen != ScreenRunWorkflowSteps || m.wfp == nil {
		t.Fatalf("→ must open the step picker, screen=%v wfp=%v", m.screen, m.wfp)
	}

	// Toggle the last step first, then the first one: the run order must
	// still follow the workflow, not the order they were picked.
	down := tea.KeyMsg{Type: tea.KeyDown}
	tab := tea.KeyMsg{Type: tea.KeyTab}
	for i := 0; i < 2; i++ {
		nm, _ = m.updateRunWorkflowSteps(down)
		m = nm.(Model)
	}
	nm, _ = m.updateRunWorkflowSteps(tab) // deploy (index 2)
	m = nm.(Model)
	nm, _ = m.updateRunWorkflowSteps(down) // wraps to index 0
	m = nm.(Model)
	nm, _ = m.updateRunWorkflowSteps(tab) // build (index 0)
	m = nm.(Model)
	if m.wfp.count() != 2 {
		t.Fatalf("selected = %d, want 2", m.wfp.count())
	}
	// Tab is the toggle everywhere in baton; Space stays unbound here so
	// it can go to a filter if this list ever gains one.
	nm, _ = m.updateRunWorkflowSteps(tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
	m = nm.(Model)
	if m.wfp.count() != 2 {
		t.Fatalf("Space must not toggle, selected = %d", m.wfp.count())
	}

	nm, _ = m.updateRunWorkflowSteps(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(Model)
	if m.resolve == nil {
		t.Fatalf("Enter must start the resolve flow, screen=%v err=%q", m.screen, m.errMsg)
	}
	if got := m.resolve.itemNames; len(got) != 2 || got[0] != "build" || got[1] != "deploy" {
		t.Fatalf("run steps = %v, want [build deploy] in workflow order", got)
	}
	if !strings.Contains(m.resolve.workflowLabel, "2/3") {
		t.Fatalf("label = %q, want it to show the partial step count", m.resolve.workflowLabel)
	}
	if m.lastWorkflow != "release" {
		t.Fatalf("a partial run must still record the workflow, got %q", m.lastWorkflow)
	}
}

// Empty selections never proceed; missing commands cannot be selected.
func TestWorkflowStepPick_RequiresSelection(t *testing.T) {
	m := wfModel()
	m.projectDir = t.TempDir()
	m.workflows[0].Commands = []string{"ghost", "build"}
	m.gotoWorkflowStepPick(0)
	nm, _ := m.updateRunWorkflowSteps(tea.KeyMsg{Type: tea.KeyTab})
	m = nm.(Model)
	if m.wfp.count() != 0 {
		t.Fatal("missing step selectable")
	}
	for _, key := range []tea.KeyType{tea.KeyEnter, tea.KeyDown, tea.KeyEnter} {
		nm, _ = m.updateRunWorkflowSteps(tea.KeyMsg{Type: key})
		m = nm.(Model)
		if m.resolve != nil || m.screen == ScreenConfirmRun {
			t.Fatal("empty selection advanced")
		}
	}
	nm, _ = m.updateRunWorkflowSteps(tea.KeyMsg{Type: tea.KeyTab})
	m = nm.(Model)
	nm, _ = m.updateRunWorkflowSteps(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(Model)
	if m.screen != ScreenConfirmRun || len(m.confirmRunItems) != 1 || m.confirmRunItems[0].Name != "build" {
		t.Fatalf("selected step not reviewed: %+v", m.confirmRunItems)
	}
}

// TestWorkflowStepPick_EmptyWorkflow checks a workflow with no steps —
// reachable by hand-editing workflows.json — leaves every key a no-op
// instead of indexing past the end of the step list.
func TestWorkflowStepPick_EmptyWorkflow(t *testing.T) {
	m := wfModel()
	m.projectDir = t.TempDir()
	m.workflows[0].Commands = nil
	m.gotoWorkflowStepPick(0)

	for _, k := range []tea.KeyMsg{
		{Type: tea.KeyUp}, {Type: tea.KeyDown}, {Type: tea.KeyTab}, {Type: tea.KeyEnter},
	} {
		nm, _ := m.updateRunWorkflowSteps(k)
		m = nm.(Model)
	}
	if m.screen != ScreenRunWorkflowSteps || m.resolve != nil {
		t.Fatalf("an empty workflow must stay put and run nothing: screen=%v resolve=%v", m.screen, m.resolve)
	}
	if strings.Count(m.viewRunWorkflowSteps(80), "\n") == 0 {
		t.Fatal("the picker must still render for an empty workflow")
	}
}

// TestUpdateRunWorkflow_SpaceStillTypesIntoSearch guards the key budget
// on the workflow list: Space belongs to the search field, so the step
// picker is opened with → instead.
func TestUpdateRunWorkflow_SpaceStillTypesIntoSearch(t *testing.T) {
	m := wfModel()
	m.screen = ScreenRunWorkflow
	m.wfSearchTI.Focus()
	for _, k := range []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune{'a'}},
		{Type: tea.KeySpace, Runes: []rune{' '}},
		{Type: tea.KeyRunes, Runes: []rune{'b'}},
	} {
		nm, _ := m.updateRunWorkflow(k)
		m = nm.(Model)
	}
	if got := m.wfSearchTI.Value(); got != "a b" {
		t.Fatalf("search value = %q, want %q", got, "a b")
	}
	if m.screen != ScreenRunWorkflow {
		t.Fatalf("Space must not leave the list, screen=%v", m.screen)
	}
}

// TestFormatDuration checks the elapsed times printed after each step
// read at a precision that suits their length, instead of Go's default
// which puts 1.170448s next to 4m12.003s.
func TestFormatDuration(t *testing.T) {
	cases := map[time.Duration]string{
		120 * time.Millisecond:                              "120ms",
		1234 * time.Millisecond:                             "1.23s",
		time.Minute + 12*time.Second:                        "1m12s",
		4*time.Minute + 12*time.Second + 3*time.Millisecond: "4m12s",
	}
	for in, want := range cases {
		if got := formatDuration(in); got != want {
			t.Errorf("formatDuration(%v) = %q, want %q", in, got, want)
		}
	}
}

// TestRunCompletion_FreezesTheTotal checks the done screen reports the
// time the run took, not the time since it finished: the screen redraws
// on every tick, so a total computed at render would keep climbing while
// the user reads it.
func TestRunCompletion_FreezesTheTotal(t *testing.T) {
	cmd := mdl.Command{Name: "a", Cmd: "echo hi"}
	m := Model{width: 80, height: 24, screen: ScreenRunning}
	m.running = &runningState{
		items:    []mdl.RunItem{{Name: "a", Cmd: &cmd}},
		runStart: time.Now().Add(-3 * time.Second),
	}

	nm, _ := m.handleRunnerDone(runner.DoneMsg{Index: 0})
	got := nm.(Model)
	if !got.running.completed {
		t.Fatal("the last step finishing must complete the run")
	}
	if got.running.total < 3*time.Second {
		t.Fatalf("total = %v, want the elapsed run time", got.running.total)
	}

	first := got.runCompletionLine()
	time.Sleep(20 * time.Millisecond)
	if second := got.runCompletionLine(); second != first {
		t.Fatalf("the done screen changed between renders:\n%q\n%q", first, second)
	}
	if !strings.Contains(first, "· 3") {
		t.Fatalf("done screen = %q, want it to report how long the run took", first)
	}
}

// TestRunCompletion_IgnoresTheKeyBurst checks the done screen drops the
// keys that arrive with it. While a command runs the terminal belongs to
// that command, so anything typed then reaches baton all at once when it
// returns — and each of those keys would otherwise walk a menu step
// closer to starting the same run again.
func TestRunCompletion_IgnoresTheKeyBurst(t *testing.T) {
	cmd := mdl.Command{Name: "a", Cmd: "echo hi"}
	newDone := func(age time.Duration) Model {
		m := Model{width: 80, height: 24, screen: ScreenRunning}
		m.spinner = spinner.New()
		m.running = &runningState{
			items:       []mdl.RunItem{{Name: "a", Cmd: &cmd}},
			completed:   true,
			completedAt: time.Now().Add(-age),
		}
		return m
	}
	enter := tea.KeyMsg{Type: tea.KeyEnter}

	m := newDone(0)
	for i := 0; i < 5; i++ { // the burst
		nm, _ := m.Update(enter)
		m = nm.(Model)
	}
	if m.screen != ScreenRunning {
		t.Fatalf("the burst left the done screen for %v — it would keep walking the menu", m.screen)
	}

	// Once it has settled, a real keypress returns to the menu.
	m = newDone(settleAfterRun + 10*time.Millisecond)
	nm, _ := m.Update(enter)
	if got := nm.(Model); got.screen != ScreenRunCommands || got.running != nil {
		t.Fatalf("after settling a key must return to the menu, screen=%v running=%v", got.screen, got.running)
	}
}

// TestUpdateConfirmRun_ScrollClampsToLastPage checks ↑↓ scroll the item
// window and stop at the edges: Up at the top and Down past the last
// page must both be no-ops, so the counter can never drift off-screen.
func TestUpdateConfirmRun_ScrollClampsToLastPage(t *testing.T) {
	m := Model{width: 80, height: 20, screen: ScreenConfirmRun}
	cmd := mdl.Command{Name: "c", Cmd: "echo hi"}
	for i := 0; i < 10; i++ {
		m.confirmRunItems = append(m.confirmRunItems, mdl.RunItem{Name: "c", Cmd: &cmd})
	}

	nm, _ := m.updateConfirmRun(tea.KeyMsg{Type: tea.KeyUp})
	m = nm.(Model)
	if m.confirmRunScroll != 0 {
		t.Fatalf("scroll = %d after Up at the top, want 0", m.confirmRunScroll)
	}

	maxScroll := len(m.confirmRunItems) - m.confirmRunPerPage()
	for i := 0; i < 50; i++ {
		nm, _ = m.updateConfirmRun(tea.KeyMsg{Type: tea.KeyDown})
		m = nm.(Model)
	}
	if m.confirmRunScroll != maxScroll {
		t.Fatalf("scroll = %d after holding Down, want clamp at %d", m.confirmRunScroll, maxScroll)
	}
}
