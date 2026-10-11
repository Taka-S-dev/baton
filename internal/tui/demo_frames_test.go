package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	mdl "github.com/Taka-S-dev/baton/internal/model"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// TestDemoFrames writes the frames of docs/demo.gif: the real View() of a
// sample project, in colour, after each key of the recorded session. It
// runs only when DEMO_FRAMES_DIR names the output directory; docs/demo
// explains how the frames become the gif.
func TestDemoFrames(t *testing.T) {
	out := os.Getenv("DEMO_FRAMES_DIR")
	if out == "" {
		t.Skip("set DEMO_FRAMES_DIR to record the demo frames")
	}
	old := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(old) })

	dir := filepath.Join(t.TempDir(), "api")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	m := Model{width: 100, height: 20, projectDir: dir, projectsDir: filepath.Dir(dir), configFile: "commands.tsv", launchDir: dir}
	m.config = mdl.Config{Base: []mdl.Command{
		{Name: "build", Group: "make", Cmd: "go build ./...", Source: "tsv"},
		{Name: "test", Group: "make", Cmd: "go test ./...", Source: "tsv"},
		{Name: "deploy", Group: "deploy", Cmd: "deploy --env {env} --dir {$root}", Source: "tsv"},
		{Name: "logs", Group: "ops", Cmd: "kubectl logs -f {service}", Source: "tsv"},
	}}
	m.workflows = []mdl.Workflow{
		{Name: "build+test", Commands: []string{"build", "test"}},
		{Name: "ship", Commands: []string{"build", "test", "deploy"}},
	}
	m.lists = map[string][]mdl.ListEntry{"env": {{Value: "staging"}, {Value: "prod", Label: "production"}}, "service": {{Value: "api"}, {Value: "web"}}}
	m.vars = map[string]string{"root": `C:\work\api`}
	m.nameInput = textinput.New()
	m.gotoPostLoad()

	// Each step is a key and how long its frame stays up, in 1/100 s.
	steps := []struct {
		key  tea.KeyMsg
		hold int
	}{
		{tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")}, 130},
		{tea.KeyMsg{Type: tea.KeyTab}, 80},
		{tea.KeyMsg{Type: tea.KeyDown}, 40},
		{tea.KeyMsg{Type: tea.KeyDown}, 40},
		{tea.KeyMsg{Type: tea.KeyTab}, 100},
		{tea.KeyMsg{Type: tea.KeyEnter}, 140},
		{tea.KeyMsg{Type: tea.KeyDown}, 60},
		{tea.KeyMsg{Type: tea.KeyEnter}, 300},
	}
	write := func(i, hold int) {
		name := filepath.Join(out, fmt.Sprintf("%02d_%d.txt", i, hold))
		if err := os.WriteFile(name, []byte(m.View()), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(0, 110)
	for i, s := range steps {
		next, _ := m.Update(s.key)
		m = exploreModel(next)
		write(i+1, s.hold)
	}
	if m.screen != ScreenConfirmRun {
		t.Fatalf("the session should end on the run review, not %v", m.screen)
	}
}
