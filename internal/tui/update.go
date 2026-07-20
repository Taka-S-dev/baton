package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Taka-S-dev/baton/internal/runner"
)

type runReadyMsg struct{}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		if m.screen == ScreenNameInput || m.screen == ScreenCreateCommandName || m.screen == ScreenEditCommandName || m.screen == ScreenCommandForm {
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
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateStepsViewport()
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
		m.errMsg = ""
		m.successMsg = ""
		switch m.screen {
		case ScreenProjectSelect:
			return m.updateProjectSelect(msg)
		case ScreenMainMenu:
			return m.updateMainMenu(msg)
		case ScreenRunWorkflow:
			return m.updateRunWorkflow(msg)
		case ScreenRunCommands, ScreenCreateWorkflow, ScreenEditWorkflowCommands:
			return m.updateMultiSelect(msg)
		case ScreenSlotPick:
			return m.updateSlotPick(msg)
		case ScreenConfirmRun:
			return m.updateConfirmRun(msg)
		case ScreenRunning:
			if m.running != nil && m.running.completed {
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
		case ScreenEditList:
			return m.updateEditList(msg)
		case ScreenManageCommands:
			return m.updateManageCommands(msg)
		case ScreenCreateCommandKind:
			return m.updateCreateCommandKind(msg)
		case ScreenCommandForm:
			return m.updateCommandForm(msg)
		case ScreenEditCommandPick:
			return m.updateEditCommandPick(msg)
		case ScreenCreateCommandName, ScreenCreateCommandTemplate:
			return m.updateCreateCommand(msg)
		case ScreenEditCommandName, ScreenEditCommandTemplate:
			return m.updateEditCommand(msg)
		case ScreenDeleteCommand:
			return m.updateDeleteCommand(msg)
		case ScreenSwitchConfig:
			return m.updateSwitchConfig(msg)
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
