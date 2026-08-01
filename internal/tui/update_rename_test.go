package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Taka-S-dev/baton/internal/config"
	mdl "github.com/Taka-S-dev/baton/internal/model"
)

// renamedProject builds a project where commands.tsv was renamed by hand
// after the last run: the snapshot still records "apistart", a workflow
// still references it, and the row now reads "api-start".
func renamedProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"commands.tsv":   "name\tgroup\tworkdir\tcmd\tshell\tslots\napi-start\t\t\tgo run .\t\t\n",
		"workflows.json": `[{"name": "wf", "commands": ["apistart", "keep"]}]`,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	snap := map[string]string{
		"apistart": config.Fingerprint(mdl.Command{Cmd: "go run ."}),
		"keep":     "ffff",
	}
	if err := config.SaveSnapshot(dir, snap); err != nil {
		t.Fatal(err)
	}
	return dir
}

// TestRenameRepair_ApplyUpdatesReferences checks the whole hand-edit
// flow: loading detects the rename, the repair screen opens before the
// main menu, and confirming rewrites workflows.json and refreshes the
// snapshot under the new name.
func TestRenameRepair_ApplyUpdatesReferences(t *testing.T) {
	dir := renamedProject(t)
	m := &Model{}
	if err := m.loadProject(dir); err != nil {
		t.Fatal(err)
	}
	if len(m.renames) != 1 || m.renames[0].Old != "apistart" || m.renames[0].New != "api-start" {
		t.Fatalf("renames = %+v, want apistart → api-start", m.renames)
	}
	if m.renames[0].WfSteps != 1 {
		t.Fatalf("rename = %+v, want one workflow step counted", m.renames[0])
	}

	m.gotoPostLoad()
	if m.screen != ScreenRenameRepair {
		t.Fatalf("detected renames must open the repair screen, screen=%v", m.screen)
	}

	m.updateRenameRepair(tea.KeyMsg{Type: tea.KeyEnter}) // Yes is preselected
	if m.screen != ScreenMainMenu {
		t.Fatalf("confirming must land on the main menu, screen=%v", m.screen)
	}
	if steps := m.workflows[0].Commands; steps[0] != "api-start" || steps[1] != "keep" {
		t.Fatalf("workflow steps = %v, want only the renamed step updated", steps)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "workflows.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "api-start") || strings.Contains(string(raw), `"apistart"`) {
		t.Fatalf("workflows.json = %q, want the step renamed on disk", raw)
	}
	for _, w := range m.loadWarnings {
		if strings.Contains(w, "apistart") {
			t.Fatalf("warnings must clear after the repair, got %q", m.loadWarnings)
		}
	}
	snap := config.LoadSnapshot(dir)
	if _, ok := snap["api-start"]; !ok {
		t.Fatalf("snapshot must record the new name, got %v", snap)
	}
}

// TestRenameRepair_DeclineKeepsFilesQuietly checks Esc leaves every file
// untouched, drops the stale snapshot entry so the offer is not repeated
// on the next start, and keeps the dangling-step warning visible.
func TestRenameRepair_DeclineKeepsFilesQuietly(t *testing.T) {
	dir := renamedProject(t)
	m := &Model{}
	if err := m.loadProject(dir); err != nil {
		t.Fatal(err)
	}
	m.gotoPostLoad()
	m.updateRenameRepair(tea.KeyMsg{Type: tea.KeyEscape})
	if m.screen != ScreenMainMenu {
		t.Fatalf("declining must land on the main menu, screen=%v", m.screen)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "workflows.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"apistart"`) {
		t.Fatalf("workflows.json = %q, declining must not rewrite it", raw)
	}
	if !strings.Contains(strings.Join(m.loadWarnings, "; "), "apistart") {
		t.Fatalf("warnings = %q, the dangling step must stay visible", m.loadWarnings)
	}

	m2 := &Model{}
	if err := m2.loadProject(dir); err != nil {
		t.Fatal(err)
	}
	if len(m2.renames) != 0 {
		t.Fatalf("a declined rename must not be offered again, got %+v", m2.renames)
	}
}

// TestGotoMainMenu_WritesSnapshot checks a plain menu visit records the
// current commands, so detection works from the very next start.
func TestGotoMainMenu_WritesSnapshot(t *testing.T) {
	dir := t.TempDir()
	tsv := "name\tgroup\tworkdir\tcmd\tshell\tslots\nbuild\t\t\tmake\t\t\n"
	if err := os.WriteFile(filepath.Join(dir, "commands.tsv"), []byte(tsv), 0o644); err != nil {
		t.Fatal(err)
	}
	m := &Model{}
	if err := m.loadProject(dir); err != nil {
		t.Fatal(err)
	}
	m.gotoPostLoad()
	snap := config.LoadSnapshot(dir)
	if snap["build"] != config.Fingerprint(mdl.Command{Cmd: "make"}) {
		t.Fatalf("snapshot = %v, want the loaded command recorded", snap)
	}
}
