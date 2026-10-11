package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func searchStateLine(query string) string {
	if query != "" {
		return gray("Filter: ") + white(query)
	}
	return gray("/ Search…")
}

func startSearchInput(input *textinput.Model, query string) tea.Cmd {
	*input = newMSTI("Search / ")
	input.Placeholder = ""
	input.SetValue(query)
	input.CursorEnd()
	return input.Focus()
}

func searchInputView(input textinput.Model, query string, editing bool) string {
	if !editing {
		return searchStateLine(query)
	}
	input.Prompt = "Search / "
	input.Placeholder = ""
	return input.View()
}

func (m Model) runSearchView(input textinput.Model) string {
	if !m.runHomeVisible() {
		return searchInputView(input, input.Value(), input.Focused())
	}
	return searchInputView(input, input.Value(), m.runHome.searching)
}

// updateSearchInput shares editing and dismissal across searchable lists.
// Down transfers focus without advancing the selected result; Enter accepts it.
func updateSearchInput(input *textinput.Model, query *string, key tea.KeyMsg, canAccept bool) (handled, changed, closed bool, cmd tea.Cmd) {
	var focus tea.Cmd
	if !input.Focused() {
		focus = startSearchInput(input, *query)
	}
	if input.Value() != *query {
		input.SetValue(*query)
	}
	switch key.String() {
	case "enter":
		if !canAccept {
			return true, false, false, focus
		}
		input.Blur()
		return false, false, true, nil
	case "esc":
		if *query != "" {
			*query = ""
			input.SetValue("")
			return true, true, false, focus
		}
		input.Blur()
		return true, false, true, nil
	case "down":
		if !canAccept {
			return true, false, false, focus
		}
		input.Blur()
		return true, false, true, nil
	case "up":
		return false, false, false, focus
	}
	*input, cmd = input.Update(key)
	changed = input.Value() != *query
	*query = input.Value()
	return true, changed, false, tea.Batch(focus, cmd)
}

// commandSearchLine reserves a stable right-hand area for list metadata.
func (m Model) commandSearchLine(w, matches int) string {
	if w <= 0 {
		return ""
	}
	count := fmt.Sprintf("%d/%d", matches, len(m.msItems))
	sortKey := "s"
	meta := count + " · " + m.commandSortLabel() + " · " + sortKey + " Sort"
	// Keep at least 20 columns for search before dropping secondary metadata.
	if w-ansi.StringWidth(meta)-2 < 20 {
		meta = count
	}
	if w-ansi.StringWidth(meta)-2 < 10 {
		meta = ""
	}
	searchW := w
	if meta != "" {
		searchW -= ansi.StringWidth(meta) + 2
	}
	input := m.msSearchTI
	promptW := ansi.StringWidth(input.Prompt)
	if m.runHomeVisible() && m.runHome.searching {
		promptW = ansi.StringWidth("Search / ")
	}
	input.Width = max(1, searchW-promptW-1)
	search := ansi.Truncate(m.runSearchView(input), searchW, "…")
	if meta == "" {
		return search
	}
	return search + strings.Repeat(" ", max(2, w-ansi.StringWidth(search)-ansi.StringWidth(meta))) + gray(meta)
}

// Use the same renderers for search placement and pointer targeting.
func (m Model) runSearchHit(x, y int) bool {
	w := m.viewWidth()
	leftW := w
	row := -1
	if m.screen == ScreenRunCommands {
		if splitPanels(w) {
			leftW = w / 2
		} else if m.runQueueFocused {
			return false
		}
		m.viewMultiSelectCandidates(leftW, !splitPanels(w), func(index, y int) {
			if index == -2 {
				row = y
			}
		})
	} else {
		if splitPanels(w) {
			leftW = w / 3
		}
		m.viewRunWorkflow(w, func(y int) { row = y })
	}
	return x >= 0 && x < leftW && y == row
}
