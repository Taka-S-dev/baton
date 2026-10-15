package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	mdl "github.com/Taka-S-dev/baton/internal/model"
	"github.com/Taka-S-dev/baton/internal/projectfs"
)

func openedProject(t *testing.T) Model {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "api")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeCommands(t, dir, "build\t\t\tgo build ./...\t\t\n")
	m := Model{projectsDir: filepath.Dir(dir)}
	if err := m.loadProject(dir); err != nil {
		t.Fatal(err)
	}
	m.workflows = []mdl.Workflow{{Name: "ship", Commands: []string{"build"}}}
	return m
}

func writeCommands(t *testing.T, dir, rows string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "commands.tsv"), []byte("name\tgroup\tworkdir\tcmd\tshell\tslots\n"+rows), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSaveRefusesAfterAnOutsideEdit(t *testing.T) {
	m := openedProject(t)
	if err := m.saveWorkflows(); err != nil {
		t.Fatal(err)
	}
	writeCommands(t, m.projectDir, "test\t\t\tgo test ./...\t\t\n")
	err := m.saveWorkflows()
	if err == nil || !strings.Contains(err.Error(), "changed on disk") {
		t.Fatalf("save after an outside edit: %v", err)
	}
	if err := m.loadProject(m.projectDir); err != nil {
		t.Fatal(err)
	}
	if err := m.saveWorkflows(); err != nil {
		t.Fatalf("save after reopening: %v", err)
	}
	if _, err := os.Stat(filepath.Join(m.projectDir, ".baton-lock")); !os.IsNotExist(err) {
		t.Fatalf("lock left behind: %v", err)
	}
}

func TestSaveReportsAProjectHeldByAnotherProcess(t *testing.T) {
	m := openedProject(t)
	release, err := projectfs.Lock(m.projectDir)
	if err != nil {
		t.Fatal(err)
	}
	err = m.saveWorkflows()
	if err == nil || !strings.Contains(err.Error(), "busy") {
		t.Fatalf("save while held: %v", err)
	}
	release()
	if err := m.saveWorkflows(); err != nil {
		t.Fatalf("save after release: %v", err)
	}
}
