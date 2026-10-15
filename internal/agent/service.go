package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Taka-S-dev/baton/internal/config"
	mdl "github.com/Taka-S-dev/baton/internal/model"
	"github.com/Taka-S-dev/baton/internal/projectfs"
	"github.com/Taka-S-dev/baton/internal/projectops"
	"github.com/Taka-S-dev/baton/internal/slot"
)

type Request struct {
	Action         string                 `json:"action"`
	Project        string                 `json:"project,omitempty"`
	Revision       string                 `json:"revision,omitempty"`
	DryRun         bool                   `json:"dry_run,omitempty"`
	Operations     []projectops.Operation `json:"operations,omitempty"`
	Steps          []Step                 `json:"steps,omitempty"`
	Workflow       string                 `json:"workflow,omitempty"`
	Cwd            string                 `json:"cwd,omitempty"`
	PlanToken      string                 `json:"plan_token,omitempty"`
	TimeoutSeconds int                    `json:"timeout_seconds,omitempty"`
}
type Response struct {
	Version int    `json:"version"`
	OK      bool   `json:"ok"`
	Code    string `json:"code,omitempty"`
	Error   string `json:"error,omitempty"`
	Data    any    `json:"data,omitempty"`
}
type Service struct{ ProjectsDir string }
type State struct {
	Project        string                     `json:"project"`
	Revision       string                     `json:"revision"`
	Commands       []mdl.Command              `json:"commands"`
	CommandSources map[string]string          `json:"command_sources"`
	Workflows      []mdl.Workflow             `json:"workflows"`
	Lists          map[string][]mdl.ListEntry `json:"lists"`
	Variables      map[string]string          `json:"variables"`
	Warnings       []string                   `json:"warnings"`
	Inputs         map[string][]Input         `json:"inputs"`
	References     []Reference                `json:"references"`
}
type Reference struct {
	Kind     string `json:"kind"`
	Target   string `json:"target"`
	Source   string `json:"source"`
	Location string `json:"location"`
}
type Change struct {
	Path   string  `json:"path"`
	Before *string `json:"before"`
	After  *string `json:"after"`
}
type Result struct {
	BaseRevision string   `json:"base_revision"`
	State        State    `json:"state"`
	Changes      []Change `json:"changes"`
	Committed    bool     `json:"committed"`
}

func (s Service) resolve(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("project is required")
	}
	path := name
	if !filepath.IsAbs(path) && !strings.ContainsAny(path, "/\\") {
		path = filepath.Join(s.ProjectsDir, path)
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	st, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("project must be a real directory")
	}
	return path, nil
}
func state(dir string) (State, error) {
	files, err := projectfs.Read(dir)
	if err != nil {
		return State{}, err
	}
	p, err := config.LoadProject(dir)
	if err != nil {
		return State{}, err
	}
	after, err := projectfs.Read(dir)
	if err != nil {
		return State{}, err
	}
	if projectfs.Revision(files) != projectfs.Revision(after) {
		return State{}, fmt.Errorf("conflict: project changed while reading")
	}
	globals := map[string]string{}
	for k, v := range p.Vars {
		if !strings.Contains(k, ".") {
			globals[k] = v
		}
	}
	inputs := map[string][]Input{}
	sources := map[string]string{}
	for _, c := range p.Config.AllCommands() {
		inputs[c.Name] = inputSpecs(slot.GetSlots(c))
		sources[c.Name] = c.Source
	}
	refs := []Reference{}
	addVars := func(text, source, location string) {
		for _, token := range slot.VarPattern.FindAllString(text, -1) {
			refs = append(refs, Reference{"variable", token[2 : len(token)-1], source, location})
		}
	}
	for _, w := range p.Workflows {
		for i, name := range w.Commands {
			refs = append(refs, Reference{"command", name, w.Name, fmt.Sprintf("workflow.steps[%d]", i)})
		}
	}
	for _, c := range p.Config.AllCommands() {
		if c.Template != "" {
			refs = append(refs, Reference{"command", c.Template, c.Name, "template"})
		}
		for _, d := range slot.GetSlots(c) {
			if d.PathKind == "" {
				refs = append(refs, Reference{"list", d.ListName, c.Name, d.Placeholder()})
			}
		}
		addVars(c.Cmd, c.Name, "cmd")
		addVars(c.Dir, c.Name, "workdir")
	}
	names := make([]string, 0, len(p.Lists))
	for name := range p.Lists {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		for i, e := range p.Lists[name] {
			addVars(e.Value, name, fmt.Sprintf("list.entries[%d]", i))
		}
	}
	return State{Project: dir, Revision: projectfs.Revision(files), Commands: p.Config.AllCommands(), CommandSources: sources, Workflows: p.Workflows, Lists: p.Lists, Variables: globals, Warnings: p.Warnings, Inputs: inputs, References: refs}, nil
}
func changes(a, b projectfs.Files) []Change {
	out := []Change{}
	for _, name := range projectfs.Changed(a, b) {
		c := Change{Path: name}
		if data, ok := a[name]; ok {
			v := string(data)
			c.Before = &v
		}
		if data, ok := b[name]; ok {
			v := string(data)
			c.After = &v
		}
		out = append(out, c)
	}
	return out
}
func (s Service) Handle(r Request) Response {
	root, rootErr := filepath.Abs(s.ProjectsDir)
	if rootErr != nil {
		return Response{Version: 1, Code: "invalid_request", Error: rootErr.Error()}
	}
	s.ProjectsDir = root
	data, err := s.handle(r)
	if err != nil {
		code := "invalid_request"
		text := err.Error()
		switch {
		case strings.Contains(text, "conflict"):
			code = "conflict"
		case strings.Contains(text, "busy"):
			code = "busy"
		case strings.Contains(text, "recovery") || strings.Contains(text, "pending transaction"):
			code = "recovery_required"
		case strings.Contains(text, "referenced by"):
			code = "referenced"
		case strings.Contains(text, "validation failed"):
			code = "validation_failed"
		case strings.Contains(text, "not found"):
			code = "not_found"
		}
		return Response{Version: 1, Code: code, Error: text}
	}
	if r.Action == "run.execute" {
		if result, ok := data.(map[string]any); ok && result["completed"] == false {
			return Response{Version: 1, Code: "execution_failed", Error: "execution stopped at a failed command", Data: data}
		}
	}
	if r.Action == "project.check" {
		if st, ok := data.(State); ok && len(st.Warnings) > 0 {
			return Response{Version: 1, Code: "validation_failed", Error: "project has validation warnings", Data: data}
		}
	}
	return Response{Version: 1, OK: true, Data: data}
}
func (s Service) handle(r Request) (any, error) {
	if r.Action == "schema" {
		return Schema(), nil
	}
	if r.Action == "project.list" {
		return config.ListProjects(s.ProjectsDir), nil
	}
	if r.Action == "project.create" {
		if err := projectops.ValidName(r.Project); err != nil {
			return nil, err
		}
		if strings.HasPrefix(r.Project, ".") {
			return nil, fmt.Errorf("project names cannot start with a dot")
		}
		if err := os.MkdirAll(s.ProjectsDir, 0700); err != nil {
			return nil, err
		}
		release, err := projectfs.Lock(s.ProjectsDir)
		if err != nil {
			return nil, err
		}
		defer release()
		dir := filepath.Join(s.ProjectsDir, r.Project)
		if _, err := os.Lstat(dir); !os.IsNotExist(err) {
			return nil, fmt.Errorf("project already exists or cannot be inspected")
		}
		if r.DryRun {
			return map[string]any{"project": dir, "committed": false}, nil
		}
		staging, err := os.MkdirTemp(s.ProjectsDir, ".baton-create-")
		if err != nil {
			return nil, err
		}
		defer os.RemoveAll(staging)
		if err = projectfs.Write(staging, "commands.tsv", []byte("name\tgroup\tworkdir\tcmd\tshell\tslots\n")); err != nil {
			return nil, err
		}
		if err = os.Rename(staging, dir); err != nil {
			return nil, err
		}
		return state(dir)
	}
	dir, err := s.resolve(r.Project)
	if err != nil {
		return nil, err
	}
	if r.Action == "project.get" || r.Action == "project.check" {
		if projectfs.Pending(dir) {
			return nil, fmt.Errorf("pending transaction: project.recover is required")
		}
		return state(dir)
	}
	if r.Action == "run.plan" {
		before, err := projectfs.Read(dir)
		if err != nil {
			return nil, err
		}
		plan, err := s.run(dir, projectfs.Revision(before), r)
		if err != nil {
			return nil, err
		}
		after, err := projectfs.Read(dir)
		if err != nil {
			return nil, err
		}
		if projectfs.Revision(before) != projectfs.Revision(after) {
			return nil, fmt.Errorf("conflict: project changed while planning")
		}
		return plan, nil
	}
	// The root lock serializes project creation and deletion with every
	// write; the project lock serializes the writers of one project, the
	// TUI's saves included.
	releaseRoot, err := projectfs.Lock(s.ProjectsDir)
	if err != nil {
		return nil, err
	}
	defer releaseRoot()
	release, err := projectfs.Lock(dir)
	if err != nil {
		return nil, err
	}
	defer release()
	if r.Action == "project.recover" {
		if err := projectfs.Recover(dir); err != nil {
			return nil, err
		}
		return state(dir)
	}
	if projectfs.Pending(dir) {
		return nil, fmt.Errorf("pending transaction: project.recover is required")
	}
	before, err := projectfs.Read(dir)
	if err != nil {
		return nil, err
	}
	revision := projectfs.Revision(before)
	if r.Action == "run.plan" || r.Action == "run.execute" {
		return s.run(dir, revision, r)
	}
	if r.Revision == "" || r.Revision != revision {
		return nil, fmt.Errorf("conflict: revision is required and must match project.get")
	}
	if r.Action == "project.delete" {
		rel, err := filepath.Rel(s.ProjectsDir, dir)
		if err != nil || rel == "." || strings.ContainsAny(rel, "/\\") || strings.HasPrefix(rel, ".") {
			return nil, fmt.Errorf("only a direct child of projects directory can be archived")
		}
		trash := filepath.Join(s.ProjectsDir, ".baton-trash", rel+"-"+time.Now().UTC().Format("20060102T150405.000000000"))
		if r.DryRun {
			return map[string]any{"archive": trash, "committed": false}, nil
		}
		if err = os.MkdirAll(filepath.Dir(trash), 0700); err != nil {
			return nil, err
		}
		release() // the root lock still excludes the other writers
		if err = os.Rename(dir, trash); err != nil {
			return nil, err
		}
		return map[string]any{"archive": trash, "committed": true}, nil
	}
	if r.Action != "apply" {
		return nil, fmt.Errorf("unknown action %q", r.Action)
	}
	if len(r.Operations) == 0 {
		return nil, fmt.Errorf("operations must not be empty")
	}
	stage, err := os.MkdirTemp("", "baton-agent-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(stage)
	for name, data := range before {
		if err = projectfs.Write(stage, name, data); err != nil {
			return nil, err
		}
	}
	for i, o := range r.Operations {
		if err = projectops.Apply(stage, o); err != nil {
			return nil, fmt.Errorf("operation %d (%s): %w", i, o.Op, err)
		}
	}
	p, err := config.LoadProject(stage)
	if err != nil {
		return nil, err
	}
	if len(p.Warnings) > 0 {
		return nil, fmt.Errorf("validation failed: %s", strings.Join(p.Warnings, "; "))
	}
	snapshots := map[string]string{}
	for _, c := range p.Config.AllCommands() {
		snapshots[c.Name] = config.Fingerprint(c)
	}
	if err = config.SaveSnapshot(stage, snapshots); err != nil {
		return nil, err
	}
	after, err := projectfs.Read(stage)
	if err != nil {
		return nil, err
	}
	st, err := state(stage)
	if err != nil {
		return nil, err
	}
	st.Project = dir
	result := Result{BaseRevision: revision, State: st, Changes: changes(before, after)}
	if r.DryRun {
		return result, nil
	}
	if err = projectfs.Commit(dir, before, after); err != nil {
		return nil, err
	}
	result.Committed = true
	return result, nil
}
