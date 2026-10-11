package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	sAccent    = lipgloss.NewStyle().Foreground(lipgloss.Color("36"))
	sGray      = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	sWhite     = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	sSuccess   = lipgloss.NewStyle().Foreground(lipgloss.Color("32"))
	sWarn      = lipgloss.NewStyle().Foreground(lipgloss.Color("220")) // gold — warnings and confirm prompts
	sHighlight = lipgloss.NewStyle().Foreground(lipgloss.Color("220")) // gold — emphasis (e.g. the {slot} being resolved), not a warning
	sError     = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	sBold      = lipgloss.NewStyle().Bold(true)
	sDim       = lipgloss.NewStyle().Faint(true)
	sCursor    = lipgloss.NewStyle().Background(lipgloss.Color("238")).Foreground(lipgloss.Color("97"))
	// The overview's selected row, drawn as a bar across the pane. It
	// dims when the pane beside it has the keys, so which half is being
	// driven is readable without a frame around either.
	sRowActive   = lipgloss.NewStyle().Background(lipgloss.Color("23")).Foreground(lipgloss.Color("255"))
	sRowInactive = lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("249"))
	sSelNum      = lipgloss.NewStyle().Foreground(lipgloss.Color("32"))
	sGroup       = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	sAccentBold  = lipgloss.NewStyle().Foreground(lipgloss.Color("36")).Bold(true)
	sSlotVar     = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true) // bright cyan, consistent across terminals

	sBtnSelected = lipgloss.NewStyle().
			Padding(0, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("36")).
			Foreground(lipgloss.Color("97")).
			Bold(true)
	sBtnNormal = lipgloss.NewStyle().
			Padding(0, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("238")).
			Foreground(lipgloss.Color("240"))
)

func accent(s string) string     { return sAccent.Render(s) }
func gray(s string) string       { return sGray.Render(s) }
func white(s string) string      { return sWhite.Render(s) }
func success(s string) string    { return sSuccess.Render(s) }
func warn(s string) string       { return sWarn.Render(s) }
func highlight(s string) string  { return sHighlight.Render(s) }
func errorText(s string) string  { return sError.Render(s) }
func accentBold(s string) string { return sAccentBold.Render(s) }
func slotVar(s string) string    { return sSlotVar.Render(s) }
func dim(s string) string        { return sDim.Render(s) }
func bold(s string) string       { return sBold.Render(s) }
func renderBtn(label string, selected bool) string {
	if selected {
		return sBtnSelected.Render(label)
	}
	return sBtnNormal.Render(label)
}

func renderBtns(selected int, labels ...string) string {
	rendered := make([]string, len(labels))
	for i, label := range labels {
		rendered[i] = renderBtn(label, i == selected)
	}
	joined := lipgloss.JoinHorizontal(lipgloss.Center, rendered...)
	return lipgloss.NewStyle().MarginLeft(2).Render(joined)
}

func hline(width int) string {
	w := max(0, (width-4)/1)
	return gray("  " + strings.Repeat("─", w))
}

func hlineLabel(width int, label string) string {
	w := max(0, width-4-len(label)-4)
	return gray("  ── " + label + " " + strings.Repeat("─", w))
}

func hlineLabelBright(width int, label string) string {
	w := max(0, width-4-len(label)-4)
	return gray("  ── ") + white(label) + gray(" "+strings.Repeat("─", w))
}

func header(title string, width int) string {
	return accentBold("  [ "+title+" ]") + "\n" + hline(width) + "\n"
}

// formatDuration renders an elapsed time at a precision that suits its
// length: hundredths below a minute, whole seconds above it. Go's own
// formatting would print 1.170448s next to 4m12.003s.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return d.Round(10 * time.Millisecond).String()
	}
	return d.Round(time.Second).String()
}

func progressBar(done, total, barWidth int) string {
	filled := 0
	pct := 0
	if total > 0 {
		filled = done * barWidth / total
		pct = done * 100 / total
	}
	bar := success(strings.Repeat("█", filled)) + gray(strings.Repeat("─", barWidth-filled))
	return "  [" + bar + "]  " + white(fmt.Sprintf("%d/%d steps", done, total)) + "  " + gray(fmt.Sprintf("%d%%", pct))
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
