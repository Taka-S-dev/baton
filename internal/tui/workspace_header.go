package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

var sEditMode = lipgloss.NewStyle().Foreground(lipgloss.Color("179")).Bold(true)

// Only the mode name changes color; project names and navigation stay neutral.
func workspaceTitle(title string, width int) string {
	mode, rest, _ := strings.Cut(title, " · ")
	style := sWhite.Bold(true)
	if mode == "Edit project" || strings.HasPrefix(mode, "Edit project ›") {
		style = sEditMode
	}
	if ansi.StringWidth(mode) >= width {
		return style.Render(ansi.Truncate(mode, width, "…"))
	}
	if rest == "" {
		return style.Render(mode)
	}
	return style.Render(mode) + white(ansi.Truncate(" · "+rest, width-ansi.StringWidth(mode), "…"))
}

// workspaceHeading shares both rendering and the action's pointer span.
func workspaceHeading(title, action string, w int) (string, int) {
	inner := max(0, w-4)
	start := w - 2 - ansi.StringWidth(action)
	if start < 8 {
		return "  " + workspaceTitle(title, inner), -1
	}
	label := ansi.Truncate(title, max(1, start-4), "…")
	return "  " + workspaceTitle(title, max(1, start-4)) + strings.Repeat(" ", max(2, start-2-ansi.StringWidth(label))) + gray(action), start
}

func (m Model) runHeaderRows() int {
	if m.rows()-m.bannerRows >= 20 {
		return 4
	}
	return 2
}

func (m Model) workspaceTabRow() int {
	if m.rows()-m.bannerRows < 20 {
		return 1
	}
	return 2
}

func (m Model) hideWorkspaceHints() bool {
	if m.rows()-m.bannerRows >= 20 {
		return false
	}
	if m.screen == ScreenExplore {
		return m.exploreListFocused() && !m.exploreSearching && m.leaveAsk == leaveNone && !m.exploreMenu
	}
	return m.runHomeVisible() && !m.runHome.searching && !m.msEscArmed && m.leaveAsk == leaveNone
}

func (m Model) workflowBodyHeight() int {
	if m.hideWorkspaceHints() {
		return m.termHeight()
	}
	return m.termHeight() - 1
}
