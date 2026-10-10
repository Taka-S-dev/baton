package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"strings"
	"testing"
)

func TestWorkflowFixedAddStep(t *testing.T) {
	for _, count := range []int{0, 4, 60} {
		m := exploreProject()
		m.exploreTab = tabWorkflows
		m.wfe = &workflowEditState{idx: 0}
		for i := 0; i < count; i++ {
			m.wfe.steps = append(m.wfe.steps, "build")
		}
		if count > 0 {
			m.wfe.cursor = count - 1
		}
		l := m.exploreLayout(m.viewWidth())
		lines := strings.Split(stripStyle(m.exploreEditorBox(l)), "\n")
		if !strings.Contains(lines[l.editH-2], "+ Add step") {
			t.Fatal("missing fixed button")
		}
		at := count
		m = publicClick(m, l.rightX+3, l.editY+l.editH-2)
		if !m.wfe.adding {
			t.Fatal("click did not open picker")
		}
		m = createKey(m, tea.KeyMsg{Type: tea.KeyEnter})
		if len(m.wfe.steps) != count+1 || m.wfe.cursor != at || !m.wfe.dirty {
			t.Fatal("added step not selected")
		}
	}
}

func TestWorkflowScrolledStepClick(t *testing.T) {
	m := exploreProject()
	m.exploreTab = tabWorkflows
	m.wfe = &workflowEditState{idx: 0, cursor: 49}
	for i := 0; i < 50; i++ {
		m.wfe.steps = append(m.wfe.steps, "build")
	}
	l := m.exploreLayout(m.viewWidth())
	start := windowStart(max(1, l.editH-4), 50, 49)
	m = publicClick(m, l.rightX+3, l.fieldY)
	if m.wfe.cursor != start {
		t.Fatalf("clicked step %d, want %d", m.wfe.cursor, start)
	}
}
