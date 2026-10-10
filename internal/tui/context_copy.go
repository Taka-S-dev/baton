package tui

import (
	"strings"

	mdl "github.com/Taka-S-dev/baton/internal/model"
	"github.com/Taka-S-dev/baton/internal/slot"
	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
)

type clipboardAction struct {
	key, label, value string
	paste             bool
}

type contextCopyMsg struct{ err error }

func (m Model) contextInputValue() string {
	if m.screen == ScreenEditList && m.le != nil {
		if m.le.adding {
			if m.le.addFld == 0 {
				return m.le.addVal
			}
			return m.le.addLbl
		}
		if m.le.editFld == 0 {
			return m.le.editValTI.Value()
		}
		return m.le.editLblTI.Value()
	}
	return m.nameInput.Value()
}

func (m Model) commandCopyActions() []clipboardAction {
	var c mdl.Command
	ok := false
	if m.wfe != nil {
		e := m.wfe
		names := e.steps
		i := e.cursor
		if e.adding {
			names = m.addCandidates()
			i = e.addCursor
		}
		if i >= 0 && i < len(names) {
			c, ok = m.workflowStepCommand(names[i])
		}
	} else if m.exploreTab == tabCommands {
		if m.cf != nil {
			f := m.cf.fields
			f[m.cf.fieldIdx] = m.nameInput.Value()
			c = mdl.Command{Cmd: f[1], Dir: f[2]}
			ok = true
		} else if it, found := m.exploreCurrent(); found {
			c, ok = m.config.FindCommand(it.name)
		}
	}
	if !ok {
		return nil
	}
	c = slot.ApplyVarsToCommand(c, m.vars)
	return []clipboardAction{{"c", "Copy command", c.Cmd, false}, {"w", "Copy workdir", displayOr(c.Dir, "."), false}}
}

func (m Model) clipboardMenuBody() string {
	var b strings.Builder
	for i, a := range m.contextActions {
		line := "   " + a.key + "  " + a.label
		if i == m.contextCursor {
			line = accent(" ▶ " + a.key + "  " + a.label)
		}
		b.WriteString(line + "\n")
	}
	b.WriteString("\n Enter Confirm  Esc Close")
	return b.String()
}

func (m Model) runClipboardAction(i int) (tea.Model, tea.Cmd) {
	m.pasteMenu = false
	if i < 0 || i >= len(m.contextActions) {
		return m, nil
	}
	a := m.contextActions[i]
	if a.paste {
		return m, readContextClipboard
	}
	return m, func() tea.Msg { return contextCopyMsg{clipboard.WriteAll(a.value)} }
}

func (m Model) updateClipboardMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch input := msg.(type) {
	case tea.KeyMsg:
		switch input.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			m.pasteMenu = false
		case "up":
			m.contextCursor = max(0, m.contextCursor-1)
		case "down":
			m.contextCursor = min(max(0, len(m.contextActions)-1), m.contextCursor+1)
		case "enter":
			return m.runClipboardAction(m.contextCursor)
		default:
			for i, a := range m.contextActions {
				if input.String() == a.key {
					return m.runClipboardAction(i)
				}
			}
		}
	case tea.MouseMsg:
		if input.Action == tea.MouseActionPress {
			p := m.pastePopup(m.viewWidth())
			m.pasteMenu = false
			row := input.Y - p.y - 1
			if input.Button == tea.MouseButtonLeft && input.X > p.x && input.X < p.x+p.w-1 && row >= 0 && row < len(m.contextActions) {
				return m.runClipboardAction(row)
			}
		}
	}
	return m, nil
}
