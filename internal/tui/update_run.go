package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	mdl "github.com/Taka-S-dev/baton/internal/model"
	"github.com/Taka-S-dev/baton/internal/runner"
	"github.com/Taka-S-dev/baton/internal/slot"
	"github.com/Taka-S-dev/baton/internal/store"
)

// ── Confirm run ───────────────────────────────────────────────────────────────

func (m Model) startConfirmRun(items []mdl.RunItem, label string) (tea.Model, tea.Cmd) {
	m.confirmRunItems = items
	m.confirmRunLabel = label
	m.confirmRunBtn = 0
	m.screen = ScreenConfirmRun
	m.resolve = nil
	m.sp = nil
	return m, nil
}

func (m Model) updateConfirmRun(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab", "left", "right":
		m.confirmRunBtn = 1 - m.confirmRunBtn
	case "enter":
		if m.confirmRunBtn == 0 {
			return m.startRunning(m.confirmRunItems, 0, m.confirmRunLabel)
		}
		m.gotoMainMenu()
	case "esc":
		m.gotoMainMenu()
	}
	return m, nil
}

// ── Running ───────────────────────────────────────────────────────────────────

func (m Model) startRunning(items []mdl.RunItem, startIdx int, label string) (tea.Model, tea.Cmd) {
	return m.startRunningRetry(items, startIdx, label, 0)
}

func (m Model) startRunningRetry(items []mdl.RunItem, startIdx int, label string, retryCount int) (tea.Model, tea.Cmd) {
	m.running = &runningState{
		items:      items,
		current:    startIdx,
		startIdx:   startIdx,
		starting:   true,
		label:      label,
		retryCount: retryCount,
	}
	m.screen = ScreenRunning
	return m, tea.Sequence(
		tea.ExitAltScreen,
		func() tea.Msg { return runReadyMsg{} },
	)
}

func (m Model) runNext() tea.Cmd {
	r := m.running
	if r.current >= len(r.items) {
		return nil
	}
	item := r.items[r.current]
	if item.IsAlias() {
		var cmds []mdl.RunItem
		for _, stepName := range item.Alias.Steps {
			for i := range m.config.Commands {
				if m.config.Commands[i].Name == stepName {
					c := m.config.Commands[i]
					if item.VarMap != nil {
						c = slot.Apply(c, item.VarMap)
					} else if item.Alias.Vars != nil {
						if vars, ok := item.Alias.Vars[stepName]; ok {
							c = slot.Apply(c, vars)
						}
					}
					cmds = append(cmds, mdl.RunItem{Name: stepName, Cmd: &c})
					break
				}
			}
		}
		newItems := make([]mdl.RunItem, 0, len(r.items)-1+len(cmds))
		newItems = append(newItems, r.items[:r.current]...)
		newItems = append(newItems, cmds...)
		newItems = append(newItems, r.items[r.current+1:]...)
		r.items = newItems
		return m.runNext()
	}
	stepHeader := fmt.Sprintf("\n── [%d/%d] %s", r.current+1, len(r.items), item.Name)
	if item.Cmd.Dir != "" {
		stepHeader += fmt.Sprintf("   workdir: %s", item.Cmd.Dir)
	}
	prefix := ""
	if r.current == r.startIdx {
		label := r.label
		if r.retryCount > 0 {
			label = fmt.Sprintf("%s (retry #%d)", r.label, r.retryCount)
		}
		sep := strings.Repeat("─", 48)
		if label != "" {
			pad := strings.Repeat("─", max(0, 48-len(label)-4))
			sep = "── " + label + " " + pad
		}
		prefix = "\n" + sep + "\n"
	}
	return tea.Sequence(tea.Println(prefix+stepHeader), runner.Exec(r.current, *item.Cmd, m.dryRun))
}

func (m Model) handleRunnerDone(msg runner.DoneMsg) (tea.Model, tea.Cmd) {
	r := m.running
	if msg.Err != nil {
		r.failed = true
		r.failErr = msg.Err
		m.screen = ScreenRetry
		m.listCursor = 0
		return m, nil
	}
	r.current++
	bar := progressBar(r.current, len(r.items), 24)
	if r.current >= len(r.items) {
		r.completed = true
		return m, tea.Sequence(tea.Println(bar), tea.ExitAltScreen)
	}
	return m, tea.Sequence(tea.Println(bar), m.runNext())
}

// ── Retry ─────────────────────────────────────────────────────────────────────

func (m Model) updateRetry(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	r := m.running
	items := []string{
		fmt.Sprintf("Retry from step %d", r.current+1),
		"Retry all",
		"Abort",
	}
	switch msg.String() {
	case "up":
		if m.listCursor > 0 {
			m.listCursor--
		}
	case "down":
		if m.listCursor < len(items)-1 {
			m.listCursor++
		}
	case "enter":
		switch m.listCursor {
		case 0:
			return m.startRunningRetry(r.items, r.current, r.label, r.retryCount+1)
		case 1:
			return m.startRunningRetry(r.items, 0, r.label, r.retryCount+1)
		case 2:
			m.gotoMainMenu()
			return m, tea.EnterAltScreen
		}
	case "esc":
		m.gotoMainMenu()
		return m, tea.EnterAltScreen
	}
	return m, nil
}

// ── Run workflow ──────────────────────────────────────────────────────────────

func (m Model) updateRunWorkflow(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		if len(m.workflows) > 0 {
			m.stepsFocused = !m.stepsFocused
		}
	case "up":
		if m.stepsFocused {
			m.stepsVP.ScrollUp(1)
		} else if m.listCursor > 0 {
			m.listCursor--
			m.updateStepsViewport()
		}
	case "down":
		if m.stepsFocused {
			m.stepsVP.ScrollDown(1)
		} else if m.listCursor < len(m.listItems)-1 {
			m.listCursor++
			m.updateStepsViewport()
		}
	case "enter":
		if len(m.workflows) == 0 {
			break
		}
		wf := m.workflows[m.listCursor]
		store.SaveLastWorkflow(m.projectDir, wf.Name)
		m.lastWorkflow = wf.Name

		// Apply saved vars first; any remaining {slots} are resolved interactively.
		var preCmds []mdl.Command
		var names []string
		for _, name := range wf.Commands {
			for i := range m.config.Commands {
				if m.config.Commands[i].Name == name {
					cmd := m.config.Commands[i]
					if wf.Vars != nil {
						if vars, ok := wf.Vars[name]; ok {
							cmd = slot.Apply(cmd, vars)
						}
					}
					preCmds = append(preCmds, cmd)
					names = append(names, name)
					break
				}
			}
		}
		msItems := make([]msItem, len(preCmds))
		for i := range preCmds {
			msItems[i] = msItem{cmd: &preCmds[i]}
		}
		m.resolve = &resolveFlowState{
			purpose:       purposeRunWorkflow,
			rawItems:      msItems,
			itemNames:     names,
			itemNotes:     make([]string, len(msItems)),
			workflowVars:  make(map[string]map[string]string),
			workflowLabel: wf.Name,
		}
		return m.advanceResolve()
	case "esc":
		m.gotoMainMenu()
	}
	return m, nil
}
