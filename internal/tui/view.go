package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"

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
	case ScreenRunManually, ScreenCreateWorkflow, ScreenCreateAlias,
		ScreenEditWorkflowCommands, ScreenEditAliasCommands:
		view = m.viewMultiSelect(w)
	case ScreenEditWorkflowMode:
		view = m.viewSingleSelect("Edit workflow", w)
	case ScreenEditAliasMode:
		view = m.viewSingleSelect("Edit alias", w)
	case ScreenSlotPick:
		view = m.viewSlotPick(w)
	case ScreenConfirmRun:
		view = m.viewConfirmRun(w)
	case ScreenRunning:
		view = m.viewRunning(w)
	case ScreenRetry:
		view = m.viewRetry(w)
	case ScreenConfirmVars:
		view = m.viewConfirmVars(w)
	case ScreenNameInput:
		view = m.viewNameInput(w)
	case ScreenEditWorkflow:
		view = m.viewSingleSelect("Edit workflow", w)
	case ScreenDeleteWorkflow:
		view = m.viewDeleteList("Delete workflow", m.listItems, w)
	case ScreenAliasMgmt:
		view = m.viewSingleSelect("Manage aliases", w)
	case ScreenEditAlias:
		view = m.viewSingleSelect("Edit alias", w)
	case ScreenDeleteAlias:
		view = m.viewDeleteList("Delete alias", m.listItems, w)
	case ScreenManageLists:
		view = m.viewManageLists(w)
	case ScreenEditList:
		view = m.viewEditList(w)
	case ScreenSwitchConfig:
		view = m.viewSingleSelect("Switch config", w)
	}
	if m.errMsg != "" {
		view += "\n" + red("  Error: "+m.errMsg) + "\n"
	}
	return view
}

// ── Project select / main menu ───────────────────────────────────────────────

func (m Model) viewProjectSelect(w int) string {
	var b strings.Builder
	b.WriteString("\n  ┌───────────────┐\n")
	b.WriteString("  │  " + bold(white("B A T O N")) + "    │\n")
	b.WriteString("  └───────────────┘\n\n")
	for i, p := range m.projects {
		if i == m.listCursor {
			b.WriteString("  " + cyanBold("▶") + " " + p + "\n")
		} else {
			b.WriteString("    " + p + "\n")
		}
	}
	b.WriteString("\n" + hline(w) + "\n")
	b.WriteString("  " + gray("↑↓ Enter") + "\n")
	return b.String()
}

type menuItemInfo struct {
	desc     string
	shortcuts [][2]string
}

var menuItemInfos = map[string]menuItemInfo{
	"Run workflow":    {desc: "Run a saved workflow.", shortcuts: [][2]string{{"Enter", "Run"}, {"Esc", "Back"}}},
	"Run manually":    {desc: "Pick commands and run them once.", shortcuts: [][2]string{{"Space", "Select"}, {"Enter", "Run"}, {"Esc", "Back"}}},
	"Create workflow": {desc: "Save a command set as a reusable workflow.", shortcuts: [][2]string{{"Space", "Select"}, {"Enter", "Save"}, {"Esc", "Back"}}},
	"Edit workflow":   {desc: "Rename or change commands in a workflow.", shortcuts: [][2]string{{"Enter", "Edit"}, {"Esc", "Back"}}},
	"Delete workflow": {desc: "Delete one or more workflows.", shortcuts: [][2]string{{"Space", "Toggle"}, {"Enter", "Confirm"}, {"Esc", "Back"}}},
	"Manage aliases":  {desc: "Reusable command groups, selectable in Run Manually.", shortcuts: [][2]string{{"Enter", "Open"}, {"Esc", "Back"}}},
	"Manage lists":    {desc: "Edit selection lists for placeholders.", shortcuts: [][2]string{{"Enter", "Edit"}, {"n", "New"}, {"d", "Delete"}, {"Esc", "Back"}}},
	"Switch config":   {desc: "Switch to a different project.", shortcuts: [][2]string{{"Enter", "Switch"}, {"Esc", "Back"}}},
	"Exit":            {desc: "Quit baton.", shortcuts: [][2]string{{"Enter", "Quit"}}},
}

func (m Model) viewMainMenu(w int) string {
	type group struct {
		label string
		items []string
	}
	groups := []group{
		{"Run", []string{"Run workflow", "Run manually"}},
		{"Workflow", []string{"Create workflow", "Edit workflow", "Delete workflow"}},
		{"Manage", []string{"Manage aliases", "Manage lists"}},
		{"", []string{"Switch config", "Exit"}},
	}

	leftW := 32
	showRight := w >= leftW+30

	// Build left pane lines
	var leftLines []string
	leftLines = append(leftLines, "")
	leftLines = append(leftLines, "  ┌───────────────┐")
	leftLines = append(leftLines, "  │  "+bold(white("B A T O N"))+"    │")
	leftLines = append(leftLines, "  └───────────────┘")
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

	if info, ok := menuItemInfos[selected]; ok {
		rightLines = append(rightLines, white(selected))
		rightLines = append(rightLines, gray(strings.Repeat("─", min(rightW-2, 24))))
		rightLines = append(rightLines, dim(info.desc))
		rightLines = append(rightLines, "")
		rightLines = append(rightLines, gray("Keys"))
		for _, sc := range info.shortcuts {
			rightLines = append(rightLines, fmt.Sprintf("  %-8s %s", white(sc[0]), gray(sc[1])))
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
		rightLines = append(rightLines, fmt.Sprintf("  "+gray("aliases:   ")+white("%d"), len(m.aliases)))
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
	if len(m.listItems) == 0 {
		b.WriteString("  " + gray("(empty)") + "\n")
	} else {
		for i, item := range m.listItems {
			if i == m.listCursor {
				b.WriteString("  " + cyanBold("▶") + " " + item + "\n")
			} else {
				b.WriteString("    " + item + "\n")
			}
		}
	}
	if m.screen == ScreenEditWorkflow {
		b.WriteString("\n" + hlineLabel(w, "steps") + "\n")
		b.WriteString(m.stepsVP.View() + "\n")
	}
	b.WriteString("\n" + hline(w) + "\n")
	b.WriteString("  " + gray("↑↓ Enter Esc") + "\n")
	return b.String()
}

// ── Run workflow ─────────────────────────────────────────────────────────────

func (m Model) viewRunWorkflow(w int) string {
	var b strings.Builder
	b.WriteString("\n" + header("Run workflow", w) + "\n")

	if len(m.workflows) == 0 {
		b.WriteString("  " + gray("(no workflows saved)") + "\n")
		b.WriteString("\n" + hline(w) + "\n")
		b.WriteString("  " + gray("Esc: back") + "\n")
		return b.String()
	}

	// Reserve lines for preview panel + footer
	previewH := 2
	if m.listCursor < len(m.workflows) {
		previewH = min(len(m.workflows[m.listCursor].Commands), 5) + 2
	}
	viewH := max(1, m.height-8-previewH)
	cur := m.listCursor
	viewStart := max(0, min(cur-viewH/2, len(m.workflows)-viewH))
	viewEnd := min(viewStart+viewH, len(m.workflows))

	if viewStart > 0 {
		b.WriteString("  " + gray("...") + "\n")
	}
	for i := viewStart; i < viewEnd; i++ {
		wf := m.workflows[i]
		suffix := ""
		if wf.Name == m.lastWorkflow {
			suffix = "  " + gray("(last)")
		}
		if i == cur {
			b.WriteString("  " + cyanBold("▶") + " " + bold(wf.Name) + suffix + "\n")
		} else {
			b.WriteString("    " + wf.Name + suffix + "\n")
		}
	}
	if viewEnd < len(m.workflows) {
		b.WriteString("  " + gray("...") + "\n")
	}

	// Step preview for hovered workflow (scrollable viewport)
	if m.stepsFocused {
		b.WriteString("\n" + hlineLabel(w, "steps ↑↓") + "\n")
	} else {
		b.WriteString("\n" + hlineLabel(w, "steps") + "\n")
	}
	b.WriteString(m.stepsVP.View() + "\n")

	b.WriteString("\n" + hline(w) + "\n")
	b.WriteString("  " + gray("↑↓ Enter Esc  Tab: focus steps") + "\n")
	return b.String()
}

// ── Multi-select ─────────────────────────────────────────────────────────────

func (m Model) viewMultiSelect(w int) string {
	title := "Select commands"
	if m.screen == ScreenEditWorkflowCommands || m.screen == ScreenEditAliasCommands {
		title = "Edit commands"
	}
	filtered := m.msFiltered()
	n := len(filtered)
	if m.msCursor >= n && n > 0 {
		// cursor out of bounds after filter — benign, view clamps it
	}

	viewH := max(1, m.height-13)
	cursor := m.msCursor
	if cursor >= n {
		cursor = max(0, n-1)
	}
	viewStart := m.msViewStart
	if cursor < viewStart {
		viewStart = cursor
	}
	if cursor >= viewStart+viewH {
		viewStart = cursor - viewH + 1
	}

	var b strings.Builder
	b.WriteString("\n" + header(title, w) + "\n")

	b.WriteString("  " + m.msSearchTI.View() + "    " + m.msGroupTI.View() + "\n\n")

	if n == 0 {
		b.WriteString("  " + gray("No results.") + "\n")
	} else {
		if viewStart > 0 {
			b.WriteString("  " + gray("...") + "\n")
		}
		viewEnd := min(viewStart+viewH, n)
		for i := viewStart; i < viewEnd; i++ {
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
			if item.isAlias() {
				steps := strings.Join(item.alias.Steps, gray(" > "))
				label = cyan("@") + " " + item.alias.Name + "  " + sGroup.Render("[alias]") + "  " + gray(steps)
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
				b.WriteString("  " + cyanBold("▶") + " " + check + " " + label + "\n")
			} else {
				b.WriteString("    " + check + " " + label + "\n")
			}
		}
		if viewEnd < n {
			b.WriteString("  " + gray("...") + "\n")
		}
	}

	// Hover preview
	b.WriteString("\n")
	if n > 0 && cursor >= 0 && cursor < n {
		hoveredIdx := filtered[cursor]
		hovered := m.msItems[hoveredIdx]
		if hovered.isAlias() {
			b.WriteString(hlineLabel(w, "alias steps") + "\n")
			steps := strings.Join(hovered.alias.Steps, gray(" > "))
			b.WriteString("  " + steps + "\n")
		} else {
			b.WriteString(hlineLabel(w, "command preview") + "\n")
			cmdStr := hovered.cmd.Cmd
			maxLen := w - 10
			if maxLen < 10 {
				maxLen = 10
			}
			if len(cmdStr) > maxLen {
				cmdStr = cmdStr[:maxLen-3] + "..."
			}
			b.WriteString("  " + gray("$ "+cmdStr) + "\n")
			workdir := hovered.cmd.Dir
			if workdir == "" {
				workdir = "."
			}
			b.WriteString("  " + dim("workdir: "+workdir) + "\n")
		}
	} else {
		b.WriteString(hline(w) + "\n")
	}

	var selNames []string
	for _, idx := range m.msSelected {
		selNames = append(selNames, m.msItems[idx].name())
	}
	orderHint := ""
	if len(m.msSelected) > 0 {
		orderHint = "  " + dim("[n] = run order")
	}
	b.WriteString("\n  " + green(fmt.Sprintf("Selected(%d)", len(m.msSelected))) + orderHint + ": " + strings.Join(selNames, ", ") + "\n")
	b.WriteString(hline(w) + "\n")
	b.WriteString("  " + gray("↑↓ Move  Space Select  Enter Confirm  Esc Back  Tab Switch") + "\n")
	return b.String()
}

// ── Slot pick ────────────────────────────────────────────────────────────────

func (m Model) viewSlotPick(w int) string {
	sp := m.sp
	var b strings.Builder
	b.WriteString("\n" + cyanBold("  [ Select value for {"+sp.slotName+"} ]") + "\n" + hline(w) + "\n\n")

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
				b.WriteString("  " + cyanBold(fmt.Sprintf("  %2d. %s", i+1, name)) + "\n")
			} else {
				isDone := i < cur && i < len(sp.contextNotes) && sp.contextNotes[i] != ""
				marker := "  "
				nameStr := gray(fmt.Sprintf("  %2d. %s", i+1, name))
				if isDone {
					marker = green("✓ ")
					note := sp.contextNotes[i]
					maxNote := w - len(name) - 14
					if maxNote > 10 && len(note) > maxNote {
						note = note[:maxNote-3] + "..."
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
			b.WriteString("\n" + hlineLabel(w, "command preview") + "\n\n")
			// Compute hovered value for inline display
			hoveredValPreview := ""
			isCustomCursorPreview := sp.cursor == len(sp.filtered)
			if !isCustomCursorPreview && sp.cursor >= 0 && sp.cursor < len(sp.filtered) {
				hoveredValPreview = sp.filtered[sp.cursor].Value
			} else if isCustomCursorPreview && sp.search != "" {
				hoveredValPreview = sp.search
			}
			pointerSuffix := ""
			if hoveredValPreview != "" {
				pointerSuffix = "  " + slotVar("{"+sp.slotName+"}") + gray(" = ") + white(hoveredValPreview)
			}

			highlighted := slot.HighlightSlot(sp.currentCmd.Cmd, sp.slotName, sp.resolvedSoFar)
			b.WriteString("    " + gray("$") + " " + highlighted + "\n")
			partialCmd := sp.currentCmd.Cmd
			for k, v := range sp.resolvedSoFar {
				partialCmd = strings.ReplaceAll(partialCmd, "{"+k+"}", v)
			}
			if idx := strings.Index(partialCmd, "{"+sp.slotName+"}"); idx >= 0 {
				b.WriteString(strings.Repeat(" ", 6+idx) + cyan("^") + pointerSuffix + "\n")
			}

			dir := sp.currentCmd.Dir
			if dir == "" {
				b.WriteString("    " + gray("workdir:") + " " + dim(".") + "\n")
			} else {
				highlighted := slot.HighlightSlot(dir, sp.slotName, sp.resolvedSoFar)
				b.WriteString("    " + gray("workdir:") + " " + highlighted + "\n")
				partialDir := dir
				for k, v := range sp.resolvedSoFar {
					partialDir = strings.ReplaceAll(partialDir, "{"+k+"}", v)
				}
				if idx := strings.Index(partialDir, "{"+sp.slotName+"}"); idx >= 0 {
					b.WriteString(strings.Repeat(" ", 13+idx) + cyan("^") + pointerSuffix + "\n")
				}
			}
		}
		b.WriteString("\n\n" + hlineLabelBright(w, "Select value") + "\n\n")
	}

	// Search field
	if sp.search == "" {
		b.WriteString("  " + dim("/ type to filter...") + "\n\n")
	} else {
		countStr := ""
		if len(sp.filtered) == 0 {
			countStr = yellow("no match")
		} else {
			countStr = dim(fmt.Sprintf("%d results", len(sp.filtered)))
		}
		b.WriteString("  " + cyan("/") + " " + white(sp.search) + dim("_") +
			"  " + countStr + "\n\n")
	}

	// List
	contextLines := 0
	if sp.contextNames != nil {
		contextLines = len(sp.contextNames) + 3
	}
	viewH := max(1, m.height-8-contextLines)
	total := len(sp.filtered) + 1
	viewStart := max(0, min(sp.cursor-viewH/2, total-viewH))
	viewEnd := min(viewStart+viewH, total)

	// Compute label column alignment
	labelCol := 0
	for _, e := range sp.filtered {
		if e.Label != "" && len(e.Value)+2 > labelCol {
			labelCol = len(e.Value) + 2
		}
	}

	if viewStart > 0 {
		b.WriteString("  " + dim("↑ more") + "\n")
	}
	for i := viewStart; i < viewEnd; i++ {
		isCustom := i == len(sp.filtered)
		selected := i == sp.cursor

		if selected {
			// Render raw text inside sCursor to avoid ANSI width miscalculation
			var rawLine string
			if isCustom {
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
			}
			b.WriteString(sCursor.Width(w-2).Render("  ▶ " + rawLine) + "\n")
		} else {
			var line string
			if isCustom {
				if sp.search != "" {
					line = cyan("[") + " + " + white(sp.search) + "  " + dim("(custom)") + cyan(" ]")
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
			}
			b.WriteString("    " + line + "\n")
		}
	}
	if viewEnd < total {
		b.WriteString("  " + dim("↓ more") + "\n")
	}

	b.WriteString("\n" + hline(w) + "\n")
	b.WriteString("  " + gray("↑↓ Enter") + "  " + gray("Esc: ") + dim("clear filter / back") + "\n")
	return b.String()
}

// ── Confirm run ──────────────────────────────────────────────────────────────

func (m Model) viewConfirmRun(w int) string {
	var b strings.Builder
	b.WriteString("\n" + header("Confirm", w) + "\n")
	for i, item := range m.confirmRunItems {
		if item.IsAlias() {
			steps := strings.Join(item.Alias.Steps, " > ")
			b.WriteString(fmt.Sprintf("  %s%2d.%s  %s %s  %s\n",
				gray(""), i+1, gray(""), cyan("@"), item.Name, gray("("+steps+")")))
		} else {
			b.WriteString(fmt.Sprintf("  %s%2d.%s  %s\n", gray(""), i+1, gray(""), item.Name))
			b.WriteString("       " + gray("$ "+item.Cmd.Cmd) + "\n")
			workdir := item.Cmd.Dir
			if workdir == "" {
				workdir = "."
			}
			b.WriteString("         " + dim("workdir: "+workdir) + "\n")
		}
	}
	b.WriteString("\n" + hline(w) + "\n\n")
	b.WriteString(renderBtns(m.confirmRunBtn, "  Run  ", "  Cancel  ") + "\n")
	b.WriteString("\n  " + gray("Tab: switch   Enter: confirm   Esc: back") + "\n")
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
	b.WriteString("\n" + progressBar(n, n, 24) + "\n\n")
	b.WriteString(greenBold("  [ Done ]") + "  " + gray(fmt.Sprintf("%d/%d", n, n)) + "\n\n")
	b.WriteString("  " + gray("Press any key to return to menu...") + "\n")
	return b.String()
}

// ── Retry ────────────────────────────────────────────────────────────────────

func (m Model) viewRetry(w int) string {
	r := m.running
	var b strings.Builder
	b.WriteString("\n" + header("Run failed", w) + "\n")
	if r != nil && r.failErr != nil {
		b.WriteString("  " + red("Error: "+r.failErr.Error()) + "\n\n")
	}

	items := []string{
		fmt.Sprintf("Retry from step %d", r.current+1),
		"Retry all",
		"Abort",
	}
	for i, item := range items {
		if i == m.listCursor {
			b.WriteString("  " + cyanBold("▶") + " " + item + "\n")
		} else {
			b.WriteString("    " + item + "\n")
		}
	}
	b.WriteString("\n" + hline(w) + "\n")
	b.WriteString("  " + gray("↑↓ Enter Esc") + "\n")
	return b.String()
}

// ── Confirm vars ─────────────────────────────────────────────────────────────

func (m Model) viewConfirmVars(w int) string {
	cv := m.cv
	var b strings.Builder
	b.WriteString("\n" + header("Confirm variables", w) + "\n")
	b.WriteString("  " + gray("Review the variable values to be saved.") + "\n\n")

	for i, cmd := range cv.cmds {
		vars, ok := cv.vars[cmd.Name]
		if !ok {
			continue
		}
		resolved := slot.Apply(cmd, vars)
		b.WriteString("  " + gray(fmt.Sprintf("  %d. %s", i+1, cmd.Name)) + "\n")
		b.WriteString("       " + gray("$ "+resolved.Cmd) + "\n")
		workdir := resolved.Dir
		if workdir == "" {
			workdir = "."
		}
		b.WriteString("         " + dim("workdir: "+workdir) + "\n")
	}

	b.WriteString("\n" + hline(w) + "\n\n")
	b.WriteString("\n" + renderBtns(cv.btn, "  Confirm  ", "  Edit  ") + "\n")
	b.WriteString("\n  " + gray("Tab: switch   Enter: select   Esc: re-edit") + "\n")
	return b.String()
}

// ── Name input ────────────────────────────────────────────────────────────────

func (m Model) viewNameInput(w int) string {
	title := "Create workflow"
	switch m.nameInputMode {
	case nameInputAlias:
		title = "Create alias"
	case nameInputEditWorkflow:
		title = "Rename workflow"
	case nameInputEditAlias:
		title = "Rename alias"
	case nameInputNewList:
		title = "New list"
	}
	var b strings.Builder
	b.WriteString("\n" + header(title, w) + "\n")
	b.WriteString("  " + gray("Esc: cancel") + "\n\n")
	b.WriteString("  " + m.nameInput.View() + "\n")
	if m.nameInputErr != "" {
		b.WriteString("\n  " + yellow(m.nameInputErr) + "\n")
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
		b.WriteString("  " + cyan("Value > ") + le.addVal)
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
	for i, e := range le.entries {
		lbl := ""
		if e.Label != "" {
			pad := strings.Repeat(" ", max(1, labelCol-len(e.Value)))
			lbl = pad + gray(e.Label)
		}
		line := e.Value + lbl
		if i == le.cursor {
			b.WriteString("  " + cyanBold("▶") + " " + line + "\n")
		} else {
			b.WriteString("    " + line + "\n")
		}
	}
	addLine := green("+ Add value")
	if le.cursor == len(le.entries) {
		b.WriteString("  " + cyanBold("▶") + " " + addLine + "\n")
	} else {
		b.WriteString("    " + gray("+ Add value") + "\n")
	}
	b.WriteString("\n" + hline(w) + "\n")
	b.WriteString("  " + gray("↑↓ Enter: edit   Del: remove   Esc: done") + "\n")
	return b.String()
}

// ── Manage lists ─────────────────────────────────────────────────────────────

func (m Model) viewManageLists(w int) string {
	var b strings.Builder
	b.WriteString("\n" + header("Manage lists", w) + "\n")
	if len(m.listItems) == 0 {
		b.WriteString("  " + gray("(empty)") + "\n")
	} else {
		for i, item := range m.listItems {
			if i == m.listCursor {
				b.WriteString("  " + cyanBold("▶") + " " + item + "\n")
			} else {
				b.WriteString("    " + item + "\n")
			}
		}
	}
	if !m.deleteConfirm && len(m.listItems) > 0 && m.listCursor < len(m.listItems) {
		listName := m.listItems[m.listCursor]
		entries := m.lists[listName]
		b.WriteString("\n" + hlineLabel(w, "entries") + "\n")
		if len(entries) == 0 {
			b.WriteString("  " + gray("(empty)") + "\n")
		} else {
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
	}
	b.WriteString("\n" + hline(w) + "\n")
	if m.deleteConfirm && len(m.listItems) > 0 {
		b.WriteString("  " + yellow(fmt.Sprintf("Delete %q?", m.listItems[m.listCursor])) + "\n\n")
		b.WriteString(renderBtns(m.deleteBtn, "  No  ", "  Yes  ") + "\n")
		b.WriteString("\n  " + gray("Tab: switch   Enter: confirm   Esc: back") + "\n")
	} else {
		b.WriteString("  " + gray("↑↓ Enter: edit   n: new   d: delete   Esc: back") + "\n")
	}
	return b.String()
}

// ── Delete list (with confirmation) ──────────────────────────────────────────

func (m Model) viewDeleteList(title string, items []string, w int) string {
	var b strings.Builder
	b.WriteString("\n" + header(title, w) + "\n")
	if len(items) == 0 {
		b.WriteString("  " + gray("(empty)") + "\n")
	} else {
		selectedSet := make(map[int]bool, len(m.deleteSelected))
		for _, s := range m.deleteSelected {
			selectedSet[s] = true
		}
		for i, item := range items {
			check := gray("[ ]")
			if selectedSet[i] {
				check = yellow("[x]")
			}
			if i == m.listCursor {
				b.WriteString("  " + cyanBold("▶") + " " + check + " " + item + "\n")
			} else {
				b.WriteString("    " + check + " " + item + "\n")
			}
		}
	}
	if m.screen == ScreenDeleteWorkflow && !m.deleteConfirm {
		b.WriteString("\n" + hlineLabel(w, "steps") + "\n")
		b.WriteString(m.stepsVP.View() + "\n")
	}
	b.WriteString("\n" + hline(w) + "\n")
	if m.deleteConfirm {
		n := len(m.deleteSelected)
		msg := fmt.Sprintf("Delete %d item", n)
		if n > 1 {
			msg += "s"
		}
		b.WriteString("  " + yellow(msg+"?") + "\n\n")
		b.WriteString(renderBtns(m.deleteBtn, "  No  ", "  Yes  ") + "\n")
		b.WriteString("\n  " + gray("Tab: switch   Enter: confirm   Esc: back") + "\n")
	} else {
		b.WriteString("  " + gray("↑↓ Space: toggle   Enter: confirm   Esc: back") + "\n")
	}
	return b.String()
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

