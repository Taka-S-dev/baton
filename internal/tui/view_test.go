package tui

import (
	"fmt"
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

// TestViewMultiSelect_FixedHeight guards against the frame overflowing the
// terminal or changing height as the cursor moves: an overflowing frame
// makes the terminal itself scroll on every repaint (visible jitter), and
// the "..." markers / hover preview appearing at different heights makes
// the footer jump around.
func TestViewMultiSelect_FixedHeight(t *testing.T) {
	cmds := make([]mdl.Command, 30)
	var items []msItem
	for i := range cmds {
		cmds[i] = mdl.Command{Name: fmt.Sprintf("cmd-%02d", i), Cmd: "echo hi"}
		items = append(items, msItem{cmd: &cmds[i]})
	}
	tmpl := mdl.Command{
		Name: "copy-all", Template: "copy", Cmd: "cp a b",
		Values: map[string]string{"src": "./src", "dest": "./dist"},
	}
	items = append(items, msItem{cmd: &tmpl})

	m := Model{width: 80, height: 20}
	m.msItems = items
	m.msSearchTI = textinput.New()

	lines := func(s string) int { return strings.Count(s, "\n") }
	first := lines(m.viewMultiSelect(80))
	if first > m.height {
		t.Fatalf("view is %d lines for a %d-line terminal — overflow scrolls the terminal on every repaint", first, m.height)
	}
	for c := range items {
		m.msCursor = c
		if got := lines(m.viewMultiSelect(80)); got != first {
			t.Fatalf("cursor=%d: view height %d != %d — layout must not shift while scrolling", c, got, first)
		}
	}

	// Filtering down to a handful of rows must not change the height either.
	m.msCursor = 0
	m.msSearchTI.SetValue("cmd-0")
	if got := lines(m.viewMultiSelect(80)); got != first {
		t.Fatalf("filtered: view height %d != %d — the list region must stay padded", got, first)
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

// TestViewEditCommandPick_ShowsPreview checks the edit picker renders a
// hover preview for the highlighted command: template and values for
// derived commands, the command line otherwise.
func TestViewEditCommandPick_ShowsPreview(t *testing.T) {
	m := Model{width: 80, height: 24}
	m.config.Commands = []mdl.Command{
		{Name: "as", Template: "build", Cmd: "make build", Values: map[string]string{"workdir": "./src"}},
		{Name: "plain", Cmd: "echo hi", Dir: "./x"},
	}
	names, refs := m.editableCommands()
	m.listItems, m.editRefs = names, refs

	view := m.viewEditCommandPick(80)
	if !strings.Contains(view, "template: build") || !strings.Contains(view, "./src") {
		t.Fatalf("derived command preview missing template/values:\n%s", view)
	}

	m.listCursor = 1
	view = m.viewEditCommandPick(80)
	if !strings.Contains(view, "$ echo hi") || !strings.Contains(view, "workdir: ./x") {
		t.Fatalf("plain command preview missing cmd/workdir:\n%s", view)
	}
}

// TestViewDeleteCommand_ShowsPreview checks the Delete command list
// renders the hovered command's preview, like the edit picker.
func TestViewDeleteCommand_ShowsPreview(t *testing.T) {
	m := Model{width: 80, height: 24, screen: ScreenDeleteCommand}
	m.config.Commands = []mdl.Command{{Name: "plain", Cmd: "echo hi", Dir: "./x", Source: "local"}}
	names, refs := m.editableCommands()
	m.listItems, m.editRefs = names, refs

	view := m.viewDeleteList("Delete commands", m.listItems, 80)
	if !strings.Contains(view, "$ echo hi") || !strings.Contains(view, "workdir: ./x") {
		t.Fatalf("delete list must preview the hovered command:\n%s", view)
	}
}

// TestViewDeleteCommand_MarksSavedCommands checks Delete command rows use
// the shared picker labels: saved (template-derived) commands carry the
// $ marker and their template name, like the edit picker.
func TestViewDeleteCommand_MarksSavedCommands(t *testing.T) {
	m := Model{width: 80, height: 24, screen: ScreenDeleteCommand}
	m.config.Base = []mdl.Command{{Name: "build", Cmd: "make {x}", Source: "tsv"}}
	m.config.Commands = []mdl.Command{{Name: "as", Template: "build", Cmd: "make src", Source: "local"}}
	names, refs := m.editableCommands()
	m.listItems, m.editRefs = names, refs

	view := m.viewDeleteList("Delete commands", m.listItems, 80)
	if !strings.Contains(view, "$") || !strings.Contains(view, "(build)") {
		t.Fatalf("saved command must carry the $ marker and its template:\n%s", view)
	}
}

// TestViewSlotPick_Variadic checks the multi-pick rendering: checkbox
// markers on entries, the joined picks in the command preview, and the
// Tab hint in the key guide.
func TestViewSlotPick_Variadic(t *testing.T) {
	cmd := mdl.Command{Name: "up", Cmd: "docker compose up {services...}"}
	m := Model{width: 80, height: 24}
	m.sp = &slotPickState{
		slotName: "services", listName: "services", variadic: true,
		entries:       []mdl.ListEntry{{Value: "api"}, {Value: "web"}, {Value: "worker"}},
		picked:        []string{"api", "web"},
		contextNames:  []string{"up"},
		contextNotes:  []string{""},
		currentCmd:    &cmd,
		resolvedSoFar: map[string]string{},
	}
	m.sp.applyFilter()

	view := m.viewSlotPick(80)
	if !strings.Contains(view, "[x]") || !strings.Contains(view, "[ ]") {
		t.Fatal("variadic picker must render checkbox markers")
	}
	if !strings.Contains(view, "{services...}") {
		t.Fatal("title and preview must show the variadic placeholder")
	}
	if !strings.Contains(view, "api web") {
		t.Fatal("preview must show the picked values joined with spaces")
	}
	if !strings.Contains(view, "Tab") {
		t.Fatal("key guide must mention Tab")
	}

	m.sp.variadic = false
	m.sp.picked = nil
	if strings.Contains(m.viewSlotPick(80), "[ ]") {
		t.Fatal("single-value picker must not render checkboxes")
	}
}

// TestViewVarRebase_LabelsShowKindAndOwner checks the offer rows say
// what each value is and where it lives: a kind tag, the owning
// command and slot for fixed values, and the list file for entries.
func TestViewVarRebase_LabelsShowKindAndOwner(t *testing.T) {
	m := Model{width: 100, height: 24}
	m.vr = &varRebaseState{
		varName:   "bbb.workdir",
		propagate: true,
		editedOld: "./src",
		editedNew: "./step",
		items: []varRebaseItem{
			{kind: 0, key: "bbb.workdir", label: fixedValueLabel("bbb.workdir"), oldValue: "./src", newValue: "./step", on: true},
			{kind: 1, listName: "source_path", entryIdx: 0, label: "lists/source_path.tsv", oldValue: "./src", newValue: "./step", on: true},
		},
	}
	view := m.viewVarRebase(100)
	for _, want := range []string{
		"[fixed value]", `command "bbb" · slot "workdir"`,
		"[list entry]", "lists/source_path.tsv",
		"(already saved)", // the committed edit is echoed, so Esc clearly keeps it
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("offer row missing %q:\n%s", want, view)
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
