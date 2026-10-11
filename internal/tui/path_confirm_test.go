package tui

import (
	"testing"

	mdl "github.com/Taka-S-dev/baton/internal/model"
	tea "github.com/charmbracelet/bubbletea"
)

func TestPathEnterAdvancesAndBackRetainsValue(t *testing.T) {
	for _, later := range []bool{false, true} {
		body := "echo {@file:.go}"
		if later {
			body += " {mode}"
		}
		c := mdl.Command{Name: "build", Cmd: body, Dir: completionFixture(t)}
		nm, read := (Model{width: 80, height: 24}).startResolveFlow([]msItem{{cmd: &c}})
		m := completionResult(nm.(Model), read)
		m = queryCompletion(m, "test.go")
		m, _ = completionKey(m, tea.KeyMsg{Type: tea.KeyEnter})
		if later {
			if m.screen != ScreenSlotPick || m.sp.slotName != "mode" {
				t.Fatal("did not advance to next input")
			}
		} else {
			if m.screen != ScreenConfirmRun || m.running != nil {
				t.Fatal("must review, never execute")
			}
			m, _ = completionKey(m, tea.KeyMsg{Type: tea.KeyEsc})
			if m.sp == nil || m.sp.completedPath != "test.go" {
				t.Fatal("Back lost value")
			}
		}
	}
}
