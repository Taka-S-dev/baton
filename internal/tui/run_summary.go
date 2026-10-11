package tui

func (m Model) runCompletionLine() string {
	r := m.running
	label := "✓ Completed"
	if m.dryRun {
		label = "✓ Dry run complete"
	}
	return "  " + success(label) + " · " + formatDuration(r.total) + " · " + r.completedAt.Format("15:04:05")
}

// runCompletionOutput keeps the final progress state in scrollback before
// the completion message.
func (m Model) runCompletionOutput() string {
	if len(m.running.items) > 1 {
		return progressBar(m.running.current, len(m.running.items), 24) + "\n" + m.runCompletionLine()
	}
	return m.runCompletionLine()
}
