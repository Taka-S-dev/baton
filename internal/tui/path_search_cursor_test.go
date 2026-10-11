package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/cursor"
	tea "github.com/charmbracelet/bubbletea"
)

func TestPathSearchCursorAndEditing(t *testing.T) {
	m := multiCompletion(t)
	m, cmd := completionKey(m, tea.KeyMsg{Type: tea.KeyTab})
	m = completionResult(m, cmd)
	if !m.sp.completion.input.Focused() || m.sp.completion.input.Cursor.Mode() != cursor.CursorBlink {
		t.Fatal("file input must focus a blinking cursor")
	}
	for _, key := range []tea.KeyMsg{{Type: tea.KeyRunes, Runes: []rune("ac")}, {Type: tea.KeyLeft}, {Type: tea.KeyRunes, Runes: []rune("b")}} {
		m, cmd = completionKey(m, key)
		if cmd != nil {
			m = completionResult(m, cmd)
		}
	}
	if m.sp.search != "abc" {
		t.Fatalf("middle insertion: %q", m.sp.search)
	}
	m, cmd = completionKey(m, tea.KeyMsg{Type: tea.KeyDelete})
	m = completionResult(m, cmd)
	if m.sp.search != "ab" {
		t.Fatal("Delete must edit at cursor")
	}
	m, cmd = completionKey(m, tea.KeyMsg{Type: tea.KeyEsc})
	m = completionResult(m, cmd)
	if m.sp.search != "" || !m.sp.completion.input.Focused() {
		t.Fatal("clear must retain input focus")
	}
}
