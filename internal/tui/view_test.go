package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/textinput"

	mdl "github.com/Taka-S-dev/baton/internal/model"
)

// TestViewMultiSelect_StableValueOrder guards against map-iteration-order
// leaking into rendered output: with 2+ values the lines would swap on
// every re-render, which the user sees as flickering while idle (cursor
// blink ticks trigger re-renders). The view must render identically every
// time for the same model.
func TestViewMultiSelect_StableValueOrder(t *testing.T) {
	cmd := mdl.Command{
		Name:     "copy-all",
		Template: "copy",
		Cmd:      "cp a b",
		Values: map[string]string{
			"src": "./src", "dest": "./dist", "mode": "fast", "env": "dev",
		},
	}
	m := Model{width: 80, height: 24}
	m.msItems = []msItem{{cmd: &cmd}}
	m.msSearchTI = textinput.New()
	m.msGroupTI = textinput.New()

	first := m.viewMultiSelect(80)
	for i := 0; i < 50; i++ {
		if got := m.viewMultiSelect(80); got != first {
			t.Fatalf("render %d differs from first render — map iteration order is leaking into the view", i)
		}
	}
}

// TestViewCommandNameInput_StableValueOrder does the same for the
// create/edit command name screen, which lists the chosen slot values.
func TestViewCommandNameInput_StableValueOrder(t *testing.T) {
	m := Model{width: 80, height: 24}
	m.nameInput = textinput.New()
	sce := &commandEditState{
		currentValues: map[string]string{
			"src": "./src", "dest": "./dist", "mode": "fast", "env": "dev",
		},
	}

	first := m.viewCommandNameInput("Create command", 80, sce)
	for i := 0; i < 50; i++ {
		if got := m.viewCommandNameInput("Create command", 80, sce); got != first {
			t.Fatalf("render %d differs from first render — map iteration order is leaking into the view", i)
		}
	}
}
