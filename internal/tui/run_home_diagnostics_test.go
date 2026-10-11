package tui

import (
	"testing"

	mdl "github.com/Taka-S-dev/baton/internal/model"
)

// TestRunHomeWarningsRecomputed guards against stale diagnostics:
// warnings must describe the current in-memory state, so fixing the
// problem inside the TUI clears them on the next return to Run home.
func TestRunHomeWarningsRecomputed(t *testing.T) {
	m := &Model{}
	m.projectDir = t.TempDir()
	m.workflows = []mdl.Workflow{{Name: "wf", Commands: []string{"ghost"}}}

	m.gotoRunHome()
	if len(m.loadWarnings) == 0 {
		t.Fatal("broken workflow must produce a warning")
	}

	m.workflows = nil // the workflow was deleted in the TUI
	m.gotoRunHome()
	if len(m.loadWarnings) != 0 {
		t.Fatalf("warnings must clear once the cause is gone, got %v", m.loadWarnings)
	}
}
