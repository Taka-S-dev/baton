package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestPathCompletionOwnsFilterAndKeepsPreview(t *testing.T) {
	m := multiCompletion(t)
	m.width = 100
	m.height = 36
	m, read := completionKey(m, tea.KeyMsg{Type: tea.KeyTab})
	m = completionResult(m, read)
	m, read = completionKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("te")})
	m = completionResult(m, read)
	view := stripStyle(m.View())
	for _, want := range []string{"Search / te", "command preview", "workdir:", "0 selected", "Tab Check", "Enter Select"} {
		if !strings.Contains(view, want) {
			t.Fatalf("missing %q: %s", want, view)
		}
	}
	if strings.Count(view, "Search / te") != 1 || strings.Contains(view, "Search >") || strings.Contains(view, "Choose input") {
		t.Fatal("completion created a second input screen")
	}
	x, anchor := m.pathCompletionAnchor(m.viewWidth())
	p := m.pathCompletionLayout(m.viewWidth())
	if p.x != x || p.y != anchor || p.y+p.h > m.termHeight() {
		t.Fatal("picker must fit below the preview")
	}
	if strings.Contains(view, "Select value\n") || strings.Contains(view, "/ type to filter") {
		t.Fatal("background retains duplicate input")
	}
	nm, _ := m.Update(tea.MouseMsg{X: x + 2, Y: p.y + 1, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	m = nm.(Model)
	if m.sp.completion == nil || len(m.sp.completion.draft) != 0 {
		t.Fatal("input click changed selection or closed completion")
	}
	t.Log("\n" + view)
}
