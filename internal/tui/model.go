package tui

import (
	"fmt"
	"path/filepath"
	"runtime"
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
	ScreenEditListPick
	ScreenEditList
	ScreenDeleteList
	ScreenManageVars
	ScreenEditVarPick
	ScreenVarForm
	ScreenDeleteVar
	ScreenVarRebase
	ScreenSwitchConfig
	ScreenManageCommands
	ScreenEditCommandPick
	ScreenEditCommandMode
	ScreenCreateCommandKind
	ScreenCreateCommandName
	ScreenCreateCommandTemplate
	ScreenEditCommandName
	ScreenEditCommandTemplate
	ScreenCommandForm
	ScreenDeleteCommand
	ScreenRenameRepair
	ScreenRunWorkflowSteps
)

type nameInputMode int

const (
	nameInputWorkflow nameInputMode = iota
	nameInputEditWorkflow
	nameInputRenameCommand
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

	// {name...} slots: Tab toggles values into picked (in toggle order)
	// and Enter joins them with spaces. With nothing picked, Enter falls
	// back to the single-value behavior.
	variadic bool
	picked   []string

	contextNames  []string
	contextNotes  []string
	contextIdx    int
	currentCmd    *mdl.Command
	resolvedSoFar map[string]string
}

// placeholder returns the literal placeholder text being resolved.
func (s *slotPickState) placeholder() string {
	if s.variadic {
		return "{" + s.slotName + "...}"
	}
	return "{" + s.slotName + "}"
}

func (s *slotPickState) togglePicked(v string) {
	for i, p := range s.picked {
		if p == v {
			s.picked = append(s.picked[:i], s.picked[i+1:]...)
			return
		}
	}
	s.picked = append(s.picked, v)
}

func (s *slotPickState) isPicked(v string) bool {
	for _, p := range s.picked {
		if p == v {
			return true
		}
	}
	return false
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

// varEditState holds state for the variable create/edit form.
type varEditState struct {
	mode     int    // 0=create, 1=edit
	name     string // being typed (create) or fixed (edit)
	fieldIdx int    // create: 0=name, 1=value
	oldValue string // edit: value before the change, drives the rebase offer
}

// varRebaseItem is one literal value that starts with a variable's old
// value and can be rewritten to a {$name} reference.
type varRebaseItem struct {
	kind     int    // 0 = scoped vars.tsv value, 1 = list entry
	key      string // kind 0: vars map key ("command.slot")
	listName string // kind 1
	entryIdx int    // kind 1
	label    string // display: where the value lives
	oldValue string
	newValue string
	on       bool
}

// varRebaseState holds the post-edit offer to rewrite matching literals
// into references (rebase), or — after a scoped value edit — to apply
// the same change to other values that shared the old value (propagate).
// Both are prefix-anchored, never substring, and always opt-in.
type varRebaseState struct {
	varName    string // global name (rebase) or the edited key (propagate)
	created    bool   // offer follows a Create (pure extraction) vs an edit
	propagate  bool   // literal-to-literal propagation after a scoped edit
	items      []varRebaseItem
	cursor     int
	confirm    bool // No/Yes window is up (same shape as the delete flows)
	confirmBtn int  // 0=No, 1=Yes

	// The offer opened from the list entry editor: return there instead
	// of the vars submenu when it closes.
	returnToList bool

	// The already-committed edit that triggered a propagate offer,
	// echoed in the window so it's clear Esc keeps it.
	editedOld string
	editedNew string
}

// checkedCount returns how many items are toggled on.
func (s *varRebaseState) checkedCount() int {
	n := 0
	for _, it := range s.items {
		if it.on {
			n++
		}
	}
	return n
}

// wfStepPickState holds a per-run step selection for one workflow.
// Only which steps run is chosen here: they always execute in the
// workflow's own order, since encoding that order is what makes a
// workflow more than a set of commands.
type wfStepPickState struct {
	wfIdx  int
	cursor int
	picked []bool // parallel to the workflow's steps
}

func (s *wfStepPickState) count() int {
	n := 0
	for _, p := range s.picked {
		if p {
			n++
		}
	}
	return n
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
	fromPick  bool // entered via Edit list (Esc returns to the pick screen)
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
	mode           int  // 0=create, 1=edit
	editIdx        int  // for edit mode
	pickedTemplate bool // edit entered via Change template (Esc returns there)
	name           string
	templateRefIdx int
	currentSlots   []slot.Def
	currentSlotIdx int
	currentValues  map[string]string
}

// editRef points at an editable command: a TSV row in the Base layer
// or a template-derived command in the local layer.
type editRef struct {
	tsv bool
	idx int
}

// commandFormState holds state for writing a concrete command directly.
type commandFormState struct {
	mode     int  // 0=create, 1=edit
	editIdx  int  // index into config.Commands, or config.Base when tsvEdit
	tsvEdit  bool // editing a TSV row (saved via UpdateCommandTSV)
	origName string
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

	// Edit/Delete command: what each list row points at
	editRefs []editRef

	// Type-to-filter for the management pick screens (Edit/Delete of
	// commands, workflows and lists): pickBase/pickTexts hold the full
	// item set, listItems the visible subset, and pickMap maps a visible
	// row to its original index (nil = identity).
	pickSearch string
	pickBase   []string
	pickTexts  []string
	pickMap    []int

	// Manage vars picks: pickBase holds display labels ("name = value"),
	// varPickNames the raw names, parallel to pickBase.
	varPickNames []string

	// Multi-select
	msItems    []msItem
	msCursor   int
	msSelected []int
	msSearchTI textinput.Model
	msEscArmed bool // first Esc with selections pressed; next Esc discards

	// Slot picking
	sp *slotPickState

	// Manage vars
	ve *varEditState
	vr *varRebaseState

	// Resolve flow (Run commands / Run workflow)
	resolve *resolveFlowState

	// Run workflow: per-run step selection (→ on the workflow list)
	wfp *wfStepPickState

	// Create workflow: command names picked, waiting for the name input
	pendingWorkflowCmds []string

	// Confirm run
	confirmRunItems  []mdl.RunItem
	confirmRunLabel  string
	confirmRunBtn    int
	confirmRunScroll int

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

	// Rename repair: renames detected at load by matching the command
	// snapshot, offered on the repair screen before the main menu.
	// snapOld is the snapshot as loaded, minus declined proposals — its
	// stale entries are carried forward for names still referenced, so
	// an unresolved rename stays detectable on a later start.
	renames   []config.Rename
	renameBtn int // 0=No, 1=Yes
	snapOld   map[string]string

	editTargetIdx  int
	mainMenuCursor int
	lastWorkflow   string
	errMsg         string
	successMsg     string   // transient "it worked" note, cleared on the next keypress
	loadWarnings   []string // diagnostics; recomputed on every return to the main menu
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
		m.gotoPostLoad()
	} else {
		m.screen = ScreenProjectSelect
		m.listItems = projects
		m.listCursor = 0
	}
	return m, nil
}

func (m *Model) loadProject(projectDir string) error {
	p, err := config.LoadProject(projectDir)
	if err != nil {
		return err
	}
	m.projectDir = projectDir
	m.configFile = p.Files
	m.config = p.Config
	m.workflows = p.Workflows
	m.lists = p.Lists
	m.vars = p.Vars
	m.lastWorkflow = store.LoadLastWorkflow(projectDir)
	m.loadWarnings = p.Warnings
	m.snapOld = config.LoadSnapshot(projectDir)
	m.renames = config.DetectRenames(p.Config, p.Workflows, p.Vars, m.snapOld)
	m.renameBtn = 1
	return nil
}

// gotoPostLoad enters the rename-repair offer when loading detected
// hand-edited renames, otherwise the main menu.
func (m *Model) gotoPostLoad() {
	if len(m.renames) > 0 {
		m.gotoRenameRepair()
		return
	}
	m.gotoMainMenu()
}

// writeSnapshot records every command's name and fingerprint for the
// next start's rename detection. Stale entries whose names are still
// referenced somewhere are carried forward. Best-effort, like
// SaveLastWorkflow: a failed write only costs future detection.
func (m *Model) writeSnapshot() {
	entries := make(map[string]string)
	for _, c := range m.config.AllCommands() {
		entries[c.Name] = config.Fingerprint(c)
	}
	for _, name := range config.DanglingCommandRefs(m.config, m.workflows, m.vars) {
		if fp, ok := m.snapOld[name]; ok {
			entries[name] = fp
		}
	}
	_ = config.SaveSnapshot(m.projectDir, entries)
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

// gotoManageLists opens the Manage lists submenu.
func (m *Model) gotoManageLists() {
	m.screen = ScreenManageLists
	m.listItems = []string{"Create list", "Edit list", "Delete list"}
	m.listCursor = 0
}

func (m *Model) gotoMainMenu() {
	m.screen = ScreenMainMenu
	m.listItems = mainMenuItems()
	// Re-diagnose so the warnings describe the current state — TUI
	// edits since load (deleted commands, workflows, …) are reflected.
	if m.projectDir != "" {
		m.loadWarnings = config.Diagnose(m.config, m.workflows, m.lists, m.vars)
		m.writeSnapshot()
	}
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

	var wf mdl.Workflow
	switch m.screen {
	case ScreenEditWorkflow, ScreenDeleteWorkflow:
		// These pick screens filter via the pick filter, not the Run
		// workflow search field.
		if len(m.listItems) == 0 || m.listCursor >= len(m.listItems) {
			m.stepsVP.SetContent("")
			return
		}
		wf = m.workflows[m.pickOrig(m.listCursor)]
	default:
		filtered := m.wfFiltered()
		if len(filtered) == 0 || m.listCursor >= len(filtered) {
			m.stepsVP.SetContent("")
			return
		}
		wf = m.workflows[filtered[m.listCursor]]
	}
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
			short := truncate(cmdStr, max(8, w-len(prefix)-8))
			lines = append(lines, prefix+"  "+gray("$ "+short))
			if dirStr != "" {
				shortDir := truncate(dirStr, max(8, w-len(indent)-8))
				lines = append(lines, indent+gray("workdir: "+shortDir))
			}
		} else {
			lines = append(lines, prefix)
		}
	}
	m.stepsVP.SetContent(strings.Join(lines, "\n"))
	m.stepsVP.GotoTop()
}
