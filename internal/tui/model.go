package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
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
	ScreenRunCommands
	ScreenSlotPick
	ScreenConfirmRun
	ScreenRunning
	ScreenRetry
	ScreenCreateWorkflow
	ScreenNameInput
	ScreenWorkflowMgmt
	ScreenEditWorkflow
	ScreenEditWorkflowMode
	ScreenEditWorkflowCommands
	ScreenDeleteWorkflow
	ScreenManageLists
	ScreenEditList
	ScreenSwitchConfig
	ScreenManageCommands
	ScreenEditCommandPick
	ScreenCreateCommandKind
	ScreenCreateCommandName
	ScreenCreateCommandTemplate
	ScreenEditCommandName
	ScreenEditCommandTemplate
	ScreenCommandForm
	ScreenDeleteCommand
)

type nameInputMode int

const (
	nameInputWorkflow nameInputMode = iota
	nameInputEditWorkflow
	nameInputNewList
)

type resolveFlowPurpose int

const (
	purposeRunCommands resolveFlowPurpose = iota
	purposeRunWorkflow
)

// msItem is a command shown in the multi-select screen.
type msItem struct {
	cmd *mdl.Command
}

func (i msItem) name() string { return i.cmd.Name }

// searchText returns the lowercase haystack the search field matches
// against: name, group, command body, template name and embedded values.
func (i msItem) searchText() string {
	parts := []string{i.cmd.Name, i.cmd.Group, i.cmd.Cmd, i.cmd.Template}
	for _, v := range i.cmd.Values {
		parts = append(parts, v)
	}
	return strings.ToLower(strings.Join(parts, " "))
}

// matchesAllTerms reports whether every term occurs in haystack (AND search).
func matchesAllTerms(haystack string, terms []string) bool {
	for _, t := range terms {
		if !strings.Contains(haystack, t) {
			return false
		}
	}
	return true
}

// slotPickState holds state for the slot-picking screen.
type slotPickState struct {
	slotName string
	listName string
	entries  []mdl.ListEntry
	filtered []mdl.ListEntry
	cursor   int
	search   string
	canSkip  bool // true when creating a template-derived command

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
	terms := strings.Fields(strings.ToLower(s.search))
	for _, e := range s.entries {
		hay := strings.ToLower(e.Value + " " + e.Label)
		if matchesAllTerms(hay, terms) {
			s.filtered = append(s.filtered, e)
		}
	}
}

// resolveFlowState tracks multi-command slot resolution.
type resolveFlowState struct {
	purpose       resolveFlowPurpose
	rawItems      []msItem
	itemNames     []string
	itemNotes     []string
	workflowLabel string // label for purposeRunWorkflow

	currentIdx     int
	currentSlots   []slot.Def
	currentSlotIdx int
	currentValues  map[string]string

	resolved []mdl.RunItem
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
	name      string
	entries   []mdl.ListEntry
	cursor    int
	adding    bool
	addVal    string
	addLbl    string
	addFld    int // 0=value, 1=label
	editing   bool
	editFld   int // 0=value, 1=label
	editValTI textinput.Model
	editLblTI textinput.Model
}

// commandEditState holds state for creating/editing a template-derived command.
type commandEditState struct {
	mode           int // 0=create, 1=edit
	editIdx        int // for edit mode
	name           string
	templateRefIdx int
	currentSlots   []slot.Def
	currentSlotIdx int
	currentValues  map[string]string
}

// commandFormState holds state for writing a concrete command directly.
type commandFormState struct {
	mode     int // 0=create, 1=edit
	editIdx  int // for edit mode
	fieldIdx int
	fields   [5]string // name, cmd, workdir, group, shell

	// Placeholder picker window: left pane picks a list name to insert
	// {name}; the right pane picks a concrete value to insert directly.
	slotPickFocus       bool
	slotPickCursor      int
	slotPickPane        int // 0=list names, 1=values
	slotPickValueCursor int
}

var commandFormLabels = [5]string{"Name", "Cmd", "Workdir (optional)", "Group (optional)", shellFormLabel()}

// shellFormLabel names the shell the command runs with when the field is
// left empty, so the form shows the actual platform default.
func shellFormLabel() string {
	if runtime.GOOS == "windows" {
		return "Shell (ps = PowerShell, leave empty for cmd)"
	}
	return "Shell (ps = PowerShell, leave empty for sh)"
}

// Model is the main bubbletea model.
type Model struct {
	dryRun           bool
	projectsDir      string
	projects         []string
	projectCmdCounts map[string]int

	projectDir string
	configFile string
	config     mdl.Config
	workflows  []mdl.Workflow
	lists      map[string][]mdl.ListEntry
	vars       map[string]string // project variables for {$name} references

	screen Screen
	width  int
	height int

	// Generic single-select / menu
	listCursor int
	listItems  []string

	// Multi-select
	msItems     []msItem
	msCursor    int
	msViewStart int
	msSelected  []int
	msSearchTI  textinput.Model
	msEscArmed  bool // first Esc with selections pressed; next Esc discards

	// Slot picking
	sp *slotPickState

	// Resolve flow (Run commands / Run workflow)
	resolve *resolveFlowState

	// Create workflow: command names picked, waiting for the name input
	pendingWorkflowCmds []string

	// Confirm run
	confirmRunItems []mdl.RunItem
	confirmRunLabel string
	confirmRunBtn   int

	// Running
	running *runningState

	// Name input
	nameInput     textinput.Model
	nameInputMode nameInputMode
	nameInputErr  string

	// Sub-models
	spinner      spinner.Model
	stepsVP      viewport.Model
	stepsFocused bool
	wfSearchTI   textinput.Model // Run workflow list search

	// List edit
	le *listEditState

	// Command create/edit (template-derived)
	sce *commandEditState

	// Command form (direct input)
	cf *commandFormState

	editTargetIdx  int
	mainMenuCursor int
	lastWorkflow   string
	errMsg         string
	successMsg     string // transient "it worked" note, cleared on the next keypress
	loadWarning    string
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

	cmdCounts := make(map[string]int, len(projects))
	for _, p := range projects {
		if cfg, err := config.LoadConfig(filepath.Join(projectsDir, p)); err == nil {
			cmdCounts[p] = len(cfg.AllCommands())
		}
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
		dryRun:           dryRun,
		projectsDir:      projectsDir,
		projects:         projects,
		projectCmdCounts: cmdCounts,
		spinner:          sp,
		nameInput:        ti,
		stepsVP:          viewport.New(80, 8),
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
	var files []string
	for _, f := range []string{"commands.json", "templates.json", "template.json", "commands.tsv", "templates.tsv", "config.tsv", "commands.local.json", "config.json"} {
		if _, err := os.Stat(filepath.Join(projectDir, f)); err == nil {
			files = append(files, f)
		}
	}
	m.configFile = strings.Join(files, " + ")
	m.config = cfg
	workflows, err := store.LoadWorkflows(projectDir)
	if err != nil {
		return err
	}
	m.workflows = workflows
	m.lists = slot.LoadLists(filepath.Join(projectDir, "lists"))
	m.lastWorkflow = store.LoadLastWorkflow(projectDir)

	vars, warnings := slot.LoadVars(projectDir)
	m.vars = vars

	// Fixed slot values live in vars.tsv ("command.slot" names) and win
	// over values still stored inside commands.local.json (legacy layout,
	// migrated on the next save). Re-bake so the merged values apply.
	for i := range m.config.Commands {
		c := &m.config.Commands[i]
		if c.Template == "" {
			continue
		}
		fromVars := slot.CommandValues(m.vars, c.Name)
		if len(fromVars) == 0 {
			continue
		}
		if c.Values == nil {
			c.Values = make(map[string]string)
		}
		for k, v := range fromVars {
			c.Values[k] = v
		}
		if baked, err := slot.MaterializeCommand(*c, m.config); err == nil {
			*c = baked
		}
	}
	for _, cmd := range cfg.Commands {
		if cmd.Template == "" {
			continue
		}
		if _, ok := cfg.FindCommand(cmd.Template); !ok {
			warnings = append(warnings, "missing template: "+cmd.Name+" → "+cmd.Template)
		}
	}
	seen := make(map[string]bool)
	undefined := make(map[string]bool)
	for _, cmd := range cfg.AllCommands() {
		if seen[cmd.Name] {
			warnings = append(warnings, "duplicate command name: "+cmd.Name)
		}
		seen[cmd.Name] = true
		if cmd.Shell != "" && cmd.Shell != "ps" {
			warnings = append(warnings, "unknown shell \""+cmd.Shell+"\" on "+cmd.Name+" (runs with the platform default)")
		}
		for _, v := range slot.UndefinedVars(cmd.Cmd+" "+cmd.Dir, m.vars) {
			if !undefined[v] {
				undefined[v] = true
				warnings = append(warnings, "undefined var {$"+v+"} on "+cmd.Name+" (define it in vars.tsv)")
			}
		}
	}
	for _, entries := range m.lists {
		for _, e := range entries {
			for _, v := range slot.UndefinedVars(e.Value, m.vars) {
				if !undefined[v] {
					undefined[v] = true
					warnings = append(warnings, "undefined var {$"+v+"} in a list value (define it in vars.tsv)")
				}
			}
		}
	}
	// Orphaned vars.tsv rows (usually a command renamed by hand in
	// commands.local.json): without a warning the next save would drop
	// their values silently.
	var orphans []string
	orphanSeen := make(map[string]bool)
	for k := range m.vars {
		if i := strings.LastIndex(k, "."); i > 0 {
			if name := k[:i]; !seen[name] && !orphanSeen[name] {
				orphanSeen[name] = true
				orphans = append(orphans, name)
			}
		}
	}
	sort.Strings(orphans)
	for _, name := range orphans {
		warnings = append(warnings, "vars.tsv has values for unknown command \""+name+"\" (removed on next save)")
	}
	m.loadWarning = strings.Join(warnings, "; ")
	return nil
}

// saveConfig persists user commands to commands.local.json.
func (m *Model) saveConfig() error {
	// Mirror every template-derived command's fixed values into vars.tsv
	// ("command.slot" names), drop entries for commands that no longer
	// exist, and persist the local layer without the values — vars.tsv is
	// their single editable home.
	for _, c := range m.config.Commands {
		if c.Template != "" {
			slot.SetCommandValues(m.vars, c.Name, c.Values)
		}
	}
	slot.PruneCommandValues(m.vars, func(name string) bool {
		_, ok := m.config.FindCommand(name)
		return ok
	})
	if err := slot.SaveVars(m.projectDir, m.vars); err != nil {
		return err
	}
	// Template-derived entries persist only their identity: cmd/workdir/
	// slots/values are all recomputed from template + vars.tsv on load,
	// and a baked copy in the file just looks editable without being so.
	stripped := m.config
	stripped.Commands = make([]mdl.Command, len(m.config.Commands))
	copy(stripped.Commands, m.config.Commands)
	for i := range stripped.Commands {
		if stripped.Commands[i].Template != "" {
			stripped.Commands[i].Values = nil
			stripped.Commands[i].Cmd = ""
			stripped.Commands[i].Dir = ""
			stripped.Commands[i].Slots = nil
		}
	}
	return store.SaveConfig(m.projectDir, stripped)
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
	m.listItems = mainMenuItems()
	m.listCursor = m.mainMenuCursor
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// workflowStepCommand resolves a workflow step name to a displayable
// command (template-derived commands are baked at load).
func (m *Model) workflowStepCommand(name string) (mdl.Command, bool) {
	cmd, ok := m.config.FindCommand(name)
	if !ok {
		return mdl.Command{}, false
	}
	return slot.ApplyVarsToCommand(cmd, m.vars), true
}

// updateStepsViewport rebuilds the steps viewport content for the currently hovered workflow.
func (m *Model) updateStepsViewport() {
	w := m.width
	if w == 0 {
		w = 80
	}
	h := max(3, m.height/3)
	m.stepsVP.Width = w - 4
	m.stepsVP.Height = h

	filtered := m.wfFiltered()
	if len(filtered) == 0 || m.listCursor >= len(filtered) {
		m.stepsVP.SetContent("")
		return
	}
	wf := m.workflows[filtered[m.listCursor]]
	var lines []string
	for j, cmdName := range wf.Commands {
		cmdStr := ""
		dirStr := ""
		if cmd, ok := m.workflowStepCommand(cmdName); ok {
			cmdStr = cmd.Cmd
			dirStr = cmd.Dir
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
				lines = append(lines, indent+gray("workdir: "+shortDir))
			}
		} else {
			lines = append(lines, prefix)
		}
	}
	m.stepsVP.SetContent(strings.Join(lines, "\n"))
	m.stepsVP.GotoTop()
}
