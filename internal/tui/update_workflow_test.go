package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	mdl "github.com/Taka-S-dev/baton/internal/model"
)

// TestWorkflowMgmt_Navigation checks the Manage workflows submenu mirrors
// the Manage aliases pattern: Enter opens the chosen action, Esc returns
// to the main menu, and finished/canceled sub-screens come back to the
// submenu rather than the main menu.
func TestWorkflowMgmt_Navigation(t *testing.T) {
	enter := tea.KeyMsg{Type: tea.KeyEnter}
	esc := tea.KeyMsg{Type: tea.KeyEscape}

	m := &Model{}
	m.workflows = []mdl.Workflow{{Name: "release", Commands: []string{"build"}}}
	m.gotoWorkflowMgmt()
	if m.screen != ScreenWorkflowMgmt || len(m.listItems) != 3 {
		t.Fatalf("submenu: screen=%v items=%v", m.screen, m.listItems)
	}

	// Enter on "Create workflow" (cursor 0) opens the command selector.
	nm, _ := m.updateWorkflowMgmt(enter)
	if got := nm.(Model); got.screen != ScreenCreateWorkflow {
		t.Fatalf("Create workflow: screen=%v", got.screen)
	}

	// Esc from the submenu returns to the main menu.
	nm, _ = m.updateWorkflowMgmt(esc)
	if got := nm.(Model); got.screen != ScreenMainMenu {
		t.Fatalf("Esc: screen=%v, want main menu", got.screen)
	}

	// Esc from the Edit workflow picker returns to the submenu.
	m2 := Model{}
	m2.workflows = m.workflows
	m2.screen = ScreenEditWorkflow
	nm, _ = m2.updateEditWorkflow(esc)
	if got := nm.(Model); got.screen != ScreenWorkflowMgmt {
		t.Fatalf("edit picker Esc: screen=%v, want submenu", got.screen)
	}
}

// TestCreateWorkflow_PickAndName checks the simplified creation: Enter on
// the selector goes straight to the name input (no slot resolution), and
// saving stores the picked command names and returns to the submenu.
func TestCreateWorkflow_PickAndName(t *testing.T) {
	build := mdl.Command{Name: "build", Cmd: "make {target}"}
	m := Model{}
	m.projectDir = t.TempDir()
	m.screen = ScreenCreateWorkflow
	m.msSearchTI = textinput.New()
	m.nameInput = textinput.New()
	m.msItems = []msItem{{cmd: &build}}
	m.msSelected = []int{0}

	nm, _ := m.updateMultiSelect(tea.KeyMsg{Type: tea.KeyEnter})
	got := nm.(Model)
	if got.screen != ScreenNameInput {
		t.Fatalf("Enter must go straight to the name input, screen=%v", got.screen)
	}
	if len(got.pendingWorkflowCmds) != 1 || got.pendingWorkflowCmds[0] != "build" {
		t.Fatalf("pending commands = %v", got.pendingWorkflowCmds)
	}

	nm, _ = got.saveWorkflow("wf")
	saved := nm.(Model)
	if saved.screen != ScreenWorkflowMgmt {
		t.Fatalf("saveWorkflow: screen=%v, want Manage workflows", saved.screen)
	}
	if len(saved.workflows) != 1 || saved.workflows[0].Commands[0] != "build" {
		t.Fatalf("workflows = %+v", saved.workflows)
	}

	// Esc out of the Create workflow command selector.
	m3 := Model{}
	m3.screen = ScreenCreateWorkflow
	m3.msSearchTI = textinput.New()
	nm, _ = m3.updateMultiSelect(tea.KeyMsg{Type: tea.KeyEscape})
	if got := nm.(Model); got.screen != ScreenWorkflowMgmt {
		t.Fatalf("create selector Esc: screen=%v, want Manage workflows", got.screen)
	}
}

// TestSuccessMsg_SetAndCleared checks a successful save leaves a visible
// confirmation and that the next keypress clears it.
func TestSuccessMsg_SetAndCleared(t *testing.T) {
	m := Model{}
	m.projectDir = t.TempDir()
	nm, _ := m.saveWorkflow("wf")
	got := nm.(Model)
	if !strings.Contains(got.successMsg, "wf") {
		t.Fatalf("successMsg = %q, want a created note naming the workflow", got.successMsg)
	}
	if !strings.Contains(got.View(), "✓") {
		t.Fatal("the success note must be rendered in the view")
	}

	nm2, _ := got.Update(tea.KeyMsg{Type: tea.KeyDown})
	if s := nm2.(Model).successMsg; s != "" {
		t.Fatalf("successMsg = %q, want cleared on the next keypress", s)
	}
}

// TestEditWorkflowPick_FilterMapsTarget checks the workflow edit picker
// filters on name and step text, and Enter targets the original index.
func TestEditWorkflowPick_FilterMapsTarget(t *testing.T) {
	m := Model{}
	m.workflows = []mdl.Workflow{
		{Name: "build-all", Commands: []string{"build"}},
		{Name: "ship", Commands: []string{"deploy"}},
	}
	m.screen = ScreenEditWorkflow
	m.setWorkflowPickBase()

	// "deploy" only appears in ship's steps — name search alone wouldn't hit.
	var nm tea.Model = m
	for _, r := range "deploy" {
		nm, _ = nm.(Model).updateEditWorkflow(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	got := nm.(Model)
	if len(got.listItems) != 1 || got.listItems[0] != "ship" {
		t.Fatalf("filtered items = %v, want [ship]", got.listItems)
	}

	nm, _ = got.updateEditWorkflow(tea.KeyMsg{Type: tea.KeyEnter})
	got = nm.(Model)
	if got.editTargetIdx != 1 {
		t.Fatalf("editTargetIdx = %d, want the ORIGINAL index 1", got.editTargetIdx)
	}
	if got.screen != ScreenEditWorkflowMode {
		t.Fatalf("screen = %v, want the mode menu", got.screen)
	}
}

// TestSuggestWorkflowName checks the pre-filled workflow name lists the
// picked commands, counts the overflow, and dodges existing names.
func TestSuggestWorkflowName(t *testing.T) {
	m := Model{}
	m.pendingWorkflowCmds = []string{"build", "test", "deploy", "smoke", "notify"}
	if got := m.suggestWorkflowName(); got != "build+test+deploy+2" {
		t.Fatalf("suggested = %q, want %q", got, "build+test+deploy+2")
	}

	m.pendingWorkflowCmds = []string{"build", "deploy"}
	m.workflows = []mdl.Workflow{{Name: "build+deploy"}}
	if got := m.suggestWorkflowName(); got != "build+deploy-2" {
		t.Fatalf("suggested = %q with a collision, want %q", got, "build+deploy-2")
	}

	m.pendingWorkflowCmds = nil
	if got := m.suggestWorkflowName(); got != "" {
		t.Fatalf("suggested = %q with no commands, want empty", got)
	}
}
