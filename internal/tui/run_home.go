package tui

import (
	"path/filepath"
	"strings"

	"github.com/Taka-S-dev/baton/internal/store"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// Keep run selections separate from the editor's temporary command picker.
// Names, rather than indices or pointers, survive edits and reordering safely.
type runHomeState struct {
	ready                        bool
	tab                          string
	commandSearch, commandName   string
	selected                     []string
	commandCursor, queueCursor   int
	queueFocused                 bool
	workflowSearch, workflowName string
	workflowCursor               int
}

func (m Model) runHomeVisible() bool {
	return m.runHome.ready && (m.screen == ScreenRunCommands || m.screen == ScreenRunWorkflow)
}

func (m *Model) captureRunHome() {
	if !m.runHomeVisible() {
		return
	}
	if m.screen == ScreenRunCommands {
		m.runHome.tab = "commands"
		m.runHome.commandSearch = m.msSearchTI.Value()
		m.runHome.commandCursor = m.msCursor
		m.runHome.queueCursor, m.runHome.queueFocused = m.runQueueCursor, m.runQueueFocused
		m.runHome.commandName = ""
		filtered := m.msFiltered()
		if m.msCursor >= 0 && m.msCursor < len(filtered) {
			m.runHome.commandName = m.msItems[filtered[m.msCursor]].name()
		}
		m.runHome.selected = nil
		for _, i := range m.msSelected {
			if i >= 0 && i < len(m.msItems) {
				m.runHome.selected = append(m.runHome.selected, m.msItems[i].name())
			}
		}
	} else {
		m.runHome.tab = "workflows"
		m.runHome.workflowSearch = m.wfSearchTI.Value()
		m.runHome.workflowCursor = m.listCursor
		m.runHome.workflowName = ""
		filtered := m.wfFiltered()
		if m.listCursor >= 0 && m.listCursor < len(filtered) {
			m.runHome.workflowName = m.workflows[filtered[m.listCursor]].Name
		}
	}
}

func (m *Model) openRunHome() {
	if !m.runHome.ready {
		m.runHome.ready = true
		m.runHome.tab = store.LoadRunTab(m.projectDir)
		if m.runHome.tab == "" {
			m.runHome.tab = "commands"
			if len(m.workflows) > 0 {
				m.runHome.tab = "workflows"
			}
		}
		// An empty project always offers its first command.
		if len(m.workflows) == 0 && len(m.config.AllCommands()) == 0 {
			m.runHome.tab = "commands"
		}
		m.runHome.workflowName = m.lastWorkflow
		if m.projectDir != "" {
			_ = store.SaveRunTab(m.projectDir, m.runHome.tab)
		}
	}
	m.msEscArmed = false
	if m.runHome.tab == "workflows" {
		m.screen = ScreenRunWorkflow
		m.wfSearchTI = newMSTI("/ ")
		m.wfSearchTI.SetValue(m.runHome.workflowSearch)
		m.wfSearchTI.Focus()
		filtered := m.wfFiltered()
		m.listCursor = min(m.runHome.workflowCursor, max(0, len(filtered)-1))
		for i, idx := range filtered {
			if m.workflows[idx].Name == m.runHome.workflowName {
				m.listCursor = i
				break
			}
		}
		m.stepsFocused = false
		m.updateStepsViewport()
	} else {
		m.screen = ScreenRunCommands
		m.setupMultiSelectWithPreSelected(m.runHome.selected)
		m.msSearchTI.SetValue(m.runHome.commandSearch)
		filtered := m.msFiltered()
		m.msCursor = min(m.runHome.commandCursor, max(0, len(filtered)-1))
		for i, idx := range filtered {
			if m.msItems[idx].name() == m.runHome.commandName {
				m.msCursor = i
				break
			}
		}
		m.runQueueCursor = min(m.runHome.queueCursor, max(0, len(m.msSelected)-1))
		m.runQueueFocused = m.runHome.queueFocused
	}
}

func (m Model) switchRunTab(tab string) (tea.Model, tea.Cmd) {
	m.captureRunHome()
	m.runHome.tab = tab
	m.openRunHome()
	if m.projectDir != "" {
		if err := store.SaveRunTab(m.projectDir, tab); err != nil {
			m.errMsg = "Could not remember run tab: " + err.Error()
		}
	}
	if tab == "commands" {
		return m, m.msSearchTI.Focus()
	}
	return m, m.wfSearchTI.Focus()
}

func (m *Model) openRunProjects() {
	m.screen = ScreenProjectSelect
	m.projectMode, m.projectFilter = "", ""
	m.refreshProjects()
	m.listItems = m.projects
	m.listCursor = 0
	for i, name := range m.projects {
		if name == filepath.Base(m.projectDir) {
			m.listCursor = i
			break
		}
	}
}

func (m Model) editFromRunHome() (tea.Model, tea.Cmd) {
	m.captureRunHome()
	m.exploreTab = tabCommands
	name := m.runHome.commandName
	if m.runHome.tab == "workflows" {
		m.exploreTab = tabWorkflows
		name = m.runHome.workflowName
	}
	m.exploreSearch, m.exploreSearching = "", false
	m.exploreHistory = nil
	m.gotoExplore()
	for i, it := range m.exploreItems(m.exploreTab) {
		if it.name == name {
			m.exploreCursor[m.exploreTab] = i
			break
		}
	}
	if len(m.config.AllCommands()) == 0 && m.exploreTab == tabCommands {
		cmd := m.openExploreCommandForm("")
		return m, cmd
	}
	return m, nil
}

// The renderer and pointer use the same visible spans, including wide project names.
type runHomeAction struct {
	id, start, end int
}

func (m Model) runHomeLayout(w int) (string, []runHomeAction) {
	labels := []string{"[F7 Workflows]", "[F8 Commands]", "F4 Edit", "Projects…", "F1 Help"}
	if w < 76 {
		labels[0], labels[1] = "[F7 WF]", "[F8 Cmd]"
	}
	name := filepath.Base(m.projectDir)
	if m.projectDir == "" {
		name = "Run"
	}
	// Reserve both tabs before allocating space to the project name.
	nameWidth := min(20, max(1, w-lipgloss.Width(labels[0]+labels[1])-5))
	prefix := ansi.Truncate(name, nameWidth, "…") + " › "
	text := white(prefix)
	col := lipgloss.Width(prefix)
	var actions []runHomeAction
	for i, label := range labels {
		sep := " "
		if i == 0 {
			sep = ""
		} else if i == 2 {
			sep = "  │  "
		} else if i > 2 {
			sep = " · "
		}
		size := lipgloss.Width(sep + label)
		// Hide whole secondary actions instead of leaving misleading fragments.
		if col+size > w {
			break
		}
		text += gray(sep)
		start := col + lipgloss.Width(sep)
		if (i == 0 && m.screen == ScreenRunWorkflow) || (i == 1 && m.screen == ScreenRunCommands) {
			text += accentBold(label)
		} else {
			text += gray(label)
		}
		col += size
		actions = append(actions, runHomeAction{i, start, col})
	}
	return ansi.Truncate(text, w, "…"), actions
}

func (m Model) runHomeHeader(w int) string {
	text, _ := m.runHomeLayout(w)
	return text
}

// Update owns the run-home frame; screen handlers keep their local coordinates.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.captureRunHome()
	if m.runHomeVisible() && !m.helpOpen && !m.msEscArmed && !m.pasteMenu && m.leaveAsk == leaveNone {
		if key, ok := msg.(tea.KeyMsg); ok {
			switch key.String() {
			case "f7":
				return m.switchRunTab("workflows")
			case "f8":
				return m.switchRunTab("commands")
			case "f4":
				return m.editFromRunHome()
			case "f6":
				m.openRunProjects()
				return m, nil
			}
		}
		if mouse, ok := msg.(tea.MouseMsg); ok {
			if mouse.Y == 0 {
				if mouse.Action == tea.MouseActionPress && mouse.Button == tea.MouseButtonLeft {
					_, actions := m.runHomeLayout(m.viewWidth())
					for _, action := range actions {
						if mouse.X < action.start || mouse.X >= action.end {
							continue
						}
						switch action.id {
						case 0:
							return m.switchRunTab("workflows")
						case 1:
							return m.switchRunTab("commands")
						case 2:
							return m.editFromRunHome()
						case 3:
							m.openRunProjects()
							return m, nil
						case 4:
							m.helpOpen = true
							m.helpOffset = 0
							return m, nil
						}
					}
				}
				return m, nil
			}
			mouse.Y--
			msg = mouse
		}
	}
	next, cmd := m.updateScreen(msg)
	if result, ok := next.(Model); ok {
		result.captureRunHome()
		return result, cmd
	}
	return next, cmd
}

func (m Model) viewRunHome(body string, w int) string {
	if m.screen == ScreenRunCommands && len(m.msItems) == 0 {
		body = gray("No commands yet. F4 Add your first command.")
	}
	if m.screen == ScreenRunWorkflow && len(m.workflows) == 0 {
		body = gray("No workflows yet. F4 Open editor to add one.")
	}
	lines := strings.Split(body, "\n")
	for i := range lines {
		lines[i] = ansi.Truncate(lines[i], w, "…")
	}
	return m.runHomeHeader(w) + "\n" + strings.Join(lines, "\n")
}
