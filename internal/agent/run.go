package agent

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Taka-S-dev/baton/internal/config"
	mdl "github.com/Taka-S-dev/baton/internal/model"
	"github.com/Taka-S-dev/baton/internal/runner"
	"github.com/Taka-S-dev/baton/internal/slot"
	"github.com/Taka-S-dev/baton/internal/store"
)

type Step struct {
	Command string            `json:"command"`
	Inputs  map[string]string `json:"inputs,omitempty"`
}
type Missing struct {
	Step    int             `json:"step"`
	Input   Input           `json:"input"`
	Choices []mdl.ListEntry `json:"choices,omitempty"`
}
type Plan struct {
	Revision string        `json:"revision"`
	Token    string        `json:"plan_token,omitempty"`
	Commands []mdl.Command `json:"commands"`
	Missing  []Missing     `json:"missing_inputs"`
}
type Output struct {
	Command   string `json:"command"`
	ExitCode  int    `json:"exit_code"`
	Stdout    string `json:"stdout"`
	Stderr    string `json:"stderr"`
	Truncated bool   `json:"truncated"`
	Error     string `json:"error,omitempty"`
}
type limitedBuffer struct {
	bytes.Buffer
	truncated bool
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	n := len(p)
	room := (1 << 20) - b.Len()
	if len(p) > room {
		b.truncated = true
		p = p[:max(0, room)]
	}
	_, err := b.Buffer.Write(p)
	return n, err
}
func (s Service) run(dir, revision string, r Request) (any, error) {
	p, err := config.LoadProject(dir)
	if err != nil {
		return nil, err
	}
	if len(p.Warnings) > 0 {
		return nil, fmt.Errorf("project has validation warnings: %v", p.Warnings)
	}
	steps := r.Steps
	if r.Workflow != "" {
		if len(steps) > 0 {
			return nil, fmt.Errorf("specify workflow or steps, not both")
		}
		found := false
		for _, w := range p.Workflows {
			if w.Name == r.Workflow {
				found = true
				for _, n := range w.Commands {
					steps = append(steps, Step{Command: n})
				}
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("workflow not found")
		}
	}
	if len(steps) == 0 {
		return nil, fmt.Errorf("steps or workflow is required")
	}
	cwd := r.Cwd
	if cwd == "" {
		cwd, err = os.Getwd()
		if err != nil {
			return nil, err
		}
	}
	cwd, err = filepath.Abs(cwd)
	if err != nil {
		return nil, err
	}
	plan := Plan{Revision: revision, Missing: []Missing{}}
	for i, step := range steps {
		c, ok := p.Config.FindCommand(step.Command)
		if !ok {
			return nil, fmt.Errorf("unknown command %q", step.Command)
		}
		defs := slot.GetSlots(c)
		allowed := map[string]bool{}
		for _, d := range defs {
			allowed[d.Name] = true
			if _, ok := step.Inputs[d.Name]; !ok {
				plan.Missing = append(plan.Missing, Missing{i, inputSpec(d), p.Lists[d.ListName]})
			}
		}
		for k := range step.Inputs {
			if !allowed[k] {
				return nil, fmt.Errorf("step %d: unknown input %q", i, k)
			}
		}
		c = slot.ApplyVarsToCommand(slot.Apply(c, step.Inputs), p.Vars)
		if c.Dir == "" {
			c.Dir = cwd
		} else if !filepath.IsAbs(c.Dir) {
			c.Dir = filepath.Join(cwd, c.Dir)
		}
		plan.Commands = append(plan.Commands, c)
	}
	if len(plan.Missing) == 0 {
		data, _ := json.Marshal(struct {
			Revision, Project string
			Commands          []mdl.Command
		}{revision, dir, plan.Commands})
		hash := sha256.Sum256(data)
		plan.Token = hex.EncodeToString(hash[:])
	}
	if r.Action == "run.plan" {
		return plan, nil
	}
	if r.DryRun {
		return nil, fmt.Errorf("use run.plan instead of dry_run for execution")
	}
	if plan.Token == "" {
		return nil, fmt.Errorf("inputs are missing; call run.plan")
	}
	if r.PlanToken == "" || r.PlanToken != plan.Token {
		return nil, fmt.Errorf("conflict: plan_token is required and must match run.plan")
	}
	timeout := r.TimeoutSeconds
	if timeout == 0 {
		timeout = 120
	}
	if timeout < 1 || timeout > 3600 {
		return nil, fmt.Errorf("timeout_seconds must be 1..3600")
	}
	outputs := []Output{}
	for _, c := range plan.Commands {
		var out, stderr limitedBuffer
		exitCode, runErr := runner.RunCaptured(c, time.Duration(timeout)*time.Second, &out, &stderr)
		result := Output{Command: c.Name, ExitCode: exitCode, Stdout: out.String(), Stderr: stderr.String(), Truncated: out.truncated || stderr.truncated}
		if runErr != nil {
			result.Error = runErr.Error()
		}
		if err := store.RecordRecent(s.ProjectsDir, cwd, dir, c.Name, time.Now()); err != nil {
			result.Error += "; recent history: " + err.Error()
		}
		outputs = append(outputs, result)
		if runErr != nil {
			break
		}
	}
	return map[string]any{"plan": plan, "results": outputs, "completed": len(outputs) == len(plan.Commands) && outputs[len(outputs)-1].ExitCode == 0}, nil
}
func Schema() any {
	return map[string]any{
		"examples":                  examples(),
		"required_operation_fields": requiredOperationFields(),
		"error_recovery":            errorRecovery(),
		"response_paths": map[string]string{
			"read_revision":         "project.get: data.revision",
			"commit_revision":       "reuse preview data.base_revision, not proposed data.state.revision",
			"after_commit_revision": "apply: data.state.revision",
			"input_key":             "run.plan: data.missing_inputs[i].input.name; step is zero-based",
			"input_choices":         "run.plan: data.missing_inputs[i].choices[].value",
			"execution_token":       "run.plan: data.plan_token; keep project/steps/inputs/cwd identical",
		},
		"version":          1,
		"transport":        "one JSON request on stdin, one JSON response on stdout; diagnostics never mix with stdout",
		"actions":          []string{"schema", "project.list", "project.get", "project.check", "project.create", "project.delete", "project.recover", "apply", "run.plan", "run.execute"},
		"operations":       []string{"command.create", "command.update", "command.rename", "command.delete", "command.inputs", "workflow.create", "workflow.update", "workflow.rename", "workflow.delete", "list.create", "list.update", "list.delete", "variable.create", "variable.update", "variable.delete"},
		"request_fields":   map[string]string{"action": "required action", "project": "project name or directory path", "revision": "project.get revision; required for apply/delete", "dry_run": "preview apply/create/delete without modifying the project", "operations": "ordered array of operations; one validated transaction", "steps": "ordered [{command, inputs: {slotKey: value}}]", "workflow": "run a saved workflow with no remaining inputs; otherwise expand its steps", "cwd": "launch directory for relative workdirs; defaults to CLI cwd", "plan_token": "required for run.execute; returned by run.plan", "timeout_seconds": "per-command execution timeout, default 120, maximum 3600"},
		"operation_fields": map[string]string{"op": "required operation", "name": "existing command/workflow or list/variable name", "new_name": "rename destination", "command": "full {name,cmd,workdir,group,shell,slots,template,values}; update replaces, not patches", "workflow": "full {name,commands:[names in execution order]}", "entries": "full list [{value,label}]", "value": "global variable value (string)", "values": "full saved input map; omitted slots prompt again"},
		"rules":            []string{"read state before changing it", "updates replace complete definitions", "delete references first or update dependents in the same batch", "commands.json stays read-only, matching TUI ownership", "run.plan never executes; run.execute requires the current plan token", "TUI saves and API writes take turns on the project lock; a TUI save refuses when the files changed underneath it", "raw project file edits do not honor locks; revision conflicts are detected before committing"},
	}
}
