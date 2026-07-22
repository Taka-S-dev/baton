package config

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Taka-S-dev/baton/internal/model"
	"github.com/Taka-S-dev/baton/internal/slot"
	"github.com/Taka-S-dev/baton/internal/store"
)

// Project is a fully loaded project: commands, workflows, lists, vars,
// and the diagnostics shown at startup and by `baton check`.
type Project struct {
	Config    model.Config
	Workflows []model.Workflow
	Lists     map[string][]model.ListEntry
	Vars      map[string]string
	Files     string // loaded command-file names, joined for display
	Warnings  []string
}

// LoadProject loads every part of a project and runs the consistency
// checks. It is the single source of truth for diagnostics: the TUI
// startup warning and `baton check` both come from here.
func LoadProject(projectDir string) (Project, error) {
	var p Project

	cfg, err := LoadConfig(projectDir)
	if err != nil {
		return p, err
	}
	var files []string
	for _, f := range []string{"commands.json", "commands.tsv", "commands.local.json"} {
		if _, err := os.Stat(filepath.Join(projectDir, f)); err == nil {
			files = append(files, f)
		}
	}
	p.Files = strings.Join(files, " + ")

	workflows, err := store.LoadWorkflows(projectDir)
	if err != nil {
		return p, err
	}
	p.Workflows = workflows
	p.Lists = slot.LoadLists(filepath.Join(projectDir, "lists"))

	vars, warnings := slot.LoadVars(projectDir)
	p.Vars = vars

	// Fixed slot values live in vars.tsv; merge them into the derived
	// commands and re-bake so they apply. vars.tsv wins over any values
	// present in commands.local.json.
	for i := range cfg.Commands {
		c := &cfg.Commands[i]
		if c.Template == "" {
			continue
		}
		fromVars := slot.CommandValues(vars, c.Name)
		if len(fromVars) == 0 {
			continue
		}
		if c.Values == nil {
			c.Values = make(map[string]string)
		}
		for k, v := range fromVars {
			c.Values[k] = v
		}
		if baked, err := slot.MaterializeCommand(*c, cfg); err == nil {
			*c = baked
		}
	}
	p.Config = cfg

	p.Warnings = append(warnings, Diagnose(cfg, workflows, p.Lists, vars)...)
	return p, nil
}

// Diagnose runs the consistency checks over an in-memory project state.
// It is re-run whenever the TUI returns to the main menu, so warnings
// always describe the current state rather than the state at load time.
func Diagnose(cfg model.Config, workflows []model.Workflow, lists map[string][]model.ListEntry, vars map[string]string) []string {
	var warnings []string
	for _, cmd := range cfg.Commands {
		if cmd.Template == "" {
			continue
		}
		if _, ok := cfg.FindCommand(cmd.Template); !ok {
			warnings = append(warnings, "command \""+cmd.Name+"\": its template \""+cmd.Template+"\" no longer exists")
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
			warnings = append(warnings, "command \""+cmd.Name+"\": unknown shell \""+cmd.Shell+"\" (runs with the platform default)")
		}
		for _, v := range slot.UndefinedVars(cmd.Cmd+" "+cmd.Dir, vars) {
			if !undefined[v] {
				undefined[v] = true
				warnings = append(warnings, "command \""+cmd.Name+"\": {$"+v+"} is not defined in vars.tsv")
			}
		}
	}
	listNames := make([]string, 0, len(lists))
	for name := range lists {
		listNames = append(listNames, name)
	}
	sort.Strings(listNames)
	for _, name := range listNames {
		for _, e := range lists[name] {
			for _, v := range slot.UndefinedVars(e.Value, vars) {
				if !undefined[v] {
					undefined[v] = true
					warnings = append(warnings, "list \""+name+"\": {$"+v+"} is not defined in vars.tsv")
				}
			}
		}
	}
	for _, wf := range workflows {
		for _, step := range wf.Commands {
			if _, ok := cfg.FindCommand(step); !ok {
				warnings = append(warnings, "workflow \""+wf.Name+"\": step \""+step+"\" is not a command (deleted or renamed?)")
			}
		}
	}
	// Orphaned vars.tsv rows (usually a command renamed by hand in
	// commands.local.json): without a warning the next save would drop
	// their values silently.
	var orphans []string
	orphanSeen := make(map[string]bool)
	for k := range vars {
		if i := strings.LastIndex(k, "."); i > 0 {
			if name := k[:i]; !seen[name] && !orphanSeen[name] {
				orphanSeen[name] = true
				orphans = append(orphans, name)
			}
		}
	}
	sort.Strings(orphans)
	for _, name := range orphans {
		warnings = append(warnings, "vars.tsv: values for unknown command \""+name+"\" (removed on next save)")
	}
	return warnings
}
