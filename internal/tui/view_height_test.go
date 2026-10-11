package tui

import (
	"fmt"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/textinput"

	mdl "github.com/Taka-S-dev/baton/internal/model"
	"github.com/Taka-S-dev/baton/internal/slot"
)

// heightFixture builds a model with every screen's state populated, so
// one sweep can render all of them. The state is deliberately busy — a
// workflow with steps, a slot picker part-way through a run, commands
// with placeholders — because the screens that overflowed did so on the
// panels that grow with the work in hand.
func heightFixture(h int, sc Screen) Model {
	cmds := make([]mdl.Command, 8)
	for i := range cmds {
		cmds[i] = mdl.Command{Name: fmt.Sprintf("cmd-%d", i), Cmd: "echo hello {env}", Group: "g", Source: "tsv"}
	}
	m := Model{width: 100, height: h, screen: sc, projectDir: "p", configFile: "commands.tsv"}
	m.config = mdl.Config{Base: cmds}
	m.workflows = []mdl.Workflow{{Name: "wf", Commands: []string{"cmd-0", "cmd-1"}}}
	m.lists = map[string][]mdl.ListEntry{"env": {{Value: "dev"}, {Value: "stg"}, {Value: "prod"}}}
	m.vars = map[string]string{"root": `C:\work`}
	m.nameInput = textinput.New()
	m.msSearchTI = textinput.New()
	m.wfSearchTI = textinput.New()
	names, refs := m.editableCommands()
	m.editRefs = refs
	m.setPickBase(names, m.commandPickTexts(refs))
	m.listItems = names
	m.varPickNames = []string{"root"}

	entries := m.lists["env"]
	m.sp = &slotPickState{
		slotName: "env", listName: "env", entries: entries, filtered: entries,
		contextNames: []string{"cmd-0", "cmd-1", "cmd-2", "cmd-3", "cmd-4", "cmd-5", "cmd-6", "cmd-7"},
		contextIdx:   4,
		contextNotes: []string{"echo a", "echo b", "echo c", "echo d", "", "", "", ""},
		currentCmd:   &cmds[0], resolvedSoFar: map[string]string{},
	}
	items := []mdl.RunItem{{Name: "cmd-0", Cmd: &cmds[0]}, {Name: "cmd-1", Cmd: &cmds[1]}}
	m.confirmRunItems = items
	m.running = &runningState{
		items: items, current: 1, label: "wf", failed: true,
		failErr: fmt.Errorf("exit status 1"), runStart: time.Now(), stepStart: time.Now(),
		total: 2 * time.Second, completedAt: time.Now(),
	}
	m.ve = &varEditState{mode: 1, name: "root", oldValue: `C:\old`}
	m.le = &listEditState{name: "env", entries: entries, editValTI: textinput.New(), editLblTI: textinput.New()}
	m.cf = &commandFormState{fields: [5]string{"n", "echo hi", "", "g", ""}}
	m.sce = &commandEditState{name: "n", currentSlots: []slot.Def{{Name: "env"}}, currentValues: map[string]string{}}
	m.vr = &varRebaseState{varName: "root", items: []varRebaseItem{{label: "cmd-0.env", oldValue: `C:\old`}}}
	m.updateStepsViewport()
	m.exploreCursor[tabCommands] = 3
	return m
}

// screenFloors is the shortest terminal each screen fits in. A frame
// taller than the terminal makes the terminal scroll on every repaint,
// and what scrolls away is the top of the screen.
//
// The numbers are upper bounds, not targets: lowering one is an
// improvement and needs no change here, raising one is a regression and
// fails. The screens on the run path are the ones held lowest, because
// that is the path someone runs from an editor panel. What is left above
// eleven is fixed content — a form's fields, a confirmation, the rename
// report — where the only way lower is to stop showing something the
// screen exists to show.
var screenFloors = map[Screen]struct {
	name  string
	floor int
}{
	ScreenProjectSelect:         {"project select", 7},
	ScreenRunCommands:           {"run commands", 7},
	ScreenRunWorkflow:           {"run workflow", 7},
	ScreenRunWorkflowSteps:      {"run workflow steps", 8},
	ScreenExplore:               {"overview", 8},
	ScreenSlotPick:              {"slot pick", 13},
	ScreenConfirmRun:            {"confirm run", 11},
	ScreenRunning:               {"running", 8},
	ScreenRetry:                 {"retry", 12},
	ScreenNameInput:             {"name input", 9},
	ScreenWorkflowMgmt:          {"manage workflows", 8},
	ScreenEditWorkflow:          {"edit workflow", 8},
	ScreenEditWorkflowMode:      {"edit workflow mode", 8},
	ScreenEditWorkflowCommands:  {"edit workflow commands", 8},
	ScreenDeleteWorkflow:        {"delete workflow", 8},
	ScreenManageLists:           {"manage lists", 8},
	ScreenEditListPick:          {"edit list pick", 8},
	ScreenEditList:              {"edit list", 11},
	ScreenDeleteList:            {"delete list", 8},
	ScreenManageVars:            {"manage vars", 8},
	ScreenEditVarPick:           {"edit var pick", 8},
	ScreenVarForm:               {"var form", 12},
	ScreenDeleteVar:             {"delete var", 8},
	ScreenVarRebase:             {"var rebase", 13},
	ScreenSwitchConfig:          {"switch config", 8},
	ScreenManageCommands:        {"manage commands", 11},
	ScreenEditCommandPick:       {"edit command pick", 7},
	ScreenEditCommandMode:       {"edit command mode", 8},
	ScreenCreateCommandKind:     {"create command kind", 8},
	ScreenCreateCommandName:     {"create command name", 10},
	ScreenCreateCommandTemplate: {"create command template", 16},
	ScreenEditCommandName:       {"edit command name", 8},
	ScreenEditCommandTemplate:   {"edit command template", 8},
	ScreenCommandForm:           {"command form", 13},
	ScreenDeleteCommand:         {"delete command", 8},
	ScreenRenameRepair:          {"rename repair", 16},
}

// TestEveryScreenFitsItsFloor sweeps every screen over every terminal
// height from its floor up. Screens were reaching twenty-one lines and
// more — the slot picker and the delete screens kept a hover panel of
// ten rows whatever the terminal had left — so this is the invariant
// that keeps them honest.
func TestEveryScreenFitsItsFloor(t *testing.T) {
	for sc, want := range screenFloors {
		for h := want.floor; h <= 40; h++ {
			view := heightFixture(h, sc).View()
			if got := frameLines(view); got > h {
				t.Errorf("%s at height %d renders %d lines\n%s", want.name, h, got, view)
				break
			}
		}
	}
}

// TestNoScreenIsTallerThanAStandardTerminal is the backstop: whatever
// each screen's floor, none of them may overflow a terminal of the size
// people actually have. The confirm screen ran to twenty-six lines on
// exactly this one.
func TestNoScreenIsTallerThanAStandardTerminal(t *testing.T) {
	for sc, want := range screenFloors {
		for _, h := range []int{24, 30, 50} {
			if got := frameLines(heightFixture(h, sc).View()); got > h {
				t.Errorf("%s renders %d lines in a %d-line terminal", want.name, got, h)
			}
		}
	}
}
