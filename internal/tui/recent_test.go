package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Taka-S-dev/baton/internal/config"
	mdl "github.com/Taka-S-dev/baton/internal/model"
	"github.com/Taka-S-dev/baton/internal/store"
	tea "github.com/charmbracelet/bubbletea"
)

func recentFixture(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	project := filepath.Join(root, "demo")
	if err := os.Mkdir(project, 0700); err != nil {
		t.Fatal(err)
	}
	writeRecentCommand(t, project, "echo {@file:.go}", "")
	return root, project
}

func writeRecentCommand(t *testing.T, project, cmd, dir string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(project, "commands.tsv"), []byte("name\tgroup\tworkdir\tcmd\tshell\tslots\nchoose\t\t"+dir+"\t"+cmd+"\t\t\n"), 0600); err != nil {
		t.Fatal(err)
	}
	p, err := config.LoadProject(project)
	if err != nil || len(p.Warnings) != 0 {
		t.Fatalf("invalid fixture: %v %v", err, p.Warnings)
	}
}

func TestRecentStartupLatestDefinitionAndFreshInputs(t *testing.T) {
	root, project := recentFixture(t)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := store.RecordRecent(root, cwd, project, "choose", time.Now()); err != nil {
		t.Fatal(err)
	}
	t.Setenv("BATON_PROJECTS_DIR", root)
	m, err := NewRecent(true)
	if err != nil {
		t.Fatal(err)
	}
	if !m.exitAfterRun || m.screen != ScreenRecent || len(m.recentRows) != 1 {
		t.Fatal("did not enter recent directly")
	}
	if projects := config.ListProjects(root); len(projects) != 1 || projects[0] != "demo" {
		t.Fatalf("history appeared as project: %v", projects)
	}
	writeRecentCommand(t, project, "echo updated {@file:.go}", project)
	nm, read := m.updateRecent(tea.KeyMsg{Type: tea.KeyEnter})
	m = completionResult(nm.(Model), read)
	if m.screen != ScreenSlotPick || m.sp.search != "" || len(m.sp.picked) != 0 || !strings.Contains(m.sp.currentCmd.Cmd, "updated") {
		t.Fatal("did not load latest definition with fresh inputs")
	}
	if base, err := m.completionBase(); err != nil || base != project {
		t.Fatalf("explicit workdir lost: %s %v", base, err)
	}
	m.sp.completion = nil
	nm, _ = m.goBackInResolve()
	m = nm.(Model)
	if m.screen != ScreenRecent {
		t.Fatal("cancel did not return to recent")
	}
	writeRecentCommand(t, project, "echo changed", "")
	nm, _ = m.updateRecent(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(Model)
	if m.screen != ScreenConfirmRun || m.confirmRunItems[0].Cmd.Cmd != "echo changed" {
		t.Fatal("plain command must still require confirmation")
	}
	nm, _ = m.updateConfirmRun(tea.KeyMsg{Type: tea.KeyEsc})
	if nm.(Model).screen != ScreenRecent {
		t.Fatal("confirm cancel did not return")
	}
}

func TestRecentMissingCommandAndEmptyBrowse(t *testing.T) {
	root, project := recentFixture(t)
	cwd, _ := os.Getwd()
	if err := store.RecordRecent(root, cwd, project, "deleted", time.Now()); err != nil {
		t.Fatal(err)
	}
	m := Model{projectsDir: root, launchDir: cwd, recentActive: true, width: 80, height: 24}
	m.gotoRecent()
	if m.recentRows[0].problem == "" {
		t.Fatal("missing command not marked")
	}
	if m.recentCursor != 1 {
		t.Fatal("all unavailable should focus Browse projects")
	}
	m.recentCursor = 0 // A user can still inspect an unavailable reference.
	nm, cmd := m.updateRecent(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(Model)
	if cmd != nil || m.screen != ScreenRecent || m.errMsg == "" {
		t.Fatal("missing command could run")
	}
	m.launchDir = filepath.Join(root, "empty")
	m.gotoRecent()
	if !strings.Contains(m.View(), "No commands run here") {
		t.Fatal("empty hint missing")
	}
	nm, _ = m.updateRecent(tea.KeyMsg{Type: tea.KeyEnter})
	if nm.(Model).screen != ScreenProjectSelect {
		t.Fatal("empty browse failed")
	}
}

func TestRecentRecordingOnlyForExecutedSteps(t *testing.T) {
	root, project := recentFixture(t)
	cwd := t.TempDir()
	c := mdl.Command{Name: "choose", Cmd: "echo secret-runtime-input", Dir: project}
	m := Model{projectsDir: root, projectDir: project, launchDir: cwd, dryRun: true, running: &runningState{items: []mdl.RunItem{{Name: c.Name, Cmd: &c}}}}
	_ = m.runNext()
	entries, err := store.LoadRecent(root, cwd)
	if err != nil || len(entries) != 0 {
		t.Fatal("dry-run recorded history")
	}
	m.dryRun = false
	_ = m.runNext() // only constructs runner command; never executes it
	entries, err = store.LoadRecent(root, cwd)
	if err != nil || len(entries) != 1 || entries[0].Command != "choose" {
		t.Fatal("run not recorded")
	}
	other, err := store.LoadRecent(root, project)
	if err != nil || len(other) != 0 {
		t.Fatal("history keyed by workdir instead of launch directory")
	}
	filepath.WalkDir(filepath.Join(root, ".baton-recent"), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			data, e := os.ReadFile(path)
			if e != nil {
				t.Error(e)
			}
			if strings.Contains(string(data), "secret-runtime-input") {
				t.Error("resolved input persisted")
			}
		}
		return nil
	})
}

func TestRecentLayoutAndFilter(t *testing.T) {
	root, project := recentFixture(t)
	cwd, _ := os.Getwd()
	if err := store.RecordRecent(root, cwd, project, "choose", time.Now()); err != nil {
		t.Fatal(err)
	}
	m := Model{projectsDir: root, launchDir: cwd, recentActive: true}
	m.gotoRecent()
	for _, size := range [][2]int{{40, 13}, {80, 24}, {120, 40}} {
		m.width, m.height = size[0], size[1]
		if frameRows(m.View()) > m.height {
			t.Fatal("recent view exceeds viewport")
		}
	}
	m.recentSearching = true
	nm, _ := m.updateRecent(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("no match")})
	m = nm.(Model)
	if len(m.recentFiltered()) != 0 || !strings.Contains(m.View(), "Browse projects") {
		t.Fatal("filter hid browse fallback")
	}
	nm, _ = m.updateRecent(tea.KeyMsg{Type: tea.KeyEsc})
	m = nm.(Model)
	if m.recentSearch != "" || len(m.recentFiltered()) != 1 {
		t.Fatal("Esc did not clear filter")
	}
}

func TestRecentFocusSkipsUnavailableAndRefreshes(t *testing.T) {
	root, project := recentFixture(t)
	cwd := t.TempDir()
	now := time.Now()
	for i, name := range []string{"choose", "removed"} {
		if err := store.RecordRecent(root, cwd, project, name, now.Add(time.Duration(i)*time.Second)); err != nil {
			t.Fatal(err)
		}
	}
	m := Model{projectsDir: root, launchDir: cwd, recentActive: true}
	m.gotoRecent()
	if m.recentCursor != 1 || m.recentRows[m.recentCursor].ref.Command != "choose" {
		t.Fatal("did not focus newest available command")
	}
	m.recentSearching = true
	nm, _ := m.updateRecent(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("removed")})
	m = nm.(Model)
	if m.recentCursor != 1 {
		t.Fatal("unavailable filter should focus Browse")
	}
	nm, _ = m.updateRecent(tea.KeyMsg{Type: tea.KeyEsc})
	m = nm.(Model)
	if m.recentCursor != 1 {
		t.Fatal("clearing filter lost preferred focus")
	}
	if err := store.RecordRecent(root, cwd, project, "choose", now.Add(3*time.Second)); err != nil {
		t.Fatal(err)
	}
	nm, _ = m.updateRecent(tea.KeyMsg{Type: tea.KeyEsc})
	m = nm.(Model)
	nm, _ = m.updateRecent(tea.KeyMsg{Type: tea.KeyCtrlR})
	m = nm.(Model)
	if m.recentCursor != 0 || m.recentRows[0].ref.Command != "choose" {
		t.Fatal("refresh did not focus last execution")
	}
}

func TestRecentCompactLayoutAndMouse(t *testing.T) {
	m := Model{screen: ScreenRecent, width: 120, height: 40, launchDir: "C:/work/baton", recentRows: []recentRow{{ref: store.RecentEntry{Command: "go-build", Project: "Go"}, command: mdl.Command{Cmd: "go build ."}}}}
	p := m.recentGeometry(m.width)
	if p.width != 96 {
		t.Fatal("launcher width not capped")
	}
	view := stripStyle(m.View())
	if strings.Index(view, "$ go build .") > strings.Index(view, "Browse projects") {
		t.Fatal("preview must precede browse action")
	}
	lines := strings.Split(view, "\n")
	if !strings.Contains(lines[p.browseY], "Browse projects") || !strings.Contains(lines[p.browseY-1], "─") {
		t.Fatal("browse not separated")
	}
	if !strings.Contains(view, "/ Search…") || !strings.Contains(view, "Selected command") || !strings.Contains(view, "Enter Review") {
		t.Fatal("missing visual hierarchy")
	}
	nm, cmd := m.updateRecentMouse(tea.MouseMsg{X: p.x + 1, Y: p.browseY - 1, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	if cmd != nil || nm.(Model).screen != ScreenRecent {
		t.Fatal("separator triggered selection")
	}
	nm, _ = m.updateRecentMouse(tea.MouseMsg{X: p.x + 1, Y: p.browseY, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	if nm.(Model).screen != ScreenProjectSelect {
		t.Fatal("browse mouse target incorrect")
	}
	t.Log("\n" + view)
}

func TestRecentRunSelectionSurvivesRestart(t *testing.T) {
	root, project := recentFixture(t)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "commands.tsv"), []byte("name\tgroup\tworkdir\tcmd\tshell\tslots\na\t\t\techo a\t\t\nb\t\t\techo b\t\t\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for i, name := range []string{"b", "a"} {
		if err := store.RecordRecent(root, cwd, project, name, time.Now().Add(time.Duration(i-10)*time.Second)); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("BATON_PROJECTS_DIR", root)
	m, err := NewRecent(false)
	if err != nil {
		t.Fatal(err)
	}
	if m.recentRows[m.recentCursor].ref.Command != "a" {
		t.Fatal("fixture order")
	}
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = nm.(Model)
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(Model)
	if m.screen != ScreenConfirmRun {
		t.Fatal("selection skipped review")
	}
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(Model)
	// Deliver the execution-start event; do not execute the resulting shell command.
	nm, _ = m.Update(runReadyMsg{})
	m = nm.(Model)
	restarted, err := NewRecent(false)
	if err != nil {
		t.Fatal(err)
	}
	if restarted.recentRows[restarted.recentCursor].ref.Command != "b" {
		t.Fatal("last started command not focused after restart")
	}
}
