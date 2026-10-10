package tui

import (
	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"strings"
)

type contextPasteMsg struct {
	text string
	err  error
}

func readContextClipboard() tea.Msg { s, err := clipboard.ReadAll(); return contextPasteMsg{s, err} }
func (m Model) contextInputActive() bool {
	if m.exploreMenu || m.leaveAsk != leaveNone {
		return false
	}
	if m.cf != nil {
		return m.cf.fieldIdx != 4 && !m.cf.pickOpen
	}
	if m.screen == ScreenEditList && m.le != nil {
		return m.le.editing || m.le.adding
	}
	switch m.screen {
	case ScreenNameInput, ScreenVarForm, ScreenCreateCommandName, ScreenEditCommandName:
		return true
	}
	return false
}
func (m Model) pastePopup(w int) actionPopupLayout {
	p := actionPopupLayout{w: min(30, w), h: min(6, m.termHeight())}
	p.x = min(max(0, m.contextX), max(0, w-p.w))
	p.y = min(max(0, m.contextY), max(0, m.termHeight()-p.h))
	return p
}
func (m Model) insertClipboard(s string) (tea.Model, tea.Cmd) {
	if !m.contextInputActive() {
		return m, nil
	}
	// Single-line editors must never interpret pasted newlines as confirmation.
	s = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == '\t' {
			return ' '
		}
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, s)
	key := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s), Paste: true}
	if m.screen == ScreenEditList {
		if m.le.adding {
			if m.le.addFld == 0 {
				m.le.addVal += s
			} else {
				m.le.addLbl += s
			}
			return m, nil
		}
		if m.le.editFld == 0 {
			m.le.editValTI, _ = m.le.editValTI.Update(key)
		} else {
			m.le.editLblTI, _ = m.le.editLblTI.Update(key)
		}
		return m, nil
	}
	m.nameInput, _ = m.nameInput.Update(key)
	return m, nil
}
