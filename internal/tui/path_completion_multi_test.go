package tui

import (
	"reflect"
	"strings"
	"testing"

	mdl "github.com/Taka-S-dev/baton/internal/model"
	tea "github.com/charmbracelet/bubbletea"
)

func multiCompletion(t *testing.T) Model {
	cmd := mdl.Command{Name: "build", Cmd: "echo {@file...}", Dir: completionFixture(t)}
	nm, _ := (Model{width: 80, height: 24}).startResolveFlow([]msItem{{cmd: &cmd}})
	m := nm.(Model)
	m.sp.completion = nil
	return m
}

func queryCompletion(m Model, text string) Model {
	m, cmd := completionKey(m, tea.KeyMsg{Type: tea.KeyCtrlU})
	m = completionResult(m, cmd)
	m, cmd = completionKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(text)})
	return completionResult(m, cmd)
}

func TestDropdownMultiSelectionAcrossFilters(t *testing.T) {
	m := multiCompletion(t)
	m.sp.search = "test.go"
	m, cmd := completionKey(m, tea.KeyMsg{Type: tea.KeyTab})
	m = completionResult(m, cmd)
	if m.sp.completion == nil {
		t.Fatal("variadic completion must stay open")
	}
	m, _ = completionKey(m, tea.KeyMsg{Type: tea.KeyTab})
	if len(m.sp.picked) != 0 || !reflect.DeepEqual(m.sp.completion.draft, []string{"test.go"}) {
		t.Fatal("selection committed before Enter")
	}
	m = queryCompletion(m, "templates")
	if len(m.sp.completion.entries) != 1 || len(m.sp.completion.draft) != 1 {
		t.Fatal("filter lost selection")
	}
	m, _ = completionKey(m, tea.KeyMsg{Type: tea.KeyTab})
	if !strings.Contains(stripStyle(m.View()), "2 selected") {
		t.Fatal("selection count not visible")
	}
	m = queryCompletion(m, "no-such-file")
	m, _ = completionKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.screen != ScreenConfirmRun || m.confirmRunItems[0].Cmd.Cmd != "echo test.go templates.txt" {
		t.Fatal("selected values did not reach execution review")
	}
}

func TestDropdownMultiEscapeKeepsSelections(t *testing.T) {
	m := multiCompletion(t)
	m.sp.search = "test.go"
	m, cmd := completionKey(m, tea.KeyMsg{Type: tea.KeyTab})
	m = completionResult(m, cmd)
	m, _ = completionKey(m, tea.KeyMsg{Type: tea.KeyTab})
	if !strings.Contains(stripStyle(m.View()), "Esc Clear filter") {
		t.Fatal("missing clear hint")
	}
	m, cmd = completionKey(m, tea.KeyMsg{Type: tea.KeyEsc})
	m = completionResult(m, cmd)
	if m.sp.completion == nil || m.sp.search != "" || !reflect.DeepEqual(m.sp.completion.draft, []string{"test.go"}) {
		t.Fatal("clear lost selection or closed popup")
	}
	if !strings.Contains(stripStyle(m.View()), "Esc Close") {
		t.Fatal("missing close hint")
	}
	m, _ = completionKey(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.sp.completion != nil || !reflect.DeepEqual(m.sp.picked, []string{"test.go"}) || m.screen != ScreenSlotPick {
		t.Fatal("close lost selections or advanced")
	}
	m, cmd = completionKey(m, tea.KeyMsg{Type: tea.KeyTab})
	m = completionResult(m, cmd)
	m = queryCompletion(m, "test.go")
	if !m.sp.completion.selected("test.go") {
		t.Fatal("reopen lost selection")
	}
	m, _ = completionKey(m, tea.KeyMsg{Type: tea.KeyTab})
	m, _ = completionKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	if len(m.sp.picked) != 0 {
		t.Fatal("deselect-all reselected hovered candidate")
	}
	m, _ = completionKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.screen != ScreenSlotPick {
		t.Fatal("empty selection should not execute")
	}
}

func TestDropdownMultiFolderAndMouse(t *testing.T) {
	m := multiCompletion(t)
	m.sp.search = "testd"
	m, cmd := completionKey(m, tea.KeyMsg{Type: tea.KeyTab})
	m = completionResult(m, cmd)
	m = queryCompletion(m, "testdir/")
	if m.sp.completion == nil || len(m.sp.completion.entries) != 2 || !strings.Contains(m.sp.search, "testdir") {
		t.Fatal("folder navigation failed")
	}
	p := m.pathCompletionLayout(m.viewWidth())
	nm, _ := m.Update(tea.MouseMsg{X: p.x + 2, Y: p.y + pathCandidateRowOffset + 1, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	m = nm.(Model)
	if m.sp.completion == nil || len(m.sp.completion.draft) != 1 || len(m.sp.picked) != 0 {
		t.Fatal("mouse should toggle without applying")
	}
	nm, _ = m.Update(tea.MouseMsg{X: 0, Y: 0, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	m = nm.(Model)
	if m.sp.completion != nil || !reflect.DeepEqual(m.sp.picked, []string{"testdir/child.go"}) || m.sp.search != "" {
		t.Fatal("outside click should close and keep selection")
	}
}
