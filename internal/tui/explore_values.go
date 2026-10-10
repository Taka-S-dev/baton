package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"strings"
)

// Keep existing form state and persistence; only its presentation changes.
func (m Model) exploreValueFlow() bool {
	if !m.exploreLaunched {
		return false
	}
	return (m.screen == ScreenEditList && m.le != nil) ||
		(m.screen == ScreenVarForm && m.ve != nil) ||
		(m.screen == ScreenNameInput && m.nameInputMode == nameInputNewList)
}

func (m Model) valuePaneKeys() string {
	if m.screen == ScreenEditList {
		if m.le.adding || m.le.editing {
			return "Enter Next / Save entry  Esc Cancel entry"
		}
		return "↑↓ Select entry  Enter Edit  a Add  Del Remove  Esc Back to list"
	}
	if m.screen == ScreenNameInput {
		return "Enter Create list  Esc Back to list"
	}
	return "Enter Next / Save  Esc Back"
}

func (m Model) exploreValueBox(l exploreLayout) string {
	var b strings.Builder
	title := "New list"
	width := max(1, l.rightW-6)
	switch m.screen {
	case ScreenNameInput:
		ti := m.nameInput
		ti.Width = width
		b.WriteString(" " + ti.View() + "\n")
		b.WriteString("\n " + dim("Example: env — choices for {env}") + "\n")
		if m.nameInputErr != "" {
			b.WriteString(" " + errorText(m.nameInputErr))
		}
	case ScreenVarForm:
		title = "New global variable"
		if m.ve.mode == 1 {
			title = "Edit global variable · " + m.ve.name
			if _, _, scoped := scopedKey(m.ve.name); scoped {
				title = "Edit fixed value · " + m.ve.name
			}
		}
		ti := m.nameInput
		ti.Width = width
		if m.ve.mode == 0 && m.ve.fieldIdx == 0 {
			b.WriteString(" " + ti.View() + "\n " + gray("value >") + "\n")
		} else {
			b.WriteString(" " + gray("name > ") + m.ve.name + "\n " + ti.View() + "\n")
		}
		b.WriteString("\n " + dim("Enter saves the value; Esc goes back.") + "\n")
	case ScreenEditList:
		title = "Edit list · " + m.le.name
		le := m.le
		switch {
		case le.editing:
			val, label := le.editValTI, le.editLblTI
			val.Width, label.Width = width, width
			b.WriteString(" " + val.View() + "\n")
			if le.editFld == 1 {
				b.WriteString(" " + label.View() + "\n")
			}
		case le.adding:
			b.WriteString(" " + gray("Value > ") + le.addVal)
			if le.addFld == 0 {
				b.WriteString(accent("_"))
			}
			b.WriteString("\n " + gray("Label > ") + le.addLbl)
			if le.addFld == 1 {
				b.WriteString(accent("_"))
			}
			b.WriteString("\n\n " + dim("Label is optional. Enter confirms each field."))
		default:
			writeWindowedList(&b, max(1, l.editH-4), len(le.entries)+1, le.cursor, "", func(i int) string {
				// The add row is an action, not an entry: it takes the
				// same accent and key hint as the Add row of the list pane.
				if i == len(le.entries) {
					marker := "   "
					if i == le.cursor {
						marker = accent(" ▶ ")
					}
					return marker + accent(truncate("+ Add value  a", width))
				}
				value := le.entries[i].Value
				if le.entries[i].Label != "" {
					value += "  · " + le.entries[i].Label
				}
				if i == le.cursor {
					return accent(" ▶ ") + truncate(value, width)
				}
				return "   " + truncate(value, width)
			})
		}
	}
	return box("← Back · "+title, b.String(), l.rightW, l.editH, true)
}

func (m Model) updateExploreValueMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	l := m.exploreLayout(m.viewWidth())
	if !l.single && msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress {
		onList := msg.X >= 0 && msg.X < l.leftW && msg.Y >= l.bodyY && msg.Y < l.bodyY+l.bodyH
		onBack := msg.Y == l.editY && msg.X >= l.rightX+3 && msg.X < l.rightX+9
		if onList || onBack {
			if m.askToLeave(leaveValue) {
				return m, nil
			}
			m.closeExploreValue()
			if onList && l.addY >= 0 && msg.Y == l.addY {
				m.exploreSearching = false
				return m.updateExplore(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
			}
			if onList {
				row := msg.Y - l.listY
				if row >= 0 && row < l.listH && l.start+row < len(l.items) && !l.items[l.start+row].draft {
					m.exploreCursor[m.exploreTab] = l.start + row
				}
			}
			return m, nil
		}
	}
	if l.single || msg.X < l.rightX || msg.X >= l.rightX+l.rightW || msg.Y < l.fieldY || msg.Y >= l.editY+l.editH-1 {
		return m, nil
	}
	if m.screen == ScreenEditList && !m.le.adding && !m.le.editing {
		rows := max(1, l.editH-4)
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			m.le.cursor = max(0, m.le.cursor-1)
		case tea.MouseButtonWheelDown:
			m.le.cursor = min(len(m.le.entries), m.le.cursor+1)
		case tea.MouseButtonLeft:
			if msg.Action == tea.MouseActionPress {
				i := windowStart(rows, len(m.le.entries)+1, m.le.cursor) + msg.Y - l.fieldY
				if i >= 0 && i <= len(m.le.entries) && msg.Y-l.fieldY < rows {
					if i == m.le.cursor {
						return m.updateEditList(tea.KeyMsg{Type: tea.KeyEnter})
					}
					m.le.cursor = i
				}
			}
		}
	}
	return m, nil
}
