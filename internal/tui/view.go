package tui

import (
	"fmt"
	"os"
	"path/filepath"
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
	var view string
	switch m.screen {
	case ScreenProjectSelect:
		view = m.viewProjectSelect(w)
	case ScreenMainMenu:
		view = m.viewMainMenu(w)
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
		view = m.viewSingleSelect("Switch config", w)
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
	}
	if m.errMsg != "" {
		view += "\n" + errorText("  Error: "+m.errMsg) + "\n"
	}
	if m.successMsg != "" {
		view += "\n" + success("  ✓ "+m.successMsg) + "\n"
	}
	if len(m.loadWarnings) > 0 && m.screen == ScreenMainMenu {
		const maxShown = 4
		shown := m.loadWarnings
		if len(shown) > maxShown {
			shown = shown[:maxShown]
		}
		view += "\n"
		for _, wmsg := range shown {
			view += warn("  Warning: "+wmsg) + "\n"
		}
		if rest := len(m.loadWarnings) - maxShown; rest > 0 {
			view += warn(fmt.Sprintf("  ... and %d more (run: baton check)", rest)) + "\n"
		}
	}
	return view
}

// ── Project select / main menu ───────────────────────────────────────────────

// logoBox renders the boxed app name. The name defaults to BATON and can be
// overridden with the BATON_DISPLAY_NAME environment variable (ASCII
// recommended — wide characters break the box alignment).
func logoBox() []string {
	name := os.Getenv("BATON_DISPLAY_NAME")
	if name == "" {
		name = "BATON"
	}
	runes := []rune(strings.ToUpper(name))
	parts := make([]string, len(runes))
	for i, r := range runes {
		parts[i] = string(r)
	}
	label := strings.Join(parts, " ")
	innerW := len([]rune(label)) + 4
	return []string{
		"  ┌" + strings.Repeat("─", innerW) + "┐",
		"  │  " + bold(white(label)) + "  │",
		"  └" + strings.Repeat("─", innerW) + "┘",
	}
}

func (m Model) viewProjectSelect(w int) string {
	var b strings.Builder
	b.WriteString("\n")
	for _, line := range logoBox() {
		b.WriteString(line + "\n")
	}
	b.WriteString("  " + gray("Command Runner") + "\n\n")
	b.WriteString(hlineLabel(w, "Projects") + "\n\n")
	viewH := m.listBudget(&b, 3)
	writeWindowedList(&b, viewH, len(m.projects), m.listCursor, "(no projects)", func(i int) string {
		p := m.projects[i]
		count := fmt.Sprintf("%d commands", m.projectCmdCounts[p])
		if i == m.listCursor {
			return "  " + accentBold("▶") + " " + p + "   " + gray(count)
		}
		return "    " + p + "   " + gray(count)
	})
	b.WriteString("\n" + hline(w) + "\n")
	b.WriteString("  " + gray("↑↓ move   Enter select   q quit") + "\n")
	return b.String()
}

type menuItemInfo struct {
	desc      string
	shortcuts [][2]string
}

// mainMenuGroups is the single source of truth for the main menu.
// viewMainMenu renders it and gotoMainMenu flattens it into listItems,
// so the rendered rows and the Enter dispatch can never drift apart.
type menuGroup struct {
	label string
	items []string
}

var mainMenuGroups = []menuGroup{
	{"Run", []string{"Run workflow", "Run commands"}},
	{"Manage", []string{"Manage workflows", "Manage commands", "Manage lists", "Manage vars"}},
	{"", []string{"Switch config", "Exit"}},
}

func mainMenuItems() []string {
	var items []string
	for _, g := range mainMenuGroups {
		items = append(items, g.items...)
	}
	return items
}

var menuItemInfos = map[string]menuItemInfo{
	"Run workflow":     {desc: "Run a saved workflow.", shortcuts: [][2]string{{"Enter", "Run"}, {"Esc", "Back"}}},
	"Run commands":     {desc: "Pick commands and run them once.", shortcuts: [][2]string{{"Tab", "Select"}, {"Enter", "Run"}, {"Esc", "Back"}}},
	"Manage workflows": {desc: "Create, edit or delete workflows.", shortcuts: [][2]string{{"Enter", "Open"}, {"Esc", "Back"}}},
	"Manage commands":  {desc: "Create commands from templates, edit or delete them.", shortcuts: [][2]string{{"Enter", "Open"}, {"Esc", "Back"}}},
	"Manage lists":     {desc: "Create, edit or delete selection lists for placeholders.", shortcuts: [][2]string{{"Enter", "Open"}, {"Esc", "Back"}}},
	"Manage vars":      {desc: "Project variables ({$name}): change one value to move every reference at once.", shortcuts: [][2]string{{"Enter", "Open"}, {"Esc", "Back"}}},
	"Switch config":    {desc: "Switch to a different project.", shortcuts: [][2]string{{"Enter", "Switch"}, {"Esc", "Back"}}},
	"Exit":             {desc: "Quit.", shortcuts: [][2]string{{"Enter", "Quit"}}},
}

func (m Model) viewMainMenu(w int) string {
	groups := mainMenuGroups

	leftW := 32
	showRight := w >= leftW+30

	// Build left pane lines
	var leftLines []string
	leftLines = append(leftLines, "")
	leftLines = append(leftLines, logoBox()...)
	if m.projectDir != "" {
		leftLines = append(leftLines, "  "+gray("project: ")+white(filepath.Base(m.projectDir)))
	} else {
		leftLines = append(leftLines, "")
	}
	leftLines = append(leftLines, "")

	idx := 0
	for _, g := range groups {
		if g.label != "" {
			leftLines = append(leftLines, "  "+gray(strings.ToUpper(g.label)))
		}
		for _, item := range g.items {
			if idx == m.listCursor {
				leftLines = append(leftLines, "    "+sMenuSelect.Render(" "+item))
			} else {
				leftLines = append(leftLines, "      "+item)
			}
			idx++
		}
		leftLines = append(leftLines, "")
	}

	if !showRight {
		var b strings.Builder
		for _, l := range leftLines {
			b.WriteString(l + "\n")
		}
		b.WriteString(hline(w) + "\n")
		b.WriteString("  " + gray("↑↓ Move  Enter Select  Esc Quit") + "\n")
		return b.String()
	}

	// Build right pane lines
	rightW := w - leftW - 2
	var rightLines []string
	rightLines = append(rightLines, "")
	rightLines = append(rightLines, "")
	rightLines = append(rightLines, "")
	rightLines = append(rightLines, "")

	selected := ""
	flatIdx := 0
	for _, g := range groups {
		for _, item := range g.items {
			if flatIdx == m.listCursor {
				selected = item
			}
			flatIdx++
		}
	}

	// The pane is padded to the same height for every item (longest
	// description and shortcut list win), so moving the cursor never
	// changes the frame height and the footer stays put.
	maxShortcuts := 0
	for _, info := range menuItemInfos {
		if len(info.shortcuts) > maxShortcuts {
			maxShortcuts = len(info.shortcuts)
		}
	}

	if info, ok := menuItemInfos[selected]; ok {
		desc := truncate(info.desc, max(10, rightW-2))
		rightLines = append(rightLines, white(selected))
		rightLines = append(rightLines, gray(strings.Repeat("─", min(rightW-2, 24))))
		rightLines = append(rightLines, dim(desc))
		rightLines = append(rightLines, "")
		rightLines = append(rightLines, gray("Keys"))
		for _, sc := range info.shortcuts {
			rightLines = append(rightLines, fmt.Sprintf("  %-8s %s", white(sc[0]), gray(sc[1])))
		}
		for i := len(info.shortcuts); i < maxShortcuts; i++ {
			rightLines = append(rightLines, "")
		}
		rightLines = append(rightLines, "")
		rightLines = append(rightLines, gray("Config"))
		rightLines = append(rightLines, "  "+gray("project: ")+white(filepath.Base(m.projectDir)))
		if m.configFile != "" {
			rightLines = append(rightLines, "  "+gray("file:    ")+white(m.configFile))
		}
		rightLines = append(rightLines, "")
		rightLines = append(rightLines, gray("Stats"))
		rightLines = append(rightLines, fmt.Sprintf("  "+gray("workflows: ")+white("%d"), len(m.workflows)))
		rightLines = append(rightLines, fmt.Sprintf("  "+gray("lists:     ")+white("%d"), len(m.lists)))
	}

	// Merge left + right
	n := max(len(leftLines), len(rightLines))
	var b strings.Builder
	dividerCol := leftW
	for i := 0; i < n; i++ {
		l := ""
		if i < len(leftLines) {
			l = leftLines[i]
		}
		r := ""
		if i < len(rightLines) {
			r = rightLines[i]
		}
		lVis := lipgloss.Width(l)
		pad := strings.Repeat(" ", max(0, dividerCol-lVis))
		b.WriteString(l + pad + gray("│") + " " + r + "\n")
	}

	b.WriteString(hline(w) + "\n")
	b.WriteString("  " + gray("↑↓ Move  Enter Select  Esc Quit") + "\n")
	return b.String()
}

// ── Generic single-select ────────────────────────────────────────────────────

func (m Model) viewSingleSelect(title string, w int) string {
	var b strings.Builder
	b.WriteString("\n" + header(title, w) + "\n")
	if m.screen == ScreenEditWorkflow {
		m.writePickFilter(&b)
	}
	reserved := 3
	if m.screen == ScreenEditWorkflow {
		reserved += 2 + m.stepsVP.Height
	}
	viewH := m.listBudget(&b, reserved)
	writeWindowedList(&b, viewH, len(m.listItems), m.listCursor, "(empty)", func(i int) string {
		if i == m.listCursor {
			return "  " + accentBold("▶") + " " + m.listItems[i]
		}
		return "    " + m.listItems[i]
	})
	if m.screen == ScreenEditWorkflow {
		b.WriteString("\n" + hlineLabel(w, "steps") + "\n")
		b.WriteString(m.stepsVP.View() + "\n")
	}
	b.WriteString("\n" + hline(w) + "\n")
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

// workflowLabel renders a workflow's name for a list row, colouring the
// joiners of a generated name so the step names read as separate chunks.
//
// The joiner has to outrank the hyphens inside step names — build-src
// carries one of its own — so it is marked rather than faded: fading put
// the boundary that matters below the one that does not. Gold is the
// emphasis colour here and, unlike the accent, is not already spoken for
// by the cursor on this screen.
func (m Model) workflowLabel(idx int, hovered bool) string {
	style := func(s string) string { return s }
	if hovered {
		style = bold
	}
	name := m.workflows[idx].Name
	if !m.workflowNameIsGenerated(idx) {
		return style(name)
	}
	parts := strings.Split(name, "+")
	for i, p := range parts {
		parts[i] = style(p)
	}
	return strings.Join(parts, highlight("+"))
}

func (m Model) viewRunWorkflow(w int) string {
	var b strings.Builder
	b.WriteString("\n" + header("Run workflow", w) + "\n")

	if len(m.workflows) == 0 {
		b.WriteString("  " + gray("(no workflows saved)") + "\n")
		b.WriteString("\n" + hline(w) + "\n")
		b.WriteString("  " + gray("Esc: back") + "\n")
		return b.String()
	}

	filtered := m.wfFiltered()
	n := len(filtered)
	b.WriteString("  " + m.wfSearchTI.View() + "  " + dim(fmt.Sprintf("%d/%d", n, len(m.workflows))) + "\n\n")
	cur := m.listCursor
	if cur >= n {
		cur = max(0, n-1)
	}

	// The steps viewport below has a fixed height, so budget against it
	// instead of the hovered workflow's step count — a per-item budget
	// makes the list grow and shrink as the cursor moves.
	viewH := m.listBudget(&b, m.stepsVP.Height+5)
	writeWindowedList(&b, viewH, n, cur, "No results.", func(i int) string {
		wf := m.workflows[filtered[i]]
		suffix := ""
		if wf.Name == m.lastWorkflow {
			suffix = "  " + gray("(last)")
		}
		if i == cur {
			return "  " + accentBold("▶") + " " + m.workflowLabel(filtered[i], true) + suffix
		}
		return "    " + m.workflowLabel(filtered[i], false) + suffix
	})

	// Step preview for hovered workflow (scrollable viewport)
	if m.stepsFocused {
		b.WriteString("\n" + hlineLabel(w, "steps ↑↓") + "\n")
	} else {
		b.WriteString("\n" + hlineLabel(w, "steps") + "\n")
	}
	b.WriteString(m.stepsVP.View() + "\n")

	b.WriteString("\n" + hline(w) + "\n")
	b.WriteString("  " + gray("Type to search  ↑↓ Enter Esc  Tab: focus steps  →: pick steps") + "\n")
	return b.String()
}

// ── Multi-select ─────────────────────────────────────────────────────────────

func (m Model) viewMultiSelect(w int) string {
	title := "Select commands"
	if m.screen == ScreenEditWorkflowCommands {
		title = "Edit commands"
	}
	filtered := m.msFiltered()
	n := len(filtered)

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
	b.WriteString("\n" + header(title, w) + "\n")

	b.WriteString("  " + m.msSearchTI.View() + "  " + dim(fmt.Sprintf("%d/%d", n, len(m.msItems))) + "\n\n")

	viewH := m.listBudget(&b, previewH+5)
	writeWindowedList(&b, viewH, n, cursor, "No results.", func(i int) string {
		origIdx := filtered[i]
		item := m.msItems[origIdx]

		selOrder := -1
		for j, s := range m.msSelected {
			if s == origIdx {
				selOrder = j
				break
			}
		}
		check := gray("[ ]")
		if selOrder >= 0 {
			check = sSelNum.Render(fmt.Sprintf("[%d]", selOrder+1))
		}

		var label string
		if item.cmd.Template != "" {
			grp := ""
			if item.cmd.Group != "" {
				grp = "  " + sGroup.Render("["+item.cmd.Group+"]")
			}
			label = accent("$") + " " + item.cmd.Name + grp + "  " + gray("("+item.cmd.Template+")")
		} else {
			grp := ""
			if item.cmd.Group != "" {
				grp = "  " + sGroup.Render("["+item.cmd.Group+"]")
			}
			hasVars := ""
			if slot.HasPlaceholders(*item.cmd) {
				hasVars = "  " + gray("{...}")
			}
			label = item.cmd.Name + grp + hasVars
		}

		if i == cursor {
			return "  " + accentBold("▶") + " " + check + " " + label
		}
		return "    " + check + " " + label
	})

	// Hover preview, padded to previewH so the footer below never moves.
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

	var selNames []string
	for _, idx := range m.msSelected {
		selNames = append(selNames, m.msItems[idx].name())
	}
	orderHint := ""
	if len(m.msSelected) > 0 {
		orderHint = "  " + dim("[n] = run order")
	}
	// Keep the Selected line to one row: a wrapped line would grow the
	// frame past the terminal height and bring the jitter back.
	joined := truncate(strings.Join(selNames, ", "), max(10, w-40))
	b.WriteString("\n  " + success(fmt.Sprintf("Selected(%d)", len(m.msSelected))) + orderHint + ": " + joined + "\n")
	b.WriteString(hline(w) + "\n")
	b.WriteString("  " + gray("↑↓ Move  Tab Select  Enter Confirm  Esc Back") + "\n")

	// Discard guard: opens as a centered floating window over the list.
	if m.msEscArmed && len(m.msSelected) > 0 {
		return m.viewDiscardWindow(w, selNames)
	}
	return b.String()
}

// viewDiscardWindow renders the discard confirmation as a centered floating
// window (same style as the placeholder picker): shown after Esc is pressed
// with items still selected. Esc again discards; any other key keeps the
// selection and returns to the list.
func (m Model) viewDiscardWindow(w int, selNames []string) string {
	names := strings.Join(selNames, ", ")
	innerW := min(max(lipgloss.Width(names), 40), max(10, w-10))
	content := warn(" Discard selection? ") + "\n\n" +
		fmt.Sprintf("%d selected:", len(selNames)) + "\n" +
		dim(lipgloss.NewStyle().Width(innerW).Render(names)) + "\n\n" +
		gray("Esc: discard   any other key: keep")

	win := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("220")).
		Padding(0, 2).
		Render(content)

	h := m.height
	if h == 0 {
		h = 24
	}
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, win)
}

// ── Slot pick ────────────────────────────────────────────────────────────────

func (m Model) viewSlotPick(w int) string {
	sp := m.sp
	var b strings.Builder
	title := "  [ Select value for " + sp.placeholder() + " ]"
	if sp.variadic {
		title = "  [ Select values for " + sp.placeholder() + " ]"
	}
	b.WriteString("\n" + accentBold(title) + "\n" + hline(w) + "\n\n")

	// Context panel — windowed: show up to 3 before current, current, up to 2 after
	if sp.contextNames != nil {
		cur := sp.contextIdx
		n := len(sp.contextNames)
		start := max(0, cur-3)
		end := min(n, cur+3)

		if start > 0 {
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
		if end < n {
			b.WriteString("  " + dim(fmt.Sprintf("    ↓ %d more", n-end)) + "\n")
		}

		// Command preview — separated from the list
		if sp.currentCmd != nil {
			b.WriteString("\n" + hlineLabelBright(w, "command preview  "+highlight(sp.placeholder())) + "\n\n")

			// Hovered value (from list cursor or custom search input)
			hoveredVal := ""
			if sp.cursor >= 0 && sp.cursor < len(sp.filtered) {
				hoveredVal = sp.filtered[sp.cursor].Value
			} else if sp.cursor == len(sp.filtered) && sp.search != "" {
				hoveredVal = sp.search
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
		b.WriteString("\n\n" + hlineLabelBright(w, selectLabel) + "\n\n")
	}

	// Search field
	if sp.search == "" {
		b.WriteString("  " + dim("/ type to filter...") + "\n\n")
	} else {
		countStr := ""
		if len(sp.filtered) == 0 {
			countStr = warn("no match")
		} else {
			countStr = dim(fmt.Sprintf("%d results", len(sp.filtered)))
		}
		b.WriteString("  " + accent("/") + " " + white(sp.search) + dim("_") +
			"  " + countStr + "\n\n")
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
				if sp.search != "" {
					rawLine = "[ + " + sp.search + "  (custom) ]"
				} else {
					rawLine = "[ + custom value ]"
				}
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
			if sp.search != "" {
				line = accent("[") + " + " + white(sp.search) + "  " + dim("(custom)") + accent(" ]")
			} else {
				line = dim("[ + custom value ]")
			}
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
	if m.pickSearch == "" {
		b.WriteString("  " + dim("/ type to filter...") + "\n\n")
		return
	}
	count := dim(fmt.Sprintf("%d results", len(m.listItems)))
	if len(m.listItems) == 0 {
		count = warn("no match")
	}
	b.WriteString("  " + accent("/") + " " + white(m.pickSearch) + dim("_") + "  " + count + "\n\n")
}

// ── Fixed-height windowed lists ──────────────────────────────────────────────
//
// Scrolling screens share one discipline: the row window and the hover
// preview are padded to constant heights, so the frame never exceeds the
// terminal (an overflowing frame makes the terminal itself scroll on every
// repaint — visible jitter) and nothing below the list shifts as the
// cursor moves.

// listBudget returns the row budget for a windowed list: the terminal
// height minus what the frame already holds, the two scroll-marker lines,
// what is still to come below the list, and one spare line.
func (m Model) listBudget(b *strings.Builder, reservedBelow int) int {
	h := m.height
	if h == 0 {
		h = 24
	}
	return max(1, h-strings.Count(b.String(), "\n")-2-reservedBelow-1)
}

// writeWindowedList renders a cursor-following window over n rows framed
// by count-aware scroll markers, padded to exactly viewH+2 lines.
// render(i) returns row i without a trailing newline; emptyMsg is shown
// when the list is empty.
func writeWindowedList(b *strings.Builder, viewH, n, cursor int, emptyMsg string, render func(i int) string) {
	viewStart := max(0, min(cursor-viewH/2, n-viewH))
	viewEnd := min(viewStart+viewH, n)
	if viewStart > 0 {
		b.WriteString("  " + dim(fmt.Sprintf("↑ %d more", viewStart)) + "\n")
	} else {
		b.WriteString("\n")
	}
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
	if viewEnd < n {
		b.WriteString("  " + dim(fmt.Sprintf("↓ %d more", n-viewEnd)) + "\n")
	} else {
		b.WriteString("\n")
	}
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
		b.WriteString("  " + gray("$ "+cmd.Cmd) + "\n")
		return
	}
	b.WriteString(hlineLabel(w, "command preview") + "\n")
	cmdStr := truncate(slot.ApplyVars(cmd.Cmd, m.vars), max(10, w-10))
	b.WriteString("  " + gray("$ "+cmdStr) + "\n")
	workdir := cmd.Dir
	if workdir == "" {
		workdir = "."
	}
	b.WriteString("  " + dim("workdir: "+workdir) + "\n")
}

// ── Edit command pick ────────────────────────────────────────────────────────

// viewEditCommandPick lists the editable commands (TSV rows and
// template-derived ones) with the same hover preview as the command
// selector, so you can see what a command does before opening it.
func (m Model) viewEditCommandPick(w int) string {
	var b strings.Builder
	b.WriteString("\n" + header("Edit command", w) + "\n")
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
	viewH := m.listBudget(&b, previewH+4)
	writeWindowedList(&b, viewH, len(m.listItems), m.listCursor, "", func(i int) string {
		label := m.commandPickLabel(m.editRefCommand(m.editRefs[m.pickOrig(i)]))
		if i == m.listCursor {
			return "  " + accentBold("▶") + " " + label
		}
		return "    " + label
	})
	b.WriteString("\n")
	var pv strings.Builder
	if m.listCursor >= 0 && m.listCursor < len(m.listItems) {
		m.writeCommandHover(&pv, m.editRefCommand(m.editRefs[m.pickOrig(m.listCursor)]), w)
	} else {
		pv.WriteString(hline(w) + "\n")
	}
	writePadded(&b, pv.String(), previewH)
	b.WriteString("\n" + hline(w) + "\n")
	b.WriteString("  " + gray("↑↓ Enter Esc") + "\n")
	return b.String()
}

// ── Confirm run ──────────────────────────────────────────────────────────────

// confirmRunPerPage returns how many three-line items fit on the confirm
// screen. The view and the ↑↓ handler share it so scrolling stops exactly
// at the last page.
func (m Model) confirmRunPerPage() int {
	var b strings.Builder
	b.WriteString("\n" + header("Confirm", m.width) + "\n")
	return max(1, m.listBudget(&b, 6)/3)
}

func (m Model) viewConfirmRun(w int) string {
	var b strings.Builder
	b.WriteString("\n" + header("Confirm", w) + "\n")

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
			b.WriteString("       " + gray("$ "+item.Cmd.Cmd) + "\n")
			workdir := item.Cmd.Dir
			if workdir == "" {
				workdir = "."
			}
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

	b.WriteString("\n" + hline(w) + "\n\n")
	b.WriteString(renderBtns(m.confirmRunBtn, "  Run  ", "  Cancel  ") + "\n")
	b.WriteString("\n  " + gray("↑↓: scroll   Tab: switch   Enter: confirm   Esc: back") + "\n")
	return b.String()
}

// ── Running ──────────────────────────────────────────────────────────────────

func (m Model) viewRunning(w int) string {
	r := m.running
	if r == nil {
		return ""
	}
	if !r.completed {
		return ""
	}
	var b strings.Builder
	n := len(r.items)
	b.WriteString("\n" + successBold("  [ Done ]") + "  " + gray(fmt.Sprintf("%d/%d", n, n)) + "\n\n")
	b.WriteString("  " + gray("Press any key to return to menu...") + "\n")
	return b.String()
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

	b.WriteString("  " + gray("Del: remove   Esc: done") + "\n\n")
	labelCol := 0
	for _, e := range le.entries {
		if e.Label != "" && len(e.Value)+2 > labelCol {
			labelCol = len(e.Value) + 2
		}
	}
	// The "+ Add value" row scrolls with the entries as the last row.
	viewH := m.listBudget(&b, 3)
	writeWindowedList(&b, viewH, len(le.entries)+1, le.cursor, "", func(i int) string {
		if i == len(le.entries) {
			if i == le.cursor {
				return "  " + accentBold("▶") + " " + success("+ Add value")
			}
			return "    " + gray("+ Add value")
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
	b.WriteString("  " + gray("↑↓ Enter: edit   Del: remove   Esc: done") + "\n")
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
	b.WriteString("\n" + header("Edit list", w) + "\n")
	m.writePickFilter(&b)
	names := m.sortedListNames()
	previewH := maxLines(len(names), 3, func(i int, tb *strings.Builder) {
		m.writeListEntriesPreview(tb, names[i], w)
	})
	viewH := m.listBudget(&b, previewH+3)
	writeWindowedList(&b, viewH, len(m.listItems), m.listCursor, "(no lists)", func(i int) string {
		if i == m.listCursor {
			return "  " + accentBold("▶") + " " + m.listItems[i]
		}
		return "    " + m.listItems[i]
	})
	var pv strings.Builder
	if len(m.listItems) > 0 && m.listCursor < len(m.listItems) {
		m.writeListEntriesPreview(&pv, m.listItems[m.listCursor], w)
	} else {
		pv.WriteString("\n" + hline(w) + "\n")
	}
	writePadded(&b, pv.String(), previewH)
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
	b.WriteString("\n" + header("Edit variable", w) + "\n")
	m.writePickFilter(&b)
	previewH := maxLines(len(m.varPickNames), 3, func(i int, tb *strings.Builder) {
		m.writeVarHover(tb, m.varPickNames[i], w)
	})
	viewH := m.listBudget(&b, previewH+3)
	writeWindowedList(&b, viewH, len(m.listItems), m.listCursor,
		"(vars.tsv is empty — create a global, or save a command with fixed values)", func(i int) string {
			if i == m.listCursor {
				return "  " + accentBold("▶") + " " + m.listItems[i]
			}
			return "    " + m.listItems[i]
		})
	var pv strings.Builder
	if len(m.listItems) > 0 && m.listCursor < len(m.listItems) {
		m.writeVarHover(&pv, m.varPickNames[m.pickOrig(m.listCursor)], w)
	} else {
		pv.WriteString("\n" + hline(w) + "\n")
	}
	writePadded(&b, pv.String(), previewH)
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
	b.WriteString("\n" + header(title, w) + "\n")
	if !m.deleteConfirm {
		m.writePickFilter(&b)
	}
	// Preview region height per screen (previews hide while confirming).
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
	footer := 3
	if m.deleteConfirm {
		footer = 7
	}

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

func (m Model) viewRunWorkflowSteps(w int) string {
	var b strings.Builder
	p := m.wfp
	if p == nil || p.wfIdx >= len(m.workflows) {
		return ""
	}
	wf := m.workflows[p.wfIdx]
	n := len(wf.Commands)

	b.WriteString("\n" + header("Run workflow: "+wf.Name, w) + "\n")
	b.WriteString("  " + gray("Pick the steps to run — they always run in the order below.") + "\n")

	viewH := m.listBudget(&b, 4)
	writeWindowedList(&b, viewH, n, p.cursor, "(no steps)", func(i int) string {
		name := wf.Commands[i]
		box := "[ ]"
		if p.picked[i] {
			box = accent("[x]")
		}
		note := ""
		if cmd, ok := m.workflowStepCommand(name); ok {
			// Same one-line shape as the step preview on the workflow
			// list: the directory decides what "make build" even means,
			// so it must not disappear on the way into this screen.
			text := "$ " + cmd.Cmd
			if cmd.Dir != "" {
				text += "  (workdir: " + cmd.Dir + ")"
			}
			note = gray(truncate(text, max(8, w-len(name)-24)))
		} else {
			box = dim("[-]")
			note = warn("(not found)")
		}
		row := fmt.Sprintf("%s %d. %-16s %s", box, i+1, name, note)
		if i == p.cursor {
			return "  " + accentBold("▶") + " " + row
		}
		return "    " + row
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
		b.WriteString("  " + gray("Nothing selected — Enter runs the highlighted step") + "\n")
	}
	b.WriteString("  " + gray("↑↓ Tab: toggle   Enter: run   Esc ←: back") + "\n")
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
	var b strings.Builder
	b.WriteString("\n" + header(title, w) + "\n\n")
	for i, label := range commandFormLabels {
		if i == cf.fieldIdx {
			b.WriteString("  " + m.nameInput.View() + "\n")
		} else if cf.fields[i] != "" {
			b.WriteString("  " + gray(label+" > ") + cf.fields[i] + "\n")
		} else {
			b.WriteString("  " + gray(label+" > ") + "\n")
		}
	}

	// Placeholder picker: opens as a centered floating window (Tab).
	if cf.slotPickFocus {
		return m.viewPlaceholderWindow(w)
	}
	if lines := m.slotValidationLines(cf); len(lines) > 0 {
		b.WriteString("\n")
		for _, l := range lines {
			b.WriteString("  " + l + "\n")
		}
	}

	b.WriteString("\n" + hline(w) + "\n")
	guide := "↑↓: field   Enter: next / save   Ctrl+S: save   Esc: previous / cancel"
	if cf.slotInsertAvailable(len(m.lists)) {
		guide = "Tab: insert placeholder   " + guide
	}
	b.WriteString("  " + gray(guide) + "\n")
	return b.String()
}

// viewPlaceholderWindow renders the placeholder picker as a centered
// floating window (LazyVim style) while it has key focus: list names in
// the left pane, the selected list's entries in the right pane. The field
// being edited is echoed at the top so the insertion context stays visible.
func (m Model) viewPlaceholderWindow(w int) string {
	cf := m.cf
	names := m.sortedListNames()

	// Left pane: list names with cursor (dimmed while the right pane is focused).
	var left strings.Builder
	for i, name := range names {
		if i == cf.slotPickCursor {
			if cf.slotPickPane == 0 {
				left.WriteString(accentBold("▶ ") + white("{"+name+"}") + "\n")
			} else {
				left.WriteString(gray("▶ ") + white("{"+name+"}") + "\n")
			}
		} else {
			left.WriteString("  " + gray("{"+name+"}") + "\n")
		}
	}

	// Right pane: entries of the selected list (value + label). The pane
	// takes focus with → and inserts the selected value directly.
	//
	// The pane width is fixed to the widest entry across ALL lists (clamped
	// to the terminal width) so the window frame keeps the same size while
	// the cursor moves between lists; values that don't fit are truncated
	// with an ellipsis instead of stretching the frame.
	leftW := 0
	for _, name := range names {
		if lw := 2 + lipgloss.Width("{"+name+"}"); lw > leftW {
			leftW = lw
		}
	}
	rightW := 9 // room for "(empty)" and the scroll markers
	for _, name := range names {
		for _, e := range m.lists[name] {
			lw := 2 + lipgloss.Width(e.Value)
			if e.Label != "" {
				lw += 2 + lipgloss.Width(e.Label)
			}
			if lw > rightW {
				rightW = lw
			}
		}
	}
	// Chrome around the right pane's content: window border+padding (6),
	// left pane + its gap (leftW+2), divider + right pane padding (3).
	if maxRight := w - leftW - 11; maxRight >= 8 && rightW > maxRight {
		rightW = maxRight
	}

	const maxRows = 8
	var right strings.Builder
	if cf.slotPickCursor < len(names) {
		entries := m.lists[names[cf.slotPickCursor]]
		if len(entries) == 0 {
			right.WriteString(dim("(empty)") + "\n")
		}
		start := 0
		if cf.slotPickPane == 1 && cf.slotPickValueCursor >= maxRows {
			start = cf.slotPickValueCursor - maxRows + 1
		}
		if start > 0 {
			right.WriteString(dim(fmt.Sprintf("…%d above", start)) + "\n")
		}
		for i := start; i < len(entries); i++ {
			if i >= start+maxRows {
				right.WriteString(dim(fmt.Sprintf("…+%d more", len(entries)-i)) + "\n")
				break
			}
			e := entries[i]
			avail := rightW - 2 // cursor prefix
			val := ansi.Truncate(e.Value, avail, "…")
			line := val
			if e.Label != "" {
				if rem := avail - lipgloss.Width(val) - 2; rem > 0 {
					line += "  " + gray(ansi.Truncate(e.Label, rem, "…"))
				}
			}
			if cf.slotPickPane == 1 && i == cf.slotPickValueCursor {
				right.WriteString(accentBold("▶ ") + line + "\n")
			} else {
				right.WriteString("  " + line + "\n")
			}
		}
	}
	rightPane := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(lipgloss.Color("240")).
		PaddingLeft(2).
		Width(rightW + 2).
		Render(strings.TrimRight(right.String(), "\n"))

	panes := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().PaddingRight(2).Render(strings.TrimRight(left.String(), "\n")),
		rightPane)

	help := "↑↓ move   →: values   Enter: insert {placeholder}   Esc: close"
	if cf.slotPickPane == 1 {
		help = "↑↓ move   ←: lists   Enter: insert value   Esc: close"
	}

	// Full-text footer: list rows are truncated to keep the frame stable,
	// so while the value pane is focused the selected entry is echoed in
	// full above the help line, wrapped to the window's existing width.
	footer := ""
	if cf.slotPickPane == 1 && cf.slotPickCursor < len(names) {
		if entries := m.lists[names[cf.slotPickCursor]]; cf.slotPickValueCursor < len(entries) {
			e := entries[cf.slotPickValueCursor]
			need := 2 + lipgloss.Width(e.Value)
			if e.Label != "" {
				need += 2 + lipgloss.Width(e.Label)
			}
			if need > rightW {
				full := e.Value
				if e.Label != "" {
					full += "  " + e.Label
				}
				innerW := lipgloss.Width(panes)
				if hw := lipgloss.Width(help); hw > innerW {
					innerW = hw
				}
				footer = dim(lipgloss.NewStyle().Width(innerW).Render(full)) + "\n\n"
			}
		}
	}

	content := accentBold(" Insert placeholder / value ") + "\n\n" +
		m.nameInput.View() + "\n\n" +
		panes + "\n\n" +
		footer +
		gray(help)

	win := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("36")).
		Padding(0, 2).
		Render(content)

	h := m.height
	if h == 0 {
		h = 24
	}
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, win)
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
		if _, ok := m.lists[s.ListName]; ok {
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
