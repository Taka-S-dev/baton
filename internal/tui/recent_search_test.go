package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/Taka-S-dev/baton/internal/store"
	tea "github.com/charmbracelet/bubbletea"
)

func TestRecentExplicitSearchAndEscape(t *testing.T) {
	root, project := recentFixture(t)
	cwd := t.TempDir()
	if err := store.RecordRecent(root, cwd, project, "choose", time.Now()); err != nil {
		t.Fatal(err)
	}
	m := Model{projectsDir: root, launchDir: cwd, recentActive: true, width: 80, height: 24}
	m.gotoRecent()
	if m.recentSearching || m.recentCursor != 0 {
		t.Fatal("startup mode/focus")
	}
	m = homeKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("choose")})
	if m.recentSearch != "" {
		t.Fatal("normal typing entered search")
	}
	m = homeKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	for _, query := range []string{"qp12", "choose"} {
		m = homeKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(query)})
		if m.recentSearch != query || m.screen != ScreenRecent {
			t.Fatal("search input navigated")
		}
		m = homeKey(m, tea.KeyMsg{Type: tea.KeyDelete})
		if len(m.recentRows) != 1 {
			t.Fatal("search deleted history")
		}
		m = homeKey(m, tea.KeyMsg{Type: tea.KeyEsc})
		if !m.recentSearching || m.recentSearch != "" {
			t.Fatal("Esc must clear and remain searching")
		}
	}
	m = homeKey(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.recentSearching || m.screen != ScreenRecent {
		t.Fatal("second Esc must only finish search")
	}
	m = homeKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m = homeKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("choose")})
	m = homeKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.screen != ScreenSlotPick || m.recentSearching {
		t.Fatal("search Enter must select the command")
	}
	m.gotoRecent()
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd == nil {
		t.Fatal("q must quit outside search")
	}
	m = homeKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	if m.screen != ScreenProjectSelect {
		t.Fatal("p must open projects")
	}
}

func TestRecentSearchPresentation(t *testing.T) {
	for _, height := range []int{8, 24} {
		m := Model{screen: ScreenRecent, width: 80, height: height}
		if !strings.Contains(m.View(), "/ Search…") {
			t.Fatal("normal search hint missing")
		}
		m = homeKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
		if !strings.Contains(m.View(), "Search / ") {
			t.Fatal("search state missing")
		}
		m = homeKey(m, tea.KeyMsg{Type: tea.KeyEnter})
		if !m.recentSearching {
			t.Fatal("Enter without results must keep searching")
		}
	}
}
