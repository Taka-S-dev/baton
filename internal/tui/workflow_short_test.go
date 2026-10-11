package tui

import (
	"strings"
	"testing"

	mdl "github.com/Taka-S-dev/baton/internal/model"
	"github.com/charmbracelet/x/ansi"
)

func TestShortWorkflowSplitTracksSelectionAndFits(t *testing.T) {
	for _, h := range []int{7, 12, 19} {
		m := Model{width: 120, height: h, workflows: []mdl.Workflow{{Name: "first", Commands: []string{"build"}}, {Name: "second", Commands: []string{"deploy"}}}}
		for i, name := range []string{"build", "deploy"} {
			m.listCursor = i
			got := stripStyle(m.viewRunWorkflow(120))
			if !strings.Contains(got, "Execution steps") || !strings.Contains(got, "1. "+name) {
				t.Fatal(got)
			}
			lines := strings.Split(got, "\n")
			if len(lines) != h {
				t.Fatal(got)
			}
			for _, line := range lines {
				if ansi.StringWidth(line) > 120 {
					t.Fatal(line)
				}
			}
		}
		if strings.Contains(stripStyle(m.viewRunWorkflow(80)), "Execution steps") {
			t.Fatal("narrow view split")
		}
	}
}
