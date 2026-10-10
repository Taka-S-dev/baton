package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"strings"
	"testing"
)

func TestWorkflowStepMouse(t *testing.T) {
	for _, width := range []int{40, 120} {
		m := wfModel()
		m.width = width
		m.height = 24
		m.gotoWorkflowStepPick(0)
		w := width
		if m.workflowDetailsInline() {
			w = m.viewWidth() / 2
		} else {
			w = m.viewWidth()
		}
		y := -1
		m.viewRunWorkflowStepsList(w, func(i, row int) {
			if i == 1 {
				y = row
			}
		})
		if y < 0 {
			t.Fatal("missing row")
		}
		click := func(x int) {
			next, _ := m.Update(tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
			m = next.(Model)
		}
		count := m.wfp.count()
		click(12)
		if m.wfp.cursor != 1 || m.wfp.count() != count {
			t.Fatal("row click must only inspect")
		}
		click(3)
		if m.wfp.count() != count+1 {
			t.Fatal("checkbox must toggle")
		}
		click(3)
		if m.wfp.count() != count {
			t.Fatal("checkbox must deselect")
		}
	}
}
func TestWorkflowStepCompactRows(t *testing.T) {
	m := wfModel()
	m.height = 24
	m.gotoWorkflowStepPick(0)
	for _, w := range []int{28, 40, 44, 60} {
		view := m.viewRunWorkflowStepsList(w)
		for _, line := range strings.Split(view, "\n") {
			if lipgloss.Width(line) > w {
				t.Fatalf("width %d overflow: %q", w, line)
			}
		}
	}
}
