package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestEmptyCategoryHelpFitsResizedPane(t *testing.T) {
	for tab := 0; tab < tabCount; tab++ {
		for _, split := range []int{30, 65, 80} {
			m := exploreProject()
			m.width = 120
			m.height = 30
			m.exploreTab = tab
			m.exploreSplit = split
			m.config.Base = nil
			m.config.Commands = nil
			m.workflows = nil
			m.lists = nil
			m.vars = nil
			l := m.exploreLayout(m.viewWidth())
			got := ansi.Strip(m.exploreEditorBox(l))
			what, how := categoryHelp(tab)
			// Remove the box edges and normalize only layout whitespace.
			var text []string
			for _, line := range strings.Split(got, "\n") {
				if strings.HasPrefix(line, "│") {
					text = append(text, strings.TrimSuffix(strings.TrimPrefix(line, "│"), "│"))
				}
				if ansi.StringWidth(line) > l.rightW {
					t.Fatal("overflow")
				}
			}
			normalized := strings.Join(strings.Fields(strings.Join(text, " ")), "")
			if !strings.Contains(normalized, strings.Join(strings.Fields(what), "")) || !strings.Contains(normalized, strings.Join(strings.Fields(how), "")) {
				t.Fatalf("tab %d width %d lost text: %s", tab, l.rightW, got)
			}
		}
	}
}
