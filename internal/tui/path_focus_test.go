package tui

import (
	"os"
	"path/filepath"
	"testing"

	mdl "github.com/Taka-S-dev/baton/internal/model"
	tea "github.com/charmbracelet/bubbletea"
)

func TestPathFocusRestoredWithoutSelecting(t *testing.T) {
	root := t.TempDir()
	base := completionFixture(t)
	start := func(name, dir string) Model {
		cmd := mdl.Command{Name: name, Dir: dir, Cmd: "echo {@file:.go}"}
		nm, read := (Model{projectsDir: root, projectDir: filepath.Join(root, "project")}).startResolveFlow([]msItem{{cmd: &cmd}})
		return completionResult(nm.(Model), read)
	}
	m := start("build", base)
	m = queryCompletion(m, "日本語")
	m, _ = completionKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = completionKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.screen != ScreenConfirmRun {
		t.Fatal("selection did not reach review")
	}
	m = start("build", base)
	p := m.sp.completion
	if p.entries[p.cursor].value != "日本語.go" || m.sp.search != "" || m.sp.completedPath != "" || len(m.sp.picked) != 0 {
		t.Fatal("focus not restored independently of selection")
	}
	other := start("other", base)
	if other.sp.completion.entries[other.sp.completion.cursor].value == "日本語.go" {
		t.Fatal("focus leaked to another command")
	}
	if err := os.Remove(filepath.Join(base, "日本語.go")); err != nil {
		t.Fatal(err)
	}
	m = start("build", base)
	if m.sp.completion.err != nil || m.sp.completion.cursor != 0 {
		t.Fatal("removed file did not fall back")
	}
}
