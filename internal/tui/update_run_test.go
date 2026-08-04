package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	mdl "github.com/Taka-S-dev/baton/internal/model"
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
	if m.screen != ScreenMainMenu {
		t.Fatalf("second Esc: screen=%v, want main menu", m.screen)
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

// TestWorkflowStepPick_HoverFallbackAndMissingSteps checks Enter with
// nothing toggled runs the hovered step, and that a step whose command
// is gone can neither be toggled nor run while the rest stay usable.
func TestWorkflowStepPick_HoverFallbackAndMissingSteps(t *testing.T) {
	m := wfModel()
	m.projectDir = t.TempDir()
	m.workflows[0].Commands = []string{"ghost", "build"}
	m.gotoWorkflowStepPick(0)

	// Cursor starts on the missing step: it cannot be toggled or run.
	nm, _ := m.updateRunWorkflowSteps(tea.KeyMsg{Type: tea.KeyTab})
	m = nm.(Model)
	if m.wfp.count() != 0 {
		t.Fatal("a missing step must not be selectable")
	}
	nm, _ = m.updateRunWorkflowSteps(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(Model)
	if m.resolve != nil || !strings.Contains(m.errMsg, "ghost") {
		t.Fatalf("running a missing step must fail with a named error, err=%q", m.errMsg)
	}

	// The intact step still runs via the hover fallback, no toggle needed.
	// It has no slots, so resolution completes and lands on the confirm
	// screen in one step.
	nm, _ = m.updateRunWorkflowSteps(tea.KeyMsg{Type: tea.KeyDown})
	m = nm.(Model)
	nm, _ = m.updateRunWorkflowSteps(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(Model)
	if m.screen != ScreenConfirmRun {
		t.Fatalf("Enter with nothing toggled must run the hovered step, screen=%v err=%q", m.screen, m.errMsg)
	}
	if len(m.confirmRunItems) != 1 || m.confirmRunItems[0].Name != "build" {
		t.Fatalf("run items = %+v, want just the hovered step", m.confirmRunItems)
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
