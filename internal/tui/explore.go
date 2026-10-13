package tui

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	mdl "github.com/Taka-S-dev/baton/internal/model"
	"github.com/Taka-S-dev/baton/internal/slot"
)

// The overview is a workspace in three parts: a tab picks what kind of
// thing is listed, the list on the left picks one, and the right side
// is where it is edited (above) and where what it will actually do is
// read back (below). Each tab holds one kind of thing, so a row in the
// list is always something you can act on, and the references between
// kinds — which workflow runs a command, which command draws on a list
// — are read in the preview rather than by seeing two lists at once.

// The tabs, in the order they are numbered on screen.
const (
	tabWorkflows = iota
	tabCommands
	tabLists
	tabVars
	tabCount
)

var tabNames = [tabCount]string{"Workflows", "Commands", "Lists", "Variables"}

// exploreKind is what a list row stands for.
type exploreKind int

const (
	exCommand exploreKind = iota
	exWorkflow
	exList
	exVar
)

// wsItem is one row of the list on the left.
type wsItem struct {
	group       string
	commandType string
	template    string
	kind        exploreKind
	name        string // what the row stands for: a command, workflow, list or var name
	label       string // the name plus its tag, as drawn
	note        string // trailing detail, dimmed
	draft       bool   // the row the open form is writing, not saved yet
}

// exploreItems lists what the given tab holds, before any search.
func (m Model) exploreItems(tab int) []wsItem {
	switch tab {
	case tabCommands:
		return m.exploreCommandItems()
	case tabWorkflows:
		return m.exploreWorkflowItems()
	case tabLists:
		return m.exploreListItems()
	case tabVars:
		return m.exploreVarItems()
	}
	return nil
}

// exploreVisible is the list as shown: the tab's items, narrowed by the
// search typed over them.
func (m Model) exploreVisible() []wsItem {
	items := m.exploreItems(m.exploreTab)
	terms := strings.Fields(strings.ToLower(m.exploreSearch))
	if len(terms) == 0 {
		return m.withValueDraft(items)
	}
	var out []wsItem
	for _, it := range items {
		if matchesAllTerms(strings.ToLower(it.label+" "+it.note), terms) {
			out = append(out, it)
		}
	}
	return m.withValueDraft(out)
}

// exploreCommandItems lists every command, grouped by their group
// column. A group is a label that nothing references, so it is a tag on
// the row rather than a level of its own; the order keeps a group's
// commands together, which is what a level would have been for.
func (m Model) exploreCommandItems() []wsItem {
	cmds := m.exploreDraftCommands()
	sort.SliceStable(cmds, func(a, b int) bool {
		if cmds[a].Group != cmds[b].Group {
			// Grouped commands first, in group order; the ungrouped
			// ones fall to the end rather than under an empty heading.
			if cmds[a].Group == "" || cmds[b].Group == "" {
				return cmds[b].Group == ""
			}
			return cmds[a].Group < cmds[b].Group
		}
		return false
	})
	out := make([]wsItem, 0, len(cmds))
	for _, c := range cmds {
		label := c.Name
		if c.Group != "" {
			label += "  [" + c.Group + "]"
		}
		switch {
		case c.Template != "":
			label += "  (" + c.Template + ")"
		case slot.HasPlaceholders(c):
			label += "  {...}"
		}
		out = append(out, wsItem{
			kind:        exCommand,
			name:        c.Name,
			label:       label,
			note:        "$ " + slot.ApplyVars(c.Cmd, m.vars),
			draft:       c.Source == draftSource,
			group:       c.Group,
			commandType: commandKind(c),
			template:    c.Template,
		})
	}
	return out
}

func (m Model) exploreWorkflowItems() []wsItem {
	out := make([]wsItem, 0, len(m.workflows))
	for _, wf := range m.workflows {
		missing := 0
		for _, step := range wf.Commands {
			if _, ok := m.workflowStepCommand(step); !ok {
				missing++
			}
		}
		note := countLabel(len(wf.Commands), "step")
		if missing > 0 {
			note += fmt.Sprintf(", %d missing", missing)
		}
		out = append(out, wsItem{kind: exWorkflow, name: wf.Name, label: wf.Name, note: note})
	}
	return out
}

func (m Model) exploreListItems() []wsItem {
	names := m.sortedListNames()
	out := make([]wsItem, 0, len(names))
	for _, n := range names {
		out = append(out, wsItem{kind: exList, name: n, label: n, note: countLabel(len(m.lists[n]), "entry")})
	}
	return out
}

// exploreVarItems lists globals first, then the fixed values baton wrote
// for saved commands. Both are rows of vars.tsv; what a row is is said
// on the row.
func (m Model) exploreVarItems() []wsItem {
	var globals []wsItem
	for _, k := range sortedKeys(m.vars) {
		it := wsItem{kind: exVar, name: k, note: m.vars[k]}
		if strings.Contains(k, ".") {
			it.label = "Fixed · " + fixedValueLabel(k)
			// Fixed values are edited with their owning command.
		} else {
			it.label = "Global · {$" + k + "}"
			globals = append(globals, it)
		}
	}
	return globals
}

// exploreDraftCommands is the command list as the overview should show
// it while the form is open: the row being written stands in the list
// with whatever has been typed so far, so where it lands — which group
// it sorts in with — is visible before it is saved.
func (m Model) exploreDraftCommands() []mdl.Command {
	cmds := m.config.AllCommands()
	if m.cf == nil {
		return cmds
	}
	f := m.cf.fields
	f[m.cf.fieldIdx] = strings.TrimSpace(m.nameInput.Value())
	// Source is display-only and never written, so it can carry the
	// one bit the list needs: which row is the draft. Matching by name
	// would mark a saved command too, the moment the draft is given a
	// name already in use — the very case the form is about to refuse.
	draft := mdl.Command{Name: f[0], Cmd: f[1], Dir: f[2], Group: f[3], Shell: f[4], Source: draftSource}
	if draft.Name == "" {
		draft.Name = draftPlaceholderName
	}
	if m.cf.mode == 1 {
		for i, c := range cmds {
			if c.Name == m.cf.origName {
				draft.Template, draft.Values, draft.Slots = c.Template, c.Values, c.Slots
				cmds[i] = draft
				return cmds
			}
		}
	}
	return append(cmds, draft)
}

// draftPlaceholderName stands in until the name field has something in
// it, so the row still has somewhere to sit in the list; draftSource is
// how the list tells the draft from a saved command of the same name.
const (
	draftPlaceholderName = "(unnamed)"
	draftSource          = "draft"
)

// exploreCurrent is the item under the cursor, if the list has one.
func (m Model) exploreCurrent() (wsItem, bool) {
	items := m.exploreVisible()
	i := m.exploreCursor[m.exploreTab]
	if i < 0 || i >= len(items) {
		return wsItem{}, false
	}
	return items[i], true
}

// commandRefLocations lists what would break if the named command went
// away: workflow steps, derived commands, and its rows in vars.tsv. The
// same walk drives the rename repair, which only ever reported counts —
// here it is something you can read before you touch anything.
func (m Model) commandRefLocations(name string) []string {
	var out []string
	for _, wf := range m.workflows {
		for i, step := range wf.Commands {
			if step == name {
				out = append(out, fmt.Sprintf("workflow %q step %d", wf.Name, i+1))
			}
		}
	}
	for _, c := range m.config.AllCommands() {
		if c.Template == name {
			out = append(out, fmt.Sprintf("derived command %q", c.Name))
		}
	}
	n := 0
	for _, k := range sortedKeys(m.vars) {
		if strings.HasPrefix(k, name+".") {
			n++
		}
	}
	if n > 0 {
		out = append(out, fmt.Sprintf("%s in vars.tsv", countLabel(n, "fixed value")))
	}
	return out
}

// listRefLocations lists the commands whose placeholders draw on the
// named list, which is what deleting or renaming it would break.
func (m Model) listRefLocations(name string) []string {
	var out []string
	for _, c := range m.config.AllCommands() {
		for _, d := range slot.GetSlots(c) {
			if d.ListName == name {
				out = append(out, fmt.Sprintf("command %q, %s", c.Name, d.Placeholder()))
			}
		}
	}
	return out
}

// countLabel renders "1 step" / "3 steps" without the caller spelling
// out the plural each time.
func countLabel(n int, unit string) string {
	if n == 1 {
		return "1 " + unit
	}
	if strings.HasSuffix(unit, "y") {
		return fmt.Sprintf("%d %sies", n, strings.TrimSuffix(unit, "y"))
	}
	return fmt.Sprintf("%d %ss", n, unit)
}

// exploreTitle says what the overview is doing, not only where you are:
// with the form open on the right, the list still fills the left, and
// nothing else would say a command is half-written.
func (m Model) exploreTitle() string {
	title := "Project editor"
	if m.projectDir != "" {
		title += " · " + filepath.Base(filepath.Clean(m.projectDir))
	}
	if m.cf == nil {
		return title
	}
	if m.cf.mode == 1 {
		return title + " › Edit " + m.cf.origName
	}
	return title + " › New command"
}

// exploreLayout is where the overview's parts land on screen. The view
// draws from it and the pointer reads it back — a click has to find the
// row the view put under it, and two copies of the same arithmetic
// would drift apart the first time either one changed.
type exploreLayout struct {
	tabY   int              // screen row of the tab line
	tabX   [tabCount][2]int // columns each tab label spans, inclusive-exclusive
	bodyY  int              // first screen row of the boxes
	bodyH  int              // rows the boxes have between them
	plain  bool             // too short even for a box: bare rows
	single bool             // too narrow or short for the right side

	leftW  int // outer width of the list box (the whole width when single)
	listY  int // screen row of the first list row
	addY   int // fixed create action row; -1 when too short
	listH  int // rows the list has for items
	rightX int // column the right boxes start at
	rightW int
	editY  int // screen row of the editor box's top edge
	editH  int
	fieldY int // screen row the form's first field is drawn on
	prevY  int
	prevH  int
	items  []wsItem
	cursor int
	start  int
}

// exploreMinList is the narrowest the list box is allowed to get, and
// exploreMinRight the narrowest right side worth drawing; below that
// the list has the width to itself.
const (
	exploreMinList  = 24
	exploreMinRight = 40
)

func (m Model) exploreLayout(w int) exploreLayout {
	// View budgets the frame against the terminal less the banner, and
	// sets that on its own copy of the model; the same has to happen
	// here or a click lands a row off whenever a message is showing.
	m.bannerRows = len(m.bannerLines())

	var b strings.Builder
	m.writeHeading(&b, m.exploreTitle(), w)
	l := exploreLayout{addY: -1, tabY: strings.Count(b.String(), "\n")}

	gap := 1
	if m.compact() {
		gap = 0
	}
	l.bodyY = l.tabY + 1 + gap
	l.bodyH = m.termHeight() - l.bodyY - 2 - gap
	l.plain = l.bodyH < 5
	l.single = w < exploreMinList+exploreMinRight+1 || l.bodyH < 9

	// The tab labels, so a click on one can be told from a click beside it.
	labels, sep := m.tabPresentation(w)
	x := len(sep)
	for i, label := range labels {
		if i == 2 && sep == "  " {
			x += 3
		}
		l.tabX[i] = [2]int{x, x + len(label)}
		x += len(label) + len(sep)
	}

	l.leftW = w
	if !l.single {
		want := m.exploreSplit
		if want == 0 {
			want = w * 28 / 100
		}
		l.leftW = min(max(want, exploreMinList), w-exploreMinRight-1)
		l.rightX = l.leftW + 1
		l.rightW = w - l.rightX
		// The editor holds five fields and a line or two under them; the
		// preview holds the resolved command, what points at it and where
		// it is defined, which is the longer of the two. The split
		// follows that, with the editor never cut below its fields.
		l.editY = l.bodyY
		l.editH = max(9, l.bodyH*40/100)
		if m.cf == nil && m.screen == ScreenExplore && m.exploreTab == tabCommands {
			it, _ := m.exploreCurrent()
			if cmd, ok := m.config.FindCommand(it.name); ok {
				need := 10
				if cmd.Template != "" {
					need += 1 + len(cmd.Values)
				}
				l.editH = max(l.editH, min(l.bodyH-3, need))
			}
		}
		if m.cf != nil && m.cf.shellOpen {
			l.editH = max(l.editH, min(l.bodyH, 10))
		}
		if m.cf != nil && m.cf.pickOpen {
			// The candidates open under the fields and need rows of
			// their own; the preview has nothing to say about a brace
			// mid-typing, so it gives them up while the popup is open.
			l.editH = min(l.bodyH-3, 5+2+pickRows+2)
		}
		if m.wfe != nil {
			// The steps are the thing being worked on, so the box holds
			// as many of them as it can before the preview has its say.
			l.editH = max(5, min(l.bodyH/2, len(m.wfe.steps)+5))
		}
		if m.cf == nil && m.wfe == nil && len(m.exploreVisible()) == 0 {
			needed := strings.Count(categoryHelpContent(m.exploreTab, l.rightW-2), "\n") + 3
			l.editH = min(l.bodyH, max(l.editH, needed))
		}
		l.prevH = l.bodyH - l.editH
		if l.prevH < 3 {
			l.editH, l.prevH = l.bodyH, 0
		}
		l.prevY = l.editY + l.editH
		l.fieldY = l.editY + 1
	}

	// Inside the list box: the search line, the rows, and one line
	// carrying what is scrolled off. Too short for a box, it is rows.
	if l.plain {
		l.listY = l.bodyY
		l.listH = max(1, l.bodyH)
	} else {
		l.listY = l.bodyY + 2
		l.listH = max(1, l.bodyH-3)
		if m.exploreTab == tabCommands && l.bodyH >= 6 {
			l.listY++
			l.listH--
		}
	}

	if !l.plain && l.bodyH >= 7 {
		l.addY = l.bodyY + l.bodyH - 2
		l.listH--
	}

	l.items = m.exploreVisible()
	l.cursor = min(max(0, m.exploreCursor[m.exploreTab]), max(0, len(l.items)-1))
	for i, it := range l.items {
		if it.draft && (m.exploreTab == tabLists || m.exploreTab == tabVars) {
			l.cursor = i
		}
	}
	l.start = windowStart(l.listH, len(l.items), l.cursor)
	if m.exploreScrollManual[m.exploreTab] {
		l.start = min(max(0, m.exploreScroll[m.exploreTab]), max(0, len(l.items)-l.listH))
	}
	return l
}

// viewWidth is the width the view draws at, including the fallback it
// uses before the first size message arrives.
func (m Model) viewWidth() int {
	if m.width == 0 {
		return 80
	}
	return m.width
}
