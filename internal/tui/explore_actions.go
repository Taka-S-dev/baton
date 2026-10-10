package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	mdl "github.com/Taka-S-dev/baton/internal/model"
	"github.com/Taka-S-dev/baton/internal/slot"
)

// An action is one thing that can be done to the row under the cursor.
// The same table drives the menu Space opens — which shows them with
// their keys, for someone who has not learnt the keys — and the keys
// themselves, for someone who has. The two cannot disagree because
// there is one list.
type exploreAction struct {
	key   string
	label string
}

// exploreActionsFor lists what can be done to a thing of the given kind.
func exploreActionsFor(kind exploreKind) []exploreAction {
	switch kind {
	case exCommand:
		return []exploreAction{
			{"e", "Edit"},
			{"d", "Duplicate"},
			{"r", "Rename"},
			{"x", "Delete"},
		}
	case exWorkflow:
		return []exploreAction{
			{"e", "Edit steps"},
			{"r", "Rename"},
			{"x", "Delete"},
		}
	case exList:
		return []exploreAction{
			{"e", "Edit entries"},
			{"x", "Delete"},
		}
	case exVar:
		return []exploreAction{
			{"e", "Edit value"},
			{"x", "Delete"},
		}
	}
	return nil
}

// runExploreAction carries out an action on an item by handing it to
// the flow that already does that job — the delete screens, the name
// input, the list and variable editors — with the item picked in
// advance so the flow opens on the step that matters. exploreLaunched
// is set so that when the flow would go back to its own menu, it comes
// back here instead.
func (m *Model) runExploreAction(key string, it wsItem) tea.Cmd {
	m.exploreMenu = false
	if it.kind == exCommand {
		for i, link := range m.commandLinks(it.name) {
			if key == linkKey(i) {
				m.openExploreLink(link.tab, link.name)
				return nil
			}
		}
	}
	m.exploreLaunched = true
	switch it.kind {
	case exCommand:
		if key == "f" {
			idx := m.getTemplateRefIdx(it.name)
			if idx < 0 {
				m.exploreLaunched = false
				return nil
			}
			m.sce = &commandEditState{mode: 0, editIdx: -1, templateRefIdx: idx}
			_, cmd := m.startCommandSlots()
			return cmd
		}
		return m.runCommandAction(key, it.name)
	case exWorkflow:
		return m.runWorkflowAction(key, it.name)
	case exList:
		return m.runListAction(key, it.name)
	case exVar:
		return m.runVarAction(key, it.name)
	}
	m.exploreLaunched = false
	return nil
}

func (m *Model) runCommandAction(key, name string) tea.Cmd {
	names, refs := m.editableCommands()
	at := -1
	for i, n := range names {
		if n == name {
			at = i
		}
	}
	if at < 0 {
		m.exploreLaunched = false
		m.errMsg = "only TSV rows and TUI-created commands can be changed here"
		return nil
	}
	ref := refs[at]
	cmd := m.editRefCommand(ref)
	switch key {
	case "e":
		if cmd.Template != "" {
			// A derived command's cmd is rebuilt from its template on
			// every load, so it is edited as its values and template,
			// not as text.
			idx := m.getTemplateRefIdx(cmd.Template)
			if idx < 0 {
				m.exploreLaunched = false
				m.errMsg = "template not found: " + cmd.Template
				return nil
			}
			m.sce = &commandEditState{mode: 1, editIdx: ref.idx, name: cmd.Name, templateRefIdx: idx, currentValues: map[string]string{}}
			m.gotoEditCommandMode()
			return nil
		}
		m.exploreLaunched = false
		return m.openExploreCommandForm(name)
	case "d":
		// A near copy is the common way a command starts: the same line
		// with a detail changed. The copy opens as a new command with
		// the fields filled and the name left to give, a free one shown
		// in place.
		m.exploreLaunched = false
		m.cf = &commandFormState{mode: 0, editIdx: -1, returnTo: ScreenExplore,
			fields: [5]string{"", cmd.Cmd, cmd.Dir, cmd.Group, cmd.Shell}}
		if m.exploreLayout(m.viewWidth()).single {
			m.screen = ScreenCommandForm
		}
		c := m.cfGotoField(0)
		m.nameInput.Placeholder = m.freeName(name)
		return c
	case "r":
		if ref.tsv {
			// A TSV row is renamed by saving the form with a new name:
			// that is the one path that rewrites the row in place and
			// moves every reference with it, so it is opened on the
			// name field rather than given a second rename of its own.
			m.exploreLaunched = false
			m.openExploreCommandForm(name)
			return m.cfGotoField(0)
		}
		m.sce = &commandEditState{mode: 1, editIdx: ref.idx, name: cmd.Name, currentValues: map[string]string{}}
		if cmd.Template != "" {
			m.sce.templateRefIdx = m.getTemplateRefIdx(cmd.Template)
		}
		m.nameInput.SetValue(cmd.Name)
		m.nameInput.Prompt = "Name > "
		m.nameInput.Placeholder = ""
		m.nameInputMode = nameInputRenameCommand
		m.nameInputErr = ""
		m.screen = ScreenNameInput
		return m.nameInput.Focus()
	case "x":
		m.screen = ScreenDeleteCommand
		m.editRefs = refs
		m.setPickBase(names, m.commandPickTexts(refs))
		m.listCursor = at
		m.openDeleteConfirm(at)
		return nil
	}
	m.exploreLaunched = false
	return nil
}

func (m *Model) runWorkflowAction(key, name string) tea.Cmd {
	at := -1
	for i, wf := range m.workflows {
		if wf.Name == name {
			at = i
		}
	}
	if at < 0 {
		m.exploreLaunched = false
		return nil
	}
	switch key {
	case "e":
		m.exploreLaunched = false
		m.openWorkflowEditor(name)
		return nil
	case "r":
		m.editTargetIdx = at
		m.nameInput.SetValue(name)
		m.nameInput.Prompt = "Name > "
		m.nameInput.Placeholder = ""
		m.nameInputMode = nameInputEditWorkflow
		m.nameInputErr = ""
		m.screen = ScreenNameInput
		return m.nameInput.Focus()
	case "x":
		m.screen = ScreenDeleteWorkflow
		m.setWorkflowPickBase()
		m.listCursor = at
		m.updateStepsViewport()
		m.openDeleteConfirm(at)
		return nil
	}
	m.exploreLaunched = false
	return nil
}

func (m *Model) runListAction(key, name string) tea.Cmd {
	switch key {
	case "e":
		m.le = &listEditState{name: name, entries: append([]mdl.ListEntry{}, m.lists[name]...)}
		m.screen = ScreenEditList
		return nil
	case "x":
		m.screen = ScreenDeleteList
		m.setListPickBase()
		at := indexOf(m.sortedListNames(), name)
		m.listCursor = max(0, at)
		m.openDeleteConfirm(at)
		return nil
	}
	m.exploreLaunched = false
	return nil
}

func (m *Model) runVarAction(key, name string) tea.Cmd {
	switch key {
	case "e":
		m.ve = &varEditState{mode: 1, name: name, oldValue: m.vars[name]}
		m.nameInput.SetValue(m.vars[name])
		m.nameInput.Prompt = "value > "
		m.nameInput.Placeholder = ""
		m.screen = ScreenVarForm
		return m.nameInput.Focus()
	case "x":
		m.screen = ScreenDeleteVar
		m.setVarPickBase()
		at := indexOf(m.varPickNames, name)
		m.listCursor = max(0, at)
		m.openDeleteConfirm(at)
		return nil
	}
	m.exploreLaunched = false
	return nil
}

// openDeleteConfirm enters a delete screen with one item already chosen
// and the question already asked, so deleting from the overview is the
// confirmation and nothing before it.
func (m *Model) openDeleteConfirm(idx int) {
	m.deleteSelected = nil
	if idx >= 0 {
		m.deleteSelected = []int{idx}
	}
	m.deleteConfirm = idx >= 0
	m.deleteBtn = 0
}

// freeName is base, or the first of base-2, base-3, … that no command
// has, for naming a copy.
func (m *Model) freeName(base string) string {
	if !m.commandNameTaken(base, -1) {
		return base
	}
	for i := 2; i < 100; i++ {
		if c := fmt.Sprintf("%s-%d", base, i); !m.commandNameTaken(c, -1) {
			return c
		}
	}
	return base
}

// indexOf is the position of s in all, or -1.
func indexOf(all []string, s string) int {
	for i, v := range all {
		if v == s {
			return i
		}
	}
	return -1
}

// returnFromFlow is what the management flows call when they would go
// back to their own menu. Launched from the overview, they come back to
// it instead, and the flag is spent on the way.
func (m *Model) returnFromFlow() bool {
	if !m.exploreLaunched {
		return false
	}
	m.exploreLaunched = false
	m.gotoExplore()
	return true
}

// selectExploreCreated reveals a saved item even when a search hid its name.
func (m *Model) selectExploreCreated(tab int, name string) {
	if !m.exploreLaunched {
		return
	}
	m.exploreTab = tab
	m.exploreSearch, m.exploreSearching = "", false
	for i, it := range m.exploreItems(tab) {
		if it.name == name {
			m.exploreCursor[tab] = i
			return
		}
	}
}

func (m Model) exploreActions(it wsItem) []exploreAction {
	actions := exploreActionsFor(it.kind)
	if it.kind == exCommand {
		if cmd, ok := m.config.FindCommand(it.name); ok && cmd.Template == "" && slot.HasPlaceholders(cmd) {
			actions = append(actions, exploreAction{"f", "From template"})
		}
	}
	if it.kind == exCommand {
		for i, link := range m.commandLinks(it.name) {
			actions = append(actions, exploreAction{linkKey(i), link.label})
		}
	}
	return actions
}
