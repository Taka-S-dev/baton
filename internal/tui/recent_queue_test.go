package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Taka-S-dev/baton/internal/store"
	tea "github.com/charmbracelet/bubbletea"
)

func TestRecentQueueOrderAndProjectIsolation(t *testing.T) {
	root := t.TempDir()
	refs := make([]store.RecentEntry, 2)
	for i, name := range []string{"alpha", "beta"} {
		dir := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Join(dir, "lists"), 0700); err != nil {
			t.Fatal(err)
		}
		for file, content := range map[string]string{
			"vars.tsv":      "command\tname\tvalue\n*\troot\t" + dir + "\n",
			"lists/env.tsv": name + "\n",
		} {
			if err := os.WriteFile(filepath.Join(dir, file), []byte(content), 0600); err != nil {
				t.Fatal(err)
			}
		}
		writeRecentCommand(t, dir, "echo {$root} {env}", "{$root}")
		refs[i] = store.RecentEntry{Project: dir, Command: "choose"}
	}
	m := Model{width: 100, height: 30, projectsDir: root}
	m.toggleRecent(recentRow{ref: refs[1]})
	m.toggleRecent(recentRow{ref: refs[0]})
	if m.recentMarker(refs[1]) != "[1] " || m.recentMarker(refs[0]) != "[2] " {
		t.Fatal("selection order lost")
	}
	nm, _ := m.resolveRecentQueue(refs[0])
	m = nm.(Model)
	if m.sp == nil || m.sp.entries[0].Value != "beta" {
		t.Fatal("wrong first project's list")
	}
	nm, _ = m.acceptSlotValue("beta")
	m = nm.(Model)
	if m.sp == nil || m.sp.entries[0].Value != "alpha" || m.sp.reuseFrom != "" {
		t.Fatal("answers or lists leaked across projects")
	}
	nm, _ = m.acceptSlotValue("alpha")
	m = nm.(Model)
	if m.screen != ScreenConfirmRun || len(m.confirmRunItems) != 2 {
		t.Fatal("missing batch review")
	}
	for i, name := range []string{"beta", "alpha"} {
		item := m.confirmRunItems[i]
		cmd := m.runItemCommand(item)
		if cmd.Dir != filepath.Join(root, name) || cmd.Cmd != "echo "+filepath.Join(root, name)+" "+name || m.runItemProject(item) != filepath.Join(root, name) {
			t.Fatalf("project context lost: %+v", cmd)
		}
		if !strings.Contains(item.Name, name) {
			t.Fatal("review lacks project")
		}
	}
}

func TestRecentQueueFilterToggleAndUnavailable(t *testing.T) {
	a := store.RecentEntry{Project: "a", Command: "build"}
	b := store.RecentEntry{Project: "b", Command: "test"}
	m := Model{recentRows: []recentRow{{ref: a}, {ref: b}}, recentSearch: "test"}
	nm, _ := m.updateRecent(tea.KeyMsg{Type: tea.KeyTab})
	m = nm.(Model)
	if len(m.recentQueue) != 1 || !sameRecent(m.recentQueue[0], b) {
		t.Fatal("filtered selection wrong")
	}
	m.recentSearch = ""
	m.toggleRecent(recentRow{ref: a})
	m.toggleRecent(recentRow{ref: b})
	if m.recentMarker(a) != "[1] " {
		t.Fatal("removal did not renumber")
	}
	m.toggleRecent(recentRow{ref: b, problem: "missing"})
	if len(m.recentQueue) != 1 || m.errMsg != "missing" {
		t.Fatal("unavailable command queued")
	}
}
