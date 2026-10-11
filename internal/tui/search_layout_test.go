package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestCommandSearchMetadataLayout(t *testing.T) {
	m := homeTab(homeTestModel(t), '2')
	for _, width := range []int{1, 12, 24, 48, 80} {
		for _, query := range []string{"", "build", strings.Repeat("日本語", 30)} {
			m.msSearchTI.SetValue(query)
			for _, active := range []bool{false, true} {
				m.runHome.searching = active
				line := m.commandSearchLine(width, 3)
				if ansi.StringWidth(line) > width {
					t.Fatalf("overflow at %d: %q", width, line)
				}
				plain := ansi.Strip(line)
				if width >= 48 && (!strings.HasSuffix(plain, "s Sort") || ansi.StringWidth(line) != width) {
					t.Fatalf("metadata not anchored: %q", plain)
				}
				if width == 24 && (!strings.HasSuffix(plain, "3/5") || strings.Contains(plain, "Sort")) {
					t.Fatalf("narrow view must keep count before sort: %q", plain)
				}
			}
		}
	}
}
