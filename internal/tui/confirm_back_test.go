package tui

import (
	"path/filepath"
	"strings"
	"testing"

	mdl "github.com/Taka-S-dev/baton/internal/model"
	tea "github.com/charmbracelet/bubbletea"
)

func TestConfirmBackPreservesSelectedFile(t *testing.T) {
	m := completionModel(t, completionFixture(t))
	m.insertPathCandidate("test.go")
	m, _ = completionKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.screen != ScreenConfirmRun || m.confirmBack == nil {
		t.Fatal("missing return state")
	}
	m, _ = completionKey(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.screen != ScreenSlotPick || m.sp.completedPath != "test.go" || m.sp.search != "test.go" {
		t.Fatal("Back lost input")
	}
	m.insertPathCandidate("日本語.go")
	m, _ = completionKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.screen != ScreenConfirmRun || !strings.Contains(m.confirmRunItems[0].Cmd.Cmd, "日本語.go") {
		t.Fatal("edited input did not replace prior value")
	}
}

func TestConfirmButtonsStayAtSameRow(t *testing.T) {
	// A short launch directory: the review cuts a line to the width,
	// and a temporary directory is long enough to be cut on some
	// machines, which is not what this test is about.
	m := Model{width: 100, height: 36, launchDir: filepath.Join("srv", "api")}
	cmd := mdl.Command{Name: "test", Cmd: "echo test"}
	m.confirmRunItems = []mdl.RunItem{{Name: "test", Cmd: &cmd}}
	line := func(view string) int {
		for i, l := range strings.Split(stripStyle(view), "\n") {
			if strings.Contains(l, "Run") && strings.Contains(l, "Cancel") {
				return i
			}
		}
		return -1
	}
	first := line(m.viewConfirmRun(100))
	for i := 0; i < 20; i++ {
		m.confirmRunItems = append(m.confirmRunItems, mdl.RunItem{Name: "test", Cmd: &cmd})
	}
	if first < 0 || line(m.viewConfirmRun(100)) != first {
		t.Fatal("button position changes with command count")
	}
	if !strings.Contains(stripStyle(m.viewConfirmRun(100)), m.launchDir) {
		t.Fatal("actual workdir missing")
	}
}
