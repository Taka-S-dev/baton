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
		if m.cf != nil && m.cf.returnTo == ScreenExplore {
			if m.exploreLayout(m.viewWidth()).single {
				m.screen = ScreenCommandForm
			} else {
				m.screen = ScreenExplore
			}
		}
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

	if m.narrowDetails && !m.helpOpen {
		if key, ok := msg.(tea.KeyMsg); ok {
			switch key.String() {
			case "esc", "v":
				m.narrowDetails = false
				return m, nil
			case "f1":
				m.helpOpen = true
				m.helpOffset = 0
				return m, nil
			case "ctrl+c":
				return m, tea.Quit
			}
		}
		return m, nil
	}
	if m.helpOpen {
		switch input := msg.(type) {
		case tea.KeyMsg:
			switch input.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "esc", "?", "f1":
				m.helpOpen = false
			case "up", "k":
				m.helpOffset = max(0, m.helpOffset-1)
			case "down", "j":
				m.helpOffset++
			case "pgdown":
				m.helpOffset += 8
			case "pgup":
				m.helpOffset = max(0, m.helpOffset-8)
			}
			m.clampHelpOffset()
			return m, nil
		case tea.MouseMsg:
			p := m.helpGeometry(m.viewWidth())
			if input.Button == tea.MouseButtonWheelDown {
				m.helpOffset += 3
			}
			if input.Button == tea.MouseButtonWheelUp {
				m.helpOffset = max(0, m.helpOffset-3)
			}
			if input.Action == tea.MouseActionPress && input.Button == tea.MouseButtonLeft && (input.X < p.x || input.X >= p.x+p.w || input.Y < p.y || input.Y >= p.y+p.h) {
				m.helpOpen = false
			}
			m.clampHelpOffset()
			return m, nil
		}
	}
	if key, ok := msg.(tea.KeyMsg); ok && m.screen != ScreenRunning {
		question := key.String() == "?" && m.screen == ScreenExplore && m.exploreListFocused() && !m.exploreSearching
		if key.String() == "f1" || question {
			m.helpOpen = true
			m.helpOffset = 0
			return m, nil
		}
	}
	if m.leaveAsk != leaveNone {
		switch input := msg.(type) {
		case tea.MouseMsg:
			return m.updateLeaveMouse(input)
		case tea.KeyMsg:
			if input.Type != tea.KeyCtrlC {
				return m, m.updateLeave(input)
			}
		}
	}

	if pasted, ok := msg.(contextPasteMsg); ok {
		if pasted.err != nil {
			m.errMsg = "Paste failed: " + pasted.err.Error()
			return m, nil
		}
		return m.insertClipboard(pasted.text)
	}
	if m.pasteMenu {
		return m.updateClipboardMenu(msg)
	}
	if input, ok := msg.(tea.MouseMsg); ok && input.Button == tea.MouseButtonRight && input.Action == tea.MouseActionPress && m.contextInputActive() {
		l := m.exploreLayout(m.viewWidth())
		if l.single || (input.X > l.rightX && input.X < l.rightX+l.rightW-1 && input.Y >= l.fieldY && input.Y < l.editY+l.editH-1) {
			m.contextActions = []clipboardAction{{"p", "Paste", "", true}, {"c", "Copy value", m.contextInputValue(), false}}
			m.contextCursor = 0
			m.pasteMenu = true
			m.contextX, m.contextY = input.X, input.Y
			return m, nil
		}
	}
	if input, ok := msg.(tea.KeyMsg); ok && input.Type == tea.KeyCtrlV && m.contextInputActive() {
		return m, readContextClipboard
	}

	if result, ok := msg.(contextCopyMsg); ok {
		if result.err != nil {
			m.errMsg = "Copy failed: " + result.err.Error()
		} else {
			m.successMsg = "Copied"
		}
		return m, nil
	}
	if mouse, ok := msg.(tea.MouseMsg); ok && mouse.Button == tea.MouseButtonRight && mouse.Action == tea.MouseActionPress && m.screen == ScreenExplore && !m.exploreMenu {
		l := m.exploreLayout(m.viewWidth())
		if !l.single && mouse.X > l.rightX && mouse.X < l.rightX+l.rightW-1 && mouse.Y >= l.fieldY && mouse.Y < l.bodyY+l.bodyH-1 {
			if actions := m.commandCopyActions(); len(actions) > 0 {
				m.contextActions = actions
				m.contextCursor = 0
				m.pasteMenu = true
				m.contextX, m.contextY = mouse.X, mouse.Y
				return m, nil
			}
		}
	}
	if key, ok := msg.(tea.KeyMsg); ok && key.String() == "v" && m.exploreLayout(m.viewWidth()).single && m.screen == ScreenExplore && m.cf == nil && !m.exploreSearching && !m.exploreMenu && (m.wfe == nil || !m.wfe.adding) {
		m.narrowDetails = true
		return m, nil
	}
	if result, ok := msg.(projectFolderMsg); ok {
		if result.err != nil {
			m.errMsg = "Open folder failed: " + result.err.Error()
		}
		return m, nil
	}
	if mouse, ok := msg.(tea.MouseMsg); ok && m.screen == ScreenProjectSelect {
		if m.projectMode != "" && m.projectMode != "search" {
			if mouse.Action == tea.MouseActionPress && mouse.Button == tea.MouseButtonLeft {
				pw, ph := min(68, m.viewWidth()), min(8, m.termHeight())
				x, y := max(0, (m.viewWidth()-pw)/2), max(0, (m.termHeight()-ph)/2)
				if mouse.X < x || mouse.X >= x+pw || mouse.Y < y || mouse.Y >= y+ph {
					m.projectMode = ""
					return m, nil
				}
				row := mouse.Y - y - 1
				if m.projectMode == "actions" && row >= 0 && row < 3 {
					return m.updateProjects(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune([]string{"o", "c", "x"}[row])})
				}
			}
			return m, nil
		}
		if mouse.Action == tea.MouseActionPress && mouse.Button == tea.MouseButtonLeft && mouse.Y == 2 {
			m.projectMode = "search"
			return m, nil
		}
		names := m.visibleProjects()
		y, rows := m.projectRows(m.viewWidth())
		if mouse.Action == tea.MouseActionPress {
			if mouse.Y == m.projectAddY(m.viewWidth()) && mouse.Button == tea.MouseButtonLeft {
				return m.updateProjects(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
			}
			row := mouse.Y - y
			i := windowStart(rows, len(names), m.listCursor) + row
			if row >= 0 && row < rows && i < len(names) {
				m.listCursor = i
				if mouse.Button == tea.MouseButtonRight {
					m.projectMode = "actions"
				}
			}
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

	if mouse, ok := msg.(tea.MouseMsg); ok && m.exploreTemplateFlow() && !m.exploreLayout(m.viewWidth()).single {
		return m.updateExploreTemplateMouse(mouse)
	}

	// Mouse input for embedded value editors must reach the pane before textinput.
	if mouse, ok := msg.(tea.MouseMsg); ok && m.exploreValueFlow() && !m.exploreLayout(m.viewWidth()).single {
		return m.updateExploreValueMouse(mouse)
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
