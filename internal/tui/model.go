package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Taka-S-dev/baton/internal/config"
	mdl "github.com/Taka-S-dev/baton/internal/model"
	"github.com/Taka-S-dev/baton/internal/slot"
	"github.com/Taka-S-dev/baton/internal/store"
)

// Screen identifies which TUI screen is active.
type Screen int

const (
	ScreenProjectSelect Screen = iota
	ScreenMainMenu
	ScreenRunWorkflow
	ScreenRunManually
	ScreenSlotPick
	ScreenConfirmRun
	ScreenRunning
	ScreenRetry
	ScreenCreateWorkflow
	ScreenConfirmVars
	ScreenNameInput
	ScreenEditWorkflow
	ScreenEditWorkflowMode
	ScreenEditWorkflowCommands
	ScreenDeleteWorkflow
	ScreenAliasMgmt
	ScreenCreateAlias
	ScreenEditAlias
	ScreenEditAliasMode
	ScreenEditAliasCommands
	ScreenDeleteAlias
	ScreenManageLists
	ScreenEditList
	ScreenSwitchConfig
)

type nameInputMode int

const (
	nameInputWorkflow nameInputMode = iota
	nameInputAlias
	nameInputEditWorkflow
	nameInputEditAlias
	nameInputNewList
)

type resolveFlowPurpose int

const (
	purposeRunManually resolveFlowPurpose = iota
	purposeCreateWorkflow
	purposeCreateAlias
	purposeEditWorkflow
	purposeEditAlias
)

// msItem is an item shown in the multi-select screen.
type msItem struct {
	cmd   *mdl.Command
	alias *mdl.Alias
}

func (i msItem) name() string {
	if i.alias != nil {
		return i.alias.Name
	}
	return i.cmd.Name
}

func (i msItem) group() string {
	if i.alias != nil {
		return "alias"
	}
	if i.cmd.Group != "" {
		return i.cmd.Group
	}
	return ""
}

func (i msItem) isAlias() bool { return i.alias != nil }

// slotPickState holds state for the slot-picking screen.
type slotPickState struct {
	slotName  string
	listName  string
	entries   []mdl.ListEntry
	filtered  []mdl.ListEntry
	cursor int
	search string

	contextNames  []string
	contextNotes  []string
	contextIdx    int
	currentCmd    *mdl.Command
	resolvedSoFar map[string]string
}

func (s *slotPickState) applyFilter() {
	if s.search == "" {
		s.filtered = s.entries
		return
	}
	s.filtered = nil
	q := strings.ToLower(s.search)
	for _, e := range s.entries {
		if strings.Contains(strings.ToLower(e.Value), q) ||
			strings.Contains(strings.ToLower(e.Label), q) {
			s.filtered = append(s.filtered, e)
		}
	}
}

// resolveFlowState tracks multi-command slot resolution.
type resolveFlowState struct {
	purpose   resolveFlowPurpose
	rawItems  []msItem
	itemNames []string
	itemNotes []string

	currentIdx    int
	currentSlots  []slot.Def
	currentSlotIdx int
	currentValues map[string]string

	resolved     []mdl.RunItem
	workflowVars map[string]map[string]string // for Create flows
}

// confirmVarsState holds state for the Confirm Variables screen.
type confirmVarsState struct {
	cmds []mdl.Command
	vars map[string]map[string]string
	btn  int // 0=Confirm, 1=Edit
}

// runningState tracks command execution.
type runningState struct {
	items      []mdl.RunItem
	current    int
	startIdx   int
	failed     bool
	failErr    error
	completed  bool
	starting   bool // true while waiting for alt screen to exit
	label      string
	retryCount int
}

// listEditState holds state for list editing.
type listEditState struct {
	name    string
	entries []mdl.ListEntry
	cursor  int
	adding  bool
	addVal  string
	addLbl  string
	addFld  int // 0=value, 1=label
	editing    bool
	editFld    int // 0=value, 1=label
	editValTI  textinput.Model
	editLblTI  textinput.Model
}

// Model is the main bubbletea model.
type Model struct {
	dryRun      bool
	projectsDir string
	projects    []string

	projectDir string
	configFile string
	config     mdl.Config
	workflows  []mdl.Workflow
	aliases    []mdl.Alias
	lists      map[string][]mdl.ListEntry

	screen Screen
	width  int
	height int

	// Generic single-select / menu
	listCursor int
	listItems  []string

	// Multi-select
	msItems      []msItem
	msCursor     int
	msViewStart  int
	msSelected   []int
	msActiveField int
	msSearchTI   textinput.Model
	msGroupTI    textinput.Model

	// Slot picking
	sp *slotPickState

	// Resolve flow (Run manually / Create workflow / Create alias)
	resolve *resolveFlowState

	// Confirm run
	confirmRunItems  []mdl.RunItem
	confirmRunLabel  string
	confirmRunBtn    int

	// Confirm vars
	cv *confirmVarsState

	// Running
	running *runningState

	// Name input
	nameInput     textinput.Model
	nameInputMode nameInputMode
	nameInputErr  string

	// Sub-models
	spinner     spinner.Model
	stepsVP     viewport.Model
	stepsFocused bool

	// List edit
	le *listEditState

	editTargetIdx    int
	mainMenuCursor   int
	lastWorkflow     string
	errMsg         string
	deleteConfirm  bool
	deleteSelected []int
	deleteBtn      int // 0=No (default), 1=Yes
}

// New creates the initial model.
func New(dryRun bool) (Model, error) {
	projectsDir, err := config.FindProjectsDir()
	if err != nil {
		return Model{}, fmt.Errorf("no projects/ directory found next to the executable")
	}
	projects := config.ListProjects(projectsDir)
	if len(projects) == 0 {
		return Model{}, fmt.Errorf("no projects found in %s", projectsDir)
	}

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("36"))

	ti := textinput.New()
	ti.Prompt = "Name > "
	ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("36"))
	ti.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("97"))
	ti.CharLimit = 64
	ti.Width = 40

	m := Model{
		dryRun:      dryRun,
		projectsDir: projectsDir,
		projects:    projects,
		spinner:     sp,
		nameInput:   ti,
		stepsVP:     viewport.New(80, 8),
	}

	if len(projects) == 1 {
		if err := m.loadProject(filepath.Join(projectsDir, projects[0])); err != nil {
			return Model{}, err
		}
		m.gotoMainMenu()
	} else {
		m.screen = ScreenProjectSelect
		m.listItems = projects
		m.listCursor = 0
	}
	return m, nil
}

func (m *Model) loadProject(projectDir string) error {
	cfg, err := config.LoadConfig(projectDir)
	if err != nil {
		return err
	}
	m.projectDir = projectDir
	m.configFile = "config.json"
	if _, err := os.Stat(filepath.Join(projectDir, "config.json")); err != nil {
		m.configFile = "config.tsv"
	}
	m.config = cfg
	workflows, err := store.LoadWorkflows(projectDir)
	if err != nil {
		return err
	}
	m.workflows = workflows
	aliases, err := store.LoadAliases(projectDir)
	if err != nil {
		return err
	}
	m.aliases = aliases
	m.lists = slot.LoadLists(filepath.Join(projectDir, "lists"))
	m.lastWorkflow = store.LoadLastWorkflow(projectDir)
	return nil
}

func (m *Model) gotoManageLists() {
	m.screen = ScreenManageLists
	var names []string
	for k := range m.lists {
		names = append(names, k)
	}
	m.listItems = names
	m.listCursor = 0
}

func (m *Model) gotoMainMenu() {
	m.screen = ScreenMainMenu
	m.listItems = []string{
		"Run workflow",
		"Run manually",
		"Create workflow",
		"Edit workflow",
		"Delete workflow",
		"Manage aliases",
		"Manage lists",
		"Switch config",
		"Exit",
	}
	m.listCursor = m.mainMenuCursor
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// updateStepsViewport rebuilds the steps viewport content for the currently hovered workflow.
func (m *Model) updateStepsViewport() {
	w := m.width
	if w == 0 {
		w = 80
	}
	h := max(3, m.height/3)
	m.stepsVP.Width = w - 4
	m.stepsVP.Height = h

	if len(m.workflows) == 0 || m.listCursor >= len(m.workflows) {
		m.stepsVP.SetContent("")
		return
	}
	wf := m.workflows[m.listCursor]
	var lines []string
	for j, cmdName := range wf.Commands {
		cmdStr := ""
		dirStr := ""
		for _, cmd := range m.config.Commands {
			if cmd.Name == cmdName {
				if wf.Vars != nil {
					if vars, ok := wf.Vars[cmdName]; ok {
						cmd = slot.Apply(cmd, vars)
					}
				}
				cmdStr = cmd.Cmd
				dirStr = cmd.Dir
				break
			}
		}
		prefix := fmt.Sprintf("  %d. %-16s", j+1, cmdName)
		indent := strings.Repeat(" ", len(prefix)+2)
		if cmdStr != "" {
			maxLen := w - len(prefix) - 8
			if maxLen < 8 {
				maxLen = 8
			}
			short := cmdStr
			if len(short) > maxLen {
				short = short[:maxLen-3] + "..."
			}
			lines = append(lines, prefix+"  "+gray("$ "+short))
			if dirStr != "" {
				maxDirLen := w - len(indent) - 8
				if maxDirLen < 8 {
					maxDirLen = 8
				}
				shortDir := dirStr
				if len(shortDir) > maxDirLen {
					shortDir = shortDir[:maxDirLen-3] + "..."
				}
				lines = append(lines, indent+gray("dir: "+shortDir))
			}
		} else {
			lines = append(lines, prefix)
		}
	}
	m.stepsVP.SetContent(strings.Join(lines, "\n"))
	m.stepsVP.GotoTop()
}
