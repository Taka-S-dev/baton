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
