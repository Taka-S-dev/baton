package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	mdl "github.com/Taka-S-dev/baton/internal/model"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func completionFixture(t *testing.T) string {
	t.Helper()
	d := t.TempDir()
	for _, name := range []string{"test.go", "templates.txt", "integration_test.go", "target example.go", "日本語.go", ".hidden"} {
		if err := os.WriteFile(filepath.Join(d, name), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(d, "testdir"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(d, "testdir", "child.go"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	return d
}
func completionModel(t *testing.T, dir string) Model {
	t.Helper()
	cmd := mdl.Command{Name: "build", Cmd: "echo {@file}", Dir: dir}
	nm, _ := (Model{width: 80, height: 24}).startResolveFlow([]msItem{{cmd: &cmd}})
	m := nm.(Model)
	m.sp.completion = nil
	return m
}
func completionKey(m Model, key tea.KeyMsg) (Model, tea.Cmd) {
	nm, cmd := m.Update(key)
	return nm.(Model), cmd
}
func completionResult(m Model, cmd tea.Cmd) Model {
	if cmd == nil {
		return m
	}
	nm, _ := m.Update(cmd())
	return nm.(Model)
}

func TestPathCompletionRankingAndNestedPaths(t *testing.T) {
	d := completionFixture(t)
	got, err := completePaths(d, "te")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) < 3 {
		t.Fatalf("missing candidates: %v", got)
	}
	for i := 1; i < len(got); i++ {
		if got[i].score < got[i-1].score {
			t.Fatal("prefix matches must rank first")
		}
	}
	found := false
	for _, c := range got {
		if c.value == "integration_test.go" {
			found = true
		}
	}
	if !found {
		t.Fatal("substring match missing")
	}
	got, err = completePaths(d, "itg")
	if err != nil || len(got) == 0 || got[0].value != "integration_test.go" {
		t.Fatalf("fuzzy match: %v %v", got, err)
	}
	got, err = completePaths(d, "testdir/ch")
	if err != nil || len(got) != 1 || got[0].value != "testdir/child.go" {
		t.Fatalf("nested relative path: %v %v", got, err)
	}
	got, err = completePaths(d, "target ex")
	if err != nil || len(got) != 1 || got[0].value != "target example.go" {
		t.Fatalf("spaces changed: %v %v", got, err)
	}
	got, err = completePaths(d, "日語")
	if err != nil || len(got) != 1 || got[0].value != "日本語.go" {
		t.Fatalf("unicode fuzzy match: %v %v", got, err)
	}
	got, err = completePaths(d, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range got {
		if c.value == ".hidden" {
			t.Fatal("hidden file suggested without dot")
		}
	}
	got, err = completePaths(d, ".h")
	if err != nil || len(got) != 1 {
		t.Fatalf("explicit hidden match: %v %v", got, err)
	}
}

func TestPathCompletionTabInsertsBeforeConfirm(t *testing.T) {
	d := completionFixture(t)
	m := completionModel(t, d)
	m.sp.search = "te"
	m, cmd := completionKey(m, tea.KeyMsg{Type: tea.KeyTab})
	m = completionResult(m, cmd)
	if m.sp.completion == nil || len(m.sp.completion.entries) < 2 {
		t.Fatal("dropdown did not open")
	}
	m, cmd = completionKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("st.go")})
	m = completionResult(m, cmd)
	if m.sp.completion == nil || m.sp.search != "test.go" {
		t.Fatal("typing closed dropdown")
	}
	m, _ = completionKey(m, tea.KeyMsg{Type: tea.KeyTab})
	if m.screen != ScreenSlotPick || m.sp.completion != nil || m.sp.search != "test.go" || m.sp.cursor != len(m.sp.filtered) {
		t.Fatal("Tab should only insert")
	}
	m, _ = completionKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.screen != ScreenConfirmRun || m.confirmRunItems[0].Cmd.Cmd != "echo test.go" {
		t.Fatal("Enter should reach execution review")
	}
}

func TestPathCompletionUniqueFolderAndStaleReads(t *testing.T) {
	d := completionFixture(t)
	m := completionModel(t, d)
	m.sp.search = "testd"
	m, cmd := completionKey(m, tea.KeyMsg{Type: tea.KeyTab})
	m = completionResult(m, cmd)
	if m.sp.completion == nil {
		t.Fatal("reserved picker must stay open")
	}
	m = queryCompletion(m, "testdir/")
	m, _ = completionKey(m, tea.KeyMsg{Type: tea.KeyTab})
	if m.sp.search != "testdir/child.go" {
		t.Fatalf("child not completed: %q", m.sp.search)
	}
	m.sp.search = "te"
	m, old := completionKey(m, tea.KeyMsg{Type: tea.KeyTab})
	m, current := completionKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("zzz")})
	m = completionResult(m, current)
	active := m.sp.completion
	m = completionResult(m, old)
	if m.sp.completion != active || len(active.entries) != 0 {
		t.Fatal("stale directory read replaced current query")
	}
	m, clear := completionKey(m, tea.KeyMsg{Type: tea.KeyEsc})
	m = completionResult(m, clear)
	if m.sp.completion == nil || m.sp.search != "" {
		t.Fatal("Esc must clear filter first")
	}
	m, _ = completionKey(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.sp.completion != nil {
		t.Fatal("second Esc must close")
	}
	m = completionResult(m, old)
	if m.sp.completion != nil {
		t.Fatal("late read reopened popup")
	}
}

func TestPathCompletionBaseAndReadError(t *testing.T) {
	d := completionFixture(t)
	m := completionModel(t, "")
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if got, err := m.completionBase(); err != nil || got != cwd {
		t.Fatalf("empty workdir: %q %v", got, err)
	}
	m.sp.currentCmd.Dir = "{$root}/{sub}"
	m.vars = map[string]string{"root": d}
	m.sp.resolvedSoFar = map[string]string{"sub": "testdir"}
	if got, err := m.completionBase(); err != nil || got != filepath.Join(d, "testdir") {
		t.Fatalf("resolved workdir: %q %v", got, err)
	}
	m.sp.resolvedSoFar = nil
	if _, err := m.completionBase(); err == nil {
		t.Fatal("unresolved workdir silently fell back")
	}
	m.sp.slotName = "sub"
	if got, err := m.completionBase(); err != nil || got != cwd {
		t.Fatalf("workdir slot itself should use cwd: %q %v", got, err)
	}
	m = completionModel(t, filepath.Join(d, "missing"))
	m, cmd := completionKey(m, tea.KeyMsg{Type: tea.KeyTab})
	m = completionResult(m, cmd)
	if m.sp.completion == nil || m.sp.completion.err == nil || m.screen != ScreenSlotPick {
		t.Fatal("missing directory error not shown safely")
	}
}

func TestPathCompletionLeavesOrdinarySlotsUnchanged(t *testing.T) {
	cmd := mdl.Command{Name: "legacy", Cmd: "echo {targets...}"}
	nm, _ := (Model{}).startResolveFlow([]msItem{{cmd: &cmd}})
	m := nm.(Model)
	m.sp.search = "value"
	m, c := completionKey(m, tea.KeyMsg{Type: tea.KeyTab})
	if c != nil || m.sp.completion != nil || len(m.sp.picked) != 1 || m.sp.picked[0] != "value" {
		t.Fatal("ordinary Tab no longer toggles")
	}
	m, c = completionKey(m, tea.KeyMsg{Type: tea.KeyCtrlAt})
	if c != nil || m.sp.completion != nil {
		t.Fatal("ordinary slot opened path picker")
	}
}

func TestPathCompletionDropdownBoundsAndMouse(t *testing.T) {
	d := completionFixture(t)
	for _, size := range [][2]int{{40, 13}, {80, 24}, {120, 40}} {
		m := completionModel(t, d)
		m.width, m.height = size[0], size[1]
		m.sp.search = "te"
		m, cmd := completionKey(m, tea.KeyMsg{Type: tea.KeyTab})
		m = completionResult(m, cmd)
		view := m.View()
		if frameRows(view) > m.height {
			t.Fatalf("popup taller than %v", size)
		}
		for _, line := range strings.Split(view, "\n") {
			if ansi.StringWidth(line) > m.width {
				t.Fatalf("popup wider than %v: %q", size, ansi.Strip(line))
			}
		}
		p := m.pathCompletionLayout(m.width)
		value := m.sp.completion.entries[p.start].value
		nm, _ := m.Update(tea.MouseMsg{X: p.x + 2, Y: p.y + pathCandidateRowOffset, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
		m = nm.(Model)
		if m.sp.completion != nil || m.sp.search != value || m.screen != ScreenSlotPick {
			t.Fatal("mouse should insert without executing")
		}
	}
}
