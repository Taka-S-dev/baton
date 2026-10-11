package tui

import (
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestParentNavigationPreservesSelectionAndFocus(t *testing.T) {
	for _, key := range []tea.KeyType{tea.KeyLeft, tea.KeyEnter} {
		m := multiCompletion(t)
		m.sp.search = "testdir/"
		m, cmd := completionKey(m, tea.KeyMsg{Type: tea.KeyTab})
		m = completionResult(m, cmd)
		if !m.sp.completion.entries[0].parent {
			t.Fatal("parent row missing")
		}
		m, _ = completionKey(m, tea.KeyMsg{Type: tea.KeyTab})
		if len(m.sp.completion.draft) != 1 {
			t.Fatal("child not checked")
		}
		m.sp.completion.cursor = 0
		m, _ = completionKey(m, tea.KeyMsg{Type: tea.KeyTab})
		if len(m.sp.completion.draft) != 1 {
			t.Fatal("parent row selectable")
		}
		m, cmd = completionKey(m, tea.KeyMsg{Type: key})
		m = completionResult(m, cmd)
		if m.sp.search != "" || len(m.sp.completion.draft) != 1 || filepath.Clean(m.sp.completion.entries[m.sp.completion.cursor].value) != "testdir" {
			t.Fatal("parent navigation lost selection or focus")
		}
	}
}
func TestCompletionParentPaths(t *testing.T) {
	base := t.TempDir()
	for _, tc := range []struct{ query, want string }{{"testdir/ch", ""}, {"src/nested/", "src/"}, {"../", "../../"}, {"../../", "../../../"}} {
		got, _, ok := completionParent(base, tc.query)
		if !ok || got != tc.want {
			t.Fatalf("%q: %q %v", tc.query, got, ok)
		}
	}
	root := filepath.VolumeName(base) + string(filepath.Separator)
	if _, _, ok := completionParent(base, root); ok {
		t.Fatal("filesystem root has parent")
	}
}
