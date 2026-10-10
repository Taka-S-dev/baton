package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"runtime"
	"strings"
)

func shellChoiceLabel(value string) string {
	if strings.EqualFold(value, "ps") {
		return "PowerShell (ps)"
	}
	if value != "" {
		return "Unknown: " + value
	}
	if runtime.GOOS == "windows" {
		return "Default (cmd.exe)"
	}
	return "Default (sh)"
}

func (m *Model) acceptShellChoice() {
	value := ""
	if m.cf.shellCursor == 1 {
		value = "ps"
	}
	m.cf.fields[4] = value
	m.nameInput.SetValue(value)
	m.cf.shellOpen = false
	m.cf.fieldErr = ""
}

// Text entry is disabled for this enum; Tab remains form navigation.
func (m *Model) updateShellChoice(msg tea.KeyMsg) (bool, tea.Cmd) {
	cf := m.cf
	switch msg.String() {
	case "tab", "shift+tab":
		cf.shellOpen = false
		return false, nil
	case "esc":
		if cf.shellOpen {
			cf.shellOpen = false
			return true, nil
		}
		return false, nil
	case "up", "down":
		if !cf.shellOpen {
			return false, nil
		}
		if msg.String() == "up" {
			cf.shellCursor = 0
		} else {
			cf.shellCursor = 1
		}
		return true, nil
	case "enter", " ":
		if cf.shellOpen {
			m.acceptShellChoice()
		} else {
			cf.shellOpen = true
		}
		return true, nil
	case "ctrl+s":
		if cf.shellOpen {
			m.acceptShellChoice()
		}
		return false, nil
	}
	return true, nil
}

func (m Model) writeShellChoices(b *strings.Builder) {
	for i, value := range []string{"", "ps"} {
		marker := "   "
		label := shellChoiceLabel(value)
		if i == m.cf.shellCursor {
			marker = " ▶ "
			label = accentBold(label)
		}
		b.WriteString(marker + label + "\n")
	}
}
