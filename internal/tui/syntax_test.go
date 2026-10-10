package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
	"strings"
	"testing"
)

func TestSyntaxColorsAndLiteralPreservation(t *testing.T) {
	old := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(old) })
	m := exploreProject()
	source := "echo {env} {$root} {services...} {$missing}"
	got := m.syntaxText(source)
	if ansi.Strip(got) != source {
		t.Fatal("coloring changed source text")
	}
	for _, want := range []string{slotVar("{env}"), slotVar("{services...}"), sFixedVar.Render("{$root}"), sError.Bold(true).Render("{$missing}")} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing semantic color: %q", got)
		}
	}
	text := m.templateSyntax("echo prod {env} {other}", map[string]string{"env": "prod", "other": "prod"}, "env")
	if strings.Count(text, sCurrentValue.Render("prod")) != 1 || ansi.Strip(text) != "echo prod prod prod" {
		t.Fatalf("highlight must follow placeholder origin: %q", text)
	}
}

func TestSyntaxVariableExpansionIsSinglePass(t *testing.T) {
	m := exploreProject()
	m.vars["root"] = "{$nested}"
	m.vars["nested"] = "expanded"
	got := ansi.Strip(m.templateSyntax("{$root}/{env}", map[string]string{"env": "{$root}"}, "env"))
	if got != "{$nested}/{$nested}" {
		t.Fatalf("recursive expansion: %q", got)
	}
	var b strings.Builder
	m.writeSyntaxWarnings(&b, "{$missing} {$missing}")
	if strings.Count(ansi.Strip(b.String()), "⚠ Undefined variable") != 1 {
		t.Fatal("missing or duplicated warning")
	}
}
