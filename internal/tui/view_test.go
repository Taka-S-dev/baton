package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"

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

	first := m.viewMultiSelect(80)
	for i := 0; i < 50; i++ {
		if got := m.viewMultiSelect(80); got != first {
			t.Fatalf("render %d differs from first render — map iteration order is leaking into the view", i)
		}
	}
}

// TestViewPlaceholderWindow_StableWidth guards against the picker window
// resizing as the cursor moves between lists: the right pane's width must
// come from the widest entry across ALL lists (clamped to the terminal),
// so the frame stays put and over-long values are truncated instead.
func TestViewPlaceholderWindow_StableWidth(t *testing.T) {
	const w = 80
	m := Model{width: w, height: 24}
	m.nameInput = textinput.New()
	m.lists = map[string][]mdl.ListEntry{
		"env":   {{Value: "dev", Label: "Development"}, {Value: "prod", Label: "Production"}},
		"empty": {},
		"paths": {{Value: `C:\Users\x\a\very\long\path\that\exceeds\the\terminal\width\node_modules\@emotion\styled`, Label: "long"}},
	}
	m.cf = &commandFormState{slotPickFocus: true}

	frameWidth := func(view string) int {
		maxW := 0
		for _, line := range strings.Split(view, "\n") {
			if lw := lipgloss.Width(line); lw > maxW {
				maxW = lw
			}
		}
		return maxW
	}

	first := frameWidth(m.viewPlaceholderWindow(w))
	if first > w {
		t.Fatalf("window width %d exceeds terminal width %d — long values must be truncated", first, w)
	}
	for i := 0; i < len(m.lists); i++ {
		m.cf.slotPickCursor = i
		for pane := 0; pane <= 1; pane++ {
			m.cf.slotPickPane = pane
			if got := frameWidth(m.viewPlaceholderWindow(w)); got != first {
				t.Fatalf("cursor=%d pane=%d: window width %d != %d — frame resizes as the cursor moves", i, pane, got, first)
			}
		}
	}
}

// TestViewPlaceholderWindow_FullTextFooter checks that a truncated entry's
// full text (including a label that was dropped from the row) is echoed in
// the footer while the value pane is focused, and that short entries add
// no footer.
func TestViewPlaceholderWindow_FullTextFooter(t *testing.T) {
	const w = 80
	m := Model{width: w, height: 24}
	m.nameInput = textinput.New()
	m.lists = map[string][]mdl.ListEntry{
		"env":   {{Value: "dev", Label: "Development"}},
		"paths": {{Value: `C:\Users\x\a\very\long\path\that\exceeds\the\terminal\width\node_modules\@emotion\styled`, Label: "emolabel"}},
	}
	// sorted order: env(0), paths(1)
	m.cf = &commandFormState{slotPickFocus: true, slotPickCursor: 1, slotPickPane: 1}

	view := m.viewPlaceholderWindow(w)
	if !strings.Contains(view, "emolabel") {
		t.Fatal("footer should echo the truncated entry's full text, including its label")
	}

	m.cf.slotPickCursor = 0
	view = m.viewPlaceholderWindow(w)
	if strings.Contains(view, "Development\n") && strings.Count(view, "Development") > 1 {
		t.Fatal("short entries must not produce a footer")
	}
}

// TestViewMultiSelect_DiscardWindow checks the discard guard renders as a
// floating window listing the selected names while msEscArmed is set.
func TestViewMultiSelect_DiscardWindow(t *testing.T) {
	cmd := mdl.Command{Name: "build", Cmd: "make all"}
	m := Model{width: 80, height: 24}
	m.msItems = []msItem{{cmd: &cmd}}
	m.msSearchTI = textinput.New()
	m.msSelected = []int{0}
	m.msEscArmed = true

	view := m.viewMultiSelect(80)
	if !strings.Contains(view, "Discard selection?") {
		t.Fatal("armed model must render the discard confirmation window")
	}
	if !strings.Contains(view, "build") {
		t.Fatal("the window must list the selected command names")
	}

	m.msEscArmed = false
	if strings.Contains(m.viewMultiSelect(80), "Discard selection?") {
		t.Fatal("disarmed model must not render the discard window")
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
