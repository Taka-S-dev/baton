package tui

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Taka-S-dev/baton/internal/store"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func TestRecentRemoveUndoIsScopedAndPreservesSearch(t *testing.T) {
	root := t.TempDir()
	cwd := filepath.Join(root, "cwd")
	project := filepath.Join(root, "project")
	now := time.Now().UTC().Truncate(time.Second)
	for _, dir := range []string{cwd, root} {
		if err := store.RecordRecent(root, dir, project, "build", now); err != nil {
			t.Fatal(err)
		}
	}
	refs, _ := store.LoadRecent(root, cwd)
	m := Model{projectsDir: root, launchDir: cwd, recentSearch: "build", recentRows: []recentRow{{ref: refs[0]}}}
	next, _ := m.updateRecent(tea.KeyMsg{Type: tea.KeyDelete})
	m = next.(Model)
	if len(m.recentRows) != 0 || m.recentUndo == nil || m.recentSearch != "build" {
		t.Fatal("remove state")
	}
	refs, _ = store.LoadRecent(root, cwd)
	other, _ := store.LoadRecent(root, root)
	if len(refs) != 0 || len(other) != 1 {
		t.Fatal("directory isolation")
	}
	next, _ = m.updateRecent(tea.KeyMsg{Type: tea.KeyCtrlZ})
	m = next.(Model)
	refs, _ = store.LoadRecent(root, cwd)
	if len(refs) != 1 || !refs[0].UsedAt.Equal(now) || m.recentUndo != nil || m.recentSearch != "build" {
		t.Fatal("undo state")
	}
}
func TestRecentMatchMaskUnicodeAndRepeatedTerms(t *testing.T) {
	text := "日本Go go"
	mask := recentMatchMask(text, "日本 GO")
	for i, r := range []rune(text) {
		if mask[i] != (r != ' ') {
			t.Fatalf("wrong match at %d", i)
		}
	}
	if got := stripStyle(recentHighlight(text, "日本 GO", true)); got != text {
		t.Fatal(got)
	}
}
func TestRecentAge(t *testing.T) {
	now := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		d    time.Duration
		want string
	}{{0, "just now"}, {2 * time.Minute, "2m ago"}, {2 * time.Hour, "2h ago"}, {48 * time.Hour, "2d ago"}} {
		if got := recentAge(now.Add(-tc.d), now); got != tc.want {
			t.Fatal(got)
		}
	}
}
func TestRecentViewShowsMatchCountAndRemovalHint(t *testing.T) {
	m := Model{width: 100, height: 30, recentSearch: "build", recentRows: []recentRow{{ref: store.RecentEntry{Command: "build", Project: "Go", UsedAt: time.Now()}}, {ref: store.RecentEntry{Command: "test", Project: "Go"}}}}
	got := stripStyle(m.viewRecent(100))
	for _, want := range []string{"1 matches / 2 recent", "just now", "Delete: Remove from history"} {
		if !strings.Contains(got, want) {
			t.Fatal(got)
		}
	}
}

// TestRecentCommandLineColoursPlaceholders: the selected command shows
// its placeholders in the slot colour like every other command preview,
// and a search match inside one is drawn as a match instead.
func TestRecentCommandLineColoursPlaceholders(t *testing.T) {
	old := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(old) })
	m := Model{vars: map[string]string{"root": "C:/w"}}
	cmd := "go build -o {@file:.exe} {$root} {$missing}"
	idle := m.recentCommandLine(cmd, "")
	if !strings.Contains(idle, slotVar("{@file:.exe}")) || !strings.Contains(idle, sFixedVar.Render("{$root}")) {
		t.Errorf("placeholders are not coloured: %q", idle)
	}
	if stripStyle(idle) != cmd {
		t.Errorf("text changed: %q", stripStyle(idle))
	}
	found := m.recentCommandLine(cmd, "exe")
	if strings.Contains(found, slotVar("{@file:.exe}")) || !strings.Contains(found, sHighlight.Bold(true).Render("exe")) {
		t.Errorf("a match inside the placeholder is not shown as a match: %q", found)
	}
	if stripStyle(found) != cmd {
		t.Errorf("text changed while searching: %q", stripStyle(found))
	}
}
