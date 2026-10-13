package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"

	mdl "github.com/Taka-S-dev/baton/internal/model"
	"github.com/Taka-S-dev/baton/internal/runner"
	"github.com/Taka-S-dev/baton/internal/slot"
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

	lines := frameLines
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
// ellipsis, and multi-byte text is cut on rune boundaries.
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

// Workdirs stay available in details while the table shows command bodies.
func TestViewRunWorkflowSteps_TableAndDetails(t *testing.T) {
	m := Model{width: 96, height: 24}
	m.config = mdl.Config{Base: []mdl.Command{{Name: "build", Cmd: "make build", Dir: "{workdir}"}}}
	m.workflows = []mdl.Workflow{{Name: "wf", Commands: []string{"build"}}}
	m.gotoWorkflowStepPick(0)
	view := m.viewRunWorkflowStepsList(96)
	for _, want := range []string{"Run", "Step", "Name", "Command", "$ make build", "Select at least one step"} {
		if !strings.Contains(view, want) {
			t.Errorf("missing %q", want)
		}
	}
	if strings.Contains(view, "workdir:") {
		t.Fatal("workdir duplicated in table")
	}
	if !strings.Contains(m.workflowRunDetail(96), "workdir:") {
		t.Fatal("workdir missing from details")
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

	lines := frameLines
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

	// Only step names are marked as steps. A generated name for four or
	// more steps ends in a "+N more" count, whose "+" joins a number
	// rather than another step.
	m2 := Model{workflows: []mdl.Workflow{
		{
			Name:     "provision-infra+migrate-database+2",
			Commands: []string{"provision-infra", "migrate-database", "deploy-services", "smoke-test"},
		},
		{Name: "a+b", Commands: []string{"a", "b"}},
		{Name: "typed+name", Commands: []string{"x"}},
	}}
	cases := []struct {
		parts []string
		tail  string
		ok    bool
	}{
		{[]string{"provision-infra", "migrate-database"}, "+2", true},
		{[]string{"a", "b"}, "", true},
		{nil, "", false},
	}
	for i, c := range cases {
		parts, tail, ok := m2.workflowNameParts(i)
		if ok != c.ok || tail != c.tail || strings.Join(parts, ",") != strings.Join(c.parts, ",") {
			t.Errorf("parts(%q) = %v/%q/%v, want %v/%q/%v",
				m2.workflows[i].Name, parts, tail, ok, c.parts, c.tail, c.ok)
		}
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

// TestViewConfirmRun_ShowsWhatWillRun checks the confirm screen resolves
// {$vars} the way the runner does. It is the screen that answers "what
// is about to run", so showing the unresolved reference there would
// describe something other than what Enter executes.
func TestViewConfirmRun_ShowsWhatWillRun(t *testing.T) {
	cmd := mdl.Command{Name: "test", Cmd: "go test {$pkgdir}/slot/", Dir: "{$root}/api"}
	m := Model{width: 100, height: 30, screen: ScreenConfirmRun}
	m.vars = map[string]string{"pkgdir": "./internal", "root": "/srv"}
	m.confirmRunItems = []mdl.RunItem{{Name: "test", Cmd: &cmd}}

	view := m.viewConfirmRun(100)
	for _, want := range []string{"$ go test ./internal/slot/", "workdir: " + m.confirmWorkdir("/srv/api")} {
		if !strings.Contains(view, want) {
			t.Errorf("confirm view missing %q\n%s", want, view)
		}
	}
	if strings.Contains(view, "{$") {
		t.Errorf("confirm view still shows an unresolved reference\n%s", view)
	}

	// What it printed is what the runner builds from the same command.
	resolved := runner.BuildExecForTest(slot.ApplyVarsToCommand(cmd, m.vars))
	if !strings.Contains(strings.Join(resolved.Args, " "), "go test ./internal/slot/") {
		t.Fatalf("runner would execute %v, which the screen no longer matches", resolved.Args)
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

	lines := frameLines
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

	lines := frameLines
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

	lines := frameLines
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

// TestViewsFitShortTerminals holds the frame inside the terminal when the
// terminal is short — an editor panel opens at well under ten lines, and
// a frame taller than the terminal scrolls it on every repaint, which
// takes the top of the frame with it. The menus shed the logo, their
// headings and finally their own entries; the list screens shed their
// spacing, their hover preview and the rule under their title. Seven
// lines hold every screen.
func TestViewsFitShortTerminals(t *testing.T) {
	cmds := make([]mdl.Command, 8)
	var items []msItem
	for i := range cmds {
		cmds[i] = mdl.Command{Name: fmt.Sprintf("cmd-%d", i), Cmd: "echo hi", Source: "tsv"}
		items = append(items, msItem{cmd: &cmds[i]})
	}
	build := func(h int, sc Screen) Model {
		m := Model{width: 120, height: h, screen: sc, projectDir: "p", configFile: "commands.tsv"}
		m.projects = []string{"a", "b", "c", "d", "e", "f"}
		m.msItems = items
		m.msSearchTI = textinput.New()
		m.wfSearchTI = textinput.New()
		m.config = mdl.Config{Base: cmds}
		m.workflows = []mdl.Workflow{{Name: "wf", Commands: []string{"cmd-0"}}}
		names, refs := m.editableCommands()
		m.editRefs = refs
		m.setPickBase(names, m.commandPickTexts(refs))
		if sc == ScreenProjectSelect {
			m.listItems = m.projects
		}
		m.updateStepsViewport()
		return m
	}
	check := func(from int, screens map[string]Screen) {
		for h := from; h <= 40; h++ {
			for name, sc := range screens {
				if got := frameLines(build(h, sc).View()); got > h {
					t.Errorf("%s at height %d renders %d lines — the frame must not outgrow the terminal", name, h, got)
				}
			}
		}
	}
	check(7, map[string]Screen{
		"project select": ScreenProjectSelect,
		"run commands":   ScreenRunCommands,
		"run workflow":   ScreenRunWorkflow,
		"edit pick":      ScreenEditCommandPick,
	})

	// Fitting is not enough: a scrolling screen that stops one row short
	// wastes the bottom line of the terminal on nothing, which is what a
	// list row is worth most on a short one. The menus are exempt — they
	// have a fixed number of choices and nothing to fill with.
	for h := 7; h <= 40; h++ {
		for name, sc := range map[string]Screen{
			"project select": ScreenProjectSelect,
			"run commands":   ScreenRunCommands,
			"run workflow":   ScreenRunWorkflow,
			"edit pick":      ScreenEditCommandPick,
		} {
			if got := frameLines(build(h, sc).View()); got != h {
				t.Errorf("%s at height %d renders %d lines — the frame should fill the terminal", name, h, got)
			}
		}
	}
}

// TestSubmenusShowEveryChoice checks a submenu is shown whole rather
// than scrolled. Windowing a list of three fixed choices left one of
// them on screen between two "1 more" counts, which is not a menu.
func TestSubmenusShowEveryChoice(t *testing.T) {
	items := []string{"Create variable", "Edit variable", "Delete variable"}
	for h := 7; h <= 40; h++ {
		m := Model{width: 90, height: h, screen: ScreenManageVars}
		m.listItems = items
		m.listCursor = 1
		view := m.View()
		for _, it := range items {
			if !strings.Contains(view, it) {
				t.Fatalf("height %d: %q is missing from the submenu\n%s", h, it, view)
			}
		}
		if got := frameLines(view); got > h {
			t.Errorf("height %d: the submenu renders %d lines", h, got)
		}
	}
}

// frameLines is how many terminal lines a view occupies. View trims the
// frame's trailing newline — it would carry the cursor onto a further
// line and cost the frame the bottom row of the terminal — so the last
// line is not counted by the separators. A sub-view is measured the same
// way, since View is what finally trims it.
func frameLines(s string) int {
	return strings.Count(strings.TrimRight(s, "\n"), "\n") + 1
}

// TestBannersStayInsideTheTerminal checks the messages under the frame.
// They used to be appended after the frame had been built to the full
// terminal height, so the first error pushed the frame past the last row
// and the terminal scrolled the top of it away — worst on the Run workspace,
// where the load warnings can run to several lines.
func TestBannersStayInsideTheTerminal(t *testing.T) {
	screens := map[string]Screen{
		"project select": ScreenProjectSelect,
		"run commands":   ScreenRunCommands,
		"run workflow":   ScreenRunWorkflow,
		"edit pick":      ScreenEditCommandPick,
	}
	cmds := make([]mdl.Command, 8)
	var items []msItem
	for i := range cmds {
		cmds[i] = mdl.Command{Name: fmt.Sprintf("cmd-%d", i), Cmd: "echo hi", Source: "tsv"}
		items = append(items, msItem{cmd: &cmds[i]})
	}
	for h := 7; h <= 40; h++ {
		for name, sc := range screens {
			m := Model{width: 120, height: h, screen: sc, projectDir: "p", configFile: "commands.tsv"}
			m.projects = []string{"a", "b", "c", "d", "e", "f"}
			m.msItems = items
			m.msSearchTI = textinput.New()
			m.wfSearchTI = textinput.New()
			m.config = mdl.Config{Base: cmds}
			m.workflows = []mdl.Workflow{{Name: "wf", Commands: []string{"cmd-0"}}}
			names, refs := m.editableCommands()
			m.editRefs = refs
			m.setPickBase(names, m.commandPickTexts(refs))
			if sc == ScreenProjectSelect {
				m.listItems = m.projects
			}
			m.updateStepsViewport()
			m.errMsg = "something went wrong"
			m.successMsg = "saved"
			for i := 0; i < 9; i++ {
				m.loadWarnings = append(m.loadWarnings, fmt.Sprintf("warning %d", i))
			}
			view := m.View()
			if got := frameLines(view); got > h {
				t.Errorf("%s at height %d renders %d lines with a banner\n%s", name, h, got, view)
			}
			// The banner is trimmed to what is left over, so on a
			// terminal at a screen's own floor there is nothing left
			// and the message is dropped rather than shown over the
			// screen it belongs to. One row above that floor is enough.
			if h >= 8 && !strings.Contains(view, "something went wrong") {
				t.Errorf("%s at height %d: the error is not on screen\n%s", name, h, view)
			}
		}
	}
}

// TestConfirmRunFitsTheTerminal checks the screen shown before every run.
// It reserved one row for the button box, which is three rows drawn by
// lipgloss, so it ran past the bottom of the terminal at every size — 26
// lines on a standard 24-line one — and the terminal scrolled away the
// commands it exists to show.
func TestConfirmRunFitsTheTerminal(t *testing.T) {
	cmds := make([]mdl.Command, 6)
	for i := range cmds {
		cmds[i] = mdl.Command{Name: fmt.Sprintf("cmd-%d", i), Cmd: "echo hi", Source: "tsv"}
	}
	for h := 11; h <= 40; h++ {
		m := Model{width: 100, height: h, screen: ScreenConfirmRun, projectDir: "p"}
		m.config = mdl.Config{Base: cmds}
		for i := range cmds {
			m.confirmRunItems = append(m.confirmRunItems, mdl.RunItem{Name: cmds[i].Name, Cmd: &cmds[i]})
		}
		view := m.View()
		if got := frameLines(view); got > h {
			t.Errorf("height %d: the confirm screen renders %d lines\n%s", h, got, view)
		}
		// An item is worth showing only whole, so the screen shows as
		// many whole items as fit and no more.
		if !strings.Contains(view, "cmd-0") || !strings.Contains(view, "$ echo hi") {
			t.Errorf("height %d: the first command is not on screen\n%s", h, view)
		}
	}
}

// TestScrollCountsKeepTheirColumns: "↓ N more" stays in the same column
// whether or not "↑ N more" is shown beside it, so the hint does not
// jump sideways as the list scrolls past its first page.
func TestScrollCountsKeepTheirColumns(t *testing.T) {
	hint := func(cursor int) string {
		var b strings.Builder
		writeWindowedList(&b, 5, 30, cursor, "", func(i int) string { return fmt.Sprintf("row %d", i) })
		lines := strings.Split(strings.TrimRight(stripStyle(b.String()), "\n"), "\n")
		return lines[len(lines)-1]
	}
	top, mid := hint(0), hint(15)
	if !strings.Contains(top, "↓") || strings.Contains(top, "↑") {
		t.Fatalf("at the top only the down count should show: %q", top)
	}
	if !strings.Contains(mid, "↑") || !strings.Contains(mid, "↓") {
		t.Fatalf("mid-list both counts should show: %q", mid)
	}
	if len([]rune(top[:strings.Index(top, "↓")])) != len([]rune(mid[:strings.Index(mid, "↓")])) {
		t.Fatalf("the down count moved: %q vs %q", top, mid)
	}
}

// TestSelectedLineNamesTheQueueOrSaysHowManyAreHidden: on its own the
// Selected line lists the queue in full while it fits; when it does not,
// it shows the first names and how many more there are, never a bare
// ellipsis. Beside the queue box it is the count alone.
func TestSelectedLineNamesTheQueueOrSaysHowManyAreHidden(t *testing.T) {
	if got := queueNamesLine([]string{"build", "recent", "vet"}, 40); got != "build, recent, vet" {
		t.Fatalf("names that fit must be shown in full: %q", got)
	}
	if got := queueNamesLine([]string{"build", "recent", "vet", "version"}, 20); got != "build, recent +2 more" && got != "build +3 more" {
		t.Fatalf("names that do not fit must say how many are hidden: %q", got)
	}
	cmds := []mdl.Command{{Name: "build", Cmd: "go build"}, {Name: "recent", Cmd: "git log"}, {Name: "vet", Cmd: "go vet"}}
	m := Model{width: 120, height: 30}
	for i := range cmds {
		m.msItems = append(m.msItems, msItem{cmd: &cmds[i]})
	}
	m.msSelected = []int{0, 1, 2}
	alone := stripStyle(m.viewMultiSelectCandidates(80, true))
	if !strings.Contains(alone, "Selected(3)") || !strings.Contains(alone, "build, recent, vet") || strings.Contains(alone, "...") {
		t.Fatalf("alone, the line should name the whole queue:\n%s", alone)
	}
	beside := stripStyle(m.viewMultiSelectCandidates(60, false))
	if !strings.Contains(beside, "Selected(3)") || strings.Contains(beside, "Run queue →") {
		t.Fatalf("beside the queue box the line should be the count alone:\n%s", beside)
	}
}
