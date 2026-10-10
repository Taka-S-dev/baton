package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestWorkflowDetailFollowsSelectionAndCandidate(t *testing.T) {
	m := exploreProject()
	m.height = 50
	m.exploreTab = tabWorkflows
	m.wfe = &workflowEditState{idx: 0, steps: []string{"build", "test"}, dirty: true}
	preview := func() string { return stripStyle(m.explorePreviewBox(m.exploreLayout(m.viewWidth()))) }
	if !strings.Contains(preview(), "Command > build") || !strings.Contains(preview(), "Shell >") {
		t.Fatal("missing step details")
	}
	m = createKey(m, tea.KeyMsg{Type: tea.KeyDown})
	if !strings.Contains(preview(), "Command > test") || !m.wfe.dirty {
		t.Fatal("selection lost draft or did not update")
	}
	m = createKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	m.wfe.addSearch = "deploy"
	if !strings.Contains(preview(), "Command > deploy") {
		t.Fatal("candidate preview not updated")
	}
	m.wfe.addSearch = "no-such-command"
	if !strings.Contains(preview(), "No matching command") {
		t.Fatal("stale candidate details")
	}
}

func TestWorkflowDetailMissingAndEmpty(t *testing.T) {
	m := exploreProject()
	m.wfe = &workflowEditState{idx: 0, steps: []string{"missing"}}
	l := m.exploreLayout(m.viewWidth())
	if !strings.Contains(stripStyle(m.workflowDetailBox(l)), "Command not found") {
		t.Fatal("missing warning")
	}
	m.wfe.steps = nil
	if !strings.Contains(stripStyle(m.workflowDetailBox(l)), "No steps yet") {
		t.Fatal("missing empty guidance")
	}
}
