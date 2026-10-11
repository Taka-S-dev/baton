package tui

import (
	"strings"
	"testing"

	mdl "github.com/Taka-S-dev/baton/internal/model"
	tea "github.com/charmbracelet/bubbletea"
)

func TestPathReadinessOnlyAfterSelection(t *testing.T) {
	cmd := mdl.Command{Name: "build", Cmd: "echo {@file:.go}", Dir: completionFixture(t)}
	nm, read := (Model{width: 120, height: 40}).startResolveFlow([]msItem{{cmd: &cmd}})
	m := completionResult(nm.(Model), read)
	if strings.Contains(stripStyle(m.View()), "✓ Inputs complete") {
		t.Fatal("hover treated as completed")
	}
	m = queryCompletion(m, "test.go")
	m, _ = completionKey(m, tea.KeyMsg{Type: tea.KeyTab})
	view := stripStyle(m.View())
	for _, want := range []string{"✓ Inputs complete", "✓ File selected: test.go", "Enter: review run"} {
		if !strings.Contains(view, want) {
			t.Fatalf("missing %q", want)
		}
	}
	m, _ = completionKey(m, tea.KeyMsg{Type: tea.KeyBackspace})
	if m.pathInputReady() || strings.Contains(stripStyle(m.View()), "✓ Inputs complete") {
		t.Fatal("editing left stale completion mark")
	}
}
func TestPathReadinessCountsLaterInputs(t *testing.T) {
	cmd := mdl.Command{Name: "build", Cmd: "echo {@file:.go} {mode}", Dir: completionFixture(t)}
	nm, read := (Model{width: 120, height: 40}).startResolveFlow([]msItem{{cmd: &cmd}})
	m := completionResult(nm.(Model), read)
	m = queryCompletion(m, "test.go")
	m, _ = completionKey(m, tea.KeyMsg{Type: tea.KeyTab})
	if m.inputProgressLabel() != "Inputs 1/2 complete" || !strings.Contains(m.pathInputKeys(), "next input") {
		t.Fatal("later input ignored")
	}
}
