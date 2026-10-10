package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	mdl "github.com/Taka-S-dev/baton/internal/model"
)

// TestUpdate_ResizeReachesEveryScreen guards the dispatch order: screens
// holding a text input forward every non-key message to it and return,
// so a resize arriving there used to be swallowed and the layout stayed
// sized for the old terminal. Every screen must see it.
func TestUpdate_ResizeReachesEveryScreen(t *testing.T) {
	screens := []Screen{
		ScreenMainMenu,
		ScreenRunWorkflow,    // wfSearchTI
		ScreenRunCommands,    // msSearchTI
		ScreenCreateWorkflow, // msSearchTI
		ScreenNameInput,      // nameInput
		ScreenCommandForm,    // nameInput
		ScreenVarForm,        // nameInput
		ScreenSlotPick,
		ScreenConfirmRun,
		ScreenRunWorkflowSteps,
		ScreenExplore,
	}
	for _, sc := range screens {
		m := Model{screen: sc, width: 80, height: 24}
		m.spinner = spinner.New()
		m.nameInput = textinput.New()
		m.msSearchTI = textinput.New()
		m.wfSearchTI = textinput.New()
		m.workflows = []mdl.Workflow{{Name: "wf", Commands: []string{"a"}}}

		nm, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
		got := nm.(Model)
		if got.width != 120 || got.height != 40 {
			t.Errorf("screen %v: size = %dx%d, want 120x40 — the resize was swallowed",
				sc, got.width, got.height)
		}
	}
}

// TestUpdate_ResizeResizesStepsPreview checks the steps viewport is
// rebuilt for the new size rather than keeping the dimensions it was
// given at startup.
func TestUpdate_ResizeResizesStepsPreview(t *testing.T) {
	m := Model{screen: ScreenRunWorkflow, width: 80, height: 24}
	m.spinner = spinner.New()
	m.wfSearchTI = textinput.New()
	m.config = mdl.Config{Base: []mdl.Command{{Name: "a", Cmd: "echo hi"}}}
	m.workflows = []mdl.Workflow{{Name: "wf", Commands: []string{"a"}}}
	m.updateStepsViewport()
	before := m.stepsVP.Width

	nm, _ := m.Update(tea.WindowSizeMsg{Width: 140, Height: 50})
	got := nm.(Model)
	if got.stepsVP.Width == before {
		t.Fatalf("steps preview width stayed %d after a resize to 140", before)
	}
	if got.stepsVP.Width != 140-4 {
		t.Fatalf("steps preview width = %d, want the new terminal width less its frame", got.stepsVP.Width)
	}
}

// TestUpdate_CtrlCQuitsEveryScreen checks the one key every terminal user
// reaches for. The terminal is in raw mode, so Ctrl+C is delivered as a
// keystroke rather than a signal; baton ignored it, which left no way out
// of a text field except walking back with Esc.
func TestUpdate_CtrlCQuitsEveryScreen(t *testing.T) {
	for sc := ScreenProjectSelect; sc <= ScreenExplore; sc++ {
		m := Model{screen: sc, width: 80, height: 24}
		m.spinner = spinner.New()
		m.nameInput = textinput.New()
		m.msSearchTI = textinput.New()
		m.wfSearchTI = textinput.New()

		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
		if cmd == nil {
			t.Errorf("screen %v: Ctrl+C did nothing", sc)
			continue
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Errorf("screen %v: Ctrl+C did not quit", sc)
		}
	}
}
