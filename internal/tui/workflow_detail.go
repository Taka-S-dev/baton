package tui

import (
	"fmt"
	"strings"

	"github.com/Taka-S-dev/baton/internal/slot"
	"github.com/charmbracelet/x/ansi"
)

// workflowDetailBox draws the details of the step or add candidate under
// the cursor, following the working copy without saving.
func (m Model) workflowDetailBox(l exploreLayout) string {
	title := "Step details"
	var b strings.Builder
	name := ""
	e := m.wfe
	if e.adding {
		title = "Add step · Command details"
		candidates := m.addCandidates()
		if e.addCursor >= 0 && e.addCursor < len(candidates) {
			name = candidates[e.addCursor]
		}
	} else if e.cursor >= 0 && e.cursor < len(e.steps) {
		name = e.steps[e.cursor]
		title = fmt.Sprintf("Step %d · Command details", e.cursor+1)
	}
	write := func(label, value string) {
		text := gray(label) + value
		b.WriteString(" " + strings.ReplaceAll(ansi.Hardwrap(text, max(1, l.rightW-4), false), "\n", "\n ") + "\n")
	}
	if name == "" {
		if e.adding {
			write("", "No matching command")
		} else {
			write("", "No steps yet · a adds a step")
		}
	} else if cmd, ok := m.workflowStepCommand(name); !ok {
		write("Command > ", name)
		write("", warn("Command not found"))
	} else {
		write("Command > ", name)
		write("$ ", m.previewVars(cmd.Cmd))
		write("Workdir > ", m.previewVars(displayOr(cmd.Dir, ".")))
		write("Shell > ", shellChoiceLabel(cmd.Shell))
		write("Type > ", commandKind(cmd))
		if cmd.Template != "" {
			write("Template > ", cmd.Template)
		}
		for _, key := range sortedKeys(cmd.Values) {
			write("Fixed > ", key+" = "+cmd.Values[key])
		}
		for _, s := range slot.GetSlots(cmd) {
			write("At run time > ", m.syntaxText(s.Placeholder()))
		}
		m.writeSyntaxWarnings(&b, cmd.Cmd+" "+cmd.Dir)
		m.writeCommandFacts(&b, cmd, max(1, l.rightW-2))
	}
	return box(title, b.String(), l.rightW, l.prevH, false)
}
