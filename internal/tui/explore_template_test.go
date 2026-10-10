package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestOverviewTemplateCreateAndEdit(t *testing.T) {
	m := exploreProject()
	m.projectDir = t.TempDir()
	m = createKey(m, tea.KeyMsg{Type: tea.KeySpace})
	if !strings.Contains(stripStyle(m.View()), "f  From template") {
		t.Fatal("template action missing")
	}
	m = createKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
	if m.sp == nil || !m.exploreTemplateFlow() {
		t.Fatal("did not open slot picker")
	}
	if !strings.Contains(stripStyle(m.View()), "[3 Lists]") {
		t.Fatal("workspace disappeared")
	}
	m = createKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.screen != ScreenCreateCommandName {
		t.Fatal("did not reach name input")
	}
	m.nameInput.SetValue("deploy-dev")
	m = createKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.screen != ScreenExplore || m.sce != nil {
		t.Fatalf("did not return after save: %s", m.errMsg)
	}
	cmd, ok := m.config.FindCommand("deploy-dev")
	if !ok || cmd.Template != "deploy" || cmd.Values["env"] != "dev" {
		t.Fatalf("wrong derived command: %+v", cmd)
	}
	it, ok := m.exploreCurrent()
	if !ok || it.name != "deploy-dev" {
		t.Fatal("saved command not selected")
	}
	m = createKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	if !m.exploreTemplateFlow() || !strings.Contains(stripStyle(m.View()), "Change values") {
		t.Fatal("derived editor is not in workspace")
	}
	m = createKey(m, tea.KeyMsg{Type: tea.KeyDown})
	m = createKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	m = createKey(m, tea.KeyMsg{Type: tea.KeyDown})
	m = createKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	m = createKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	cmd, _ = m.config.FindCommand("deploy-dev")
	if m.screen != ScreenExplore || cmd.Values["env"] != "prod" {
		t.Fatalf("edit did not persist: %+v error=%s", cmd, m.errMsg)
	}
}

func TestOverviewTemplateActionEligibilityAndCancel(t *testing.T) {
	m := exploreProject()
	for _, it := range m.exploreItems(tabCommands) {
		found := false
		for _, action := range m.exploreActions(it) {
			if action.key == "f" {
				found = true
			}
		}
		if found != (it.name == "deploy") {
			t.Fatalf("incorrect eligibility for %s", it.name)
		}
	}
	m = createKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
	m = createKey(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.screen != ScreenExplore || m.sce != nil || m.sp != nil {
		t.Fatal("cancel did not return to overview")
	}
}

func TestOverviewTemplatePaneResizes(t *testing.T) {
	m := exploreProject()
	m = createKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
	for _, width := range []int{110, 60, 110} {
		m = exploreModel(exploreFirst(m.Update(tea.WindowSizeMsg{Width: width, Height: 30})))
		embedded := strings.Contains(stripStyle(m.View()), "[3 Lists]")
		if embedded != (width == 110) || m.sp == nil {
			t.Fatal("resize lost picker or used wrong layout")
		}
		if frameLines(m.View()) > m.height {
			t.Fatal("template picker overflows terminal")
		}
	}
}
