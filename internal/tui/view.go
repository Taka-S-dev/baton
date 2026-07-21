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
		view = m.viewManageLists(w)
	case ScreenEditList:
		view = m.viewEditList(w)
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
	}
	if m.errMsg != "" {
		view += "\n" + errorText("  Error: "+m.errMsg) + "\n"
	}
	if m.successMsg != "" {
		view += "\n" + success("  ✓ "+m.successMsg) + "\n"
	}
	if m.loadWarning != "" && m.screen == ScreenMainMenu {
		view += "\n" + warn("  Warning: "+m.loadWarning) + "\n"
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
	for i, p := range m.projects {
		count := fmt.Sprintf("%d commands", m.projectCmdCounts[p])
		if i == m.listCursor {
			b.WriteString("  " + accentBold("▶") + " " + p + "   " + gray(count) + "\n")
		} else {
			b.WriteString("    " + p + "   " + gray(count) + "\n")
		}
	}
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
	{"Manage", []string{"Manage workflows", "Manage commands", "Manage lists"}},
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
	"Manage lists":     {desc: "Edit selection lists for placeholders.", shortcuts: [][2]string{{"Enter", "Edit"}, {"n", "New"}, {"d", "Delete"}, {"Esc", "Back"}}},
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
				b.WriteString("  " + accentBold("▶") + " " + item + "\n")
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

	b.WriteString("  " + m.wfSearchTI.View() + "\n\n")

	filtered := m.wfFiltered()
	n := len(filtered)
	cur := m.listCursor
	if cur >= n {
		cur = max(0, n-1)
	}

	if n == 0 {
		b.WriteString("  " + gray("No results.") + "\n")
	} else {
		// Reserve lines for preview panel + footer
		previewH := min(len(m.workflows[filtered[cur]].Commands), 5) + 2
		viewH := max(1, m.height-10-previewH)
		viewStart := max(0, min(cur-viewH/2, n-viewH))
		viewEnd := min(viewStart+viewH, n)

		if viewStart > 0 {
			b.WriteString("  " + gray("...") + "\n")
		}
		for i := viewStart; i < viewEnd; i++ {
			wf := m.workflows[filtered[i]]
			suffix := ""
			if wf.Name == m.lastWorkflow {
				suffix = "  " + gray("(last)")
			}
			if i == cur {
				b.WriteString("  " + accentBold("▶") + " " + bold(wf.Name) + suffix + "\n")
			} else {
				b.WriteString("    " + wf.Name + suffix + "\n")
			}
		}
		if viewEnd < n {
			b.WriteString("  " + gray("...") + "\n")
		}
	}

	// Step preview for hovered workflow (scrollable viewport)
	if m.stepsFocused {
		b.WriteString("\n" + hlineLabel(w, "steps ↑↓") + "\n")
	} else {
		b.WriteString("\n" + hlineLabel(w, "steps") + "\n")
	}
	b.WriteString(m.stepsVP.View() + "\n")

	b.WriteString("\n" + hline(w) + "\n")
	b.WriteString("  " + gray("Type to search  ↑↓ Enter Esc  Tab: focus steps") + "\n")
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

	b.WriteString("  " + m.msSearchTI.View() + "\n\n")

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
				b.WriteString("  " + accentBold("▶") + " " + check + " " + label + "\n")
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
		hovered := m.msItems[filtered[cursor]]
		if hovered.cmd != nil {
			m.writeCommandHover(&b, hovered.cmd, w)
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
	b.WriteString("\n  " + success(fmt.Sprintf("Selected(%d)", len(m.msSelected))) + orderHint + ": " + strings.Join(selNames, ", ") + "\n")
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
	b.WriteString("\n" + accentBold("  [ Select value for {"+sp.slotName+"} ]") + "\n" + hline(w) + "\n\n")

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
			b.WriteString("\n" + hlineLabelBright(w, "command preview  "+highlight("{"+sp.slotName+"}")) + "\n\n")

			// Hovered value (from list cursor or custom search input)
			hoveredVal := ""
			if sp.cursor >= 0 && sp.cursor < len(sp.filtered) {
				hoveredVal = sp.filtered[sp.cursor].Value
			} else if sp.cursor == len(sp.filtered) && sp.search != "" {
				hoveredVal = sp.search
			}

			// Substitute resolvedSoFar then current slot with the hovered value
			preview := func(s string) string {
				for k, v := range sp.resolvedSoFar {
					s = strings.ReplaceAll(s, "{"+k+"}", v)
				}
				if hoveredVal != "" {
					s = strings.ReplaceAll(s, "{"+sp.slotName+"}", highlight(hoveredVal))
				} else {
					s = strings.ReplaceAll(s, "{"+sp.slotName+"}", slotVar("{"+sp.slotName+"}"))
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
		b.WriteString("\n\n" + hlineLabelBright(w, "Select value") + "\n\n")
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
	contextLines := 0
	if sp.contextNames != nil {
		contextLines = len(sp.contextNames) + 3
	}
	viewH := max(1, m.height-8-contextLines)
	skipRow := len(sp.filtered) + 1
	total := skipRow
	if sp.canSkip {
		total++
	}
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
		isSkip := sp.canSkip && i == skipRow
		selected := i == sp.cursor

		if selected {
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
			}
			b.WriteString(sCursor.Width(w-2).Render("    "+rawLine) + "\n")
		} else {
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
	cmdStr := slot.ApplyVars(cmd.Cmd, m.vars)
	maxLen := w - 10
	if maxLen < 10 {
		maxLen = 10
	}
	if len(cmdStr) > maxLen {
		cmdStr = cmdStr[:maxLen-3] + "..."
	}
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
	for i, ref := range m.editRefs {
		cmd := m.editRefCommand(ref)
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
		if i == m.listCursor {
			b.WriteString("  " + accentBold("▶") + " " + label + "\n")
		} else {
			b.WriteString("    " + label + "\n")
		}
	}
	b.WriteString("\n")
	if m.listCursor >= 0 && m.listCursor < len(m.editRefs) {
		m.writeCommandHover(&b, m.editRefCommand(m.editRefs[m.listCursor]), w)
	}
	b.WriteString("\n" + hline(w) + "\n")
	b.WriteString("  " + gray("↑↓ Enter Esc") + "\n")
	return b.String()
}

// ── Confirm run ──────────────────────────────────────────────────────────────

func (m Model) viewConfirmRun(w int) string {
	var b strings.Builder
	b.WriteString("\n" + header("Confirm", w) + "\n")
	for i, item := range m.confirmRunItems {
		b.WriteString(fmt.Sprintf("  %s%2d.%s  %s\n", gray(""), i+1, gray(""), item.Name))
		if item.Cmd != nil {
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
	var b strings.Builder
	b.WriteString("\n" + header(title, w) + "\n")
	b.WriteString("  " + gray("Esc: cancel") + "\n\n")
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
	for i, e := range le.entries {
		lbl := ""
		if e.Label != "" {
			pad := strings.Repeat(" ", max(1, labelCol-len(e.Value)))
			lbl = pad + gray(e.Label)
		}
		line := e.Value + lbl
		if i == le.cursor {
			b.WriteString("  " + accentBold("▶") + " " + line + "\n")
		} else {
			b.WriteString("    " + line + "\n")
		}
	}
	addLine := success("+ Add value")
	if le.cursor == len(le.entries) {
		b.WriteString("  " + accentBold("▶") + " " + addLine + "\n")
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
				b.WriteString("  " + accentBold("▶") + " " + item + "\n")
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
		b.WriteString("  " + warn(fmt.Sprintf("Delete %q?", m.listItems[m.listCursor])) + "\n\n")
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
				check = warn("[x]")
			}
			if i == m.listCursor {
				b.WriteString("  " + accentBold("▶") + " " + check + " " + item + "\n")
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
		b.WriteString("  " + warn(msg+"?") + "\n\n")
		b.WriteString(renderBtns(m.deleteBtn, "  No  ", "  Yes  ") + "\n")
		b.WriteString("\n  " + gray("Tab: switch   Enter: confirm   Esc: back") + "\n")
	} else {
		b.WriteString("  " + gray("↑↓ Tab/Space: toggle   Enter: confirm   Esc: back") + "\n")
	}
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
	b.WriteString("  " + gray("Enter: confirm   Esc: back") + "\n")
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
			lines = append(lines, success("✓")+" "+gray("{"+s.Name+"} → "+s.ListName))
		} else {
			lines = append(lines, warn("⚠")+" "+gray("{"+s.Name+"} → ")+warn("no list (free input at run time)"))
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
