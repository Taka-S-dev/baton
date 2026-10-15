package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Taka-S-dev/baton/internal/config"
	mdl "github.com/Taka-S-dev/baton/internal/model"
	"github.com/Taka-S-dev/baton/internal/projectfs"
	"github.com/Taka-S-dev/baton/internal/projectops"
)

func fixture(t *testing.T) (Service, string) {
	t.Helper()
	s := Service{ProjectsDir: t.TempDir()}
	resp := s.Handle(Request{Action: "project.create", Project: "demo"})
	if !resp.OK {
		t.Fatal(resp.Error)
	}
	return s, filepath.Join(s.ProjectsDir, "demo")
}
func str(s string) *string { return &s }
func apply(t *testing.T, s Service, dir string, ops ...projectops.Operation) Result {
	t.Helper()
	st, err := state(dir)
	if err != nil {
		t.Fatal(err)
	}
	resp := s.Handle(Request{Action: "apply", Project: dir, Revision: st.Revision, Operations: ops})
	if !resp.OK {
		t.Fatal(resp.Error)
	}
	return resp.Data.(Result)
}
func TestAgentBatchReferencesAndSavedInputs(t *testing.T) {
	s, dir := fixture(t)
	original := "name\tgroup\tworkdir\tcmd\tshell\tslots\r\nuntouched\t\t\techo unchanged\t\t\r\n"
	if err := os.WriteFile(filepath.Join(dir, "commands.tsv"), []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	result := apply(t, s, dir,
		projectops.Operation{Op: "variable.create", Name: "root", Value: str("C:/demo")},
		projectops.Operation{Op: "list.create", Name: "target", Entries: []mdl.ListEntry{{Value: "{$root}/api", Label: "API"}}},
		projectops.Operation{Op: "command.create", Command: &mdl.Command{Name: "build", Cmd: "echo {target}"}},
		projectops.Operation{Op: "command.create", Command: &mdl.Command{Name: "build-api", Template: "build", Values: map[string]string{"target": "{$root}/api"}}},
		projectops.Operation{Op: "workflow.create", Workflow: &mdl.Workflow{Name: "release", Commands: []string{"build-api", "build"}}},
	)
	if !result.Committed {
		t.Fatal("not committed")
	}
	result = apply(t, s, dir, projectops.Operation{Op: "command.rename", Name: "build", NewName: "compile"}, projectops.Operation{Op: "command.rename", Name: "build-api", NewName: "compile-api"})
	p, err := config.LoadProject(dir)
	if err != nil || len(p.Warnings) > 0 {
		t.Fatalf("%v %v", err, p.Warnings)
	}
	c, ok := p.Config.FindCommand("compile-api")
	if !ok || c.Template != "compile" || c.Cmd != "echo {$root}/api" || p.Vars["compile-api.target"] != "{$root}/api" {
		t.Fatalf("lost references: %+v %+v", c, p.Vars)
	}
	if p.Workflows[0].Commands[0] != "compile-api" || p.Workflows[0].Commands[1] != "compile" {
		t.Fatal("workflow not retargeted")
	}
	b, _ := os.ReadFile(filepath.Join(dir, "commands.tsv"))
	if !strings.Contains(string(b), "untouched\t\t\techo unchanged\t\t\r\n") {
		t.Fatal("untargeted line changed")
	}
	apply(t, s, dir, projectops.Operation{Op: "command.inputs", Name: "compile-api", Values: map[string]string{}})
	p, _ = config.LoadProject(dir)
	c, _ = p.Config.FindCommand("compile-api")
	if c.Cmd != "echo {target}" {
		t.Fatal("cleared input did not resume prompting")
	}
}
func TestAgentDryRunConflictAndInvalidBatchLeaveFilesUntouched(t *testing.T) {
	s, dir := fixture(t)
	st, _ := state(dir)
	before, _ := projectfs.Read(dir)
	ops := []projectops.Operation{{Op: "command.create", Command: &mdl.Command{Name: "hello", Cmd: "echo hello"}}}
	r := s.Handle(Request{Action: "apply", Project: dir, Revision: st.Revision, DryRun: true, Operations: ops})
	if !r.OK || r.Data.(Result).Committed || len(r.Data.(Result).Changes) == 0 {
		t.Fatal(r)
	}
	after, _ := projectfs.Read(dir)
	if projectfs.Revision(before) != projectfs.Revision(after) {
		t.Fatal("dry run wrote files")
	}
	r = s.Handle(Request{Action: "apply", Project: dir, Revision: "stale", Operations: ops})
	if r.OK || r.Code != "conflict" {
		t.Fatal(r)
	}
	r = s.Handle(Request{Action: "apply", Project: dir, Revision: st.Revision, Operations: append(ops, projectops.Operation{Op: "workflow.create", Workflow: &mdl.Workflow{Name: "bad", Commands: []string{"missing"}}})})
	if r.OK {
		t.Fatal("invalid batch committed")
	}
	after, _ = projectfs.Read(dir)
	if projectfs.Revision(before) != projectfs.Revision(after) {
		t.Fatal("partial batch committed")
	}
}
func TestAgentRefusesReferencedDeletesAndTraversal(t *testing.T) {
	s, dir := fixture(t)
	apply(t, s, dir, projectops.Operation{Op: "variable.create", Name: "root", Value: str("demo")}, projectops.Operation{Op: "command.create", Command: &mdl.Command{Name: "show", Cmd: "echo {$root}"}}, projectops.Operation{Op: "workflow.create", Workflow: &mdl.Workflow{Name: "wf", Commands: []string{"show"}}})
	for _, op := range []projectops.Operation{{Op: "command.delete", Name: "show"}, {Op: "variable.delete", Name: "root"}, {Op: "list.create", Name: "../escape"}, {Op: "variable.create", Name: "x", Value: str("bad\nvalue")}} {
		st, _ := state(dir)
		r := s.Handle(Request{Action: "apply", Project: dir, Revision: st.Revision, Operations: []projectops.Operation{op}})
		if r.OK {
			t.Fatalf("accepted %+v", op)
		}
		after, _ := state(dir)
		if after.Revision != st.Revision {
			t.Fatal("failure changed revision")
		}
	}
	apply(t, s, dir, projectops.Operation{Op: "workflow.delete", Name: "wf"}, projectops.Operation{Op: "command.delete", Name: "show"}, projectops.Operation{Op: "variable.delete", Name: "root"})
}
func TestAgentPlanTokenAndExecution(t *testing.T) {
	s, dir := fixture(t)
	apply(t, s, dir, projectops.Operation{Op: "command.create", Command: &mdl.Command{Name: "show", Cmd: "echo {value}"}})
	req := Request{Action: "run.plan", Project: dir, Steps: []Step{{Command: "show"}}, Cwd: dir}
	resp := s.Handle(req)
	if !resp.OK {
		t.Fatal(resp)
	}
	plan := resp.Data.(Plan)
	if len(plan.Missing) != 1 || plan.Token != "" {
		t.Fatal(plan)
	}
	req.Steps[0].Inputs = map[string]string{"value": "baton-agent-test"}
	resp = s.Handle(req)
	if !resp.OK {
		t.Fatal(resp)
	}
	plan = resp.Data.(Plan)
	if plan.Token == "" || plan.Commands[0].Dir != dir {
		t.Fatal(plan)
	}
	req.Action = "run.execute"
	req.PlanToken = "bad"
	if r := s.Handle(req); r.OK || r.Code != "conflict" {
		t.Fatal(r)
	}
	req.PlanToken = plan.Token
	resp = s.Handle(req)
	if !resp.OK {
		t.Fatal(resp)
	}
	result := resp.Data.(map[string]any)
	outputs := result["results"].([]Output)
	if result["completed"] != true || !strings.Contains(outputs[0].Stdout, "baton-agent-test") {
		t.Fatal(result)
	}
}
func TestAgentBusyLock(t *testing.T) {
	s, dir := fixture(t)
	st, _ := state(dir)
	release, err := projectfs.Lock(s.ProjectsDir)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	r := s.Handle(Request{Action: "apply", Project: dir, Revision: st.Revision, Operations: []projectops.Operation{{Op: "variable.create", Name: "x", Value: str("a")}}})
	if r.OK || r.Code != "busy" {
		t.Fatal(r)
	}
	r = s.Handle(Request{Action: "project.get", Project: dir})
	if !r.OK {
		t.Fatal(r)
	}
}

func TestAgentArchiveAndFailedExecution(t *testing.T) {
	s, dir := fixture(t)
	apply(t, s, dir, projectops.Operation{Op: "command.create", Command: &mdl.Command{Name: "fail", Cmd: "exit 7"}}, projectops.Operation{Op: "command.create", Command: &mdl.Command{Name: "later", Cmd: "echo should-not-run"}})
	req := Request{Action: "run.plan", Project: dir, Steps: []Step{{Command: "fail"}, {Command: "later"}}}
	resp := s.Handle(req)
	if !resp.OK {
		t.Fatal(resp)
	}
	req.PlanToken = resp.Data.(Plan).Token
	req.Action = "run.execute"
	resp = s.Handle(req)
	if resp.OK || resp.Code != "execution_failed" {
		t.Fatal(resp)
	}
	outputs := resp.Data.(map[string]any)["results"].([]Output)
	if len(outputs) != 1 || outputs[0].ExitCode != 7 {
		t.Fatal(outputs)
	}
	st, _ := state(dir)
	resp = s.Handle(Request{Action: "project.delete", Project: dir, Revision: st.Revision, DryRun: true})
	if !resp.OK {
		t.Fatal(resp)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatal("dry-run removed project")
	}
	resp = s.Handle(Request{Action: "project.delete", Project: dir, Revision: st.Revision})
	if !resp.OK {
		t.Fatal(resp)
	}
	archive := resp.Data.(map[string]any)["archive"].(string)
	if _, err := os.Stat(filepath.Join(archive, "commands.tsv")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("original still exists")
	}
}
func TestAgentJSONOwnershipAndPendingJournal(t *testing.T) {
	s, dir := fixture(t)
	if err := os.WriteFile(filepath.Join(dir, "commands.json"), []byte(`{"commands":[{"name":"handwritten","cmd":"echo original"}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	st, _ := state(dir)
	resp := s.Handle(Request{Action: "apply", Project: dir, Revision: st.Revision, Operations: []projectops.Operation{{Op: "command.update", Name: "handwritten", Command: &mdl.Command{Name: "handwritten", Cmd: "echo changed"}}}})
	if resp.OK || !strings.Contains(resp.Error, "read-only") {
		t.Fatal(resp)
	}
	if err := os.WriteFile(filepath.Join(dir, projectfs.Journal), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := config.LoadProject(dir); err == nil {
		t.Fatal("loaded interrupted transaction")
	}
}

func TestAgentRequiresExplicitClearing(t *testing.T) {
	s, dir := fixture(t)
	apply(t, s, dir,
		projectops.Operation{Op: "list.create", Name: "env", Entries: []mdl.ListEntry{{Value: "dev"}}},
		projectops.Operation{Op: "command.create", Command: &mdl.Command{Name: "choose", Cmd: "echo {env}"}},
		projectops.Operation{Op: "command.create", Command: &mdl.Command{Name: "saved", Template: "choose", Values: map[string]string{"env": "dev"}}},
	)
	for _, op := range []projectops.Operation{{Op: "list.update", Name: "env"}, {Op: "command.inputs", Name: "saved"}} {
		st, _ := state(dir)
		resp := s.Handle(Request{Action: "apply", Project: dir, Revision: st.Revision, Operations: []projectops.Operation{op}})
		if resp.OK || !strings.Contains(resp.Error, "is required") {
			t.Fatal(resp)
		}
		after, _ := state(dir)
		if after.Revision != st.Revision {
			t.Fatal("omitted replacement erased data")
		}
	}
	// Empty values must survive Go JSON round trips as explicit empty values.
	ops := []projectops.Operation{{Op: "list.update", Name: "env", Entries: []mdl.ListEntry{}}, {Op: "command.inputs", Name: "saved", Values: map[string]string{}}}
	b, err := json.Marshal(ops)
	if err != nil {
		t.Fatal(err)
	}
	var decoded []projectops.Operation
	if err = json.Unmarshal(b, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded[0].Entries == nil || decoded[1].Values == nil {
		t.Fatalf("explicit clears omitted: %s", b)
	}
	result := apply(t, s, dir, decoded...)
	if entries, ok := result.State.Lists["env"]; !ok || entries == nil || len(entries) != 0 || len(result.State.Inputs["saved"]) != 1 {
		t.Fatal("explicit clear failed")
	}
	if input := result.State.Inputs["saved"][0]; input.Name != "env" || input.Placeholder != "{env}" {
		t.Fatal(input)
	}
	b, _ = json.Marshal(result.State.Inputs)
	if strings.Contains(string(b), `"Name"`) || !strings.Contains(string(b), `"name":"env"`) {
		t.Fatalf("unstable wire keys: %s", b)
	}
	updated := apply(t, s, dir, projectops.Operation{Op: "list.update", Name: "env", Entries: []mdl.ListEntry{{Value: "prod"}}})
	if len(updated.State.Lists["env"]) != 1 {
		t.Fatal("cannot update an empty list after reload")
	}
}
