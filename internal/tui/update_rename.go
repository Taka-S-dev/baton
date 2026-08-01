package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Taka-S-dev/baton/internal/slot"
	"github.com/Taka-S-dev/baton/internal/store"
)

// ── Rename repair ─────────────────────────────────────────────────────────────
//
// Commands renamed by hand in commands.tsv / commands.json are detected
// at load (see config.DetectRenames) and offered here, before the main
// menu: one Enter rewrites every reference to the new names, Esc keeps
// the files as they are and leaves the usual warnings.

func (m *Model) gotoRenameRepair() {
	m.screen = ScreenRenameRepair
	m.renameBtn = 1 // repair is the expected outcome, so Yes starts focused
}

func (m *Model) updateRenameRepair(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "left", "right", "tab":
		m.renameBtn = 1 - m.renameBtn
	case "enter":
		if m.renameBtn == 1 {
			m.applyRenames()
		} else {
			m.declineRenames()
		}
	case "esc":
		m.declineRenames()
	}
	return m, nil
}

// applyRenames retargets every detected rename, persists the changed
// files, and reloads the project so saved commands pick their re-keyed
// vars.tsv values back up in the same run.
func (m *Model) applyRenames() {
	var wfSteps, tplRefs, varKeys int
	for _, r := range m.renames {
		wf, tpl, vk := m.retargetCommandRefs(r.Old, r.New)
		wfSteps += wf
		tplRefs += tpl
		varKeys += vk
	}
	var err error
	if wfSteps > 0 {
		err = store.SaveWorkflows(m.projectDir, m.workflows)
	}
	if err == nil && tplRefs > 0 {
		// saveConfig persists the moved template references and, via its
		// vars mirror, the re-keyed scoped values in one pass.
		err = m.saveConfig()
	} else if err == nil && varKeys > 0 {
		err = slot.SaveVars(m.projectDir, m.vars)
	}
	count := len(m.renames)
	m.renames = nil

	if lerr := m.loadProject(m.projectDir); lerr != nil && err == nil {
		err = lerr
	}
	m.renames = nil // a partial failure must not reopen the offer as a loop
	if err != nil {
		m.errMsg = "failed to update references: " + err.Error()
	} else {
		m.successMsg = fmt.Sprintf("updated references for %d renamed command(s)", count)
	}
	m.gotoMainMenu()
}

// declineRenames keeps every file untouched. The stale snapshot entries
// are dropped so the same offer is not repeated on every start; the
// dangling-reference warnings still point at the problem.
func (m *Model) declineRenames() {
	for _, r := range m.renames {
		delete(m.snapOld, r.Old)
	}
	m.renames = nil
	m.gotoMainMenu()
}
