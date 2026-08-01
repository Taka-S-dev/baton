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
	m.confirmRunScroll = 0
	m.screen = ScreenConfirmRun
	m.resolve = nil
	m.sp = nil
	return m, nil
}

func (m Model) updateConfirmRun(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up":
		if m.confirmRunScroll > 0 {
			m.confirmRunScroll--
		}
	case "down":
		if m.confirmRunScroll < max(0, len(m.confirmRunItems)-m.confirmRunPerPage()) {
			m.confirmRunScroll++
		}
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

// stepHeader builds the lines printed before a step runs: position, name,
// workdir, and the resolved command line so the exact invocation is visible
// in the scrollback.
func stepHeader(pos, total int, name string, cmd mdl.Command) string {
	h := fmt.Sprintf("\n── [%d/%d] %s", pos, total, name)
	if cmd.Dir != "" {
		h += fmt.Sprintf("   workdir: %s", cmd.Dir)
	}
	return h + fmt.Sprintf("\n   $ %s", cmd.Cmd)
}

func (m Model) runNext() tea.Cmd {
	r := m.running
	if r.current >= len(r.items) {
		return nil
	}
	item := r.items[r.current]
	cmd := slot.ApplyVarsToCommand(*item.Cmd, m.vars)
	header := stepHeader(r.current+1, len(r.items), item.Name, cmd)
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
	return tea.Sequence(tea.Println(prefix+header), runner.Exec(r.current, cmd, m.dryRun))
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
	case "up", "down":
		m.moveListCursor(msg.String(), len(items))
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

// wfSearchText returns the lowercase haystack a workflow is matched
// against: its name and the commands it runs (name and resolved body,
// as shown in the steps preview).
func (m *Model) wfSearchText(wf mdl.Workflow) string {
	parts := []string{wf.Name}
	parts = append(parts, wf.Commands...)
	for _, name := range wf.Commands {
		if cmd, ok := m.workflowStepCommand(name); ok {
			parts = append(parts, cmd.Cmd, cmd.Group)
		}
	}
	return strings.ToLower(strings.Join(parts, " "))
}

func (m *Model) wfFiltered() []int {
	var out []int
	terms := strings.Fields(strings.ToLower(m.wfSearchTI.Value()))
	for i, wf := range m.workflows {
		if matchesAllTerms(m.wfSearchText(wf), terms) {
			out = append(out, i)
		}
	}
	return out
}

func (m Model) updateRunWorkflow(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	filtered := m.wfFiltered()
	n := len(filtered)

	switch msg.String() {
	case "tab":
		if n > 0 {
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
		} else if m.listCursor < n-1 {
			m.listCursor++
			m.updateStepsViewport()
		}
	case "enter":
		if n == 0 || m.listCursor >= n {
			break
		}
		wf := m.workflows[filtered[m.listCursor]]
		// A step that no longer resolves refuses the whole run: silently
		// executing 4 of 5 steps looks like success while skipping real
		// work, which is worse than making the user fix the workflow.
		var missing []string
		for _, name := range wf.Commands {
			if _, ok := m.config.FindCommand(name); !ok {
				missing = append(missing, name)
			}
		}
		if len(missing) > 0 {
			m.errMsg = fmt.Sprintf("cannot run %q — step(s) not found: %s (renamed or deleted? fix it in Manage workflows)",
				wf.Name, strings.Join(missing, ", "))
			break
		}
		store.SaveLastWorkflow(m.projectDir, wf.Name)
		m.lastWorkflow = wf.Name

		// Remaining {slots} are resolved interactively per step.
		var msItems []msItem
		var names []string
		for _, name := range wf.Commands {
			cmd, _ := m.config.FindCommand(name)
			cmdCopy := cmd
			msItems = append(msItems, msItem{cmd: &cmdCopy})
			names = append(names, name)
		}
		m.resolve = &resolveFlowState{
			purpose:       purposeRunWorkflow,
			rawItems:      msItems,
			itemNames:     names,
			itemNotes:     make([]string, len(msItems)),
			workflowLabel: wf.Name,
		}
		return m.advanceResolve()
	case "esc":
		if m.wfSearchTI.Value() != "" {
			m.wfSearchTI.SetValue("")
			m.listCursor = 0
			m.updateStepsViewport()
			return m, nil
		}
		m.gotoMainMenu()
	default:
		prev := m.wfSearchTI.Value()
		var cmd tea.Cmd
		m.wfSearchTI, cmd = m.wfSearchTI.Update(msg)
		if m.wfSearchTI.Value() != prev {
			m.listCursor = 0
			m.updateStepsViewport()
		}
		return m, cmd
	}
	return m, nil
}
