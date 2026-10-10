package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"reflect"
	"testing"
)

func TestSearchMatchMaskPreservesRunePositions(t *testing.T) {
	for _, tc := range []struct {
		text  string
		terms []string
		want  []bool
	}{
		{"Sa-sa", []string{"sa"}, []bool{true, true, false, true, true}},
		{"日本語Go", []string{"日本", "go"}, []bool{true, true, false, true, true}},
		{"ababa", []string{"aba"}, []bool{true, true, true, true, true}},
		{"alpha", []string{""}, []bool{false, false, false, false, false}},
	} {
		if got := searchMatchMask(tc.text, tc.terms); !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("%s: %v", tc.text, got)
		}
	}
}
func TestHighlightDoesNotChangeTextOrWidth(t *testing.T) {
	for _, selected := range []bool{false, true} {
		text := " ▶ 日本語 Sample  12 commands"
		got := searchHighlight(text, []string{"日本", "sample"}, selected, true)
		if ansi.Strip(got) != text || lipgloss.Width(got) != lipgloss.Width(text) {
			t.Fatal("highlight changed content")
		}
		original := sRowActive.Render(text)
		if searchHighlight(original, nil, true, true) != original {
			t.Fatal("empty query changed styling")
		}
	}
}
