package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	mdl "github.com/Taka-S-dev/baton/internal/model"
	"github.com/Taka-S-dev/baton/internal/slot"
)

func (m Model) View() string {
	w := m.width
	if w == 0 {
		w = 80
	}
	// Messages and load warnings sit under the frame, so the frame is
	// built against a terminal that much shorter. Appending them to a
	// frame that already fills the terminal would scroll it, and what
	// scrolls away is the top of the frame — the screen you are reading.
	//
	// Where the two still do not fit, the banner is what gives way: every
	// screen has a floor it cannot shrink past, and a screen you cannot
	// read is worse than a message you cannot. Trimming from the end
	// keeps the error, which is the line worth keeping.
	banner := m.bannerLines()
	var view string
	for {
		m.bannerRows = len(banner)
		// No trailing newline: it would carry the cursor onto a further
		// line and cost the frame the bottom row of the terminal, which
		// the list can have instead.
		view = strings.TrimRight(m.renderScreen(w), "\n")
		over := strings.Count(view, "\n") + 1 + len(banner) - m.rows()
		if over <= 0 || len(banner) == 0 {
			break
		}
		banner = trimBanner(banner, len(banner)-over)
	}
	if len(banner) > 0 {
		view += "\n" + strings.Join(banner, "\n")
	}
	return view
}

// trimBanner shortens the banner to n lines, dropping it altogether when
// what is left is only the blank line that set it off from the frame.
func trimBanner(banner []string, n int) []string {
	if n < 0 {
		n = 0
	}
	banner = banner[:n]
	if len(banner) == 1 && banner[0] == "" {
		return nil
	}
	return banner
}

// renderScreen draws the active screen. The frame it returns is budgeted
// against m.termHeight, which is the terminal less the banner rows.
func (m Model) renderScreen(w int) string {
	if m.narrowDetails && !m.helpOpen {
		return m.viewNarrowDetails(w)
	}
	if m.helpOpen {
		bg := m
		bg.helpOpen = false
		return m.overlayHelp(bg.renderScreen(w), w)
	}
	if m.screen == ScreenSlotPick && m.sp != nil && m.sp.completion != nil {
		return m.overlayPathCompletion(w)
	}
	if m.pasteMenu {
		bg := m
		bg.pasteMenu = false
		p := m.pastePopup(w)
		return overlayBox(bg.renderScreen(w), box("Clipboard", m.clipboardMenuBody(), p.w, p.h, true), p.x, p.y, w)
	}
	if m.leaveAsk != leaveNone {
		background := m
		background.leaveAsk = leaveNone
		return m.overlayLeaveAsk(background.renderScreen(w), w)
	}
	if (m.exploreValueFlow() || m.exploreTemplateFlow()) && !m.exploreLayout(w).single {
		return m.viewExplore(w)
	}
	var view string
	switch m.screen {
	case ScreenRecent:
		view = m.viewRecent(w)
	case ScreenProjectSelect:
		view = m.viewProjectSelect(w)
	case ScreenRunWorkflow:
		view = m.viewRunWorkflow(w)
	case ScreenRunWorkflowSteps:
		view = m.viewRunWorkflowSteps(w)
	case ScreenRunCommands, ScreenCreateWorkflow, ScreenEditWorkflowCommands:
		view = m.viewMultiSelect(w)
	case ScreenWorkflowMgmt:
		view = m.viewSingleSelect("Manage workflows", w)
	case ScreenEditWorkflowMode:
		view = m.viewSingleSelect("Edit workflow", w)
	case ScreenSlotPick:
		view = m.viewSlotPick(w)
	case ScreenConfirmRun:
		view = m.viewConfirmRun(w)
	case ScreenRunning:
		view = m.viewRunning(w)
	case ScreenRetry:
		view = m.viewRetry(w)
	case ScreenNameInput:
		view = m.viewNameInput(w)
	case ScreenEditWorkflow:
		view = m.viewSingleSelect("Edit workflow", w)
	case ScreenDeleteWorkflow:
		view = m.viewDeleteList("Delete workflow", m.listItems, w)
	case ScreenManageLists:
		view = m.viewSingleSelect("Manage lists", w)
	case ScreenEditListPick:
		view = m.viewEditListPick(w)
	case ScreenEditList:
		view = m.viewEditList(w)
	case ScreenDeleteList:
		view = m.viewDeleteList("Delete list", m.listItems, w)
	case ScreenManageVars:
		view = m.viewSingleSelect("Manage vars", w)
	case ScreenEditVarPick:
		view = m.viewEditVarPick(w)
	case ScreenVarForm:
		view = m.viewVarForm(w)
	case ScreenDeleteVar:
		view = m.viewDeleteList("Delete variable", m.listItems, w)
	case ScreenVarRebase:
		view = m.viewVarRebase(w)
	case ScreenSwitchConfig:
		view = m.viewSingleSelect("Switch project", w)
	case ScreenManageCommands:
		view = m.viewManageCommands(w)
	case ScreenCreateCommandKind:
		view = m.viewSingleSelect("Create command", w)
	case ScreenCommandForm:
		view = m.viewCommandForm(w)
	case ScreenEditCommandPick:
		view = m.viewEditCommandPick(w)
	case ScreenEditCommandMode:
		view = m.viewSingleSelect("Edit command", w)
	case ScreenCreateCommandName, ScreenCreateCommandTemplate:
		view = m.viewCreateCommand(w)
	case ScreenEditCommandName, ScreenEditCommandTemplate:
		view = m.viewEditCommand(w)
	case ScreenDeleteCommand:
		view = m.viewDeleteList("Delete commands", m.listItems, w)
	case ScreenRenameRepair:
		view = m.viewRenameRepair(w)
	case ScreenExplore:
		if m.exploreLayout(w).single && m.cf != nil {
			view = m.viewNarrowCommand(w)
		} else if m.exploreLayout(w).single && m.wfe != nil {
			view = m.viewNarrowWorkflow(w)
		} else {
			view = m.viewExplore(w)
		}
	}
	if m.runHomeVisible() {
		view = m.viewRunHome(view, w)
	}
	return view
}

func (m Model) viewProjectSelect(w int) string { return m.viewProjects(w) }

// ── Generic single-select ────────────────────────────────────────────────────

func (m Model) viewSingleSelect(title string, w int) string {
	var b strings.Builder
	m.writeHeading(&b, title, w)
	if m.screen == ScreenEditWorkflow {
		m.writePickFilter(&b)
	}
	// Spacing above the footer gives way to list rows on a short terminal.
	gap, reserved := "\n", 3
	if m.compact() {
		gap, reserved = "", 2
	}
	// The steps preview is budgeted like every other hover panel: all of
	// it or none, so a short terminal spends the rows on the list rather
	// than on a preview it has no room for.
	stepsH := 0
	if m.screen == ScreenEditWorkflow {
		stepsH = m.previewBudget(&b, 2+m.stepsVP.Height, reserved, 3)
	}
	viewH := m.listBudget(&b, reserved+stepsH)
	// A submenu is a handful of fixed choices, and windowing it defeats
	// its purpose: on a short terminal "Manage vars" showed one of its
	// three items between two "1 more" counts. Few enough entries are
	// shown whole even when that carries the frame past the terminal,
	// because a menu you cannot see is worse than one that scrolls.
	if n := len(m.listItems); n <= shortMenu && n > viewH {
		viewH = n
	}
	writeWindowedList(&b, viewH, len(m.listItems), m.listCursor, "(empty)", func(i int) string {
		if i == m.listCursor {
			return "  " + accentBold("▶") + " " + m.listItems[i]
		}
		return "    " + m.listItems[i]
	})
	if stepsH > 0 {
		b.WriteString("\n" + hlineLabel(w, "steps") + "\n")
		b.WriteString(m.stepsVP.View() + "\n")
	}
	b.WriteString(gap + hline(w) + "\n")
	b.WriteString("  " + gray("↑↓ Enter Esc") + "\n")
	return b.String()
}

// ── Run workflow ─────────────────────────────────────────────────────────────

// workflowNameIsGenerated reports whether the workflow's name is exactly
// the one baton would build from its steps. It is the difference between
// knowing and guessing: a matching name provably decomposes into step
// names, so its "+" are joiners, while a name the user typed is left
// alone even if it happens to contain one.
func (m Model) workflowNameIsGenerated(idx int) bool {
	if idx < 0 || idx >= len(m.workflows) {
		return false
	}
	wf := m.workflows[idx]
	return wf.Name != "" && wf.Name == m.suggestWorkflowNameFor(wf.Commands, idx)
}

// workflowNameParts decomposes a generated workflow name into the step
// names it shows and its "+N more" tail, or reports ok=false when the
// name is not one baton built or no longer decomposes into its steps —
// the 48-rune cap can cut a generated name mid-step.
//
// The pieces come from rebuilding the name out of the workflow's steps,
// never from splitting on "+": the tail's "+" joins a count rather than
// a step, and a "+" the user typed joins nothing at all.
func (m Model) workflowNameParts(idx int) (parts []string, tail string, ok bool) {
	if !m.workflowNameIsGenerated(idx) {
		return nil, "", false
	}
	wf := m.workflows[idx]
	shown, tail := workflowNameSegments(wf.Commands)
	if strings.Join(shown, "+")+tail != wf.Name {
		return nil, "", false
	}
	return shown, tail, true
}

// workflowLabel renders a workflow's name for a list row, colouring the
// joiners between step names so the names read as separate chunks.
//
// The joiner has to outrank the hyphens inside step names — build-src
// carries one of its own — so it is marked rather than faded: fading put
// the boundary that matters below the one that does not. Gold is the
// emphasis colour here and, unlike the accent, is not already spoken for
// by the cursor on this screen. The "+N" tail stays grey, since it
// counts steps rather than naming one.
func (m Model) workflowLabel(idx int, hovered bool) string {
	style := func(s string) string { return s }
	if hovered {
		style = bold
	}
	parts, tail, ok := m.workflowNameParts(idx)
	if !ok {
		return style(m.workflows[idx].Name)
	}
	styled := make([]string, len(parts))
	for i, p := range parts {
		styled[i] = style(p)
	}
	return strings.Join(styled, highlight("+")) + gray(tail)
}

func (m Model) viewRunWorkflowList(w int) string {
	var b strings.Builder
	m.writeHeading(&b, "Run workflow", w)

	if len(m.workflows) == 0 {
		b.WriteString("  " + gray("(no workflows saved)") + "\n")
		if m.termHeight() >= 18 {
			b.WriteString("\n" + hline(w) + "\n")
			b.WriteString("  " + gray("Esc Back · F1 Help") + "\n")
		}
		return b.String()
	}

	filtered := m.wfFiltered()
	n := len(filtered)
	// Spacing gives way to list rows on a short terminal.
	gap, below := "\n", 5
	if m.compact() {
		gap, below = "", 4
	}
	b.WriteString("  " + m.runSearchView(m.wfSearchTI) + "  " + dim(fmt.Sprintf("%d/%d", n, len(m.workflows))) + "\n" + gap)
	cur := m.listCursor
	if cur >= n {
		cur = max(0, n-1)
	}

	// The steps viewport below has a fixed height, so budget against it
	// instead of the hovered workflow's step count — a per-item budget
	// makes the list grow and shrink as the cursor moves.
	stepsH := m.previewBudget(&b, m.stepsVP.Height, below, 3)
	if stepsH == 0 {
		// below counts the blank line and the "steps" rule the preview
		// opens with; without the preview the list gets both rows.
		below -= 2
	}
	if m.termHeight() < 18 {
		stepsH, below = 0, 0
	}
	viewH := m.listBudget(&b, stepsH+below)
	writeWindowedList(&b, viewH, n, cur, "No results.", func(i int) string {
		wf := m.workflows[filtered[i]]
		suffix := ""
		if wf.Name == m.lastWorkflow {
			suffix = "  " + gray("(last)")
		}
		if i == cur {
			return "  " + accentBold("▶") + " " + searchHighlight(m.workflowLabel(filtered[i], true)+suffix, strings.Fields(m.wfSearchTI.Value()), false, true)
		}
		return "    " + searchHighlight(m.workflowLabel(filtered[i], false)+suffix, strings.Fields(m.wfSearchTI.Value()), false, true)
	})

	// Step preview for hovered workflow (scrollable viewport), left out
	// when the terminal cannot hold it and the list at once.
	if stepsH > 0 {
		if m.stepsFocused {
			b.WriteString("\n" + hlineLabel(w, "steps ↑↓") + "\n")
		} else {
			b.WriteString("\n" + hlineLabel(w, "steps") + "\n")
		}
		b.WriteString(m.stepsVP.View() + "\n")
	}

	if m.termHeight() >= 18 {
		b.WriteString(gap + hline(w) + "\n")
		keys := "↑↓ Move  Enter Continue  Tab/→ Steps  F2 Details  Esc Back  F1 Help"
		if m.runHomeVisible() {
			keys = m.runHomeKeys()
		}
		b.WriteString("  " + gray(keys) + "\n")
	}
	if m.termHeight() < 18 {
		lines := strings.Split(strings.TrimRight(b.String(), "\n"), "\n")
		for len(lines) < m.termHeight() {
			lines = append(lines, " ")
		}
		return strings.Join(lines, "\n")
	}
	return b.String()
}

// ── Multi-select ─────────────────────────────────────────────────────────────

func (m Model) viewMultiSelectList(w int) string {
	return m.viewMultiSelectCandidates(w, true)
}

func (m Model) viewMultiSelectCandidates(w int, showPreview bool, hitRow ...func(int, int)) string {
	if m.shortPanels() {
		return m.viewCompactCandidates(w, hitRow...)
	}
	title := "Select commands"
	if m.screen == ScreenEditWorkflowCommands {
		title = "Edit commands"
	}
	filtered := m.msFiltered()
	n := len(filtered)

	terms := strings.Fields(m.msSearchTI.Value())
	orderW := max(3, len(fmt.Sprint(len(m.msSelected)))+2)
	columnW, matchW := max(1, w-3-orderW), 0
	if len(terms) > 0 && columnW >= 48 {
		matchW = min(32, columnW/3)
		columnW -= matchW + 2
	}
	commandW := 0
	if columnW >= 76 {
		commandW = columnW - 58
		columnW = 56
	}
	tableHeader := selectCommandColumns(m.sortHeading("name", "Name"), m.sortHeading("group", "Group"), m.sortHeading("type", "Type"), columnW)
	if commandW > 0 {
		tableHeader += "  " + fmt.Sprintf("%-*s", commandW, "Command")
	}
	if matchW > 0 {
		tableHeader += "  " + "Match"
	}

	previewH := maxLines(len(m.msItems), 3, func(i int, tb *strings.Builder) {
		if cmd := m.msItems[i].cmd; cmd != nil {
			m.writeCommandHover(tb, cmd, w)
		}
	})
	cursor := m.msCursor
	if cursor >= n {
		cursor = max(0, n-1)
	}

	var b strings.Builder
	m.writeHeading(&b, title, w)

	// On a short terminal the blank lines around the list become list
	// rows: spacing is the cheapest thing in the frame to give up.
	gap, below := "\n", 5
	if m.compact() {
		gap, below = "", 4
	}
	b.WriteString("  " + m.commandSearchLine(max(0, w-2), n) + "\n" + gap)

	if !showPreview {
		previewH = 0
	}
	if m.termHeight() >= 10 {
		if len(hitRow) > 0 {
			hitRow[0](-1, strings.Count(b.String(), "\n"))
		}
		b.WriteString("  " + gray(fmt.Sprintf("%-*s ", orderW, "Run")+tableHeader) + "\n")
	}
	previewH = m.previewBudget(&b, previewH, below, 3)
	if previewH == 0 {
		// below counts the blank line the preview opens with. Dropped
		// preview, dropped blank line — the list gets the row.
		below--
	}
	viewH := m.listBudget(&b, previewH+below)
	writeWindowedList(&b, viewH, n, cursor, "No results.", func(i int) string {
		if len(hitRow) > 0 {
			hitRow[0](i, strings.Count(b.String(), "\n"))
		}
		origIdx := filtered[i]
		item := m.msItems[origIdx]

		check := gray(fmt.Sprintf("%-*s", orderW, "[ ]"))
		for order, selected := range m.msSelected {
			if selected == origIdx {
				check = sSelNum.Render(fmt.Sprintf("%-*s", orderW, fmt.Sprintf("[%d]", order+1)))
				break
			}
		}
		label := selectCommandColumns(item.cmd.Name, item.cmd.Group, commandKind(*item.cmd), columnW)
		if commandW > 0 {
			body := ansi.Truncate(m.syntaxText(strings.Join(strings.Fields(item.cmd.Cmd), " ")), commandW, "…")
			label += "  " + body + strings.Repeat(" ", max(0, commandW-lipgloss.Width(body)))
		}

		if matchW > 0 {
			reason := commandMatchReason(item.cmd, label, terms, matchW)
			label += "  " + ansi.Truncate(reason, matchW, "…")
		}

		label = searchHighlight(label, terms, false, true)
		if i == cursor {
			row := "▶ " + check + " " + label
			row += strings.Repeat(" ", max(0, w-lipgloss.Width(row)))
			style := sRowActive
			if m.runQueueFocused {
				style = sRowInactive
			}
			return searchHighlight(style.Render(ansi.Strip(row)), terms, true, !m.runQueueFocused)
		}
		return "  " + check + " " + label
	})

	// Hover preview, padded to previewH so the footer below never moves.
	// A terminal with no room for it gets the list instead.
	if previewH > 0 {
		b.WriteString("\n")
		var pv strings.Builder
		if n > 0 && cursor >= 0 && cursor < n {
			hovered := m.msItems[filtered[cursor]]
			if hovered.cmd != nil {
				m.writeCommandHover(&pv, hovered.cmd, w)
			}
		}
		if pv.Len() == 0 {
			pv.WriteString(hline(w) + "\n")
		}
		writePadded(&b, pv.String(), previewH)
	}

	var selNames []string
	for _, idx := range m.msSelected {
		selNames = append(selNames, m.msItems[idx].name())
	}
	orderHint := ""
	if len(m.msSelected) > 0 {
		orderHint = "  " + dim("Run queue →")
	}
	// Keep the Selected line to one row: a wrapped line would grow the
	// frame past the terminal height and bring the jitter back.
	joined := truncate(strings.Join(selNames, ", "), max(10, w-40))
	b.WriteString(gap + "  " + success(fmt.Sprintf("Selected(%d)", len(m.msSelected))) + orderHint + ": " + joined + "\n")
	b.WriteString(hline(w) + "\n")
	keys := "Tab Select  Enter Review  F2 Queue  s Sort  / Search  Esc Back"
	if m.runHomeVisible() {
		keys = m.runHomeKeys()
	}
	b.WriteString("  " + gray(keys) + "\n")

	// Discard guard: opens as a centered floating window over the list.
	if m.msEscArmed && len(m.msSelected) > 0 {
		return m.viewDiscardWindow(w, selNames)
	}
	return b.String()
}

// viewDiscardWindow renders the discard confirmation as a centered floating
// window (same style as the placeholder picker): shown after Esc is pressed
// with items still selected. Esc cancels the dialog and preserves selection.
func (m Model) viewDiscardWindow(w int, selNames []string) string {
	background := m
	background.msEscArmed = false
	bg := background.viewMultiSelect(w)
	height := min(m.termHeight(), frameRows(bg))
	pw, ph := min(60, w), min(10, height)
	names := truncate(strings.Join(selNames, ", "), max(1, pw-4))
	content := fmt.Sprintf(" %d commands selected:\n %s", len(selNames), names)
	if ph >= 9 {
		content += "\n\n " + strings.ReplaceAll(m.discardOptions(), "\n", "\n ") + "\n\n ↑↓ Select · Enter Confirm · Esc Cancel"
	} else if ph >= 7 {
		content = " " + strings.ReplaceAll(m.discardOptions(), "\n", "\n ") + "\n\n Enter Confirm · Esc Cancel"
	} else {
		content = " Esc Cancel · d Discard and go back"
	}
	window := box("Discard selection?", content, pw, ph, true)
	return overlayBox(bg, window, max(0, (w-pw)/2), max(0, (height-ph)/2), w)
}

// ── Slot pick ────────────────────────────────────────────────────────────────

// viewSlotPick draws the slot picker, tightening it when it does not
// fit. How tall this screen wants to be depends on the run — the context
// panel grows with the steps still to resolve — so no fixed height can
// say when it has to give its spacing up. It asks the frame instead.
func (m Model) viewSlotPick(w int) string {
	if !m.compact() {
		if view := m.slotPickFrame(w, false, 3); frameRows(view) <= m.termHeight() {
			return view
		}
	}
	// Spacing goes first, then the context panel narrows a step at a
	// time, down to the step being resolved on its own with the counts
	// either side of it.
	for around := 3; around > 0; around-- {
		if view := m.slotPickFrame(w, true, around); frameRows(view) <= m.termHeight() {
			return view
		}
	}
	return m.slotPickFrame(w, true, 0)
}

// frameRows is how many terminal lines a frame occupies once View has
// trimmed the trailing newline off it.
func frameRows(s string) int {
	return strings.Count(strings.TrimRight(s, "\n"), "\n") + 1
}

func (m Model) slotPickFrame(w int, tight bool, around int) string {
	sp := m.sp
	var b strings.Builder
	title := "  [ Select value for " + sp.placeholder() + " ]"
	if sp.variadic {
		title = "  [ Select values for " + sp.placeholder() + " ]"
	}
	// This screen carries more fixed furniture than any other — a context
	// panel, a command preview, two labelled rules — so on a short
	// terminal it gives up the spacing between them and narrows the
	// context to the steps either side of the current one. It sits on the
	// run path, so it has to work in the panel the rest works in.
	gap := "\n"
	if tight {
		gap = ""
	}
	b.WriteString(gap + accentBold(title) + "\n" + hline(w) + "\n" + gap)

	// Context panel — windowed around the step being resolved.
	if sp.contextNames != nil {
		cur := sp.contextIdx
		n := len(sp.contextNames)
		// around is how far either side of the current step is shown;
		// at zero the current step stands alone between the counts.
		start := max(0, cur-around)
		end := min(n, cur+max(1, around))

		// At the tightest the step stands alone: its own row carries its
		// number, so the counts either side are what goes last.
		if start > 0 && around > 0 {
			b.WriteString("  " + dim(fmt.Sprintf("  ↑ %d more", start)) + "\n")
		}
		for i := start; i < end; i++ {
			name := sp.contextNames[i]
			if i == cur {
				b.WriteString("  " + accentBold(fmt.Sprintf("▶ %2d. %s", i+1, name)) + "\n")
			} else {
				isDone := i < cur && i < len(sp.contextNotes) && sp.contextNotes[i] != ""
				marker := "  "
				nameStr := gray(fmt.Sprintf("  %2d. %s", i+1, name))
				if isDone {
					marker = success("✓ ")
					note := sp.contextNotes[i]
					if maxNote := w - len(name) - 14; maxNote > 10 {
						note = truncate(note, maxNote)
					}
					nameStr = gray(fmt.Sprintf("  %2d. %-14s", i+1, name)) + dim(note)
				}
				b.WriteString("  " + marker + nameStr + "\n")
			}
		}
		if end < n && around > 0 {
			b.WriteString("  " + dim(fmt.Sprintf("    ↓ %d more", n-end)) + "\n")
		}

		// Command preview — separated from the list
		if sp.currentCmd != nil {
			previewLabel := "command preview  " + highlight(sp.placeholder())
			if progress := m.inputProgressLabel(); progress != "" {
				previewLabel += "  ·  " + progress
			}
			b.WriteString(gap + hlineLabelBright(w, previewLabel) + "\n" + gap)

			// Hovered value (from list cursor or custom search input)
			hoveredVal := ""
			if sp.cursor >= 0 && sp.cursor < len(sp.filtered) {
				hoveredVal = sp.filtered[sp.cursor].Value
			} else if sp.cursor == len(sp.filtered) && sp.search != "" {
				hoveredVal = sp.search
			}

			if sp.pathDef.PathKind != "" && sp.completedPath == "" {
				hoveredVal = ""
			}

			// Substitute resolvedSoFar, then the current slot: the joined
			// picks for a variadic slot (they win on Enter), the hovered
			// value otherwise.
			preview := func(s string) string {
				for k, v := range sp.resolvedSoFar {
					s = slot.Replace(s, k, v)
				}
				switch {
				case sp.variadic && len(sp.picked) > 0:
					s = slot.Replace(s, sp.slotName, highlight(strings.Join(sp.picked, " ")))
				case hoveredVal != "":
					s = slot.Replace(s, sp.slotName, highlight(hoveredVal))
				default:
					s = slot.Replace(s, sp.slotName, slotVar(sp.placeholder()))
				}
				return slot.ApplyVars(s, m.vars)
			}

			b.WriteString("    " + gray("$") + " " + preview(sp.currentCmd.Cmd) + "\n")

			dir := sp.currentCmd.Dir
			if dir == "" {
				b.WriteString("    " + gray("workdir:") + " " + dim(".") + "\n")
			} else {
				b.WriteString("    " + gray("workdir:") + " " + preview(dir) + "\n")
			}
		}
		// The reuse hint rides on the divider rather than taking its own
		// line: this screen is height-critical, and the value itself is
		// already visible on the cursor row and in the preview above.
		selectLabel := "Select value"
		if sp.reuseFrom != "" {
			selectLabel += "  ·  same as " + sp.reuseFrom + " — Enter keeps it"
		}
		b.WriteString(gap + gap + hlineLabelBright(w, selectLabel) + "\n" + gap)
	}

	// Search field
	if sp.search == "" {
		b.WriteString("  " + dim(sp.inputHint()) + "\n" + gap)
	} else {
		countStr := dim(sp.inputStatus())
		if m.pathInputReady() {
			countStr = ""
		}
		b.WriteString("  " + accent("/") + " " + white(sp.search) + dim("_") +
			"  " + countStr + "\n" + gap)
	}

	if sp.pathDef.PathKind != "" {
		if len(sp.picked) > 0 {
			label := dim("Selected: ")
			if m.pathInputReady() {
				label = success("✓ Selected: ")
			}
			b.WriteString("  " + label + white(strings.Join(sp.picked, "  ·  ")) + "\n")
		} else if sp.completedPath != "" {
			b.WriteString("  " + success("✓ ") + white(sp.completedPathTitle()+" selected: "+sp.completedPath) + "\n")
		} else {
			b.WriteString("  " + dim("Tab: open choices") + "\n")
		}
		keys := m.pathInputKeys()
		b.WriteString(hline(w) + "\n  " + gray(keys) + "\n")
		return b.String()
	}
	// List
	skipRow := len(sp.filtered) + 1
	total := skipRow
	if sp.canSkip {
		total++
	}

	// Compute label column alignment
	labelCol := 0
	for _, e := range sp.filtered {
		if e.Label != "" && len(e.Value)+2 > labelCol {
			labelCol = len(e.Value) + 2
		}
	}

	viewH := m.listBudget(&b, 3)
	writeWindowedList(&b, viewH, total, sp.cursor, "", func(i int) string {
		isCustom := i == len(sp.filtered)
		isSkip := sp.canSkip && i == skipRow

		if i == sp.cursor {
			// Render raw text inside sCursor to avoid ANSI width miscalculation
			var rawLine string
			if isSkip {
				rawLine = "[ → skip — resolve at run time ]"
			} else if isCustom {
				rawLine = sp.typedValueLabel()
			} else {
				e := sp.filtered[i]
				if e.Label != "" {
					pad := strings.Repeat(" ", max(1, labelCol-len(e.Value)))
					rawLine = e.Value + pad + "·  " + e.Label
				} else {
					rawLine = e.Value
				}
				if sp.variadic {
					if sp.isPicked(e.Value) {
						rawLine = "[x] " + rawLine
					} else {
						rawLine = "[ ] " + rawLine
					}
				}
			}
			return sCursor.Width(w - 2).Render("    " + rawLine)
		}
		var line string
		if isSkip {
			line = dim("[ → skip — resolve at run time ]")
		} else if isCustom {
			line = dim(sp.typedValueLabel())
		} else {
			e := sp.filtered[i]
			if e.Label != "" {
				pad := strings.Repeat(" ", max(1, labelCol-len(e.Value)))
				line = white(e.Value) + dim(pad+"·  "+e.Label)
			} else {
				line = white(e.Value)
			}
			if sp.variadic {
				if sp.isPicked(e.Value) {
					line = success("[x] ") + line
				} else {
					line = dim("[ ] ") + line
				}
			}
		}
		return "    " + line
	})

	b.WriteString("\n" + hline(w) + "\n")
	if sp.variadic {
		picked := ""
		if len(sp.picked) > 0 {
			picked = "  " + dim(fmt.Sprintf("%d picked", len(sp.picked)))
		}
		b.WriteString("  " + gray("↑↓") + "  " + gray("Tab: ") + dim("toggle") +
			"  " + gray("Enter: ") + dim("confirm") +
			"  " + gray("Esc: ") + dim("clear filter / back") + picked + "\n")
	} else {
		b.WriteString("  " + gray("↑↓ Enter") + "  " + gray("Esc: ") + dim("clear filter / back") + "\n")
	}
	return b.String()
}

// writePickFilter renders the type-to-filter line for the management
// pick screens (Edit/Delete of commands, workflows and lists).
func (m Model) writePickFilter(b *strings.Builder) {
	// The blank line under the field is spacing, and on a short terminal
	// spacing is the cheapest thing in the frame to give up.
	gap := "\n"
	if m.compact() {
		gap = ""
	}
	if m.pickSearch == "" {
		b.WriteString("  " + dim("/ type to filter...") + "\n" + gap)
		return
	}
	count := dim(fmt.Sprintf("%d results", len(m.listItems)))
	if len(m.listItems) == 0 {
		count = warn("no match")
	}
	b.WriteString("  " + accent("/") + " " + white(m.pickSearch) + dim("_") + "  " + count + "\n" + gap)
}

// ── Fixed-height windowed lists ──────────────────────────────────────────────
//
// Scrolling screens share one discipline: the row window and the hover
// preview are padded to constant heights, so the frame never exceeds the
// terminal (an overflowing frame makes the terminal itself scroll on every
// repaint — visible jitter) and nothing below the list shifts as the
// cursor moves.

// listBudget returns the row budget for a windowed list: the terminal
// height minus what the frame already holds, the one scroll-marker line
// the list writes under itself, and what is still to come below it.
// Nothing is held back beyond that — the view carries no trailing
// newline, so a frame of exactly termHeight rows fits.
func (m Model) listBudget(b *strings.Builder, reservedBelow int) int {
	return max(1, m.termHeight()-strings.Count(b.String(), "\n")-1-reservedBelow)
}

// rows is the terminal height, or a conventional 24 before the first
// size message arrives.
func (m Model) rows() int {
	if m.height == 0 {
		return 24
	}
	return m.height
}

// termHeight is what the frame may occupy: the terminal less the rows
// the banner takes under it.
func (m Model) termHeight() int {
	reserved := 0
	if m.runHomeVisible() {
		reserved = 1
	}
	return max(1, m.rows()-m.bannerRows-reserved)
}

// bannerLines renders what goes under the frame: an error, a success
// note, and the warnings the Run workspace carries from load. They are
// returned rather than written so the frame can be budgeted against the
// height they leave — see View.
func (m Model) bannerLines() []string {
	var out []string
	if m.errMsg != "" {
		out = append(out, errorText("  Error: "+m.errMsg))
	}
	if m.successMsg != "" {
		out = append(out, success("  ✓ "+m.successMsg))
	}
	if len(m.loadWarnings) > 0 && m.runHomeVisible() {
		// The warnings are a pointer to `baton check`, not the report
		// itself, so they never take more of the terminal than the workspace
		// they are pointing away from.
		maxShown := 4
		if m.height > 0 {
			maxShown = min(maxShown, max(1, m.height/3))
		}
		shown := m.loadWarnings
		if len(shown) > maxShown {
			shown = shown[:maxShown]
		}
		for _, wmsg := range shown {
			out = append(out, warn("  Warning: "+wmsg))
		}
		if rest := len(m.loadWarnings) - len(shown); rest > 0 {
			out = append(out, warn(fmt.Sprintf("  ... and %d more (run: baton check)", rest)))
		}
	}
	if len(out) == 0 {
		return nil
	}
	// A blank line sets the banner off from the frame. A short terminal
	// has no line to spare for spacing, here as everywhere else.
	if m.height == 0 || m.height >= 12 {
		out = append([]string{""}, out...)
	}
	return out
}

// previewBudget caps a hover preview at what the frame can still spare
// once the list has minRows to itself. want is the height the preview
// would like; the result is 0 when there is no room at all, and the
// caller then leaves the preview out entirely. Shrinking the list alone
// could not keep the frame inside a ten-line terminal, because the list
// is already floored at one row while the preview kept its full height.
// It is all or nothing: the preview is written at its natural height by
// a writer that does not truncate, so capping it lower would not shrink
// the frame — it would only hand the list a budget the preview then
// overran.
func (m Model) previewBudget(b *strings.Builder, want, reservedBelow, minRows int) int {
	spare := m.termHeight() - strings.Count(b.String(), "\n") - 1 - reservedBelow - 1 - (minRows + 1)
	if spare >= want {
		return want
	}
	return 0
}

// writeHeading writes a scrolling screen's heading, shrinking it to the
// title alone on a terminal too short to spend four lines naming the
// screen — the blank lines around it and the rule under it are worth
// less than the three list rows they cost.
func (m Model) writeHeading(b *strings.Builder, title string, w int) {
	m.writeHeadingUnder(b, title, w, 12)
}

// writeHeadingUnder is writeHeading with the height it gives way at. A
// screen whose rows come in threes reaches that point sooner: three
// lines of heading is a whole item.
func (m Model) writeHeadingUnder(b *strings.Builder, title string, w, minRows int) {
	if m.termHeight() < minRows {
		b.WriteString(accentBold("  [ "+title+" ]") + "\n")
		return
	}
	b.WriteString("\n" + header(title, w) + "\n")
}

// shortMenu is how many entries a list may hold and still be treated as
// a menu — shown whole rather than scrolled through.
const shortMenu = 6

// compact reports whether the frame is tight enough that the blank lines
// between sections have to become list rows instead. Spacing is the
// cheapest thing to give up: it carries no information.
func (m Model) compact() bool { return m.termHeight() < 20 }

// writeWindowedList renders a cursor-following window over n rows above
// one line carrying what is hidden in either direction, padded to
// exactly viewH+1 lines. render(i) returns row i without a trailing
// newline; emptyMsg is shown when the list is empty.
//
// Both counts share a line because a slot above and a slot below cost
// two rows whatever is in them, and on a ten-line terminal that was two
// of the four the list had to itself. The line is always written, empty
// or not, so the frame keeps one height as the window scrolls.
func writeWindowedList(b *strings.Builder, viewH, n, cursor int, emptyMsg string, render func(i int) string) {
	viewStart := windowStart(viewH, n, cursor)
	viewEnd := min(viewStart+viewH, n)
	rows := 0
	if n == 0 && emptyMsg != "" {
		b.WriteString("  " + gray(emptyMsg) + "\n")
		rows++
	}
	for i := viewStart; i < viewEnd; i++ {
		b.WriteString(render(i) + "\n")
		rows++
	}
	for ; rows < viewH; rows++ {
		b.WriteString("\n")
	}
	var marks []string
	if viewStart > 0 {
		marks = append(marks, fmt.Sprintf("↑ %d more", viewStart))
	}
	if viewEnd < n {
		marks = append(marks, fmt.Sprintf("↓ %d more", n-viewEnd))
	}
	if len(marks) > 0 {
		b.WriteString("  " + dim(strings.Join(marks, "   ")) + "\n")
	} else {
		b.WriteString("\n")
	}
}

// windowStart is the index of the first row a cursor-following window
// shows. It is shared with the click handling, which has to work out
// which row the pointer landed on from the same arithmetic.
func windowStart(viewH, n, cursor int) int {
	return max(0, min(cursor-viewH/2, n-viewH))
}

// truncate shortens s to at most maxLen display runes, marking the cut
// with an ellipsis. Counting runes keeps multi-byte text from being cut
// mid-character.
func truncate(s string, maxLen int) string {
	r := []rune(s)
	if len(r) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return string(r[:max(0, maxLen)])
	}
	return string(r[:maxLen-3]) + "..."
}

// writePadded appends s, then blank lines until it spans exactly h lines.
func writePadded(b *strings.Builder, s string, h int) {
	b.WriteString(s)
	for i := strings.Count(s, "\n"); i < h; i++ {
		b.WriteString("\n")
	}
}

// maxLines measures the tallest of n candidate renders, so a preview
// region can be reserved before drawing. floor is the minimum returned.
func maxLines(n, floor int, render func(i int, b *strings.Builder)) int {
	h := floor
	for i := 0; i < n; i++ {
		var tb strings.Builder
		render(i, &tb)
		if c := strings.Count(tb.String(), "\n"); c > h {
			h = c
		}
	}
	return h
}

// commandPickLabel renders a command's row label the same way in every
// picker: $ and the template name for derived commands, the group tag,
// and {...} for slotted ones.
func (m Model) commandPickLabel(cmd *mdl.Command) string {
	label := cmd.Name
	if cmd.Group != "" {
		label += "  " + sGroup.Render("["+cmd.Group+"]")
	}
	switch {
	case cmd.Template != "":
		label = accent("$") + " " + label + "  " + gray("("+cmd.Template+")")
	case slot.HasPlaceholders(*cmd):
		label += "  " + gray("{...}")
	}
	return label
}

// writeCommandHover renders the hover panel for a command: template and
// values for derived commands, the resolved command line otherwise.
func (m Model) writeCommandHover(b *strings.Builder, cmd *mdl.Command, w int) {
	if cmd.Template != "" {
		b.WriteString(hlineLabel(w, "command") + "\n")
		b.WriteString("  " + dim("template: ") + cmd.Template + "\n")
		if len(cmd.Values) > 0 {
			b.WriteString("  " + dim("values:") + "\n")
			for _, k := range sortedKeys(cmd.Values) {
				b.WriteString("    " + gray(k) + " = " + cmd.Values[k] + "\n")
			}
		}
		b.WriteString("  " + m.previewVars("$ "+cmd.Cmd) + "\n")
		b.WriteString("  " + dim("workdir: ") + m.previewVars(displayOr(cmd.Dir, ".")) + "\n")
		b.WriteString("  " + dim("shell: "+shellChoiceLabel(cmd.Shell)) + "\n")
		return
	}
	b.WriteString(hlineLabel(w, "command preview") + "\n")
	cmdStr := ansi.Truncate(m.previewVars(cmd.Cmd), max(10, w-10), "…")
	b.WriteString("  " + "$ " + cmdStr + "\n")
	workdir := cmd.Dir
	if workdir == "" {
		workdir = "."
	}
	b.WriteString("  " + dim("workdir: ") + m.previewVars(workdir) + "\n")
	b.WriteString("  " + dim("shell: "+shellChoiceLabel(cmd.Shell)) + "\n")
}

// ── Edit command pick ────────────────────────────────────────────────────────

// viewEditCommandPick lists the editable commands (TSV rows and
// template-derived ones) with the same hover preview as the command
// selector, so you can see what a command does before opening it.
func (m Model) viewEditCommandPick(w int) string {
	var b strings.Builder
	m.writeHeading(&b, "Edit command", w)
	if len(m.editRefs) == 0 {
		b.WriteString("  " + gray("(no commands yet)") + "\n")
		b.WriteString("\n" + hline(w) + "\n")
		b.WriteString("  " + gray("Esc: back") + "\n")
		return b.String()
	}
	m.writePickFilter(&b)
	previewH := maxLines(len(m.editRefs), 3, func(i int, tb *strings.Builder) {
		m.writeCommandHover(tb, m.editRefCommand(m.editRefs[i]), w)
	})
	previewH = m.previewBudget(&b, previewH, 4, 3)
	// 4 counts the blank line the preview opens with; without the
	// preview that line is a list row instead.
	below := 4
	if previewH == 0 {
		below = 3
	}
	viewH := m.listBudget(&b, previewH+below)
	writeWindowedList(&b, viewH, len(m.listItems), m.listCursor, "", func(i int) string {
		label := m.commandPickLabel(m.editRefCommand(m.editRefs[m.pickOrig(i)]))
		if i == m.listCursor {
			return "  " + accentBold("▶") + " " + label
		}
		return "    " + label
	})
	if previewH > 0 {
		b.WriteString("\n")
		var pv strings.Builder
		if m.listCursor >= 0 && m.listCursor < len(m.listItems) {
			m.writeCommandHover(&pv, m.editRefCommand(m.editRefs[m.pickOrig(m.listCursor)]), w)
		} else {
			pv.WriteString(hline(w) + "\n")
		}
		writePadded(&b, pv.String(), previewH)
	}
	b.WriteString("\n" + hline(w) + "\n")
	b.WriteString("  " + gray("↑↓ Enter Esc") + "\n")
	return b.String()
}

// ── Confirm run ──────────────────────────────────────────────────────────────

// confirmRunPerPage returns how many three-line items fit on the confirm
// screen. The view and the ↑↓ handler share it so scrolling stops exactly
// at the last page.
func (m Model) confirmRunPerPage() int {
	if m.termHeight() < 18 {
		_, same := m.confirmCommonDirectory()
		if same {
			return max(1, m.termHeight()-5)
		}
		return max(1, (m.termHeight()-5)/2)
	}
	var b strings.Builder
	m.writeHeadingUnder(&b, "Confirm", m.width, 20)
	// Below the items: the "n more" line, a blank, the rule, a blank, the
	// three rows of the button box, a blank and the key line. The box was
	// counted as one row, which is why the screen ran past the terminal
	// on a terminal of any size.
	return max(1, m.listBudget(&b, m.confirmRunBelow())/3)
}

// confirmRunBelow is how many rows the confirm screen holds under its
// item list: the "n more" line, the rule, the three rows of the button
// box, the key line, and the three blank lines that set them apart on a
// terminal with room for spacing.
func (m Model) confirmRunBelow() int {
	if m.compact() {
		return 6
	}
	return 9
}

// confirmGap is the blank line between the confirm screen's lower
// sections, which a short terminal spends on an item instead.
func confirmGap(m Model) string {
	if m.compact() {
		return ""
	}
	return "\n"
}

func (m Model) viewConfirmRun(w int) string {
	if m.termHeight() < 18 {
		return m.viewCompactConfirm(w)
	}
	var b strings.Builder
	m.writeHeadingUnder(&b, "Review & run", w, 20)

	// Every item spans three lines, so the window scrolls whole items.
	// There is no row cursor here — ↑↓ move the window directly.
	n := len(m.confirmRunItems)
	perPage := m.confirmRunPerPage()
	scroll := min(max(0, m.confirmRunScroll), max(0, n-perPage))
	end := min(scroll+perPage, n)

	if scroll > 0 {
		b.WriteString("  " + dim(fmt.Sprintf("↑ %d more", scroll)) + "\n")
	} else {
		b.WriteString("\n")
	}
	for i := scroll; i < end; i++ {
		item := m.confirmRunItems[i]
		b.WriteString(fmt.Sprintf("  %s%2d.%s  %s\n", gray(""), i+1, gray(""), item.Name))
		if item.Cmd != nil {
			// Resolve {$vars} here as every other preview does, and as
			// the runner will: this screen answers "what is about to
			// run", so it must not show something else.
			cmd := slot.ApplyVarsToCommand(*item.Cmd, m.vars)
			b.WriteString("       " + white("$ "+cmd.Cmd) + "\n")
			workdir := m.confirmWorkdir(cmd.Dir)
			b.WriteString("         " + dim("workdir: "+workdir) + "\n")
		} else {
			b.WriteString("\n\n")
		}
	}
	for pad := end - scroll; pad < perPage; pad++ {
		b.WriteString("\n\n\n")
	}
	if end < n {
		b.WriteString("  " + dim(fmt.Sprintf("↓ %d more", n-end)) + "\n")
	} else {
		b.WriteString("\n")
	}

	b.WriteString(confirmGap(m) + hline(w) + "\n" + confirmGap(m))
	b.WriteString(renderBtns(m.confirmRunBtn, "  Run  ", "  "+m.confirmBackLabel()+"  ") + "\n")
	b.WriteString(confirmGap(m) + "  " + gray("↑↓: scroll   Tab: switch   Enter: select   Esc: "+strings.ToLower(m.confirmBackLabel())) + "\n")
	return b.String()
}

// ── Running ──────────────────────────────────────────────────────────────────

func (m Model) viewRunning(w int) string {
	if m.running == nil || !m.running.completed || m.exitAfterRun {
		return ""
	}
	return "\n  " + gray("Press Enter to return to menu…") + "\n"
}

// ── Retry ────────────────────────────────────────────────────────────────────

func (m Model) viewRetry(w int) string {
	r := m.running
	var b strings.Builder
	b.WriteString("\n" + header("Run failed", w) + "\n")
	if r != nil && r.failErr != nil {
		b.WriteString("  " + errorText("Error: "+r.failErr.Error()) + "\n\n")
	}

	items := []string{
		fmt.Sprintf("Retry from step %d", r.current+1),
		"Retry all",
		"Abort",
	}
	for i, item := range items {
		if i == m.listCursor {
			b.WriteString("  " + accentBold("▶") + " " + item + "\n")
		} else {
			b.WriteString("    " + item + "\n")
		}
	}
	b.WriteString("\n" + hline(w) + "\n")
	b.WriteString("  " + gray("↑↓ Enter Esc") + "\n")
	return b.String()
}

// ── Name input ────────────────────────────────────────────────────────────────

func (m Model) viewNameInput(w int) string {
	title := "Create workflow"
	switch m.nameInputMode {
	case nameInputEditWorkflow:
		title = "Rename workflow"
	case nameInputRenameCommand:
		title = "Rename command"
	case nameInputNewList:
		title = "New list"
	}
	hint := "Esc: cancel"
	if m.nameSuggestion() != "" {
		hint = "Tab: suggest name   " + hint
	}
	var b strings.Builder
	b.WriteString("\n" + header(title, w) + "\n")
	b.WriteString("  " + gray(hint) + "\n\n")
	b.WriteString("  " + m.nameInput.View() + "\n")
	if m.nameInputErr != "" {
		b.WriteString("\n  " + errorText(m.nameInputErr) + "\n")
	}
	b.WriteString("\n" + hline(w) + "\n")
	return b.String()
}

// ── Edit list ─────────────────────────────────────────────────────────────────

func (m Model) viewEditList(w int) string {
	le := m.le
	var b strings.Builder
	b.WriteString("\n" + header("List: "+le.name, w) + "\n")

	if le.editing {
		b.WriteString("  " + gray("Esc: cancel") + "\n\n")
		b.WriteString("  " + le.editValTI.View() + "\n")
		if le.editFld == 1 {
			b.WriteString("  " + le.editLblTI.View() + "\n")
		} else {
			b.WriteString("  " + gray("Enter to next: label") + "\n")
		}
		return b.String()
	}

	if le.adding {
		b.WriteString("  " + gray("Esc: cancel") + "\n\n")
		b.WriteString("  " + accent("Value > ") + le.addVal)
		if le.addFld == 0 {
			b.WriteString(dim("_"))
		}
		b.WriteString("\n")
		if le.addFld == 1 {
			b.WriteString("  " + gray("Label (Enter to skip) > ") + le.addLbl + dim("_") + "\n")
		}
		return b.String()
	}

	b.WriteString("  " + gray("a: add   Del: remove   Esc: done") + "\n\n")
	labelCol := 0
	for _, e := range le.entries {
		if e.Label != "" && len(e.Value)+2 > labelCol {
			labelCol = len(e.Value) + 2
		}
	}
	// The "+ Add value" row scrolls with the entries as the last row.
	viewH := m.listBudget(&b, 3)
	writeWindowedList(&b, viewH, len(le.entries)+1, le.cursor, "", func(i int) string {
		// The add row is an action and is drawn like every other Add
		// row: in the accent, with its key beside it.
		if i == len(le.entries) {
			if i == le.cursor {
				return "  " + accentBold("▶") + " " + accent("+ Add value  a")
			}
			return "    " + accent("+ Add value  a")
		}
		e := le.entries[i]
		lbl := ""
		if e.Label != "" {
			pad := strings.Repeat(" ", max(1, labelCol-len(e.Value)))
			lbl = pad + gray(e.Label)
		}
		if i == le.cursor {
			return "  " + accentBold("▶") + " " + e.Value + lbl
		}
		return "    " + e.Value + lbl
	})
	b.WriteString("\n" + hline(w) + "\n")
	b.WriteString("  " + gray("↑↓ Enter: edit   a: add   Del: remove   Esc: done") + "\n")
	return b.String()
}

// ── Manage lists ─────────────────────────────────────────────────────────────

// writeListEntriesPreview renders the hovered list's entries (up to 5),
// shared by the Edit list pick and Delete list screens.
func (m Model) writeListEntriesPreview(b *strings.Builder, listName string, w int) {
	entries := m.lists[listName]
	b.WriteString("\n" + hlineLabel(w, "entries") + "\n")
	if len(entries) == 0 {
		b.WriteString("  " + gray("(empty)") + "\n")
		return
	}
	maxShow := min(5, len(entries))
	for i := 0; i < maxShow; i++ {
		e := entries[i]
		lbl := ""
		if e.Label != "" {
			lbl = "  " + dim(e.Label)
		}
		b.WriteString("  " + gray("·") + " " + e.Value + lbl + "\n")
	}
	if len(entries) > maxShow {
		b.WriteString("  " + dim(fmt.Sprintf("... +%d more", len(entries)-maxShow)) + "\n")
	}
}

func (m Model) viewEditListPick(w int) string {
	var b strings.Builder
	m.writeHeading(&b, "Edit list", w)
	m.writePickFilter(&b)
	names := m.sortedListNames()
	previewH := maxLines(len(names), 3, func(i int, tb *strings.Builder) {
		m.writeListEntriesPreview(tb, names[i], w)
	})
	previewH = m.previewBudget(&b, previewH, 3, 3)
	viewH := m.listBudget(&b, previewH+3)
	writeWindowedList(&b, viewH, len(m.listItems), m.listCursor, "(no lists)", func(i int) string {
		if i == m.listCursor {
			return "  " + accentBold("▶") + " " + m.listItems[i]
		}
		return "    " + m.listItems[i]
	})
	if previewH > 0 {
		var pv strings.Builder
		if len(m.listItems) > 0 && m.listCursor < len(m.listItems) {
			m.writeListEntriesPreview(&pv, m.listItems[m.listCursor], w)
		} else {
			pv.WriteString("\n" + hline(w) + "\n")
		}
		writePadded(&b, pv.String(), previewH)
	}
	b.WriteString("\n" + hline(w) + "\n")
	b.WriteString("  " + gray("↑↓ Enter: edit   Esc: back") + "\n")
	return b.String()
}

// ── Manage vars ──────────────────────────────────────────────────────────────

// writeVarHover renders the preview for a vars.tsv row: reference sites
// for a global, owner and resolved value for a saved fixed value.
func (m Model) writeVarHover(b *strings.Builder, key string, w int) {
	cmdName, slotName, scoped := scopedKey(key)
	if !scoped {
		m.writeVarRefsPreview(b, key, w)
		return
	}
	b.WriteString("\n" + hlineLabel(w, "fixed value") + "\n")
	b.WriteString("  " + gray("command:") + " " + cmdName + "   " + gray("slot:") + " " + slotName + "\n")
	if resolved := slot.ApplyVars(m.vars[key], m.vars); resolved != m.vars[key] {
		b.WriteString("  " + gray("resolves to:") + " " + resolved + "\n")
	}
	b.WriteString("  " + dim("deleting un-fixes the slot (prompted at run time)") + "\n")
}

// writeVarRefsPreview renders where the hovered variable is referenced.
func (m Model) writeVarRefsPreview(b *strings.Builder, name string, w int) {
	b.WriteString("\n" + hlineLabel(w, "referenced by") + "\n")
	refs := m.varRefLocations(name)
	if len(refs) == 0 {
		b.WriteString("  " + gray("(nothing — safe to delete)") + "\n")
		return
	}
	maxShow := min(5, len(refs))
	for i := 0; i < maxShow; i++ {
		b.WriteString("  " + gray("·") + " " + refs[i] + "\n")
	}
	if len(refs) > maxShow {
		b.WriteString("  " + dim(fmt.Sprintf("... +%d more", len(refs)-maxShow)) + "\n")
	}
}

func (m Model) viewEditVarPick(w int) string {
	var b strings.Builder
	m.writeHeading(&b, "Edit variable", w)
	m.writePickFilter(&b)
	previewH := maxLines(len(m.varPickNames), 3, func(i int, tb *strings.Builder) {
		m.writeVarHover(tb, m.varPickNames[i], w)
	})
	// All of the preview or none of it, as on every other pick screen.
	previewH = m.previewBudget(&b, previewH, 3, 3)
	viewH := m.listBudget(&b, previewH+3)
	writeWindowedList(&b, viewH, len(m.listItems), m.listCursor,
		"(vars.tsv is empty — create a global, or save a command with fixed values)", func(i int) string {
			if i == m.listCursor {
				return "  " + accentBold("▶") + " " + m.listItems[i]
			}
			return "    " + m.listItems[i]
		})
	if previewH > 0 {
		var pv strings.Builder
		if len(m.listItems) > 0 && m.listCursor < len(m.listItems) {
			m.writeVarHover(&pv, m.varPickNames[m.pickOrig(m.listCursor)], w)
		} else {
			pv.WriteString("\n" + hline(w) + "\n")
		}
		writePadded(&b, pv.String(), previewH)
	}
	b.WriteString("\n" + hline(w) + "\n")
	b.WriteString("  " + gray("↑↓ Enter: edit   Esc: back") + "\n")
	return b.String()
}

func (m Model) viewVarForm(w int) string {
	ve := m.ve
	var b strings.Builder
	title := "Create variable"
	if ve.mode == 1 {
		title = "Edit variable"
	}
	b.WriteString("\n" + header(title, w) + "\n\n")
	// The textinput renders its own "name > " / "value > " prompt; the
	// inactive field is echoed as plain text with a matching label.
	if ve.mode == 0 && ve.fieldIdx == 0 {
		b.WriteString("  " + m.nameInput.View() + "\n")
		b.WriteString("  " + gray("value >") + "\n")
	} else {
		b.WriteString("  " + gray("name  > ") + white(ve.name) + "\n")
		b.WriteString("  " + m.nameInput.View() + "\n")
	}
	if cmdName, slotName, scoped := scopedKey(ve.name); scoped {
		b.WriteString("\n  " + dim("fixed value of command \""+cmdName+"\", slot \""+slotName+"\"") + "\n")
	} else {
		b.WriteString("\n  " + dim("referenced as ") + slotVar("{$"+displayOr(ve.name, "name")+"}") +
			dim(" in cmd / workdir / list values") + "\n")
	}
	b.WriteString("\n" + hline(w) + "\n")
	b.WriteString("  " + gray("Enter: next / save   Esc: back") + "\n")
	return b.String()
}

// displayOr returns s, or fallback when s is empty.
func displayOr(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

func (m Model) viewVarRebase(w int) string {
	vr := m.vr
	var b strings.Builder
	if vr.propagate {
		b.WriteString("\n" + header("Change matching values too?", w) + "\n\n")
		// The edit that opened this window is already saved — say so, so
		// Esc clearly keeps it and only skips the values below.
		b.WriteString("  " + success("✓") + " " + white(vr.varName) + "   " +
			vr.editedOld + "  " + gray("→") + "  " + accent(vr.editedNew) + "  " + dim("(already saved)") + "\n\n")
		b.WriteString("  " + fmt.Sprintf("%d other value(s) shared the old value.", len(vr.items)) + "\n")
		b.WriteString("  " + gray("Apply the same change to the checked ones:") + "\n")
		b.WriteString("  " + dim("(tip: values that should always move together can share a") + "\n")
		b.WriteString("  " + dim(" global — Create variable (global) extracts them into {$name})") + "\n\n")
	} else {
		b.WriteString("\n" + header("Rebase values onto {$"+vr.varName+"}", w) + "\n\n")
		if vr.created {
			b.WriteString("  " + fmt.Sprintf("%d literal value(s) match the new variable's value.", len(vr.items)) + "\n")
		} else {
			b.WriteString("  " + fmt.Sprintf("%d literal value(s) start with the old value.", len(vr.items)) + "\n")
		}
		b.WriteString("  " + gray("Rewrite the checked ones to ") + slotVar("{$"+vr.varName+"}") +
			gray(" references so they follow future changes:") + "\n\n")
	}

	// Rows: kind tag, then where the value lives, then the change.
	kindTag := func(kind int) string {
		if kind == 0 {
			return "fixed value"
		}
		return "list entry"
	}
	const tagCol = len("fixed value")
	labelCol := 0
	for _, it := range vr.items {
		if lw := lipgloss.Width(it.label); lw > labelCol {
			labelCol = lw
		}
	}
	reserved := 3
	if vr.confirm {
		reserved = 7
	}
	viewH := m.listBudget(&b, reserved)
	writeWindowedList(&b, viewH, len(vr.items), vr.cursor, "", func(i int) string {
		it := vr.items[i]
		check := dim("[ ]")
		if it.on {
			check = success("[x]")
		}
		tag := kindTag(it.kind)
		tagPad := strings.Repeat(" ", tagCol-len(tag)+2)
		pad := strings.Repeat(" ", labelCol-lipgloss.Width(it.label)+2)
		row := check + " " + sGroup.Render("["+tag+"]") + tagPad + white(it.label) + pad +
			it.oldValue + "  " + gray("→") + "  " + accent(it.newValue)
		if i == vr.cursor {
			return "  " + accentBold("▶") + " " + row
		}
		return "    " + row
	})
	b.WriteString("\n" + hline(w) + "\n")
	if vr.confirm {
		b.WriteString("  " + warn(fmt.Sprintf("Apply %d change(s)?", vr.checkedCount())) + "\n\n")
		b.WriteString(renderBtns(vr.confirmBtn, "  No  ", "  Yes  ") + "\n")
		b.WriteString("\n  " + gray("Tab: switch   Enter: confirm   Esc: back") + "\n")
	} else if vr.propagate {
		b.WriteString("  " + gray("↑↓  Tab: toggle   Enter: apply   Esc: keep others") + "\n")
	} else {
		b.WriteString("  " + gray("↑↓  Tab: toggle   Enter: apply   Esc: keep literals") + "\n")
	}
	return b.String()
}

// ── Delete list (with confirmation) ──────────────────────────────────────────

func (m Model) viewDeleteList(title string, items []string, w int) string {
	var b strings.Builder
	m.writeHeading(&b, title, w)
	if !m.deleteConfirm {
		m.writePickFilter(&b)
	}
	footer := 3
	if m.deleteConfirm {
		footer = 7
	}

	// Preview region height per screen (previews hide while confirming).
	// The steps of a workflow alone run to ten rows, and unlike every
	// other pick screen these kept them whatever the terminal had left,
	// which is what made the delete screens the tallest in the app.
	previewH := 0
	if !m.deleteConfirm {
		switch m.screen {
		case ScreenDeleteWorkflow:
			previewH = 2 + m.stepsVP.Height
		case ScreenDeleteCommand:
			previewH = 1 + maxLines(len(m.editRefs), 3, func(i int, tb *strings.Builder) {
				m.writeCommandHover(tb, m.editRefCommand(m.editRefs[i]), w)
			})
		case ScreenDeleteList:
			names := m.sortedListNames()
			previewH = maxLines(len(names), 3, func(i int, tb *strings.Builder) {
				m.writeListEntriesPreview(tb, names[i], w)
			})
		case ScreenDeleteVar:
			previewH = maxLines(len(m.varPickNames), 3, func(i int, tb *strings.Builder) {
				m.writeVarHover(tb, m.varPickNames[i], w)
			})
		}
	}
	previewH = m.previewBudget(&b, previewH, footer, 3)

	selectedSet := make(map[int]bool, len(m.deleteSelected))
	for _, s := range m.deleteSelected {
		selectedSet[s] = true
	}
	viewH := m.listBudget(&b, previewH+footer)
	writeWindowedList(&b, viewH, len(items), m.listCursor, "(empty)", func(i int) string {
		item := items[i]
		check := gray("[ ]")
		if selectedSet[m.pickOrig(i)] {
			check = warn("[x]")
		}
		// Delete command rows carry the same markers as every other
		// command picker ($ template, group, {...}).
		if m.screen == ScreenDeleteCommand {
			item = m.commandPickLabel(m.editRefCommand(m.editRefs[m.pickOrig(i)]))
		}
		if i == m.listCursor {
			return "  " + accentBold("▶") + " " + check + " " + item
		}
		return "    " + check + " " + item
	})

	if previewH > 0 {
		var pv strings.Builder
		switch {
		case m.screen == ScreenDeleteWorkflow:
			pv.WriteString("\n" + hlineLabel(w, "steps") + "\n")
			pv.WriteString(m.stepsVP.View() + "\n")
		case m.screen == ScreenDeleteCommand && m.listCursor >= 0 && m.listCursor < len(items):
			pv.WriteString("\n")
			m.writeCommandHover(&pv, m.editRefCommand(m.editRefs[m.pickOrig(m.listCursor)]), w)
		case m.screen == ScreenDeleteList && m.listCursor >= 0 && m.listCursor < len(items):
			m.writeListEntriesPreview(&pv, items[m.listCursor], w)
		case m.screen == ScreenDeleteVar && m.listCursor >= 0 && m.listCursor < len(items):
			m.writeVarHover(&pv, m.varPickNames[m.pickOrig(m.listCursor)], w)
		}
		if pv.Len() == 0 {
			pv.WriteString("\n" + hline(w) + "\n")
		}
		writePadded(&b, pv.String(), previewH)
	}
	b.WriteString("\n" + hline(w) + "\n")
	if m.deleteConfirm {
		n := len(m.deleteSelected)
		msg := fmt.Sprintf("Delete %d item", n)
		if n > 1 {
			msg += "s"
		}
		b.WriteString("  " + warn(msg+"?") + "\n\n")
		b.WriteString(renderBtns(m.deleteBtn, "  No  ", "  Yes  ") + "\n")
		b.WriteString("\n  " + gray("Tab: switch   Enter: confirm   Esc: back") + "\n")
	} else {
		b.WriteString("  " + gray("↑↓ Tab: toggle   Enter: confirm   Esc: back") + "\n")
	}
	return b.String()
}

// ── Run workflow: per-run step selection ──────────────────────────────────────

func (m Model) viewRunWorkflowStepsList(w int, hitRow ...func(int, int)) string {
	if m.shortPanels() {
		return m.shortWorkflowSteps(w, hitRow...)
	}
	var b strings.Builder
	p := m.wfp
	if p == nil || p.wfIdx >= len(m.workflows) {
		return ""
	}
	wf := m.workflows[p.wfIdx]
	n := len(wf.Commands)

	m.writeHeading(&b, "Run workflow: "+wf.Name, w)
	b.WriteString("  " + gray(truncate("Pick steps to run · saved order", max(1, w-2))) + "\n")

	stepW := max(4, len(fmt.Sprint(n)))
	nameW := min(20, max(1, w-2-4-stepW-1))
	showCommand := w >= 44
	if showCommand {
		nameW = min(20, w/4)
	}
	commandW := max(1, w-2-4-stepW-1-nameW-2)
	heading := fmt.Sprintf("%-3s %*s %-*s", "Run", stepW, "Step", nameW, "Name")
	if showCommand {
		heading += "  Command"
	}
	b.WriteString("  " + gray(truncate(heading, max(1, w-2))) + "\n")
	viewH := m.listBudget(&b, 4)
	writeWindowedList(&b, viewH, n, p.cursor, "(no steps)", func(i int) string {
		for _, hit := range hitRow {
			hit(i, strings.Count(b.String(), "\n"))
		}
		name := wf.Commands[i]
		box := "[ ]"
		if p.picked[i] {
			box = accent("[x]")
		}
		note := ""
		if cmd, ok := m.workflowStepCommand(name); ok {
			note = m.syntaxText(truncate("$ "+cmd.Cmd, commandW))
		} else {
			box = dim("[-]")
			note = warn(truncate("(not found)", commandW))
		}
		name = truncate(name, nameW)
		name += strings.Repeat(" ", max(0, nameW-lipgloss.Width(name)))
		row := fmt.Sprintf("%s %*d %s", box, stepW, i+1, name)
		if showCommand {
			row += "  " + note
		}
		if i == p.cursor {
			return m.selectedSyntaxRow("▶ "+row, max(1, w))
		}
		return ansi.Truncate("  "+row, max(1, w), "…")
	})

	b.WriteString("\n" + hline(w) + "\n")
	if c := p.count(); c > 0 {
		var picked []string
		for i, name := range wf.Commands {
			if p.picked[i] {
				picked = append(picked, name)
			}
		}
		b.WriteString("  " + success(fmt.Sprintf("Selected(%d)", c)) + ": " +
			truncate(strings.Join(picked, " → "), max(8, w-20)) + "\n")
	} else {
		b.WriteString("  " + gray(truncate("Select at least one step with Tab to review", max(1, w-2))) + "\n")
	}
	keys := "↑↓ Select  Tab Toggle  Enter Review  Esc Back"
	if !m.workflowDetailsInline() {
		keys += "  F2 Details"
	}
	b.WriteString("  " + gray(truncate(keys, max(1, w-2))) + "\n")
	return b.String()
}

// ── Rename repair ─────────────────────────────────────────────────────────────

func (m Model) viewRenameRepair(w int) string {
	var b strings.Builder
	b.WriteString("\n" + header("Rename detected", w) + "\n\n")
	b.WriteString("  " + warn(fmt.Sprintf("%d command(s) look renamed outside baton:", len(m.renames))) + "\n\n")
	for _, r := range m.renames {
		var refs []string
		if r.WfSteps > 0 {
			refs = append(refs, fmt.Sprintf("%d workflow step(s)", r.WfSteps))
		}
		if r.TplRefs > 0 {
			refs = append(refs, fmt.Sprintf("%d derived command(s)", r.TplRefs))
		}
		if r.VarKeys > 0 {
			refs = append(refs, fmt.Sprintf("%d vars.tsv row(s)", r.VarKeys))
		}
		b.WriteString("    " + r.Old + " " + accentBold("→") + " " + r.New + "   " + gray(strings.Join(refs, ", ")) + "\n")
	}
	b.WriteString("\n  Update these references to the new names?\n\n")
	b.WriteString(renderBtns(m.renameBtn, "  No  ", "  Yes  ") + "\n")
	b.WriteString("\n" + hline(w) + "\n")
	b.WriteString("  " + gray("←→ Tab: switch   Enter: confirm   Esc: keep as is") + "\n")
	return b.String()
}

// ── Manage commands ───────────────────────────────────────────────────────────

func (m Model) viewManageCommands(w int) string {
	var b strings.Builder
	b.WriteString("\n" + header("Manage commands", w) + "\n\n")
	items := []string{"Create command", "Edit command", "Delete command"}
	for i, item := range items {
		if i == m.listCursor {
			b.WriteString("  " + accentBold("▶") + " " + item + "\n")
		} else {
			b.WriteString("    " + item + "\n")
		}
	}
	b.WriteString("\n" + hline(w) + "\n")
	b.WriteString("  " + gray("↑↓ Enter: select   Esc: back") + "\n")
	return b.String()
}

func (m Model) viewCreateCommand(w int) string {
	if m.sce == nil {
		return ""
	}
	sce := m.sce
	if m.screen == ScreenCreateCommandTemplate {
		return m.viewTemplatePick(w, sce.templateRefIdx)
	}
	return m.viewCommandNameInput("Create command", w, sce)
}

func (m Model) viewTemplatePick(w, cursor int) string {
	var b strings.Builder
	b.WriteString("\n" + header("Select template", w) + "\n\n")
	candidates := m.templateCandidates()
	if len(candidates) == 0 {
		b.WriteString("  " + gray("(no commands with {slots} defined)") + "\n")
	}
	for i, cmd := range candidates {
		if i == cursor {
			b.WriteString("  " + accentBold("▶") + " " + cmd.Name + "   " + gray("$ "+cmd.Cmd) + "\n")
		} else {
			b.WriteString("    " + cmd.Name + "   " + gray("$ "+cmd.Cmd) + "\n")
		}
	}
	b.WriteString("\n" + hline(w) + "\n")
	b.WriteString("  " + gray("↑↓ Enter: select   Esc: back") + "\n")
	return b.String()
}

func (m Model) viewCommandNameInput(title string, w int, sce *commandEditState) string {
	var b strings.Builder
	b.WriteString("\n" + header(title, w) + "\n\n")
	b.WriteString("  " + m.nameInput.View() + "\n")
	if candidates := m.templateCandidates(); sce.templateRefIdx >= 0 && sce.templateRefIdx < len(candidates) {
		b.WriteString("  Template: " + candidates[sce.templateRefIdx].Name + "\n")
	}
	if len(sce.currentValues) > 0 {
		b.WriteString("\n" + hlineLabel(w, "values") + "\n")
		for _, k := range sortedKeys(sce.currentValues) {
			b.WriteString("  " + gray(k+" = "+sce.currentValues[k]) + "\n")
		}
	}
	b.WriteString("\n" + hline(w) + "\n")
	hint := "Enter: confirm   Esc: back"
	if m.suggestCommandName() != "" {
		hint = "Tab: suggest name   " + hint
	}
	b.WriteString("  " + gray(hint) + "\n")
	return b.String()
}

func (m Model) viewCommandForm(w int) string {
	if m.cf == nil {
		return ""
	}
	cf := m.cf
	title := "Create command"
	if cf.mode == 1 {
		title = "Edit command"
	}
	if m.formDirty() {
		title += "  ● unsaved"
	}
	var b strings.Builder
	b.WriteString("\n" + header(title, w) + "\n\n")
	// The same rows the box beside the list draws, with the same marks
	// on changed fields and the same error under the field it is about.
	m.writeCommandFields(&b, cf.fields, cf.fieldIdx, w-4)

	if m.leaveAsk != leaveNone {
		b.WriteString("\n")
		m.writeLeaveAsk(&b)
		b.WriteString("\n" + hline(w) + "\n")
		b.WriteString("  " + gray("↑↓ Choose  Enter Answer  s / d / k") + "\n")
		return b.String()
	}
	if cf.pickOpen {
		// What is left under the fields once the rule and the key line
		// below, the blank around the candidates and the count line are
		// paid for — so the popup never pushes the frame past the bottom.
		rows := m.termHeight() - strings.Count(b.String(), "\n") - 6
		m.writePickCandidates(&b, cf, w-4, rows)
	} else if lines := m.slotValidationLines(cf); len(lines) > 0 {
		b.WriteString("\n")
		for _, l := range lines {
			b.WriteString("  " + l + "\n")
		}
	}

	b.WriteString("\n" + hline(w) + "\n")
	guide := m.formKeys()
	b.WriteString("  " + gray(guide) + "\n")
	return b.String()
}

// slotValidationLines checks every {slot} in the cmd/workdir fields against
// the selection lists, so typos surface while typing instead of at run time.
func (m Model) slotValidationLines(cf *commandFormState) []string {
	cmdStr, dirStr := cf.fields[1], cf.fields[2]
	switch cf.fieldIdx {
	case 1:
		cmdStr = m.nameInput.Value()
	case 2:
		dirStr = m.nameInput.Value()
	}
	probe := mdl.Command{Cmd: cmdStr, Dir: dirStr}
	var lines []string
	for _, s := range slot.GetSlots(probe) {
		if s.PathKind != "" {
			lines = append(lines, gray(s.Placeholder()+" → "+s.PathKind+" picker"))
		} else if _, ok := m.lists[s.ListName]; ok {
			lines = append(lines, success("✓")+" "+gray(s.Placeholder()+" → "+s.ListName))
		} else {
			lines = append(lines, warn("⚠")+" "+gray(s.Placeholder()+" → ")+warn("no list (free input at run time)"))
		}
	}
	return lines
}

func (m Model) viewEditCommand(w int) string {
	if m.sce == nil || m.sce.editIdx < 0 || m.sce.editIdx >= len(m.config.Commands) {
		return ""
	}
	sce := m.sce
	if m.screen == ScreenEditCommandTemplate {
		return m.viewTemplatePick(w, sce.templateRefIdx)
	}
	return m.viewCommandNameInput("Edit command", w, sce)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// sortedKeys returns the map's keys in stable order. Views must never range
// over a map directly: Go randomizes iteration order per run, so lines would
// swap on every re-render (visible as flickering while a cursor blinks).
func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
