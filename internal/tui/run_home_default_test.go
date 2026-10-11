package tui

import (
	mdl "github.com/Taka-S-dev/baton/internal/model"
	"testing"
)

func TestProjectOpensRunHomeByContent(t *testing.T) {
	for _, tc := range []struct {
		workflows, commands bool
		want                Screen
	}{
		{true, true, ScreenRunWorkflow}, {false, true, ScreenRunCommands}, {false, false, ScreenRunCommands},
	} {
		m := exploreProject()
		m.projectDir = t.TempDir()
		if !tc.workflows {
			m.workflows = nil
		}
		if !tc.commands {
			m.config = mdl.Config{}
		}
		m.gotoPostLoad()
		if m.screen != tc.want {
			t.Fatalf("got %v want %v", m.screen, tc.want)
		}
	}
}
