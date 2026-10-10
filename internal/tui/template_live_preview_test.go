package tui

import (
	"strings"
	"testing"

	mdl "github.com/Taka-S-dev/baton/internal/model"
	tea "github.com/charmbracelet/bubbletea"
)

func TestTemplatePreviewFollowsCandidateWithoutCommitting(t *testing.T) {
	m := exploreProject()
	m = createKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
	preview := func() string { var b strings.Builder; m.writeTemplatePreview(&b); return stripStyle(b.String()) }
	if !strings.Contains(preview(), "deploy --env dev") {
		t.Fatal("initial candidate not previewed")
	}
	m = createKey(m, tea.KeyMsg{Type: tea.KeyDown})
	if !strings.Contains(preview(), "deploy --env prod") {
		t.Fatal("cursor movement not previewed")
	}
	if len(m.sce.currentValues) != 0 {
		t.Fatal("preview committed candidate")
	}
	m.sp.search = "custom"
	m.sp.applyFilter()
	m.sp.cursor = 0
	if !strings.Contains(preview(), "deploy --env custom") {
		t.Fatal("typed value not previewed")
	}
	m.sp.cursor = len(m.sp.filtered) + 1
	if !strings.Contains(preview(), "deploy --env {env}") {
		t.Fatal("interactive option must retain placeholder")
	}
	if len(m.sce.currentValues) != 0 {
		t.Fatal("preview mutated values")
	}
}

func TestTemplateVariadicPreviewMatchesConfirmation(t *testing.T) {
	m := exploreProject()
	for i := range m.config.Base {
		if m.config.Base[i].Name == "deploy" {
			m.config.Base[i].Cmd = "deploy {env...}"
			m.config.Base[i].Dir = "{env...}"
		}
	}
	m = createKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
	m.sp.picked = []string{"dev", "prod"}
	m.sp.cursor = 0
	var b strings.Builder
	m.writeTemplatePreview(&b)
	if !strings.Contains(b.String(), "deploy dev prod") || !strings.Contains(b.String(), "dev prod") {
		t.Fatal("multi-selection not previewed")
	}
	m = createKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.sce.currentValues["env"] != "dev prod" {
		t.Fatal("preview differs from confirmation")
	}
}

func TestTemplatePreviewPreservesEarlierValuesAndResolvesWorkdir(t *testing.T) {
	m := exploreProject()
	m.config.Commands = nil
	m.config.Base = []mdl.Command{{Name: "example", Cmd: "echo {first} {env}", Dir: "{$root}/{env}", Source: "tsv"}}
	m = createKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
	m.sce.currentValues = map[string]string{"first": "hello"}
	m.sp = &slotPickState{slotName: "env", filtered: []mdl.ListEntry{{Value: "prod"}}, canSkip: true}
	var b strings.Builder
	m.writeTemplatePreview(&b)
	text := stripStyle(b.String())
	if !strings.Contains(text, "echo hello prod") || !strings.Contains(text, m.vars["root"]+"/prod") {
		t.Fatalf("incomplete preview: %s", text)
	}
	if len(m.sce.currentValues) != 1 || m.sce.currentValues["first"] != "hello" {
		t.Fatal("preview modified confirmed values")
	}
}
