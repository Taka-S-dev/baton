package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Taka-S-dev/baton/internal/config"
	tea "github.com/charmbracelet/bubbletea"
)

func TestProjectCreateAndRecoverableDelete(t *testing.T) {
	m := exploreProject()
	m.projectsDir = t.TempDir()
	m.screen = ScreenProjectSelect
	m.projectMode = "new"
	m.projectInput = "sample"
	m = exploreModel(exploreFirst(m.updateProjects(tea.KeyMsg{Type: tea.KeyEnter})))
	if m.errMsg != "" {
		t.Fatal(m.errMsg)
	}
	dir := filepath.Join(m.projectsDir, "sample")
	if _, err := config.LoadConfig(dir); err != nil {
		t.Fatal(err)
	}
	m = exploreModel(exploreFirst(m.updateProjects(tea.KeyMsg{Type: tea.KeyDelete})))
	if _, err := os.Stat(dir); err != nil {
		t.Fatal("deleted before confirmation")
	}
	m = exploreModel(exploreFirst(m.updateProjects(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})))
	if m.errMsg != "" {
		t.Fatal(m.errMsg)
	}
	if len(config.ListProjects(m.projectsDir)) != 0 {
		t.Fatal("trash exposed as project")
	}
	entries, err := os.ReadDir(filepath.Join(m.projectsDir, ".baton-trash"))
	if err != nil || len(entries) != 1 {
		t.Fatal("settings not recoverable")
	}
}

func TestProjectRejectsTraversalAndEmptySelection(t *testing.T) {
	for _, name := range []string{"", "..", "../outside", "a/b", "a\\b", ".baton-trash"} {
		if _, err := projectPath(t.TempDir(), name); err == nil {
			t.Fatalf("accepted %q", name)
		}
	}
	m := exploreProject()
	m.projects = nil
	m.screen = ScreenProjectSelect
	m = exploreModel(exploreFirst(m.updateProjects(tea.KeyMsg{Type: tea.KeyEnter})))
	if m.screen != ScreenProjectSelect {
		t.Fatal("empty selection opened")
	}
}
