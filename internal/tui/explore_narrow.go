package tui

import "strings"

func (m Model) viewNarrowWorkflow(w int) string {
	var b strings.Builder
	m.writeHeading(&b, m.exploreTitle(), w)
	rows := max(3, m.termHeight()-strings.Count(b.String(), "\n")-2)
	var body strings.Builder
	m.writeWorkflowEditor(&body, max(1, w-2), max(1, rows-2))
	title := "Edit workflow"
	if m.wfe.idx < len(m.workflows) {
		title += " · " + m.workflows[m.wfe.idx].Name
	}
	b.WriteString(box(title, body.String(), w, rows, true))
	b.WriteString("\n" + gray(m.exploreKeys()+"  v Details  F1 Help"))
	return b.String()
}

func (m Model) viewNarrowDetails(w int) string {
	var b strings.Builder
	m.writeHeading(&b, m.exploreTitle(), w)
	l := m.exploreLayout(w)
	l.rightW = w
	l.prevH = max(3, m.termHeight()-strings.Count(b.String(), "\n")-2)
	b.WriteString(m.explorePreviewBox(l))
	b.WriteString("\n" + gray("Esc / v Back  F1 Help"))
	return b.String()
}

func (m Model) viewNarrowCommand(w int) string {
	var body strings.Builder
	m.writeCommandFields(&body, m.cf.fields, m.cf.fieldIdx, max(1, w-2))
	lines := strings.Split(strings.TrimRight(body.String(), "\n"), "\n")
	rows := max(1, m.termHeight()-3)
	start := 0
	if len(lines) > rows {
		for i, line := range lines {
			if strings.Contains(line, m.nameInput.View()) {
				start = max(0, i-rows+1)
				break
			}
		}
	}
	end := min(len(lines), start+rows)
	return box(m.exploreTitle(), strings.Join(lines[start:end], "\n"), w, rows+2, true) + "\n" + gray(m.formKeys())
}
