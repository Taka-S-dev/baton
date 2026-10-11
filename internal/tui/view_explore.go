package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	mdl "github.com/Taka-S-dev/baton/internal/model"
	"github.com/Taka-S-dev/baton/internal/slot"
)

// ── Overview ─────────────────────────────────────────────────────────────────

// viewExplore draws the workspace: the tabs, the list of what the open
// tab holds, and beside it the editor box and the preview box. Every
// box is cut to the rows the layout gave it, so the frame never outgrows
// the terminal; a terminal too narrow or short for the right side gets
// the list on its own, and editing takes the screen.
func (m Model) viewExplore(w int) string {
	l := m.exploreLayout(w)

	var b strings.Builder
	m.writeHeading(&b, m.exploreTitle(), w)
	b.WriteString(m.exploreTabLine() + "\n")
	gap := "\n"
	if m.compact() {
		gap = ""
	}
	b.WriteString(gap)

	left := m.exploreListPane(l)
	if l.single {

		b.WriteString(left)
	} else {
		right := m.exploreEditorBox(l)
		if l.prevH > 0 {
			right = lipgloss.JoinVertical(lipgloss.Left, right, m.explorePreviewBox(l))
		}
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, left, " ", right))
	}
	b.WriteString("\n")

	b.WriteString(gap + hline(w) + "\n")
	b.WriteString(gray(m.compactFooter(w)) + "\n")
	view := b.String()
	if m.exploreMenu {
		view = m.overlayExploreActions(view, w)
	}
	return view
}

// exploreTabLine draws the tabs. The columns each label spans are what
// exploreLayout records for clicks, so the spacing here and there is the
// same arithmetic: two leading columns, two between labels.
func (m Model) exploreTabLine() string {
	labels, sep := m.tabPresentation(m.viewWidth())
	parts := make([]string, len(labels))
	for i, label := range labels {
		if i == m.exploreTab {
			parts[i] = accentBold(label)
		} else {
			parts[i] = gray(label)
		}
	}
	if sep == "  " {
		return "  " + strings.Join(parts[:2], sep) + "  │  " + strings.Join(parts[2:], sep)
	}
	return " " + strings.Join(parts, sep)
}

// exploreKeys is the key line, for the state the screen is in.
func (m Model) exploreKeys() string {
	if m.exploreTemplateFlow() {
		return m.templatePaneKeys()
	}
	if m.exploreValueFlow() {
		return m.valuePaneKeys()
	}
	if m.leaveAsk != leaveNone {
		return "↑↓ Choose  Enter Confirm  s Save  d Discard  Esc Keep editing"
	}
	if m.cf != nil {
		return m.formKeys()
	}
	if m.wfe != nil {
		if m.wfe.adding {
			return "Type to search  ↑↓ Choose  Enter Add after step  Esc Cancel"
		}
		return "↑↓ Step  J/K Move  a Add  x Remove  Enter Open command  Ctrl+S Save  Esc Back"
	}
	if m.exploreSearching {
		if m.exploreSearch == "" {
			return "Type to search  ↓ Results  Enter Open  Esc Done"
		}
		return "Type to search  ↓ Results  Enter Open  Esc Clear"
	}
	if m.exploreMenu {
		return "↑↓ Choose  Enter Do it  Esc Close"
	}
	keys := "↑↓ Select  Enter Open  Space Actions  / Search  1-4 Tab"
	keys += "  a Add"
	if m.exploreLayout(m.viewWidth()).single {
		keys += "  v Details"
	}
	if m.exploreTab == tabLists || m.exploreTab == tabVars {
		keys += "  Del/x Delete"
	}
	if len(m.exploreHistory) > 0 {
		return keys + "  Esc Return"
	}
	return keys + "  Esc Back"
}

// exploreListFocused reports whether navigation keys belong to the list.
func (m Model) exploreListFocused() bool {
	return m.cf == nil && m.wfe == nil && !m.exploreMenu && m.leaveAsk == leaveNone && !m.exploreValueFlow() && !m.exploreTemplateFlow()
}

// exploreListPane draws the list box: the search line, the rows, and
// one line carrying what is scrolled off. Too short for a box, it is the
// rows alone.
func (m Model) exploreListPane(l exploreLayout) string {
	inner := l.leftW
	if !l.plain {
		inner = l.leftW - 2
	}
	var rows strings.Builder
	writeWindowedList(&rows, l.listH, len(l.items), l.cursor, "(none)", func(i int) string {
		if m.exploreTab == tabCommands {
			return searchHighlight(commandTableRow(l.items[i], i == l.cursor, m.exploreListFocused(), inner), strings.Fields(m.exploreSearch), i == l.cursor, m.exploreListFocused())
		}
		return searchHighlight(exploreRow(l.items[i], i == l.cursor, m.exploreListFocused(), inner), strings.Fields(m.exploreSearch), i == l.cursor, m.exploreListFocused())
	})
	body := strings.TrimRight(rows.String(), "\n")
	if l.plain {
		return body
	}

	all := len(m.exploreItems(m.exploreTab))
	title := fmt.Sprintf("%s (%d)", tabNames[m.exploreTab], all)
	if m.exploreSearch != "" {
		title = fmt.Sprintf("%s (%d/%d)", tabNames[m.exploreTab], len(l.items), all)
	}
	if m.exploreTab == tabCommands && l.bodyH >= 6 {
		body = gray(commandTableLine(wsItem{name: "Name", group: "Group", commandType: "Type", note: "Command"}, inner)) + "\n" + body
	}
	content := m.exploreSearchLine() + "\n" + body
	if l.addY >= 0 {
		lines := strings.Split(content, "\n")
		for len(lines) < l.bodyH-3 {
			lines = append(lines, "")
		}
		label := " + Add " + []string{"workflow", "command", "list", "global variable"}[m.exploreTab]
		if m.exploreListFocused() {
			label = accent(label + "  a")
		} else {
			label = dim(label)
		}
		content = strings.Join(lines, "\n") + "\n" + label
	}
	return box(title, content, l.leftW, l.bodyH, m.exploreListFocused())
}

// exploreSearchLine is the first line inside the list box.
func (m Model) exploreSearchLine() string {
	return " " + searchInputView(m.exploreSearchTI, m.exploreSearch, m.exploreSearching)
}

// exploreRow renders one list row. The selected one is a bar across the
// box: a marker says which row, a bar says which row and how far the
// box reaches, and it dims when the editor beside it has the keys.
//
// A selected row is built as plain text and styled once. Styling the
// pieces first and the bar over them does not work: the reset at the end
// of an inner style takes the background with it.
func exploreRow(it wsItem, selected, focused bool, w int) string {
	// One column of margin inside the edge, so the text does not sit
	// on the border; the bar still reaches from edge to edge.
	marker := "   "
	if selected {
		marker = " ▶ "
	}
	avail := max(4, w-len(marker))
	text := truncate(it.label, avail)
	note := ""
	if it.note != "" {
		if spare := avail - lipgloss.Width(text) - 2; spare >= 8 {
			note = "  " + truncate(it.note, spare)
		}
	}
	if selected {
		bar := sRowInactive
		if focused {
			bar = sRowActive
		}
		plain := marker + text + note
		return bar.Render(plain + strings.Repeat(" ", max(0, w-lipgloss.Width(plain))))
	}
	label := text
	if it.draft {
		// Gold is what baton already uses for the thing in play.
		label = highlight(text)
	}
	if note != "" {
		note = dim(note)
	}
	return marker + label + note
}

// exploreEditorBox is the box above the preview: the thing under the
// cursor as it is defined, and the form once one is open.
func (m Model) exploreEditorBox(l exploreLayout) string {
	if m.exploreTemplateFlow() {
		return m.exploreTemplateBox(l)
	}
	if m.exploreValueFlow() {
		return m.exploreValueBox(l)
	}
	inner := l.rightW - 2
	it, ok := wsItem{}, false
	if l.cursor < len(l.items) {
		it, ok = l.items[l.cursor], true
	}
	var b strings.Builder
	title := tabNames[m.exploreTab]
	switch {
	case m.leaveAsk != leaveNone:
		// Leaving with changes unsaved is asked about in the box the
		// changes are in, with the three answers and their keys.
		m.writeLeaveAsk(&b)
		return box("Unsaved changes", b.String(), l.rightW, l.editH, true)
	case m.wfe != nil && m.wfe.idx < len(m.workflows):
		title = "Edit workflow · " + m.workflows[m.wfe.idx].Name
		if m.wfe.dirty {
			title += "  ● unsaved"
		}
		if m.wfe.adding {
			title = "Add a step · " + m.workflows[m.wfe.idx].Name
		}
		if m.wfe.adding {
			m.writeWorkflowEditor(&b, inner, l.editH-2)
		} else {
			m.writeWorkflowEditor(&b, inner, l.editH-3)
			content := strings.Split(strings.TrimRight(b.String(), "\n"), "\n")
			for len(content) < l.editH-3 {
				content = append(content, "")
			}
			b.Reset()
			b.WriteString(strings.Join(content, "\n") + "\n")
			hint := "first step"
			if len(m.wfe.steps) > 0 {
				hint = fmt.Sprintf("after step %d", m.wfe.cursor+1)
			}
			b.WriteString(accent(" + Add step  a") + dim(" · "+hint))
		}
		return box(title, b.String(), l.rightW, l.editH, true)
	case m.exploreTab == tabCommands && m.cf != nil:
		title = "New command"
		if m.cf.mode == 1 {
			title = "Edit command · " + m.cf.origName
		}
		if m.formDirty() {
			title += "  ● unsaved"
		}
		m.writeCommandFields(&b, m.cf.fields, m.cf.fieldIdx, inner)
		if m.cf.pickOpen {
			// The fields, a blank line, the candidates, and the line
			// counting what is off the window.
			m.writePickCandidates(&b, m.cf, inner-2, l.editH-2-5-2)
		} else {
			for _, line := range m.slotValidationLines(m.cf) {
				b.WriteString(" " + line + "\n")
			}
		}
	case !ok:
		// A name is no help for a kind of thing you have not met, and a
		// project with none of it is exactly when you need to know.
		what, how := categoryHelp(m.exploreTab)
		b.WriteString(" " + gray(what) + "\n\n " + dim("none yet · "+how) + "\n")
	case it.kind == exCommand:
		title = "Command · " + it.name
		cmd, _ := m.config.FindCommand(it.name)
		m.writeCommandFields(&b, commandFields(cmd), -1, inner)
		m.writeCommandIdentity(&b, cmd, inner)
		b.WriteString("\n " + dim("Enter to edit") + "\n")
	case it.kind == exWorkflow:
		title = "Workflow · " + it.name
		m.writeWorkflowSteps(&b, it.name, false, inner)
	case it.kind == exList:
		title = "List · " + it.name
		for _, e := range m.lists[it.name] {
			line := " " + e.Value
			if e.Label != "" {
				line += "  " + dim("· "+e.Label)
			}
			b.WriteString(line + "\n")
		}
	case it.kind == exVar:
		title = "Global variable · " + it.name
		if _, _, scoped := scopedKey(it.name); scoped {
			title = "Fixed value · " + it.name
		}
		m.writeVarDefinition(&b, it.name)
	}
	return box(title, b.String(), l.rightW, l.editH, m.cf != nil)
}

// explorePreviewBox is the box under the editor: what the thing under
// the cursor will actually do, and what points at it.
func (m Model) explorePreviewBox(l exploreLayout) string {
	inner := l.rightW - 2
	it, ok := wsItem{}, false
	if l.cursor < len(l.items) {
		it, ok = l.items[l.cursor], true
	}
	draft, valueDraft := m.newValueDraft()
	var b strings.Builder
	switch {
	case m.exploreTemplateFlow():
		m.writeTemplatePreview(&b)
	case valueDraft:
		label := "New list"
		if draft.kind == exVar {
			label = "New global variable"
		}
		b.WriteString(" " + white(label) + " · " + highlight("unsaved") + "\n")
		b.WriteString(" " + gray("name  ") + displayOr(draft.name, "(unnamed)") + "\n")
		if draft.kind == exVar {
			b.WriteString(" " + gray("value ") + draft.note + "\n")
		}
	case m.exploreValueFlow() && m.screen == ScreenEditList:
		return m.listValueDetailBox(l)
	case m.wfe != nil:
		return m.workflowDetailBox(l)
	case m.exploreTab == tabCommands && m.cf != nil:
		f := m.cf.fields
		f[m.cf.fieldIdx] = strings.TrimSpace(m.nameInput.Value())
		draft := mdl.Command{Name: f[0], Cmd: f[1], Dir: f[2], Group: f[3], Shell: f[4]}
		if draft.Cmd != "" {
			m.writeResolved(&b, draft)
		}
		if m.cf.mode == 1 {
			m.writeRefList(&b, "used by", m.commandRefLocations(m.cf.origName), inner)
		}
	case !ok:
	case it.kind == exCommand:
		cmd, _ := m.config.FindCommand(it.name)
		m.writeResolved(&b, cmd)
		m.writeRefList(&b, "used by", m.commandRefLocations(it.name), inner)
		m.writeCommandFacts(&b, cmd, inner)
	case it.kind == exWorkflow:
		m.writeWorkflowSteps(&b, it.name, true, inner)
	case it.kind == exList:
		m.writeRefList(&b, "Commands using list: "+it.name, m.listRefLocations(it.name), inner)
		return box("List usage", b.String(), l.rightW, l.prevH, false)
	case it.kind == exVar:
		if cmdName, _, scoped := scopedKey(it.name); scoped {
			b.WriteString(" " + gray("fixed value of command ") + white(cmdName) + "\n")
		} else {
			m.writeRefList(&b, "referenced by", m.varRefLocations(it.name), inner)
		}
	}
	return box("Preview", b.String(), l.rightW, l.prevH, false)
}

// writeResolved writes what a command will actually run: its workdir
// and its cmd with every {$var} substituted. A {slot} stays as it is,
// since it is answered at run time.
func (m Model) writeResolved(b *strings.Builder, cmd mdl.Command) {
	b.WriteString(" " + dim("workdir  ") + m.previewVars(displayOr(cmd.Dir, ".")) + "\n")
	b.WriteString(" " + gray("$ ") + m.previewVars(cmd.Cmd) + "\n")
	m.writeSyntaxWarnings(b, cmd.Dir+" "+cmd.Cmd)
}

// writeWorkflowSteps lists a workflow's steps: by name in the editor,
// and as what each will actually run in the preview.
func (m Model) writeWorkflowSteps(b *strings.Builder, name string, resolved bool, w int) {
	for _, wf := range m.workflows {
		if wf.Name != name {
			continue
		}
		for i, step := range wf.Commands {
			cmd, ok := m.workflowStepCommand(step)
			switch {
			case !ok:
				b.WriteString(fmt.Sprintf(" %2d  %s  %s\n", i+1, step, warn("(not found)")))
			case resolved:
				r := slot.ApplyVarsToCommand(cmd, m.vars)
				b.WriteString(fmt.Sprintf(" %2d  %s\n", i+1, gray("$ ")+truncate(r.Cmd, max(8, w-7))))
			default:
				b.WriteString(fmt.Sprintf(" %2d  %s\n", i+1, step))
			}
		}
		return
	}
}

// writeVarDefinition writes a variable as it is defined: a global as
// {$name} = value, a fixed value as the command and slot it belongs to.
func (m Model) writeVarDefinition(b *strings.Builder, key string) {
	cmdName, slotName, scoped := scopedKey(key)
	if !scoped {
		b.WriteString(" " + gray("Global variable · shared across commands") + "\n")
		b.WriteString(" " + m.syntaxText("{$"+key+"}") + " = " + m.vars[key] + "\n")
		return
	}
	b.WriteString(" " + gray("Fixed value · saved for this command") + "\n")
	b.WriteString(" " + dim("Deleting this value restores the prompt at run time.") + "\n")
	b.WriteString(" " + gray("command  ") + cmdName + "\n")
	b.WriteString(" " + gray("slot     ") + slotName + "\n")
	b.WriteString(" " + gray("value    ") + m.vars[key] + "\n")
	if r := slot.ApplyVars(m.vars[key], m.vars); r != m.vars[key] {
		b.WriteString(" " + gray("resolves ") + r + "\n")
	}
}

// writeRefList prints what points at the selected item, which is the one
// thing the TUI never showed: the reference walk existed only to report a
// rename. Seeing it is what makes an edit or a delete safe to attempt.
func (m Model) writeRefList(b *strings.Builder, label string, refs []string, w int) {
	b.WriteString("\n" + hlineLabel(w, label) + "\n")
	if len(refs) == 0 {
		b.WriteString("  " + dim("nothing") + "\n")
		return
	}
	for _, r := range refs {
		b.WriteString("  " + gray(truncate(r, max(10, w-4))) + "\n")
	}
}

// writeCommandFacts says which file a command is defined in and whether
// this screen can change it. That is what the file decides — a
// commands.json entry runs like any other but has to be edited by hand
// — and the only way to find out was to try and be told no.
func (m Model) writeCommandFacts(b *strings.Builder, cmd mdl.Command, w int) {
	file, editable := "commands.tsv", true
	switch cmd.Source {
	case "json":
		file, editable = "commands.json", false
	case "local":
		file = "commands.local.json"
	}
	b.WriteString("\n" + hlineLabel(w, "defined in") + "\n")
	note := gray("edit by hand")
	if editable {
		note = gray("editable here")
	}
	b.WriteString("  " + white(file) + "  " + dim("·") + "  " + note + "\n")
}

// commandFormPaneLabels are the form's labels without their hints: the
// editor box is a fraction of the terminal wide, and "Shell (ps =
// PowerShell, leave empty for cmd)" would spend all of it on a label.
var commandFormPaneLabels = [5]string{"Name", "Cmd", "Workdir", "Group", "Shell"}

// writeCommandFields draws the five fields of a command in the order
// the form asks for them. Browsing and editing draw the same rows in
// the same places — the box is not a description that turns into a
// form, it is the form, with or without the caret in it. That is what
// lets a click on a field mean "edit this one".
func (m Model) writeCommandFields(b *strings.Builder, fields [5]string, focused, w int) {
	editing := focused >= 0 && m.cf != nil
	for _, i := range formOrder {
		label := commandFormPaneLabels[i]
		// A field that no longer holds what was opened is marked, so
		// what Ctrl+S would change is readable before it is pressed.
		mark := " "
		if editing {
			now := fields[i]
			if i == focused {
				now = strings.TrimSpace(m.nameInput.Value())
			}
			if now != m.cf.original[i] {
				mark = highlight("●")
			}
		}
		if i == 4 {
			value := fields[4]
			if i == focused {
				value = m.nameInput.Value()
			}
			label := gray("Shell > ")
			if i == focused {
				label = accent("Shell > ")
			}
			description := shellChoiceLabel(value)
			if editing && m.cf.shellRemembered && value == m.cf.original[4] {
				description += " (last used)"
			}
			b.WriteString(mark + label + description + "\n")
			if i == focused && m.cf.shellOpen && m.leaveAsk == leaveNone {
				m.writeShellChoices(b)
			}
		} else if i == focused {
			// Only the prompt is set here. The width belongs to the
			// moment the field is focused, because that is when the
			// textinput decides which part of a long value to show.
			m.nameInput.Prompt = label + " > "
			b.WriteString(mark + m.nameInput.View() + "\n")
		} else {
			value := dim("-")
			if fields[i] != "" {
				value = fields[i]
				if i == 1 || i == 2 {
					value = m.syntaxText(value)
				}
				value = ansi.Truncate(value, max(8, w-len(label)-4), "…")
			}
			b.WriteString(mark + gray(label+" > ") + value + "\n")
		}
		// What is wrong with a field is said under that field, and
		// stays there until the field is typed in.
		if editing && m.cf.fieldErr != "" && m.cf.errField == i {
			b.WriteString("   " + errorText("⚠ "+m.cf.fieldErr) + "\n")
		}
	}
}

// commandFields is a command as the five form fields.
func commandFields(c mdl.Command) [5]string {
	return [5]string{c.Name, c.Cmd, c.Dir, c.Group, c.Shell}
}

// categoryHelp says what a tab holds and how to get one. The list can
// only show names, and a name is no help for a kind of thing you have
// not met: "Lists (0)" told a newcomer nothing at all.
func categoryHelp(tab int) (what, how string) {
	switch tab {
	case tabWorkflows:
		return "Commands in a fixed order, run as one.", "press a to create a workflow"
	case tabCommands:
		return "The things baton runs.", "press a"
	case tabLists:
		return "The choices a {slot} offers when a command asks for one.", "press a to create a list (example: env with dev / prod)"
	case tabVars:
		return "Shared constants referenced as {$name}. Command-specific fixed values are edited in Commands.", "press a to create a global variable (example: root = C:/work)"
	}
	return "", ""
}

// box draws content inside a rounded border of exactly w by h cells,
// with the title set into the top edge. Content is cut to fit rather
// than allowed to grow: a box that stretches to its content is a frame
// that outgrows the terminal. The edge is bright on the box that has
// the keys, which is how a glance tells which side is being driven.
func box(title, content string, w, h int, focused bool) string {
	if w < 4 || h < 2 {
		return ""
	}
	inner, rows := w-2, h-2
	var lines []string
	if content = strings.TrimRight(content, "\n"); content != "" {
		lines = strings.Split(content, "\n")
	}
	if len(lines) > rows {
		lines = lines[:rows]
	}
	for len(lines) < rows {
		lines = append(lines, "")
	}

	edge := gray
	if focused {
		edge = accent
	}
	t := truncate(title, max(0, inner-4))
	fill := max(0, inner-3-lipgloss.Width(t))
	var b strings.Builder
	b.WriteString(edge("╭─ ") + white(t) + edge(" "+strings.Repeat("─", fill)+"╮") + "\n")
	for _, line := range lines {
		line = ansi.Truncate(line, inner, "")
		pad := strings.Repeat(" ", max(0, inner-lipgloss.Width(line)))
		b.WriteString(edge("│") + line + pad + edge("│") + "\n")
	}
	b.WriteString(edge("╰" + strings.Repeat("─", inner) + "╯"))
	return b.String()
}

// formKeys is the key line while the form has the caret. Tab is the
// next field everywhere; the one key that changes with the field is the
// arrow that takes the suggested name, and it is shown only when there
// is one to take.
func (m Model) formKeys() string {
	if m.cf.fieldIdx == 4 {
		if m.cf.shellOpen {
			return "↑↓ Select shell  Enter Confirm  Tab/Shift+Tab Field  Esc Close"
		}
		return "Enter Choose shell  Tab/Shift+Tab Field  Ctrl+S Save  Esc Back to list"
	}
	keys := "Tab/Shift+Tab Field  Enter Next  Ctrl+S Save  Esc Back"
	if m.cf.returnTo == ScreenExplore {
		keys = "Tab/Shift+Tab Field  Enter Next  Ctrl+S Save  Esc Back to list"
	}
	switch {
	case m.cf.pickOpen:
		return "↑↓ Choose  Enter Insert  Esc Close  Type to narrow"
	case m.cf.fieldIdx == 0 && m.nameInput.Value() == "" && m.nameInput.Placeholder != "":
		return "→ Take suggested name  " + keys
	case (m.cf.fieldIdx == 1 || m.cf.fieldIdx == 2) && len(m.pickCandidates("")) > 0:
		return "{ Insert placeholder  " + keys
	}
	return keys
}

// pickRows is the most candidates shown at once; the window scrolls
// past that, with a count of what is off it.
const pickRows = 6

// writePickCandidates draws the candidates under the field the brace
// was typed in, the chosen one marked, each with what it means. A
// window of them follows the choice, with a count of what is off it.
func (m Model) writePickCandidates(b *strings.Builder, cf *commandFormState, w, rows int) {
	// No rows means no popup drawn — never a popup drawn over the bottom
	// of the terminal. The keys still work on it: Enter takes the first
	// match, Esc closes it, typing narrows it.
	if rows < 1 {
		return
	}
	cands := m.pickCandidates(cf.pickTyped(m.nameInput))
	b.WriteString("\n")
	if len(cands) == 0 {
		b.WriteString("  " + dim("nothing matches") + "\n")
		return
	}
	show := min(pickRows, rows)
	cursor := min(cf.pickCursor, len(cands)-1)
	start := windowStart(show, len(cands), cursor)
	for i := start; i < len(cands) && i < start+show; i++ {
		c := cands[i]
		marker, text := "   ", slotVar(c.text)
		if i == cursor {
			marker, text = " ▶ ", highlight(c.text)
		}
		b.WriteString(searchHighlight(marker+text+"  "+dim(truncate(c.what, max(8, w-len(c.text)-5))), []string{cf.pickTyped(m.nameInput)}, i == cursor, true) + "\n")
	}
	if rest := len(cands) - start - show; rest > 0 {
		b.WriteString("   " + dim(fmt.Sprintf("↓ %d more", rest)) + "\n")
	}
}

// writeWorkflowEditor draws the steps being edited, the one under the
// cursor as a bar, each with what it runs — or, while a step is being
// added, the commands to choose from with the search over them.
func (m Model) writeWorkflowEditor(b *strings.Builder, w, rows int) {
	e := m.wfe
	if e.adding {
		search := " " + accent("/") + " " + white(e.addSearch) + dim("_")
		b.WriteString(search + "\n")
		cands := m.addCandidates()
		var list strings.Builder
		writeWindowedList(&list, max(1, rows-1-1), len(cands), e.addCursor, "(no command matches)", func(i int) string {
			cmd, _ := m.config.FindCommand(cands[i])
			marker, name := "   ", cands[i]
			if i == e.addCursor {
				marker, name = " ▶ ", white(cands[i])
			}
			return searchHighlight(marker+name+"  "+dim(truncate(cmd.Cmd, max(8, w-len(cands[i])-6))), strings.Fields(e.addSearch), i == e.addCursor, true)
		})
		b.WriteString(list.String())
		return
	}

	if len(e.steps) == 0 {
		b.WriteString(" " + dim("no steps · a adds one") + "\n")
		return
	}
	writeWindowedList(b, max(1, rows-1), len(e.steps), e.cursor, "", func(i int) string {
		step := e.steps[i]
		cmd, ok := m.workflowStepCommand(step)
		what := warn("(not found)")
		if ok {
			what = dim(truncate("$ "+cmd.Cmd, max(8, w-len(step)-9)))
		}
		line := fmt.Sprintf(" %2d  %s  %s", i+1, step, what)
		if i == e.cursor {
			plain := fmt.Sprintf(" %2d  %s  ", i+1, step)
			if ok {
				plain += truncate("$ "+cmd.Cmd, max(8, w-len(step)-9))
			} else {
				plain += "(not found)"
			}
			return sRowActive.Render(plain + strings.Repeat(" ", max(0, w-lipgloss.Width(plain))))
		}
		return line
	})
}
