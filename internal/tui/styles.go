package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	sCyan     = lipgloss.NewStyle().Foreground(lipgloss.Color("36"))
	sGray     = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	sWhite    = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	sGreen    = lipgloss.NewStyle().Foreground(lipgloss.Color("32"))
	sYellow   = lipgloss.NewStyle().Foreground(lipgloss.Color("33"))
	sRed      = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	sBold     = lipgloss.NewStyle().Bold(true)
	sDim      = lipgloss.NewStyle().Faint(true)
	sCursor     = lipgloss.NewStyle().Background(lipgloss.Color("238")).Foreground(lipgloss.Color("97"))
	sMenuSelect = lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("255")).Bold(true).Width(26)
	sSelNum   = lipgloss.NewStyle().Foreground(lipgloss.Color("32"))
	sGroup    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	sCyanBold  = lipgloss.NewStyle().Foreground(lipgloss.Color("36")).Bold(true)
	sSlotVar   = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true) // bright cyan, consistent across terminals
	sGreenBold = lipgloss.NewStyle().Foreground(lipgloss.Color("92")).Bold(true)

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

func cyan(s string) string      { return sCyan.Render(s) }
func gray(s string) string      { return sGray.Render(s) }
func white(s string) string     { return sWhite.Render(s) }
func green(s string) string     { return sGreen.Render(s) }
func yellow(s string) string    { return sYellow.Render(s) }
func red(s string) string       { return sRed.Render(s) }
func cyanBold(s string) string  { return sCyanBold.Render(s) }
func slotVar(s string) string   { return sSlotVar.Render(s) }
func greenBold(s string) string { return sGreenBold.Render(s) }
func dim(s string) string       { return sDim.Render(s) }
func bold(s string) string      { return sBold.Render(s) }
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
	return cyanBold("  [ "+title+" ]") + "\n" + hline(width) + "\n"
}

func progressBar(done, total, barWidth int) string {
	filled := 0
	pct := 0
	if total > 0 {
		filled = done * barWidth / total
		pct = done * 100 / total
	}
	bar := green(strings.Repeat("#", filled)) + gray(strings.Repeat("-", barWidth-filled))
	return "  [" + bar + "]  " + white(fmt.Sprintf("%d/%d", done, total)) + "  " + gray(fmt.Sprintf("%d%%", pct))
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
