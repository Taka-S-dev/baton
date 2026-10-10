package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Taka-S-dev/baton/internal/runner"
)

type runReadyMsg struct{}

// settleAfterRun is how long the done screen ignores input. It only has
// to outlast the burst the terminal delivers when it comes back, not a
// deliberate keypress, so it stays short enough to go unnoticed by
// someone reading the screen.
const settleAfterRun = 500 * time.Millisecond

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// A resize is every screen's business, so it is handled before the
	// forwarding below — which returns early and would otherwise leave
	// the layout stale on any screen holding a text input.
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = ws.Width
		m.height = ws.Height
		m.updateStepsViewport()
		// A field already holding a value has to be told the new width
		// before it is asked to draw, and then re-shown the value so it
		// works out what fits all over again.
		if m.cf != nil {
			m.resizeNameInput()
			pos := m.nameInput.Position()
			m.nameInput.SetValue(m.nameInput.Value())
			m.nameInput.SetCursor(pos)
		}
		return m, nil
	}

	// Always forward to spinner so TickMsg animates it.
	{
		s, c := m.spinner.Update(msg)
		m.spinner = s
		if c != nil {
			if _, ok := msg.(interface{ isSpinnerTick() }); ok {
				return m, c
			}
			_ = c
		}
	}

	// Forward non-key messages to textinput (handles internal blink tick).
	if _, ok := msg.(tea.KeyMsg); !ok {
		if m.screen == ScreenNameInput || m.screen == ScreenCreateCommandName || m.screen == ScreenEditCommandName || m.screen == ScreenCommandForm || m.screen == ScreenVarForm {
			ti, c := m.nameInput.Update(msg)
			m.nameInput = ti
			return m, c
		}
		if m.screen == ScreenEditList && m.le != nil && m.le.editing {
			if m.le.editFld == 0 {
				ti, c := m.le.editValTI.Update(msg)
				m.le.editValTI = ti
				return m, c
			}
			ti, c := m.le.editLblTI.Update(msg)
			m.le.editLblTI = ti
			return m, c
		}
		switch m.screen {
		case ScreenRunCommands, ScreenCreateWorkflow, ScreenEditWorkflowCommands:
			ti, c := m.msSearchTI.Update(msg)
			m.msSearchTI = ti
			return m, c
		case ScreenRunWorkflow:
			ti, c := m.wfSearchTI.Update(msg)
			m.wfSearchTI = ti
			return m, c
		}
	}

	switch msg := msg.(type) {
	case tea.MouseMsg:
		// Only the overview is pointable so far: it is the screen built
		// to be looked over rather than walked through, which is what
		// makes a pointer worth having.
		if m.screen == ScreenExplore {
			return m.updateExploreMouse(msg)
		}
		return m, nil
	case runReadyMsg:
		if m.running != nil {
			m.running.starting = false
			return m, m.runNext()
		}
		return m, nil
	case runner.DoneMsg:
		return m.handleRunnerDone(msg)
	case tea.KeyMsg:
		// Ctrl+C is not a signal here: the terminal is in raw mode, so it
		// arrives as an ordinary keystroke, and a program that ignores it
		// cannot be left at all from a text field. A run is the one place
		// it does not reach — the command owns the terminal and takes it.
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
		m.errMsg = ""
		m.successMsg = ""
		switch m.screen {
		case ScreenProjectSelect:
			return m.updateProjectSelect(msg)
		case ScreenMainMenu:
			return m.updateMainMenu(msg)
		case ScreenRunWorkflow:
			return m.updateRunWorkflow(msg)
		case ScreenRunWorkflowSteps:
			return m.updateRunWorkflowSteps(msg)
		case ScreenRunCommands, ScreenCreateWorkflow, ScreenEditWorkflowCommands:
			return m.updateMultiSelect(msg)
		case ScreenSlotPick:
			return m.updateSlotPick(msg)
		case ScreenConfirmRun:
			return m.updateConfirmRun(msg)
		case ScreenRunning:
			if m.running != nil && m.running.completed {
				// Keys pressed while a command was running went to that
				// command, not here, and arrive in a burst the moment
				// the terminal comes back. Dropping the burst is what
				// stops a run the user is still tapping Enter over from
				// walking the menu and starting itself again.
				if time.Since(m.running.completedAt) < settleAfterRun {
					return m, nil
				}
				m.running = nil
				m.gotoMainMenu()
				return m, tea.EnterAltScreen
			}
			return m, nil
		case ScreenRetry:
			return m.updateRetry(msg)
		case ScreenNameInput:
			return m.updateNameInput(msg)
		case ScreenEditWorkflow:
			return m.updateEditWorkflow(msg)
		case ScreenEditWorkflowMode:
			return m.updateEditWorkflowMode(msg)
		case ScreenDeleteWorkflow:
			return m.updateDeleteWorkflow(msg)
		case ScreenWorkflowMgmt:
			return m.updateWorkflowMgmt(msg)
		case ScreenManageLists:
			return m.updateManageLists(msg)
		case ScreenEditListPick:
			return m.updateEditListPick(msg)
		case ScreenEditList:
			return m.updateEditList(msg)
		case ScreenDeleteList:
			return m.updateDeleteLists(msg)
		case ScreenManageVars:
			return m.updateVarsMgmt(msg)
		case ScreenEditVarPick:
			return m.updateEditVarPick(msg)
		case ScreenVarForm:
			return m.updateVarForm(msg)
		case ScreenDeleteVar:
			return m.updateDeleteVars(msg)
		case ScreenVarRebase:
			return m.updateVarRebase(msg)
		case ScreenManageCommands:
			return m.updateManageCommands(msg)
		case ScreenCreateCommandKind:
			return m.updateCreateCommandKind(msg)
		case ScreenCommandForm:
			return m.updateCommandForm(msg)
		case ScreenEditCommandPick:
			return m.updateEditCommandPick(msg)
		case ScreenEditCommandMode:
			return m.updateEditCommandMode(msg)
		case ScreenCreateCommandName, ScreenCreateCommandTemplate:
			return m.updateCreateCommand(msg)
		case ScreenEditCommandName, ScreenEditCommandTemplate:
			return m.updateEditCommand(msg)
		case ScreenDeleteCommand:
			return m.updateDeleteCommand(msg)
		case ScreenSwitchConfig:
			return m.updateSwitchConfig(msg)
		case ScreenRenameRepair:
			return m.updateRenameRepair(msg)
		case ScreenExplore:
			// The overview edits in its own pane, so while a form is
			// open the keys belong to it, not to the tree behind it.
			if m.cf != nil {
				return m.updateCommandForm(msg)
			}
			if m.wfe != nil {
				return m.updateWorkflowEditor(msg)
			}
			return m.updateExplore(msg)
		}
	}
	return m, nil
}

func copyMap(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
