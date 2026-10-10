package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestCommandRelatedNavigationReturnsToSelection(t *testing.T) {
	m := exploreProject()
	m.exploreTab = tabCommands
	m.exploreSearch = "deploy"
	it, ok := m.exploreCurrent()
	if !ok {
		t.Fatal("missing command")
	}
	links := m.commandLinks(it.name)
	if len(links) != 1 || links[0].name != "env" {
		t.Fatal("missing related list")
	}
	m.runExploreAction(linkKey(0), it)
	if m.exploreTab != tabLists {
		t.Fatal("did not open list")
	}
	m = press(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.exploreTab != tabCommands || m.exploreSearch != "deploy" {
		t.Fatal("did not restore origin")
	}
}

func TestSharedVariablesExcludeFixedValuesWithoutDeletingThem(t *testing.T) {
	m := exploreProject()
	for _, it := range m.exploreItems(tabVars) {
		if strings.Contains(it.name, ".") {
			t.Fatal("fixed value in shared variables")
		}
	}
	if m.vars["build-api.projDir"] == "" {
		t.Fatal("fixed data removed")
	}
	links := m.commandLinks("build")
	if len(links) != 1 || links[0].tab != tabVars || links[0].name != "root" {
		t.Fatal("missing global link")
	}
	if tabNames[0] != "Workflows" || tabNames[1] != "Commands" {
		t.Fatal("wrong tab order")
	}
}
