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

// demoStep is one key of a recorded session and how long its frame
// stays up, in 1/100 s.
type demoStep struct {
	key  tea.KeyMsg
	hold int
}

func runes(s string) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }

// typed is one step per character, so the recording shows the typing.
func typed(s string, hold int) []demoStep {
	var steps []demoStep
	for _, r := range s {
		steps = append(steps, demoStep{runes(string(r)), hold})
	}
	return steps
}

// demoProject is the sample project every recording starts from, on
// the Run screen, in a temporary directory so that saving works.
func demoProject(t *testing.T) Model {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "api")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	m := Model{width: 100, height: 20, projectDir: dir, projectsDir: filepath.Dir(dir), configFile: "commands.tsv", launchDir: `C:\work\api`}
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
	m.lists = map[string][]mdl.ListEntry{
		"env":     {{Value: "staging"}, {Value: "prod", Label: "production"}},
		"service": {{Value: "api"}, {Value: "web"}},
		"pkg":     {{Value: "./internal/..."}, {Value: "./cmd/..."}},
	}
	m.vars = map[string]string{"root": `C:\work\api`}
	m.nameInput = textinput.New()
	m.gotoPostLoad()
	return m
}

// recordDemo writes the frames of one session under out/name: the real
// View() in colour before the first key and after each one.
func recordDemo(t *testing.T, out, name string, m Model, steps []demoStep) Model {
	t.Helper()
	dir := filepath.Join(out, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(i, hold int) {
		path := filepath.Join(dir, fmt.Sprintf("%02d_%d.txt", i, hold))
		if err := os.WriteFile(path, []byte(m.View()), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(0, 110)
	for i, s := range steps {
		next, _ := m.Update(s.key)
		m = exploreModel(next)
		write(i+1, s.hold)
	}
	return m
}

// TestDemoFrames writes the frames of the gifs in docs/: it runs only
// when DEMO_FRAMES_DIR names the output directory, and docs/demo
// explains how the frames become a gif.
func TestDemoFrames(t *testing.T) {
	out := os.Getenv("DEMO_FRAMES_DIR")
	if out == "" {
		t.Skip("set DEMO_FRAMES_DIR to record the demo frames")
	}
	old := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(old) })

	// docs/demo.gif: pick two commands, answer a placeholder, review.
	run := []demoStep{
		{runes("2"), 130},
		{tea.KeyMsg{Type: tea.KeyTab}, 80},
		{tea.KeyMsg{Type: tea.KeyDown}, 40},
		{tea.KeyMsg{Type: tea.KeyDown}, 40},
		{tea.KeyMsg{Type: tea.KeyTab}, 100},
		{tea.KeyMsg{Type: tea.KeyEnter}, 140},
		{tea.KeyMsg{Type: tea.KeyDown}, 60},
		{tea.KeyMsg{Type: tea.KeyEnter}, 300},
	}
	if m := recordDemo(t, out, "run", demoProject(t), run); m.screen != ScreenConfirmRun {
		t.Fatalf("the run session should end on the review, not %v", m.screen)
	}

	// docs/demo-add.gif: open the editor, write a command with a
	// placeholder from the candidates, take the suggested name, save.
	add := []demoStep{
		{runes("2"), 100},
		{runes("e"), 120},
		{runes("a"), 120},
	}
	add = append(add, typed("go test ./", 7)...)
	add = append(add, demoStep{runes("{"), 120})
	add = append(add, typed("p", 90)...)
	add = append(add,
		demoStep{tea.KeyMsg{Type: tea.KeyEnter}, 120},
		demoStep{tea.KeyMsg{Type: tea.KeyTab}, 120},
		demoStep{tea.KeyMsg{Type: tea.KeyRight}, 110},
		demoStep{tea.KeyMsg{Type: tea.KeyCtrlS}, 300},
	)
	// The editor needs a few more rows than the Run screen for the
	// placeholder candidates to show under the fields.
	tall := demoProject(t)
	tall.height = 26
	m := recordDemo(t, out, "add", tall, add)
	if m.screen != ScreenExplore || m.cf != nil {
		t.Fatalf("the add session should end back on the command list, not screen %v", m.screen)
	}
	if _, ok := m.config.FindCommand("go-test"); !ok {
		t.Fatal("the add session did not save go-test")
	}
}
