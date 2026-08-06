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

// TestTruncate checks the shared shortener every view uses: text that
// fits is untouched, longer text is cut to the limit including the
// ellipsis, and multi-byte text is cut on rune boundaries — the reason
// this replaced the byte slicing each view used to do.
func TestTruncate(t *testing.T) {
	cases := []struct {
		in     string
		maxLen int
		want   string
	}{
		{"go build ./...", 20, "go build ./..."},
		{"go build ./...", 14, "go build ./..."},
		{"go build ./cmd/baton", 14, "go build ./..."},
		{"日本語のパスを含むコマンド", 6, "日本語..."},
		{"abcdef", 3, "abc"},
		{"abcdef", 0, ""},
	}
	for _, c := range cases {
		got := truncate(c.in, c.maxLen)
		if got != c.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", c.in, c.maxLen, got, c.want)
		}
		if len([]rune(got)) > c.maxLen {
			t.Errorf("truncate(%q, %d) = %q — longer than the limit", c.in, c.maxLen, got)
		}
	}
}

// TestViewRunWorkflowSteps_ShowsWorkdir checks each step keeps the
// context the workflow list already showed: the same command means
// something different per directory, so the workdir travels into the
// picker — resolved for {$vars}, left literal for run-time slots, and
// absent when the command has none.
func TestViewRunWorkflowSteps_ShowsWorkdir(t *testing.T) {
	m := Model{width: 96, height: 24}
	m.vars = map[string]string{"root": `C:\work`}
	m.config = mdl.Config{Base: []mdl.Command{
		{Name: "build", Cmd: "make build", Dir: "{workdir}"},
		{Name: "pack", Cmd: "zip -r out.zip .", Dir: `{$root}\dist`},
		{Name: "hello", Cmd: "echo hi"},
	}}
	m.workflows = []mdl.Workflow{{Name: "wf", Commands: []string{"build", "pack", "hello"}}}
	m.gotoWorkflowStepPick(0)

	view := m.viewRunWorkflowSteps(96)
	for _, want := range []string{
		"$ make build  (workdir: {workdir})",
		`$ zip -r out.zip .  (workdir: C:\work\dist)`,
		"$ echo hi",
	} {
		if !strings.Contains(view, want) {
			t.Errorf("view missing %q\n%s", want, view)
		}
	}
	if strings.Contains(view, "echo hi  (workdir") {
		t.Error("a command with no workdir must not show an empty one")
	}
}

// TestViewRunWorkflowSteps_FixedHeight holds the step picker to the same
// discipline as the other scrolling screens: it must fit the terminal and
// keep one height as the cursor moves and steps get toggled.
func TestViewRunWorkflowSteps_FixedHeight(t *testing.T) {
	steps := make([]string, 30)
	var cmds []mdl.Command
	for i := range steps {
		steps[i] = fmt.Sprintf("step-%02d", i)
		cmds = append(cmds, mdl.Command{Name: steps[i], Cmd: "echo hi"})
	}
	steps = append(steps, "ghost") // a step whose command is gone

	m := Model{width: 80, height: 20}
	m.config = mdl.Config{Base: cmds}
	m.workflows = []mdl.Workflow{{Name: "long", Commands: steps}}
	m.gotoWorkflowStepPick(0)

	lines := func(s string) int { return strings.Count(s, "\n") }
	first := lines(m.viewRunWorkflowSteps(80))
	if first > m.height {
		t.Fatalf("view is %d lines for a %d-line terminal — overflow scrolls the terminal on every repaint", first, m.height)
	}
	for c := range steps {
		m.wfp.cursor = c
		if got := lines(m.viewRunWorkflowSteps(80)); got != first {
			t.Fatalf("cursor=%d: view height %d != %d — layout must not shift while scrolling", c, got, first)
		}
	}

	// The selection summary replaces the hint line rather than adding one.
	m.wfp.cursor = 0
	m.wfp.picked[0] = true
	if got := lines(m.viewRunWorkflowSteps(80)); got != first {
		t.Fatalf("with a selection: height %d != %d — the summary must not add a line", got, first)
	}
}

// TestWorkflowNameIsGenerated checks the test that decides whether a
// workflow name may be shown as composed of step names. It must rest on
// rebuilding the name from the steps, never on spotting a "+": a name
// the user typed is not baton's to reinterpret, and the rendered text
// stays byte-for-byte the name either way.
func TestWorkflowNameIsGenerated(t *testing.T) {
	m := Model{}
	m.workflows = []mdl.Workflow{
		{Name: "build+test", Commands: []string{"build", "test"}},   // generated
		{Name: "build+test-2", Commands: []string{"build", "test"}}, // generated, collision suffix
		{Name: "C++ build", Commands: []string{"compile", "link"}},  // typed, contains "+"
		{Name: "nightly", Commands: []string{"clean", "build"}},     // typed
		{Name: "deploy+extra", Commands: []string{"deploy"}},        // typed, has a "+" of its own
	}
	want := []bool{true, true, false, false, false}
	for i, w := range want {
		if got := m.workflowNameIsGenerated(i); got != w {
			t.Errorf("workflow %q (steps %v): generated = %v, want %v",
				m.workflows[i].Name, m.workflows[i].Commands, got, w)
		}
	}
	if m.workflowNameIsGenerated(len(m.workflows)) || m.workflowNameIsGenerated(-1) {
		t.Error("an out-of-range index must not report a generated name")
	}

	// Styling never edits the text: what is displayed is the name.
	for i := range m.workflows {
		for _, hovered := range []bool{false, true} {
			if got := m.workflowLabel(i, hovered); got != m.workflows[i].Name {
				t.Errorf("label(%d, %v) = %q, want the name unchanged", i, hovered, got)
			}
		}
	}
}

// TestViewMainMenu_StableHeight checks the two-pane menu keeps a constant
// frame height while the cursor moves: the right pane's shortcut list
// varies per item and must be padded, or the footer bounces around.
func TestViewMainMenu_StableHeight(t *testing.T) {
	m := Model{width: 100, height: 30, projectDir: "proj"}
	lines := func(s string) int { return strings.Count(s, "\n") }
	first := lines(m.viewMainMenu(100))
	for i := range mainMenuItems() {
		m.listCursor = i
		if got := lines(m.viewMainMenu(100)); got != first {
			t.Fatalf("cursor=%d: height %d != %d — pad the right pane to a constant height", i, got, first)
		}
	}
}

// TestViewMultiSelect_ScrollMarkersAndCounter checks the overflow hints:
// the search row shows a filtered/total counter and the scroll markers
// carry the hidden-row counts instead of a bare "...".
func TestViewMultiSelect_ScrollMarkersAndCounter(t *testing.T) {
	cmds := make([]mdl.Command, 30)
	var items []msItem
	for i := range cmds {
		cmds[i] = mdl.Command{Name: fmt.Sprintf("cmd-%02d", i), Cmd: "echo hi"}
		items = append(items, msItem{cmd: &cmds[i]})
	}
	m := Model{width: 80, height: 20}
	m.msItems = items
	m.msSearchTI = textinput.New()

	view := m.viewMultiSelect(80)
	if !strings.Contains(view, "30/30") {
		t.Fatal("search row must show the filtered/total counter")
	}
	if !strings.Contains(view, "↓") || !strings.Contains(view, "more") {
		t.Fatal("rows below the fold must be announced as \"↓ N more\"")
	}

	m.msCursor = len(items) - 1
	view = m.viewMultiSelect(80)
	if !strings.Contains(view, "↑") {
		t.Fatal("rows above the fold must be announced as \"↑ N more\"")
	}

	m.msSearchTI.SetValue("cmd-0")
	if view = m.viewMultiSelect(80); !strings.Contains(view, "10/30") {
		t.Fatal("counter must reflect the active filter")
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

// TestViewPickScreens_FixedHeight checks the management pickers hold a
// constant frame height that fits the terminal while the cursor sweeps
// the whole list — the same guarantee the command selector has.
func TestViewPickScreens_FixedHeight(t *testing.T) {
	m := Model{width: 80, height: 20, screen: ScreenDeleteCommand}
	for i := 0; i < 30; i++ {
		m.config.Commands = append(m.config.Commands,
			mdl.Command{Name: fmt.Sprintf("c%02d", i), Cmd: "echo hi", Source: "local"})
	}
	names, refs := m.editableCommands()
	m.listItems, m.editRefs = names, refs

	lines := func(s string) int { return strings.Count(s, "\n") }
	views := map[string]func() string{
		"edit pick":   func() string { return m.viewEditCommandPick(80) },
		"delete list": func() string { return m.viewDeleteList("Delete commands", m.listItems, 80) },
	}
	for name, render := range views {
		m.listCursor = 0
		first := lines(render())
		if first > m.height {
			t.Fatalf("%s: view is %d lines for a %d-line terminal", name, first, m.height)
		}
		for c := range names {
			m.listCursor = c
			if got := lines(render()); got != first {
				t.Fatalf("%s: cursor=%d height %d != %d — layout must not shift while scrolling", name, c, got, first)
			}
		}
	}
}

// TestViewSlotPick_FixedHeight checks the slot picker frame stays a
// constant size that fits the terminal while the cursor sweeps the
// entries, the custom row and the skip row.
func TestViewSlotPick_FixedHeight(t *testing.T) {
	cmd := mdl.Command{Name: "up", Cmd: "up {env}"}
	var entries []mdl.ListEntry
	for i := 0; i < 30; i++ {
		entries = append(entries, mdl.ListEntry{Value: fmt.Sprintf("env-%02d", i)})
	}
	m := Model{width: 80, height: 20}
	m.sp = &slotPickState{
		slotName: "env", listName: "env", entries: entries,
		canSkip: true, currentCmd: &cmd, resolvedSoFar: map[string]string{},
	}
	m.sp.applyFilter()

	lines := func(s string) int { return strings.Count(s, "\n") }
	first := lines(m.viewSlotPick(80))
	if first > m.height {
		t.Fatalf("view is %d lines for a %d-line terminal", first, m.height)
	}
	total := len(m.sp.filtered) + 2 // + custom row + skip row
	for c := 0; c < total; c++ {
		m.sp.cursor = c
		if got := lines(m.viewSlotPick(80)); got != first {
			t.Fatalf("cursor=%d: view height %d != %d — layout must not shift while scrolling", c, got, first)
		}
	}
}

// TestViewRunWorkflow_FixedHeight checks the workflow list budgets
// against the fixed steps viewport, not the hovered workflow's step
// count, so the frame fits the terminal at every cursor position.
func TestViewRunWorkflow_FixedHeight(t *testing.T) {
	m := Model{width: 80, height: 20}
	for i := 0; i < 20; i++ {
		m.workflows = append(m.workflows, mdl.Workflow{
			Name: fmt.Sprintf("wf-%02d", i), Commands: []string{"a", "b", "c"},
		})
	}
	m.wfSearchTI = textinput.New()
	m.updateStepsViewport()

	lines := func(s string) int { return strings.Count(s, "\n") }
	first := lines(m.viewRunWorkflow(80))
	if first > m.height {
		t.Fatalf("view is %d lines for a %d-line terminal", first, m.height)
	}
	for c := range m.workflows {
		m.listCursor = c
		m.updateStepsViewport()
		if got := lines(m.viewRunWorkflow(80)); got != first {
			t.Fatalf("cursor=%d: view height %d != %d — layout must not shift while scrolling", c, got, first)
		}
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
	// Tall enough that the entries, custom row and skip row all fit the
	// row window — the context panel and command preview eat ~16 lines.
	m := Model{width: 80, height: 30}
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
