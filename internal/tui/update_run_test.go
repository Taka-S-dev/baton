package tui

import (
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
