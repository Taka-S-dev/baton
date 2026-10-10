package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestRightClickSelectsActionTarget(t *testing.T) {
	m := exploreProject()
	l := m.exploreLayout(m.viewWidth())
	m = exploreModel(exploreFirst(m.Update(tea.MouseMsg{X: 3, Y: l.listY + 1, Button: tea.MouseButtonRight, Action: tea.MouseActionPress})))
	if !m.exploreMenu || m.exploreCursor[tabCommands] != l.start+1 {
		t.Fatal("wrong action target")
	}
	p := m.actionPopup(m.viewWidth(), 4)
	if p.x < 0 || p.x+p.w > m.viewWidth() {
		t.Fatal("popup overflow")
	}
	m = createKey(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.exploreMenu {
		t.Fatal("Esc did not close")
	}
}

func TestPasteMenuAndLiteralClipboard(t *testing.T) {
	m := exploreProject()
	m = createKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	l := m.exploreLayout(m.viewWidth())
	m = exploreModel(exploreFirst(m.Update(tea.MouseMsg{X: l.rightX + 5, Y: l.fieldY, Button: tea.MouseButtonRight, Action: tea.MouseActionPress})))
	if !m.pasteMenu || !strings.Contains(stripStyle(m.renderScreen(m.viewWidth())), "Paste") {
		t.Fatal("missing paste menu")
	}
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(Model)
	if cmd == nil || m.pasteMenu {
		t.Fatal("paste action failed")
	}
	m = exploreModel(exploreFirst(m.Update(contextPasteMsg{text: "echo hello\nctrl+s"})))
	if m.cf == nil || m.nameInput.Value() != "echo hello ctrl+s" {
		t.Fatal("paste executed action or lost text")
	}
}

func TestPasteListValueAndMenuCancel(t *testing.T) {
	m := exploreProject()
	m.exploreTab = tabLists
	m = createKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	m = createKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	m = exploreModel(exploreFirst(m.Update(contextPasteMsg{text: "multiple characters"})))
	if m.le.addVal != "multiple characters" || !m.le.adding {
		t.Fatal("list paste failed")
	}
	m.pasteMenu = true
	m = createKey(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.pasteMenu || m.le.addVal != "multiple characters" {
		t.Fatal("cancel changed input")
	}
}
