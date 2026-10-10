package tui

import (
	mdl "github.com/Taka-S-dev/baton/internal/model"
	"github.com/Taka-S-dev/baton/internal/slot"
	"strings"
)

func (m Model) exploreTemplateFlow() bool {
	if !m.exploreLaunched || m.sce == nil {
		return false
	}
	switch m.screen {
	case ScreenSlotPick, ScreenCreateCommandName, ScreenEditCommandName, ScreenEditCommandMode, ScreenEditCommandTemplate:
		return true
	case ScreenNameInput:
		return m.nameInputMode == nameInputRenameCommand
	}
	return false
}

func (m Model) templatePaneKeys() string {
	switch m.screen {
	case ScreenSlotPick:
		if m.sp.variadic {
			return "Type Filter / Value  ↑↓ Select  Tab Toggle  Enter Confirm  Esc Back"
		}
		return "Type Filter / Value  ↑↓ Select  Enter Confirm  Esc Back"
	case ScreenCreateCommandName, ScreenEditCommandName:
		return "Tab Suggest name  Enter Save  Esc Back"
	case ScreenNameInput:
		return "Enter Save name  Esc Back"
	}
	return "↑↓ Select  Enter Open  Esc Back"
}

func (m Model) exploreTemplateBox(l exploreLayout) string {
	var b strings.Builder
	title := "From template"
	if m.sce.mode == 1 {
		title = "Edit · " + m.sce.name
	}
	rows := max(1, l.editH-5)
	draw := func(items []string, cursor int) {
		writeWindowedList(&b, rows, len(items), cursor, "(none)", func(i int) string {
			mark := "   "
			if i == cursor {
				mark = " ▶ "
			}
			text := truncate(items[i], max(1, l.rightW-6))
			if i == cursor && m.screen == ScreenSlotPick {
				text = sCurrentValue.Render(text)
			}
			if m.screen == ScreenSlotPick && i < len(m.sp.filtered) {
				return searchHighlight(mark+text, []string{m.sp.search}, i == cursor, true)
			}
			return mark + text
		})
	}
	switch m.screen {
	case ScreenEditCommandMode:
		draw(m.listItems, m.listCursor)
	case ScreenEditCommandTemplate:
		title = "Change template"
		var names []string
		for _, cmd := range m.templateCandidates() {
			names = append(names, cmd.Name)
		}
		draw(names, m.sce.templateRefIdx)
	case ScreenSlotPick:
		sp := m.sp
		title += " · " + sp.placeholder()
		b.WriteString(" / " + sp.search + "_\n")
		var choices []string
		for _, entry := range sp.filtered {
			label := entry.Value
			if entry.Label != "" {
				label += " · " + entry.Label
			}
			if sp.variadic {
				if sp.isPicked(entry.Value) {
					label = "[x] " + label
				} else {
					label = "[ ] " + label
				}
			}
			choices = append(choices, label)
		}
		choices = append(choices, "Use typed value: "+sp.search)
		if sp.canSkip {
			choices = append(choices, "Keep interactive (choose at run time)")
		}
		draw(choices, sp.cursor)
	default:
		input := m.nameInput
		input.Prompt = "Name > "
		input.Width = max(1, l.rightW-12)
		b.WriteString(" " + input.View() + "\n")
		if m.nameInputErr != "" {
			b.WriteString(" " + errorText(m.nameInputErr) + "\n")
		}
		for _, name := range sortedKeys(m.sce.currentValues) {
			b.WriteString(" " + name + " = " + m.sce.currentValues[name] + "\n")
		}
	}
	return box("← Back · "+title, b.String(), l.rightW, l.editH, true)
}

func (m Model) writeTemplatePreview(b *strings.Builder) {
	cands := m.templateCandidates()
	if m.sce.templateRefIdx < 0 || m.sce.templateRefIdx >= len(cands) {
		return
	}
	tpl := cands[m.sce.templateRefIdx]
	b.WriteString(" " + gray("Template: ") + tpl.Name + "\n")
	values := copyMap(m.sce.currentValues)
	if m.screen == ScreenEditCommandMode && m.sce.editIdx >= 0 && m.sce.editIdx < len(m.config.Commands) {
		values = m.config.Commands[m.sce.editIdx].Values
	}
	// Preview what Enter would accept, without committing the hovered value.
	if m.screen == ScreenSlotPick && m.sp != nil {
		sp := m.sp
		delete(values, sp.slotName)
		switch {
		case sp.canSkip && sp.cursor == len(sp.filtered)+1:
			// Keep the placeholder when the interactive option is selected.
		case sp.variadic && len(sp.picked) > 0:
			values[sp.slotName] = strings.Join(sp.picked, " ")
		case sp.cursor >= 0 && sp.cursor < len(sp.filtered):
			values[sp.slotName] = sp.filtered[sp.cursor].Value
		case sp.cursor == len(sp.filtered) && sp.search != "":
			values[sp.slotName] = sp.search
		}
		b.WriteString(" " + dim("Preview only · Enter confirms this value") + "\n")
	}
	draft, err := slot.MaterializeCommand(mdl.Command{Template: tpl.Name, Values: values}, m.config)
	if err == nil {
		active := ""
		if m.screen == ScreenSlotPick && m.sp != nil {
			active = m.sp.slotName
		}
		b.WriteString(" " + dim("workdir  ") + m.templateSyntax(displayOr(tpl.Dir, "."), values, active) + "\n")
		b.WriteString(" " + gray("$ ") + m.templateSyntax(tpl.Cmd, values, active) + "\n")
		m.writeSyntaxWarnings(b, draft.Dir+" "+draft.Cmd)
	}
}
