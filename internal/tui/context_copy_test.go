package tui

import (
	"errors"
	tea "github.com/charmbracelet/bubbletea"
	"testing"
)

func TestCopyValueSnapshotsInput(t *testing.T) {
	m := exploreProject()
	m = createKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	m.nameInput.SetValue("echo copy me")
	l := m.exploreLayout(m.viewWidth())
	m = exploreModel(exploreFirst(m.Update(tea.MouseMsg{X: l.rightX + 3, Y: l.fieldY, Button: tea.MouseButtonRight, Action: tea.MouseActionPress})))
	if len(m.contextActions) != 2 || m.contextActions[1].label != "Copy value" || m.contextActions[1].value != "echo copy me" {
		t.Fatal("wrong input copy target")
	}
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	m = next.(Model)
	if cmd == nil || m.pasteMenu || m.nameInput.Value() != "echo copy me" {
		t.Fatal("copy changed input")
	}
	// Do not modify the system clipboard in tests.
	m = exploreModel(exploreFirst(m.Update(contextCopyMsg{errors.New("unavailable")})))
	if m.errMsg == "" {
		t.Fatal("copy failure not reported")
	}
}

func TestWorkflowCopyFollowsSelection(t *testing.T) {
	m := exploreProject()
	m.exploreTab = tabWorkflows
	m.wfe = &workflowEditState{idx: 0, steps: []string{"build", "test"}, cursor: 1, dirty: true}
	expected, ok := m.workflowStepCommand("test")
	if !ok {
		t.Fatal("fixture missing")
	}
	actions := m.commandCopyActions()
	if len(actions) != 2 || actions[0].value != expected.Cmd || actions[1].label != "Copy workdir" {
		t.Fatal("wrong step copy target")
	}
	l := m.exploreLayout(m.viewWidth())
	m = exploreModel(exploreFirst(m.Update(tea.MouseMsg{X: l.rightX + 3, Y: l.prevY + 1, Button: tea.MouseButtonRight, Action: tea.MouseActionPress})))
	if !m.pasteMenu || m.contextActions[0].value != expected.Cmd {
		t.Fatal("detail menu missing")
	}
	m = createKey(m, tea.KeyMsg{Type: tea.KeyEsc})
	if !m.wfe.dirty || m.pasteMenu {
		t.Fatal("menu changed draft")
	}
}
