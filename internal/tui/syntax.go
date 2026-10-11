package tui

import (
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/Taka-S-dev/baton/internal/slot"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

var syntaxToken = regexp.MustCompile(slot.VarPattern.String() + "|" + slot.Pattern.String())
var sFixedVar = lipgloss.NewStyle().Foreground(lipgloss.Color("141"))
var sCurrentValue = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)

// syntaxText styles source references without changing their text.
func (m Model) syntaxText(text string) string {
	return syntaxToken.ReplaceAllStringFunc(text, func(token string) string {
		if strings.HasPrefix(token, "{$") {
			if _, ok := m.vars[token[2:len(token)-1]]; !ok {
				return sError.Bold(true).Render(token)
			}
			return sFixedVar.Render(token)
		}
		return slotVar(token)
	})
}

// previewVars resolves once, retaining the provenance of each substituted span.
func (m Model) previewVars(text string) string {
	var b strings.Builder
	end := 0
	for _, at := range slot.VarPattern.FindAllStringIndex(text, -1) {
		b.WriteString(m.syntaxText(text[end:at[0]]))
		ref := text[at[0]:at[1]]
		if value, ok := m.vars[ref[2:len(ref)-1]]; ok {
			b.WriteString(sFixedVar.Render(value))
		} else {
			b.WriteString(sError.Bold(true).Render(ref))
		}
		end = at[1]
	}
	b.WriteString(m.syntaxText(text[end:]))
	return b.String()
}

// templateSyntax highlights by placeholder origin, never by matching the
// selected value's text. A literal "prod" beside {env} must not light up
// when env resolves to prod.
func (m Model) templateSyntax(text string, values map[string]string, active string) string {
	var b strings.Builder
	end := 0
	for _, at := range slot.Pattern.FindAllStringIndex(text, -1) {
		b.WriteString(m.previewVars(text[end:at[0]]))
		ref := text[at[0]:at[1]]
		name := slot.RefName(ref)
		value, ok := values[name]
		switch {
		case name == active:
			if !ok {
				value = ref
			}
			b.WriteString(sCurrentValue.Render(slot.ApplyVars(value, m.vars)))
		case ok:
			b.WriteString(m.previewVars(value))
		default:
			b.WriteString(slotVar(ref))
		}
		end = at[1]
	}
	b.WriteString(m.previewVars(text[end:]))
	return b.String()
}

func (m Model) writeSyntaxWarnings(b *strings.Builder, text string) {
	seen := map[string]bool{}
	for _, name := range slot.UndefinedVars(text, m.vars) {
		if !seen[name] {
			b.WriteString(" " + errorText("⚠ Undefined variable: {$"+name+"}") + "\n")
			seen[name] = true
		}
	}
}

// selectedSyntaxRow renders every span with the selected background;
// nested ANSI resets must not erase it between highlighted tokens.
func (m Model) selectedSyntaxRow(text string, width int) string {
	text = ansi.Truncate(ansi.Strip(text), max(1, width), "…")
	text += strings.Repeat(" ", max(0, width-lipgloss.Width(text)))
	var b strings.Builder
	end := 0
	for _, at := range syntaxToken.FindAllStringIndex(text, -1) {
		b.WriteString(sRowActive.Render(text[end:at[0]]))
		token := text[at[0]:at[1]]
		style := sSlotVar
		if strings.HasPrefix(token, "{$") {
			style = sFixedVar
			if _, ok := m.vars[token[2:len(token)-1]]; !ok {
				style = sError.Bold(true)
			}
		}
		b.WriteString(style.Background(sRowActive.GetBackground()).Render(token))
		end = at[1]
	}
	b.WriteString(sRowActive.Render(text[end:]))
	return b.String()
}

// recentCommandLine draws a command with its placeholders coloured as
// everywhere else, and the search matches on top. A match inside a
// placeholder shows as a match: that is what was typed to find it.
func (m Model) recentCommandLine(text, query string) string {
	mask := recentMatchMask(text, query)
	r := []rune(text)
	var b strings.Builder
	end := 0 // in runes, as the mask is
	for _, at := range syntaxToken.FindAllStringIndex(text, -1) {
		start, stop := utf8.RuneCountInString(text[:at[0]]), utf8.RuneCountInString(text[:at[1]])
		b.WriteString(recentHighlightMask(string(r[end:start]), mask[end:start], false))
		if slices.Contains(mask[start:stop], true) {
			b.WriteString(recentHighlightMask(string(r[start:stop]), mask[start:stop], false))
		} else {
			b.WriteString(m.syntaxText(string(r[start:stop])))
		}
		end = stop
	}
	b.WriteString(recentHighlightMask(string(r[end:]), mask[end:], false))
	return b.String()
}
