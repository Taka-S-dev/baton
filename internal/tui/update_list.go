package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	mdl "github.com/Taka-S-dev/baton/internal/model"
	"github.com/Taka-S-dev/baton/internal/slot"
)

// ── Name input ────────────────────────────────────────────────────────────────

func (m Model) openNameInput(mode nameInputMode) (tea.Model, tea.Cmd) {
	m.nameInput.SetValue("")
	// Other forms (command fields, vars) repurpose the shared input with
	// their own prompts — reset so a leftover prompt never leaks in here.
	m.nameInput.Prompt = "Name > "
	m.nameInputMode = mode
	m.nameInputErr = ""
	m.screen = ScreenNameInput
	return m, m.nameInput.Focus()
}

func (m Model) updateNameInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		name := strings.TrimSpace(m.nameInput.Value())
		if name == "" {
			return m, nil
		}
		switch m.nameInputMode {
		case nameInputWorkflow:
			return m.saveWorkflow(name)
		case nameInputEditWorkflow:
			return m.renameWorkflow(m.editTargetIdx, name)
		case nameInputRenameCommand:
			return m.renameCommand(name)
		case nameInputNewList:
			return m.saveNewList(name)
		}
	case "esc":
		switch m.nameInputMode {
		case nameInputEditWorkflow:
			m.screen = ScreenEditWorkflowMode
			m.listItems = []string{"Rename", "Change commands"}
			m.listCursor = 0
		case nameInputRenameCommand:
			m.gotoEditCommandMode()
		case nameInputNewList:
			m.gotoManageLists()
		case nameInputWorkflow:
			m.gotoWorkflowMgmt()
		default:
			m.gotoMainMenu()
		}
		return m, nil
	}
	ti, cmd := m.nameInput.Update(msg)
	m.nameInput = ti
	return m, cmd
}

// ── Manage lists ──────────────────────────────────────────────────────────────

func (m Model) updateManageLists(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "down":
		m.moveListCursor(msg.String(), len(m.listItems))
	case "enter":
		switch m.listItems[m.listCursor] {
		case "Create list":
			return m.openNameInput(nameInputNewList)
		case "Edit list":
			m.screen = ScreenEditListPick
			m.setListPickBase()
			m.listCursor = 0
		case "Delete list":
			m.screen = ScreenDeleteList
			m.setListPickBase()
			m.listCursor = 0
		}
	case "esc":
		m.gotoMainMenu()
	}
	return m, nil
}

// ── Edit list: pick which one ─────────────────────────────────────────────────

// setListPickBase fills the pick filter with the list names, searchable
// by name and by the entry values/labels they contain.
func (m *Model) setListPickBase() {
	names := m.sortedListNames()
	texts := make([]string, len(names))
	for i, n := range names {
		parts := []string{n}
		for _, e := range m.lists[n] {
			parts = append(parts, e.Value, e.Label)
		}
		texts[i] = strings.Join(parts, " ")
	}
	m.setPickBase(names, texts)
}

func (m Model) updateEditListPick(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "down":
		m.moveListCursor(msg.String(), len(m.listItems))
	case "enter":
		if len(m.listItems) == 0 {
			break
		}
		name := m.listItems[m.listCursor]
		m.le = &listEditState{
			name:     name,
			entries:  append([]mdl.ListEntry{}, m.lists[name]...),
			fromPick: true,
		}
		m.screen = ScreenEditList
	case "esc":
		if m.pickSearch != "" {
			m.clearPickFilter()
			break
		}
		m.gotoManageLists()
	default:
		m.handlePickTyping(msg, nil)
	}
	return m, nil
}

// ── Delete list ───────────────────────────────────────────────────────────────

func (m Model) updateDeleteLists(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	names := m.pickBase // onDelete receives original indices
	return m.updateDeleteList(msg, len(m.listItems), nil,
		m.gotoManageLists,
		func(indices []int) {
			listsDir := filepath.Join(m.projectDir, "lists")
			failed := false
			for _, i := range indices {
				name := names[i]
				if err := os.Remove(filepath.Join(listsDir, name+".tsv")); err != nil {
					m.errMsg = "failed to delete list: " + err.Error()
					failed = true
				}
				delete(m.lists, name)
			}
			if !failed {
				if len(indices) == 1 {
					m.successMsg = "deleted list \"" + names[indices[0]] + "\""
				} else {
					m.successMsg = fmt.Sprintf("deleted %d lists", len(indices))
				}
			}
		})
}

func (m Model) saveNewList(name string) (tea.Model, tea.Cmd) {
	if _, exists := m.lists[name]; exists {
		m.nameInputErr = "name already in use: " + name
		return m, nil
	}
	listsDir := filepath.Join(m.projectDir, "lists")
	if err := slot.SaveList(listsDir, name, nil); err != nil {
		m.errMsg = "failed to create list: " + err.Error()
	} else {
		m.successMsg = "created list \"" + name + "\""
	}
	m.lists[name] = []mdl.ListEntry{}
	m.le = &listEditState{
		name:    name,
		entries: []mdl.ListEntry{},
	}
	m.screen = ScreenEditList
	return m, nil
}

// ── Edit list ─────────────────────────────────────────────────────────────────

func (m Model) updateEditList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	le := m.le
	total := len(le.entries) + 1

	if le.editing {
		switch msg.String() {
		case "enter":
			if le.editFld == 0 {
				le.editFld = 1
				le.editValTI.Blur()
				cmd := le.editLblTI.Focus()
				return m, cmd
			}
			oldVal := le.entries[le.cursor].Value
			le.entries[le.cursor] = mdl.ListEntry{
				Value: le.editValTI.Value(),
				Label: le.editLblTI.Value(),
			}
			le.editing = false
			listsDir := filepath.Join(m.projectDir, "lists")
			if err := slot.SaveList(listsDir, le.name, le.entries); err != nil {
				m.errMsg = "failed to save list: " + err.Error()
			}
			m.lists[le.name] = append([]mdl.ListEntry{}, le.entries...)
			// Other values that shared the old value — fixed values and
			// entries of any list — can follow, offered explicitly.
			if newVal := le.editValTI.Value(); oldVal != "" && oldVal != newVal {
				if vr := m.buildPropagate("list \""+le.name+"\" entry", oldVal, newVal, "", le.name, le.cursor); vr != nil {
					vr.returnToList = true
					vr.editedOld, vr.editedNew = oldVal, newVal
					m.vr = vr
					m.screen = ScreenVarRebase
				}
			}
		case "esc":
			le.editing = false
		default:
			if le.editFld == 0 {
				ti, cmd := le.editValTI.Update(msg)
				le.editValTI = ti
				return m, cmd
			}
			ti, cmd := le.editLblTI.Update(msg)
			le.editLblTI = ti
			return m, cmd
		}
		return m, nil
	}

	if le.adding {
		switch msg.String() {
		case "enter":
			if le.addFld == 0 {
				if le.addVal != "" {
					le.addFld = 1
				}
			} else {
				le.entries = append(le.entries, mdl.ListEntry{Value: le.addVal, Label: le.addLbl})
				le.cursor = len(le.entries) - 1
				le.adding = false
				le.addVal = ""
				le.addLbl = ""
				le.addFld = 0
				listsDir := filepath.Join(m.projectDir, "lists")
				if err := slot.SaveList(listsDir, le.name, le.entries); err != nil {
					m.errMsg = "failed to save list: " + err.Error()
				}
				m.lists[le.name] = append([]mdl.ListEntry{}, le.entries...)
			}
		case "backspace":
			if le.addFld == 0 && len(le.addVal) > 0 {
				le.addVal = le.addVal[:len(le.addVal)-1]
			} else if le.addFld == 1 && len(le.addLbl) > 0 {
				le.addLbl = le.addLbl[:len(le.addLbl)-1]
			}
		case "esc":
			le.adding = false
		default:
			if len(msg.Runes) == 1 && msg.Runes[0] >= 32 {
				if le.addFld == 0 {
					le.addVal += string(msg.Runes[0])
				} else {
					le.addLbl += string(msg.Runes[0])
				}
			}
		}
		return m, nil
	}

	switch msg.String() {
	case "up":
		if le.cursor > 0 {
			le.cursor--
		}
	case "down":
		if le.cursor < total-1 {
			le.cursor++
		}
	case "enter":
		if le.cursor == len(le.entries) {
			le.adding = true
			le.addFld = 0
		} else {
			entry := le.entries[le.cursor]
			valTI := newListTextinput("Value > ", entry.Value)
			lblTI := newListTextinput("Label > ", entry.Label)
			le.editValTI = valTI
			le.editLblTI = lblTI
			le.editFld = 0
			le.editing = true
			return m, le.editValTI.Focus()
		}
	case "delete", "d":
		if le.cursor < len(le.entries) {
			le.entries = append(le.entries[:le.cursor], le.entries[le.cursor+1:]...)
			if le.cursor >= len(le.entries) && le.cursor > 0 {
				le.cursor--
			}
			listsDir := filepath.Join(m.projectDir, "lists")
			if err := slot.SaveList(listsDir, le.name, le.entries); err != nil {
				m.errMsg = "failed to save list: " + err.Error()
			}
			m.lists[le.name] = append([]mdl.ListEntry{}, le.entries...)
		}
	case "esc":
		if le.fromPick {
			// Return to the pick screen, cursor on the list just edited.
			m.screen = ScreenEditListPick
			m.setListPickBase()
			m.listCursor = 0
			for i, n := range m.listItems {
				if n == le.name {
					m.listCursor = i
					break
				}
			}
		} else {
			m.gotoManageLists()
		}
	}
	return m, nil
}

func newListTextinput(prompt, value string) textinput.Model {
	ti := textinput.New()
	ti.Prompt = prompt
	ti.SetValue(value)
	ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("36"))
	ti.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("97"))
	ti.Width = 48
	ti.CharLimit = 256
	return ti
}
