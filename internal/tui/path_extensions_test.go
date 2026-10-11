package tui

import (
	"strings"
	"testing"

	mdl "github.com/Taka-S-dev/baton/internal/model"
	tea "github.com/charmbracelet/bubbletea"
)

func TestReservedExtensionsAndFolderSelection(t *testing.T) {
	d := completionFixture(t)
	got, err := completePathsFor(d, "", "file", []string{".go"})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range got {
		if c.directory {
			continue
		}
		if !strings.HasSuffix(c.value, ".go") {
			t.Fatalf("extension leaked: %s", c.value)
		}
	}
	dirs, err := completePathsFor(d, "", "dir", nil)
	if err != nil || len(dirs) != 1 || !dirs[0].directory {
		t.Fatalf("directory picker includes files: %v %v", dirs, err)
	}
	cmd := mdl.Command{Name: "build", Cmd: "echo {@file...:.go}", Dir: d}
	nm, read := (Model{width: 80, height: 24}).startResolveFlow([]msItem{{cmd: &cmd}})
	m := completionResult(nm.(Model), read)
	if m.sp.completion == nil || len(m.sp.entries) != 0 {
		t.Fatal("reserved slot did not open dedicated choices")
	}
	m = queryCompletion(m, "testd")
	m, _ = completionKey(m, tea.KeyMsg{Type: tea.KeyTab})
	if len(m.sp.completion.draft) != 0 {
		t.Fatal("file picker selected folder")
	}
	if len(m.sp.completion.entries) != 1 || !m.sp.completion.entries[0].directory {
		t.Fatal("navigation folder missing")
	}
	m, read = completionKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	m = completionResult(m, read)
	if m.sp.completion == nil || len(m.sp.completion.entries) != 2 {
		t.Fatal("explicit directory prefix did not complete child file")
	}
	m, _ = completionKey(m, tea.KeyMsg{Type: tea.KeyTab})
	m, _ = completionKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.screen != ScreenConfirmRun || !strings.Contains(m.confirmRunItems[0].Cmd.Cmd, "child.go") {
		t.Fatal("reserved token did not resolve")
	}
}

func TestFileCompletionFilesBeforeNavigationFolders(t *testing.T) {
	entries, err := completePathsFor(completionFixture(t), "", "file", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatal("missing file candidates")
	}
	folderSeen := false
	for _, entry := range entries {
		if entry.directory {
			folderSeen = true
			continue
		}
		if folderSeen || strings.ContainsAny(entry.value, "/\\") {
			t.Fatalf("file must be local and precede folders: %v", entry)
		}
	}
}

func TestFolderNavigationKeepsCheckedFiles(t *testing.T) {
	m := multiCompletion(t)
	m.sp.search = "test.go"
	m, cmd := completionKey(m, tea.KeyMsg{Type: tea.KeyTab})
	m = completionResult(m, cmd)
	m, _ = completionKey(m, tea.KeyMsg{Type: tea.KeyTab})
	m = queryCompletion(m, "testd")
	m, cmd = completionKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	m = completionResult(m, cmd)
	if m.sp.completion == nil || !m.sp.completion.selected("test.go") || len(m.sp.completion.entries) != 2 || !strings.Contains(m.sp.completion.entries[1].value, "child.go") {
		t.Fatal("folder navigation lost checked files or applied prematurely")
	}
	m, _ = completionKey(m, tea.KeyMsg{Type: tea.KeyTab})
	m, _ = completionKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.screen != ScreenConfirmRun || !strings.Contains(m.confirmRunItems[0].Cmd.Cmd, "test.go") || !strings.Contains(m.confirmRunItems[0].Cmd.Cmd, "child.go") {
		t.Fatal("cross-directory selections lost")
	}
}
