package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestReservedPathSyntaxCandidates(t *testing.T) {
	m := Model{}
	c := m.pickCandidates("@")
	if len(c) != 5 {
		t.Fatalf("reserved candidates: %v", c)
	}
	for _, query := range []string{"@dir", "@file:", "@file...:"} {
		if len(m.pickCandidates(query)) != 1 {
			t.Fatalf("filter %q", query)
		}
	}
}

func TestReservedSyntaxInsertionInCommandAndWorkdir(t *testing.T) {
	for _, field := range []int{1, 2} {
		m := Model{cf: &commandFormState{fieldIdx: field}, nameInput: newMSTI("")}
		m.nameInput.Focus()
		m.nameInput.SetValue("日本語 ")
		m.nameInput.CursorEnd()
		for _, r := range "{@dir" {
			_, _ = m.updateCommandForm(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		}
		if !m.cf.pickOpen {
			t.Fatal("picker did not open without lists or globals")
		}
		_, _ = m.updateCommandForm(tea.KeyMsg{Type: tea.KeyEnter})
		if m.nameInput.Value() != "日本語 {@dir}" || m.cf.pickOpen || m.cf.fieldIdx != field {
			t.Fatalf("insertion: %q", m.nameInput.Value())
		}
	}
}
