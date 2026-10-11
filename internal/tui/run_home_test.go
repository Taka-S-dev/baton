package tui

import (
	"strings"
	"testing"

	"github.com/Taka-S-dev/baton/internal/store"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func homeTestModel(t *testing.T) Model {
	t.Helper()
	m := exploreProject()
	m.projectDir = t.TempDir()
	m.openRunHome()
	return m
}

func homeKey(m Model, key tea.KeyMsg) Model { next, _ := m.Update(key); return next.(Model) }
func homeTab(m Model, n rune) Model {
	key := tea.KeyF7
	if n == '2' {
		key = tea.KeyF8
	}
	return homeKey(m, tea.KeyMsg{Type: key})
}

func TestRunHomeTabKeepsSearchCursorAndOrder(t *testing.T) {
	m := homeTestModel(t)
	m.wfSearchTI.SetValue("smoke")
	m = homeTab(m, '2')
	// Select in a different order from the displayed list.
	m.msSelected = []int{2, 0}
	m.msSearchTI.SetValue("build")
	m.msCursor = 1
	m = homeTab(m, '1')
	if m.wfSearchTI.Value() != "smoke" {
		t.Fatal("workflow search lost")
	}
	m = homeTab(m, '2')
	if m.msSearchTI.Value() != "build" || m.msCursor != 1 {
		t.Fatal("command search/cursor lost")
	}
	if len(m.msSelected) != 2 || m.msItems[m.msSelected[0]].name() != "deploy" || m.msItems[m.msSelected[1]].name() != "build" {
		t.Fatal("execution order changed")
	}
	if got := store.LoadRunTab(m.projectDir); got != "commands" {
		t.Fatalf("saved tab %q", got)
	}
	// Ordinary letters/numbers remain search input, not navigation shortcuts.
	m.msSearchTI.SetValue("")
	m = homeKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e12pq")})
	if m.screen != ScreenRunCommands || m.msSearchTI.Value() != "e12pq" {
		t.Fatal("navigation stole search text")
	}
}

func TestRunHomeEditorRebuildsSelectionByName(t *testing.T) {
	m := homeTab(homeTestModel(t), '2')
	m.msSelected = []int{2, 0}
	m.msSearchTI.SetValue("build")
	m = homeKey(m, tea.KeyMsg{Type: tea.KeyF4})
	if m.screen != ScreenExplore || m.exploreTab != tabCommands {
		t.Fatal("editor tab")
	}
	// The editor can overwrite the shared picker and remove a command.
	m.config.Base = m.config.Base[:2]
	m.setupMultiSelect()
	m.gotoMainMenu()
	if m.screen != ScreenRunCommands || m.msSearchTI.Value() != "build" {
		t.Fatal("editor return lost search")
	}
	if len(m.msSelected) != 1 || m.msItems[m.msSelected[0]].name() != "build" {
		t.Fatal("deleted item restored or index reused")
	}
}

func TestRunHomeRememberedTabAndRecentRemainSeparate(t *testing.T) {
	m := homeTab(homeTestModel(t), '2')
	fresh := exploreProject()
	fresh.projectDir = m.projectDir
	fresh.openRunHome()
	if fresh.screen != ScreenRunCommands {
		t.Fatal("remembered tab ignored")
	}
	recent := Model{screen: ScreenRecent}
	if recent.runHomeVisible() {
		t.Fatal("recent must stay separate")
	}
}

func TestRunHomeHeaderMouseAndSize(t *testing.T) {
	for _, width := range []int{40, 65, 100, 140} {
		for _, height := range []int{6, 10, 24} {
			m := homeTab(homeTestModel(t), '2')
			m.width, m.height = width, height
			out := m.View()
			if lipgloss.Height(out) > height {
				t.Fatalf("%dx%d overflow height", width, height)
			}
			for _, line := range strings.Split(out, "\n") {
				if ansi.StringWidth(line) > width {
					t.Fatalf("%dx%d overflow width: %q", width, height, line)
				}
			}
			m = homeKey(m, tea.KeyMsg{Type: tea.KeyF1})
			if !m.helpOpen {
				t.Fatal("help inaccessible")
			}
			m = homeKey(m, tea.KeyMsg{Type: tea.KeyEsc})
			_, actions := m.runHomeLayout(width)
			next, _ := m.Update(tea.MouseMsg{X: actions[0].start, Y: 0, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
			if next.(Model).screen != ScreenRunWorkflow {
				t.Fatal("tab click")
			}
		}
	}
}

func TestRunHomeCandidateMouseOffset(t *testing.T) {
	m := homeTab(homeTestModel(t), '2')
	m.width, m.height = 80, 10
	y := -1
	m.viewMultiSelectCandidates(80, true, func(i, row int) {
		if i == 1 {
			y = row
		}
	})
	if y < 0 {
		t.Fatal("missing candidate")
	}
	next, _ := m.Update(tea.MouseMsg{X: 5, Y: y + 1, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	if next.(Model).msCursor != 1 {
		t.Fatal("header offset selected the wrong row")
	}
	if strings.Count(ansi.Strip(m.View()), "F7") != 1 {
		t.Fatal("duplicate tab header")
	}
}

func TestEmptyRunHomeCanAddCommand(t *testing.T) {
	m := exploreProject()
	m.projectDir = t.TempDir()
	m.config.Base, m.config.Commands, m.workflows = nil, nil, nil
	m.openRunHome()
	m = homeKey(m, tea.KeyMsg{Type: tea.KeyF4})
	if m.cf == nil || m.cf.mode != 0 {
		t.Fatal("empty state did not open command creation")
	}
}

func TestRunHomeProjectsNavigation(t *testing.T) {
	for _, tab := range []rune{'1', '2'} {
		m := homeTab(homeTestModel(t), tab)
		m.projectsDir = t.TempDir()
		next := homeKey(m, tea.KeyMsg{Type: tea.KeyF6})
		if next.screen != ScreenProjectSelect {
			t.Fatal("F6 event must open projects when delivered")
		}
		next = homeKey(m, tea.KeyMsg{Type: tea.KeyEsc})
		if next.screen != ScreenProjectSelect {
			t.Fatal("Esc must open projects with no pending input")
		}
		x := -1
		_, actions := m.runHomeLayout(m.viewWidth())
		for _, action := range actions {
			if action.id == 3 {
				x = action.start
			}
		}
		if x < 0 {
			t.Fatal("Projects action missing")
		}
		clicked, _ := m.Update(tea.MouseMsg{X: x, Y: 0, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
		if clicked.(Model).screen != ScreenProjectSelect {
			t.Fatal("Projects click must open picker")
		}
	}
}

func TestRunHomeHeaderGroupsAndVisibleActions(t *testing.T) {
	m := homeTab(homeTestModel(t), '2')
	m.projectDir = "日本語プロジェクト名が長い場合"
	for _, width := range []int{32, 40, 65, 100, 140} {
		m.width = width
		header, actions := m.runHomeLayout(width)
		plain := ansi.Strip(header)
		if strings.Contains(plain, "\n") || ansi.StringWidth(header) > width {
			t.Fatalf("invalid header: %q", plain)
		}
		if len(actions) < 2 {
			t.Fatalf("tabs missing at %d: %q", width, plain)
		}
		if !strings.Contains(plain, " › ") {
			t.Fatal("project context missing")
		}
		for _, a := range actions {
			if a.start < 0 || a.end > width || a.start >= a.end {
				t.Fatal("invalid click span")
			}
			if a.id == 4 {
				next, _ := m.Update(tea.MouseMsg{X: a.start, Y: 0, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
				if !next.(Model).helpOpen {
					t.Fatal("Help click failed")
				}
			}
		}
		if width >= 100 && (!strings.Contains(plain, "  │  ") || !strings.Contains(plain, "Edit · Projects… · F1 Help")) {
			t.Fatalf("actions not separated: %q", plain)
		}
	}
}
