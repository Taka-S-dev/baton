package tui

import (
	"path/filepath"
	"strings"

	"github.com/Taka-S-dev/baton/internal/store"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// Keep run selections separate from the editor's temporary command picker.
// Names, rather than indices or pointers, survive edits and reordering safely.
type runHomeState struct {
	ready                        bool
	searching                    bool
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
	m.runHome.searching = false
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
	m.syncRunSearch()
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
	return m, nil
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
	labels := []string{"[1 Workflows]", "[2 Commands]"}
	text := ""
	col := 0
	var actions []runHomeAction
	for i, label := range labels {
		if i > 0 {
			text += " "
			col++
		}
		end := col + lipgloss.Width(label)
		if end > w {
			break
		}
		if (i == 0 && m.screen == ScreenRunWorkflow) || (i == 1 && m.screen == ScreenRunCommands) {
			text += accentBold(label)
		} else {
			text += gray(label)
		}
		actions = append(actions, runHomeAction{i, col, end})
		col = end
	}
	if name := filepath.Base(m.projectDir); m.projectDir != "" && w-col > 5 {
		text += gray(" · " + ansi.Truncate(name, w-col-3, "…"))
	}
	return text, actions
}

// runHomeKeys is the key hint line of the Run workspace, in the editor's
// navigation vocabulary; compact screens keep the list first.
func (m Model) runHomeKeys() string {
	if m.runHome.searching {
		selectKey := ""
		if m.screen == ScreenRunCommands {
			selectKey = " · Tab Select"
		}
		if m.runSearchInput().Value() == "" {
			return "Search: ↓ Results · Enter Review" + selectKey + " · Esc Done · F1 Help"
		}
		return "Search: ↓ Results · Enter Review" + selectKey + " · Esc Clear · F1 Help"
	}
	if m.runQueueFocused && m.screen == ScreenRunCommands {
		return "J/K Reorder · Del Remove · F2 List · / Search · e Edit · p Projects · q Quit · F1 Help"
	}
	return "↑↓ Move · Enter Review · / Search · e Edit · p Projects · q Quit · F1 Help"
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
			if m.runHome.searching && key.String() != "f1" && key.String() != "ctrl+c" {
				return m.updateRunSearch(key)
			}
			switch key.String() {
			case "/":
				m.runQueueFocused = false
				m.runHome.searching = true
				return m, m.syncRunSearch()
			case "1":
				return m.switchRunTab("workflows")
			case "2":
				return m.switchRunTab("commands")
			case "s":
				if m.screen == ScreenRunCommands && !m.runQueueFocused {
					m.cycleCommandSort()
				}
				return m, nil
			case "e":
				return m.editFromRunHome()
			case "p":
				m.openRunProjects()
				return m, nil
			case "q":
				return m, tea.Quit
			}
			if key.Type == tea.KeyRunes || key.Type == tea.KeySpace || key.Type == tea.KeyBackspace {
				if !(m.screen == ScreenRunCommands && m.runQueueFocused && (key.String() == "J" || key.String() == "K" || key.String() == "x")) {
					return m, nil
				}
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
						}
					}
				}
				return m, nil
			}
			mouse.Y--
			if mouse.Button == tea.MouseButtonLeft && mouse.Action == tea.MouseActionPress && m.runSearchHit(mouse.X, mouse.Y) {
				m.runQueueFocused = false
				m.stepsFocused = false
				m.runHome.searching = true
				return m, m.syncRunSearch()
			}
			if mouse.Button == tea.MouseButtonRight && m.screen == ScreenRunCommands {
				return m.updateRunSelectMouse(mouse)
			}
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
		body = gray("No commands yet. e Add your first command.")
	}
	if m.screen == ScreenRunWorkflow && len(m.workflows) == 0 {
		body = gray("No workflows yet. e Open editor to add one.")
	}
	lines := strings.Split(body, "\n")
	for i := range lines {
		lines[i] = ansi.Truncate(lines[i], w, "…")
	}
	content := m.runHomeHeader(w) + "\n" + strings.Join(lines, "\n")
	return content
}

func (m *Model) runSearchInput() *textinput.Model {
	if m.screen == ScreenRunWorkflow {
		return &m.wfSearchTI
	}
	return &m.msSearchTI
}

func (m *Model) syncRunSearch() tea.Cmd {
	input := m.runSearchInput()
	if m.runHome.searching {
		input.Prompt = "Search / "
		input.Placeholder = ""
		return input.Focus()
	}
	input.Prompt = "/ "
	input.Placeholder = "Search…"
	input.Blur()
	return nil
}

func (m Model) updateRunSearch(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.String() == "down" && ((m.screen == ScreenRunCommands && len(m.msFiltered()) == 0) || (m.screen == ScreenRunWorkflow && len(m.wfFiltered()) == 0)) {
		return m, nil
	}
	if key.String() == "tab" && m.screen == ScreenRunCommands {
		return m.updateMultiSelect(key)
	}
	input := m.runSearchInput()
	query := input.Value()
	handled, changed, closed, cmd := updateSearchInput(input, &query, key, m.runSearchCanAccept())
	if changed {
		if m.screen == ScreenRunWorkflow {
			m.listCursor = 0
			m.updateStepsViewport()
		} else {
			m.msCursor = 0
		}
	}
	if closed {
		m.runHome.searching = false
		m.syncRunSearch()
	}
	if handled {
		return m, cmd
	}
	if m.screen == ScreenRunWorkflow {
		return m.updateRunWorkflow(key)
	}
	return m.updateMultiSelect(key)
}

func (m Model) runSearchCanAccept() bool {
	if m.screen == ScreenRunWorkflow {
		return len(m.wfFiltered()) > 0
	}
	return len(m.msFiltered()) > 0 || len(m.msSelected) > 0
}
