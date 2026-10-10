package tui

import (
	"testing"

	mdl "github.com/Taka-S-dev/baton/internal/model"
)

func TestMainMenuRunFirstAndContentDefault(t *testing.T) {
	items := mainMenuItems()
	want := []string{"Run workflow", "Run commands", "Edit project", "Switch project", "Exit"}
	for i, name := range want {
		if items[i] != name {
			t.Fatalf("menu %v", items)
		}
	}
	m := exploreProject()
	if items[m.defaultMainMenuCursor()] != "Run workflow" {
		t.Fatal("workflow default")
	}
	m.workflows = nil
	if items[m.defaultMainMenuCursor()] != "Run commands" {
		t.Fatal("command default")
	}
	m.config = mdl.Config{}
	if items[m.defaultMainMenuCursor()] != "Edit project" {
		t.Fatal("empty project default")
	}
	m.projectDir = ""
	m.mainMenuCursor = 3
	m.gotoMainMenu()
	if m.listCursor != 3 {
		t.Fatal("return lost selection")
	}
}
