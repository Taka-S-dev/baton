package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestListDetailFollowsSelectionAndDraft(t *testing.T) {
	m := exploreProject()
	m.height = 50
	m.exploreTab = tabLists
	m = createKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	preview := func() string { return stripStyle(m.explorePreviewBox(m.exploreLayout(m.viewWidth()))) }
	if !strings.Contains(preview(), "Value > dev") || !strings.Contains(preview(), "Commands with this fixed value") {
		t.Fatal("missing value or list usage")
	}
	m = createKey(m, tea.KeyMsg{Type: tea.KeyDown})
	if !strings.Contains(preview(), "Value > prod") || !strings.Contains(preview(), "Label > production") {
		t.Fatal("selection not followed")
	}
	m = createKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	m.le.editValTI.SetValue("{$root}/api")
	m.le.editLblTI.SetValue("Draft label")
	text := preview()
	if !strings.Contains(text, "Resolved >") || !strings.Contains(text, "Draft label") || !strings.Contains(text, "Phase1") {
		t.Fatal("draft not resolved")
	}
	if m.lists["env"][1].Value != "prod" {
		t.Fatal("preview saved draft")
	}
}

func TestListDetailAddDoesNotShowPreviousValue(t *testing.T) {
	m := exploreProject()
	m.exploreTab = tabLists
	m = createKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	m.le.cursor = len(m.le.entries)
	text := stripStyle(m.explorePreviewBox(m.exploreLayout(m.viewWidth())))
	if !strings.Contains(text, "Enter or a") || strings.Contains(text, "Value > dev") {
		t.Fatal("add shows stale value")
	}
	m.le.startAdding()
	m.le.addVal = "new"
	m.le.addLbl = "new label"
	text = stripStyle(m.explorePreviewBox(m.exploreLayout(m.viewWidth())))
	if !strings.Contains(text, "Value > new") || !strings.Contains(text, "new label") {
		t.Fatal("new draft not shown")
	}
}
