package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestMainMenuKeepsLeftAnchorAndNameAlignment(t *testing.T) {
	m := Model{width: 200, height: 40, screen: ScreenMainMenu, listItems: mainMenuItems()}
	m.listCursor = 1
	view := ansi.Strip(m.viewMainMenu(200))
	lines := strings.Split(view, "\n")
	found := false
	for _, line := range lines {
		if strings.Contains(line, "▶ Run commands") {
			if !strings.HasPrefix(line, "    ▶ Run commands") {
				t.Fatalf("shifted selected row: %q", line)
			}
			found = true
		}
		if strings.Contains(line, "Run workflow") && !strings.HasPrefix(line, "      Run workflow") {
			t.Fatalf("shifted unselected row: %q", line)
		}
	}
	if !found {
		t.Fatal("selected row missing")
	}
	if !strings.Contains(lines[len(lines)-1], "Esc Quit") {
		t.Fatal("footer missing")
	}
}
