package tui

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"

	mdl "github.com/Taka-S-dev/baton/internal/model"
	"github.com/Taka-S-dev/baton/internal/slot"
)

// ── Name input ────────────────────────────────────────────────────────────────

func (m Model) openNameInput(mode nameInputMode) (tea.Model, tea.Cmd) {
	m.nameInput.SetValue("")
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
		case nameInputAlias:
			return m.saveAlias(name)
		case nameInputEditWorkflow:
			return m.renameWorkflow(m.editTargetIdx, name)
		case nameInputEditAlias:
			return m.renameAlias(m.editTargetIdx, name)
		case nameInputNewList:
			return m.saveNewList(name)
		}
	case "esc":
		switch m.nameInputMode {
		case nameInputEditWorkflow:
			m.screen = ScreenEditWorkflowMode
			m.listItems = []string{"Rename", "Change commands"}
			m.listCursor = 0
		case nameInputEditAlias:
			m.screen = ScreenEditAliasMode
			m.listItems = []string{"Rename", "Change commands"}
			m.listCursor = 0
		case nameInputNewList:
			m.gotoManageLists()
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
	if m.deleteConfirm {
		switch msg.String() {
		case "tab", "left", "right", "h", "l":
			m.deleteBtn = 1 - m.deleteBtn
		case "enter":
			if m.deleteBtn == 1 && len(m.listItems) > 0 {
				name := m.listItems[m.listCursor]
				listsDir := filepath.Join(m.projectDir, "lists")
				if err := os.Remove(filepath.Join(listsDir, name+".tsv")); err != nil {
					m.errMsg = "failed to delete list: " + err.Error()
				}
				delete(m.lists, name)
				m.listItems = append(m.listItems[:m.listCursor], m.listItems[m.listCursor+1:]...)
				if m.listCursor >= len(m.listItems) && m.listCursor > 0 {
					m.listCursor--
				}
			}
			m.deleteConfirm = false
			m.deleteBtn = 0
		case "esc":
			m.deleteConfirm = false
			m.deleteBtn = 0
		}
		return m, nil
	}
	switch msg.String() {
	case "up", "k":
		if m.listCursor > 0 {
			m.listCursor--
		}
	case "down", "j":
		if m.listCursor < len(m.listItems)-1 {
			m.listCursor++
		}
	case "enter":
		if len(m.listItems) == 0 {
			break
		}
		name := m.listItems[m.listCursor]
		m.le = &listEditState{
			name:    name,
			entries: append([]mdl.ListEntry{}, m.lists[name]...),
		}
		m.screen = ScreenEditList
	case "n", "a":
		return m.openNameInput(nameInputNewList)
	case "d", "delete":
		if len(m.listItems) > 0 {
			m.deleteConfirm = true
			m.deleteBtn = 0
		}
	case "esc":
		m.gotoMainMenu()
	}
	return m, nil
}

func (m Model) saveNewList(name string) (tea.Model, tea.Cmd) {
	if _, exists := m.lists[name]; exists {
		m.nameInputErr = "list already exists"
		return m, nil
	}
	listsDir := filepath.Join(m.projectDir, "lists")
	if err := slot.SaveList(listsDir, name, nil); err != nil {
		m.errMsg = "failed to create list: " + err.Error()
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
	case "up", "k":
		if le.cursor > 0 {
			le.cursor--
		}
	case "down", "j":
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
		m.gotoManageLists()
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
