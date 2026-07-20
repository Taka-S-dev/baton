package tui

import (
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

// TestSaveFlows_ReturnToSubmenu checks that finishing (or canceling) a
// create flow lands on the owning submenu, not the main menu.
func TestSaveFlows_ReturnToSubmenu(t *testing.T) {
	m := Model{}
	m.projectDir = t.TempDir()
	m.resolve = &resolveFlowState{}
	nm, _ := m.saveWorkflow("wf")
	if got := nm.(Model); got.screen != ScreenWorkflowMgmt {
		t.Fatalf("saveWorkflow: screen=%v, want Manage workflows", got.screen)
	}

	m2 := Model{}
	m2.projectDir = t.TempDir()
	m2.resolve = &resolveFlowState{}
	nm, _ = m2.saveAlias("al")
	if got := nm.(Model); got.screen != ScreenAliasMgmt {
		t.Fatalf("saveAlias: screen=%v, want Manage aliases", got.screen)
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
