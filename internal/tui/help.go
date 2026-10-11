package tui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

func (m Model) helpHint() string {
	return "F1 Help"
}

func (m Model) helpGeometry(w int) actionPopupLayout {
	return actionPopupLayout{w: min(72, w), h: min(24, max(3, m.termHeight())), x: max(0, (w-min(72, w))/2), y: max(0, (m.termHeight()-min(24, max(3, m.termHeight())))/2)}
}

func (m Model) helpLines(w int) []string {
	lines := []string{"Current screen"}
	if m.runHomeVisible() {
		lines = append(lines, "1 Workflows · 2 Commands · click a tab to switch", "e Edit selected item · p Projects · q Quit · Esc Back", "Esc first clears search or leaves queue focus; selected commands require discard confirmation. Press p to open the project picker directly.", "/ Start search · Enter Select / review · Esc Clear text, then exit search", "While searching, letters and digits are text; Enter selects the result; execution still requires review.", "No result: Enter keeps search active. Tab toggles command selection.", "Down focuses search results; e edits the selected result. / resumes search.", "Tab changes preserve search, cursor and command execution order.")
	}
	if m.screen == ScreenRecent {
		lines = append(lines, "/ Start search · ↑↓ Move · Enter Select", "Search: ↓ Results · Enter Select · Esc Clear text, then exit search", "p Projects · q Quit · Esc Clear filter / Quit · Ctrl+R Refresh", "Delete Remove from this directory’s history", "Ctrl+Z Undo the last removal", "Command definitions are never deleted.", "F1 / Esc Close help")
	} else if m.screen == ScreenProjectSelect {
		lines = append(lines, "a Add project · Enter Open · / Search", "r Refresh · Space / right-click Actions", "o Open folder · c Copy path · Del/x Delete", "Deletion moves settings into .baton-trash; working directories are untouched.", "Restore by moving the archived folder back under projects.", "Esc closes a form or clears search; q quits.")
	} else if m.screen == ScreenRunCommands || m.screen == ScreenCreateWorkflow || m.screen == ScreenEditWorkflowCommands {
		lines = append(lines, func() string {
			if m.runHomeVisible() {
				return "/ Search · ↑↓ Move · Tab Select / deselect"
			}
			return "Type to filter · ↑↓ Move · Tab Select / deselect"
		}(), func() string {
			if m.runHomeVisible() {
				return "F2 Switch between candidates and execution queue · s Sort candidates"
			}
			return "F2 Switch between candidates and execution queue · F3 Sort candidates"
		}(), "Queue: J/K or Shift+↓/↑ Reorder · Del/x Remove from queue", "Enter Continue · Esc Back (keeps selection in the confirmation)", "Removing a queue entry does not delete its command.")
	} else if m.screen == ScreenRunWorkflowSteps {
		lines = append(lines, "↑↓ Choose step · Tab Select / deselect", "F2 Details / Steps on narrow screens · Enter Review · Esc Back", "Selections affect this run only; saved workflow is unchanged.", "Select at least one step before reviewing. Click a row to inspect; click its checkbox to toggle.")
	} else if m.screen == ScreenRunWorkflow {
		lines = append(lines, func() string {
			if m.runHomeVisible() {
				return "/ Search · ↑↓ Choose workflow"
			}
			return "Type to filter · ↑↓ Choose workflow"
		}(), "Enter Continue to inputs and execution review", "Tab / → Choose steps for this run · F2 Details · Esc Back")
	} else if m.screen == ScreenSlotPick {
		if m.sp != nil && m.sp.pathDef.PathKind != "" {
			lines = append(lines, "Reserved path slot: Tab / Ctrl+Space opens choices", "Type to filter · ↑↓ Choose · → Open folder", "Single value: Tab inserts · Multiple values: Tab checks", "File pickers show folders for navigation only. Extensions filter files.", "Enter confirms and advances to the next input or execution review; it never runs the command.", "Paths use workdir, or the launch directory when it is empty.", "Template creation: Ctrl+K keeps this slot interactive.", "Values are literal; quote single-value slots when paths may contain spaces.")
		} else {
			lines = append(lines, "Type to filter / enter a value · ↑↓ Choose · Enter Confirm", "Tab toggles multiple selections; single-value slots move to the next row.", "File choices require a reserved slot such as {@file:.go}.")
		}
	} else if m.pasteMenu {
		for _, a := range m.contextActions {
			lines = append(lines, a.key+"  "+a.label)
		}
		lines = append(lines, "↑↓ Select · Enter Confirm · Esc Close")
	} else if m.leaveAsk != leaveNone {
		for _, a := range leaveChoices {
			lines = append(lines, a.key+"  "+a.label)
		}
	} else if m.screen == ScreenExplore || m.exploreValueFlow() || m.exploreTemplateFlow() || m.cf != nil {
		if m.exploreMenu {
			if it, ok := m.exploreCurrent(); ok {
				for _, a := range m.exploreActions(it) {
					lines = append(lines, a.key+"  "+a.label)
				}
			}
		} else {
			lines = append(lines, strings.Split(m.exploreKeys(), "  ")...)
		}
		if m.contextInputActive() {
			lines = append(lines, "Ctrl+V Paste · Right-click: Paste / Copy value")
		}
	} else {
		lines = append(lines, "Use the shortcuts shown at the bottom of this screen.", "Esc Back")
	}
	lines = append(lines, "", "Concepts")
	lines = append(lines, conceptDiagram(max(1, w-4))...)
	lines = append(lines, "",
		"Workflow: an ordered sequence of command references.",
		"Command: executable text, workdir, shell and input slots.",
		"List: shared choices for an interactive {slot}; not owned by a command.",
		"Variable: shared constant substituted as {$name} in commands and list values.",
		"Fixed values: saved slot answers owned by a command. A fixed answer skips the run-time prompt; edit via Change values.",
		"", "Navigation",
		"1 Workflows · 2 Commands · 3 Lists · 4 Variables (Project editor)",
		"Space / right-click: actions for the selected item.",
		"Command actions include links to referenced lists and variables.",
		"Esc returns from a related item, preserving your previous selection.",
		"Run/editor search: Esc clears text and keeps searching; Esc on empty input exits search.",
		"a adds to the focused pane; n also creates a project item.",
		"F1 opens help, including while typing. Esc closes it; ? remains ordinary input while typing.",
	)
	var wrapped []string
	for _, line := range lines {
		wrapped = append(wrapped, strings.Split(ansi.Hardwrap(line, max(1, w-4), false), "\n")...)
	}
	return wrapped
}

func (m *Model) clampHelpOffset() {
	p := m.helpGeometry(m.viewWidth())
	m.helpOffset = min(max(0, m.helpOffset), max(0, len(m.helpLines(p.w))-max(1, p.h-4)))
}

func (m Model) overlayHelp(bg string, w int) string {
	p := m.helpGeometry(w)
	lines := m.helpLines(p.w)
	rows := max(1, p.h-4)
	start := min(m.helpOffset, max(0, len(lines)-rows))
	end := min(len(lines), start+rows)
	body := " " + strings.Join(lines[start:end], "\n ") + "\n\n ↑↓ Scroll · Esc Close"
	return overlayBox(bg, box("Help", body, p.w, p.h, true), p.x, p.y, w)
}

// Keep connectors intact on narrow terminals by switching to a compact map.
func conceptDiagram(width int) []string {
	if width < 48 {
		return []string{
			"Workflow",
			" +-> Command A (step 1)",
			" +-> Command B (step 2)",
			"",
			"Command B",
			" +-> List: {env}",
			" +-> Variable: {$root}",
			" +-- Fixed: env=prod",
			"",
			"-> references (shared)",
			"-- owns (command only)",
		}
	}
	return []string{
		"Workflow",
		" |",
		" +-- Step 1 --> Command A",
		" +-- Step 2 --> Command B",
		"                  |",
		"                  +-- {env} --> List: env",
		"                  |            dev / staging / prod",
		"                  +-- {$root} --> Variable: root",
		"                  |              /workspace",
		"                  +-- Fixed values (owned)",
		"                       env = prod",
		"",
		"--> references a shared item",
	}
}
