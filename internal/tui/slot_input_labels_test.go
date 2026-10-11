package tui

import (
	"strings"
	"testing"

	mdl "github.com/Taka-S-dev/baton/internal/model"
	tea "github.com/charmbracelet/bubbletea"
)

func TestCompletedPathLabelsAndInputEdits(t *testing.T) {
	m := completionModel(t, completionFixture(t))
	m.insertPathCandidate("test.go")
	view := stripStyle(m.viewSlotPick(100))
	if !strings.Contains(view, "✓ Inputs complete") || !strings.Contains(view, "✓ File selected: test.go") {
		t.Fatalf("missing file labels: %s", view)
	}
	if strings.Contains(view, "no match") || strings.Contains(view, "custom") {
		t.Fatal("file completion was presented as a list mismatch")
	}
	m, _ = completionKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	if m.sp.completedPath != "" || m.sp.inputStatus() != "" || m.sp.typedValueLabel() != "Use value: test.gox" {
		t.Fatal("editing retained stale completion status")
	}
	m.insertPathCandidate("testdir/")
	if m.sp.inputStatus() != "Folder path completed" || m.sp.typedValueLabel() != "Use folder: testdir/" {
		t.Fatal("folder labels missing")
	}
	m, _ = completionKey(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.sp.completedPath != "" || m.sp.search != "" {
		t.Fatal("clear retained completion status")
	}
}

func TestSlotInputListAndFreeTextStatus(t *testing.T) {
	s := slotPickState{search: "te"}
	if s.inputStatus() != "" || !strings.Contains(s.inputHint(), "type a value") {
		t.Fatal("free input claims a list mismatch")
	}
	s.entries = []mdl.ListEntry{{Value: "prod"}}
	s.applyFilter()
	if s.inputStatus() != "No list matches" {
		t.Fatal("list scope not explicit")
	}
	s.search = "pr"
	s.applyFilter()
	if s.inputStatus() != "1 list matches" {
		t.Fatal("list count missing")
	}
	s.variadic = true
	if strings.Contains(s.inputHint(), "Tab") {
		t.Fatal("variadic completion shortcut missing")
	}
}
