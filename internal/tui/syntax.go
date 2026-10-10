package tui

import (
	"github.com/Taka-S-dev/baton/internal/slot"
	"github.com/charmbracelet/lipgloss"
	"regexp"
	"strings"
)

var syntaxToken = regexp.MustCompile(`\{(?:\$\w+|\w+(?:\.\.\.)?)\}`)
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

// Highlight by placeholder origin, never by matching the selected value's text.
// A literal "prod" beside {env} must not light up when env resolves to prod.
func (m Model) templateSyntax(text string, values map[string]string, active string) string {
	var b strings.Builder
	end := 0
	for _, at := range slot.Pattern.FindAllStringIndex(text, -1) {
		b.WriteString(m.previewVars(text[end:at[0]]))
		ref := text[at[0]:at[1]]
		name := strings.TrimSuffix(ref[1:len(ref)-1], "...")
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
