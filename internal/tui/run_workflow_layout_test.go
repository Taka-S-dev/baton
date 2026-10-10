package tui

import (
	mdl "github.com/Taka-S-dev/baton/internal/model"
	tea "github.com/charmbracelet/bubbletea"
	"strings"
	"testing"
)

func TestWorkflowRunDetailsPreserveSelection(t *testing.T) {
	m := Model{workflows: []mdl.Workflow{{Name: "test", Commands: []string{"missing"}}}, width: 60, height: 24}
	m.gotoWorkflowStepPick(0)
	next, _ := m.updateRunWorkflowSteps(tea.KeyMsg{Type: tea.KeyCtrlQ})
	m = next.(Model)
	if !m.runWorkflowDetail {
		t.Fatal("details did not open")
	}
	if !strings.Contains(m.viewRunWorkflowSteps(60), "Command not found") {
		t.Fatal("missing reference not explained")
	}
	next, _ = m.updateRunWorkflowSteps(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(Model)
	if m.runWorkflowDetail || m.screen != ScreenRunWorkflowSteps || m.wfp == nil {
		t.Fatal("Esc should return to steps")
	}
	if m.workflows[0].Commands[0] != "missing" {
		t.Fatal("saved workflow changed")
	}
}
