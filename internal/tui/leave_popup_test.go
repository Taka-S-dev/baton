package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func dirtyPopupModel(width int) Model {
	m := exploreProject()
	m.width = width
	m = createKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	m.nameInput.SetValue("echo unsaved")
	return createKey(m, tea.KeyMsg{Type: tea.KeyEsc})
}

func TestLeavePopupPreservesFormAndDefaultsToKeep(t *testing.T) {
	for _, width := range []int{60, 110} {
		m := dirtyPopupModel(width)
		if m.leaveAsk != leaveForm {
			t.Fatal("confirmation missing")
		}
		bg := m
		bg.leaveAsk = leaveNone
		before := strings.Split(bg.renderScreen(width), "\n")
		after := strings.Split(m.renderScreen(width), "\n")
		p := m.leavePopup(width)
		if len(before) != len(after) {
			t.Fatal("dialog changed frame height")
		}
		for i := range before {
			if (i < p.y || i >= p.y+p.h) && before[i] != after[i] {
				t.Fatal("dialog replaced background")
			}
		}
		kept := createKey(m, tea.KeyMsg{Type: tea.KeyEnter})
		if kept.cf == nil || kept.leaveAsk != leaveNone || kept.nameInput.Value() != "echo unsaved" {
			t.Fatal("default Enter must preserve draft")
		}
		outside := exploreModel(exploreFirst(m.Update(tea.MouseMsg{X: 0, Y: 0, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})))
		if outside.cf == nil || outside.leaveAsk != leaveNone || outside.exploreCursor != m.exploreCursor {
			t.Fatal("outside click must only keep editing")
		}
		wheel := exploreModel(exploreFirst(m.Update(tea.MouseMsg{X: 0, Y: 0, Button: tea.MouseButtonWheelDown, Action: tea.MouseActionPress})))
		if wheel.exploreCursor != m.exploreCursor || wheel.leaveAsk != leaveForm {
			t.Fatal("wheel reached background")
		}
	}
}

func TestLeavePopupSaveFailureKeepsDraft(t *testing.T) {
	m := exploreProject()
	m.projectDir = t.TempDir()
	if err := os.Mkdir(filepath.Join(m.projectDir, "commands.tsv"), 0700); err != nil {
		t.Fatal(err)
	}
	m.openExploreCommandForm("")
	m.cf.fields[0] = "new-command"
	m.nameInput.SetValue("echo unsaved")
	m = createKey(m, tea.KeyMsg{Type: tea.KeyEsc})
	m = createKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	if m.cf == nil || m.leaveAsk != leaveNone || m.nameInput.Value() != "echo unsaved" || m.errMsg == "" {
		t.Fatal("failed save lost draft or hid error")
	}
}

func TestLeavePopupClickDiscard(t *testing.T) {
	m := dirtyPopupModel(110)
	p := m.leavePopup(m.viewWidth())
	m = exploreModel(exploreFirst(m.Update(tea.MouseMsg{X: p.x + 4, Y: p.y + 3, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})))
	if m.cf != nil || m.leaveAsk != leaveNone || m.screen != ScreenExplore {
		t.Fatal("discard click did not return to list")
	}
}
