package projectops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Taka-S-dev/baton/internal/config"
	mdl "github.com/Taka-S-dev/baton/internal/model"
	"github.com/Taka-S-dev/baton/internal/store"
)

const header = "name\tgroup\tworkdir\tcmd\tshell\tslots\n"

// project writes a project whose commands.tsv has the given rows.
func project(t *testing.T, rows ...string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "commands.tsv"), []byte(header+strings.Join(rows, "")), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func load(t *testing.T, dir string) config.Project {
	t.Helper()
	p, err := config.LoadProject(dir)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func wantErr(t *testing.T, err error, text string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), text) {
		t.Fatalf("want an error containing %q, got %v", text, err)
	}
}

func TestValidName(t *testing.T) {
	for _, name := range []string{"build", "api.v2", "deploy-prod", "日本語"} {
		if err := ValidName(name); err != nil {
			t.Errorf("%q rejected: %v", name, err)
		}
	}
	for _, name := range []string{"", " build", "build ", ".", "..", "a/b", `a\b`, "a:b", "a\tb", "con", "COM1", "lpt9.txt", "trailing."} {
		if err := ValidName(name); err == nil {
			t.Errorf("%q accepted", name)
		}
	}
}

func TestCreateRejectsInvalidCommands(t *testing.T) {
	dir := project(t, "deploy\t\t\tdeploy --env {env}\t\t\n")
	cases := []struct {
		cmd  mdl.Command
		text string
	}{
		{mdl.Command{Name: "x", Cmd: "echo", Shell: "bash"}, "shell must be empty or ps"},
		{mdl.Command{Name: "x"}, "cmd or template is required"},
		{mdl.Command{Name: "x", Cmd: "echo", Values: map[string]string{"env": "prod"}}, "saved inputs require a template"},
		{mdl.Command{Name: "x", Cmd: "echo", Group: "a\tb"}, "tabs, newlines and NUL"},
		{mdl.Command{Name: "x", Template: "deploy", Values: map[string]string{"region": "eu"}}, `"region" is not a slot of "deploy"`},
		{mdl.Command{Name: "x", Template: "missing"}, "unknown or derived template"},
		{mdl.Command{Name: "x", Cmd: "echo {a}", Slots: map[string]string{"a": "bad,list"}}, "invalid slot mapping"},
	}
	for _, c := range cases {
		wantErr(t, Apply(dir, Operation{Op: "command.create", Command: &c.cmd}), c.text)
	}
	if _, ok := load(t, dir).Config.FindCommand("x"); ok {
		t.Fatal("a rejected command was saved")
	}
}

func TestCreateAppendsRowsAndDerivedCommands(t *testing.T) {
	dir := project(t, "deploy\t\t\tdeploy --env {env}\t\t\n")
	if err := Apply(dir, Operation{Op: "command.create", Command: &mdl.Command{Name: "hello", Cmd: "echo hello", Group: "demo"}}); err != nil {
		t.Fatal(err)
	}
	wantErr(t, Apply(dir, Operation{Op: "command.create", Command: &mdl.Command{Name: "hello", Cmd: "echo again"}}), "already exists")
	if err := Apply(dir, Operation{Op: "command.create", Command: &mdl.Command{Name: "deploy-prod", Template: "deploy", Values: map[string]string{"env": "prod"}}}); err != nil {
		t.Fatal(err)
	}
	p := load(t, dir)
	hello, ok := p.Config.FindCommand("hello")
	if !ok || hello.Source != "tsv" || hello.Group != "demo" {
		t.Fatalf("hello after create: %+v %v", hello, ok)
	}
	prod, ok := p.Config.FindCommand("deploy-prod")
	if !ok || prod.Template != "deploy" || prod.Cmd != "deploy --env prod" {
		t.Fatalf("derived command after create: %+v %v", prod, ok)
	}
	if p.Vars["deploy-prod.env"] != "prod" {
		t.Fatalf("saved input not mirrored into vars.tsv: %v", p.Vars)
	}
	local, _ := os.ReadFile(filepath.Join(dir, "commands.local.json"))
	if strings.Contains(string(local), `"values"`) || strings.Contains(string(local), `"cmd"`) {
		t.Fatalf("the local layer carries a baked value:\n%s", local)
	}
}

func TestUpdateKeepsNamesAndReadOnlyLayers(t *testing.T) {
	dir := project(t, "build\t\t\tgo build ./...\t\t\n")
	if err := os.WriteFile(filepath.Join(dir, "commands.json"), []byte(`{"commands":[{"name":"lint","cmd":"golangci-lint run"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	wantErr(t, Apply(dir, Operation{Op: "command.update", Name: "build", Command: &mdl.Command{Name: "compile", Cmd: "go build"}}), "use command.rename")
	wantErr(t, Apply(dir, Operation{Op: "command.update", Name: "lint", Command: &mdl.Command{Name: "lint", Cmd: "x"}}), "commands.json is read-only")
	wantErr(t, Apply(dir, Operation{Op: "command.update", Name: "nope", Command: &mdl.Command{Name: "nope", Cmd: "x"}}), "not found")
	if err := Apply(dir, Operation{Op: "command.update", Name: "build", Command: &mdl.Command{Name: "build", Cmd: "go build -v ./...", Shell: "ps"}}); err != nil {
		t.Fatal(err)
	}
	build, _ := load(t, dir).Config.FindCommand("build")
	if build.Cmd != "go build -v ./..." || build.Shell != "ps" || build.Source != "tsv" {
		t.Fatalf("build after update: %+v", build)
	}
}

func TestDeleteRefusesReferencedCommands(t *testing.T) {
	dir := project(t, "build\t\t\tgo build ./...\t\t\n", "deploy\t\t\tdeploy --env {env}\t\t\n")
	if err := store.SaveWorkflows(dir, []mdl.Workflow{{Name: "ship", Commands: []string{"build"}}}); err != nil {
		t.Fatal(err)
	}
	if err := Apply(dir, Operation{Op: "command.create", Command: &mdl.Command{Name: "deploy-prod", Template: "deploy", Values: map[string]string{"env": "prod"}}}); err != nil {
		t.Fatal(err)
	}
	wantErr(t, Apply(dir, Operation{Op: "command.delete", Name: "build"}), `referenced by workflow "ship"`)
	wantErr(t, Apply(dir, Operation{Op: "command.delete", Name: "deploy"}), `referenced by saved command "deploy-prod"`)
	if err := Apply(dir, Operation{Op: "workflow.delete", Name: "ship"}); err != nil {
		t.Fatal(err)
	}
	if err := Apply(dir, Operation{Op: "command.delete", Name: "build"}); err != nil {
		t.Fatal(err)
	}
	if err := Apply(dir, Operation{Op: "command.delete", Name: "deploy-prod"}); err != nil {
		t.Fatal(err)
	}
	p := load(t, dir)
	if _, ok := p.Config.FindCommand("build"); ok {
		t.Fatal("build still present")
	}
	if _, ok := p.Vars["deploy-prod.env"]; ok {
		t.Fatalf("scoped values of a deleted command kept: %v", p.Vars)
	}
}

func TestRenameMovesEveryReference(t *testing.T) {
	dir := project(t, "deploy\t\t\tdeploy --env {env}\t\t\n")
	if err := store.SaveWorkflows(dir, []mdl.Workflow{{Name: "ship", Commands: []string{"deploy", "deploy"}}}); err != nil {
		t.Fatal(err)
	}
	if err := Apply(dir, Operation{Op: "command.create", Command: &mdl.Command{Name: "deploy-prod", Template: "deploy", Values: map[string]string{"env": "prod"}}}); err != nil {
		t.Fatal(err)
	}
	wantErr(t, Apply(dir, Operation{Op: "command.rename", Name: "deploy", NewName: "deploy-prod"}), "already exists")
	if err := Apply(dir, Operation{Op: "command.rename", Name: "deploy", NewName: "release"}); err != nil {
		t.Fatal(err)
	}
	p := load(t, dir)
	if _, ok := p.Config.FindCommand("deploy"); ok {
		t.Fatal("old name still present")
	}
	if got := strings.Join(p.Workflows[0].Commands, ","); got != "release,release" {
		t.Fatalf("workflow steps after rename: %s", got)
	}
	prod, _ := p.Config.FindCommand("deploy-prod")
	if prod.Template != "release" || prod.Cmd != "deploy --env prod" {
		t.Fatalf("derived command after rename: %+v", prod)
	}
	if len(p.Warnings) > 0 {
		t.Fatalf("warnings after rename: %v", p.Warnings)
	}
}

func TestRetargetCommandRefs(t *testing.T) {
	cfg := mdl.Config{Commands: []mdl.Command{{Name: "a-prod", Template: "a"}, {Name: "b-prod", Template: "b"}}}
	workflows := []mdl.Workflow{{Name: "w", Commands: []string{"a", "b", "a"}}}
	vars := map[string]string{"a.env": "prod", "a.sub.env": "kept", "ab.env": "kept", "*": "x"}
	wf, tpl, keys := RetargetCommandRefs(&cfg, workflows, vars, "a", "z")
	if wf != 2 || tpl != 1 || keys != 1 {
		t.Fatalf("counts: %d %d %d", wf, tpl, keys)
	}
	if got := strings.Join(workflows[0].Commands, ","); got != "z,b,z" {
		t.Fatalf("steps: %s", got)
	}
	if cfg.Commands[0].Template != "z" || cfg.Commands[1].Template != "b" {
		t.Fatalf("templates: %+v", cfg.Commands)
	}
	if vars["z.env"] != "prod" || vars["a.sub.env"] != "kept" || vars["ab.env"] != "kept" {
		t.Fatalf("vars: %v", vars)
	}
	if _, ok := vars["a.env"]; ok {
		t.Fatalf("old key kept: %v", vars)
	}
}

func TestWorkflowOperations(t *testing.T) {
	dir := project(t, "build\t\t\tgo build ./...\t\t\n")
	wantErr(t, Apply(dir, Operation{Op: "workflow.create", Workflow: &mdl.Workflow{Name: "ship", Commands: []string{"nope"}}}), `unknown command "nope"`)
	wantErr(t, Apply(dir, Operation{Op: "workflow.create", Workflow: &mdl.Workflow{Name: "ship"}}), "must contain commands")
	if err := Apply(dir, Operation{Op: "workflow.create", Workflow: &mdl.Workflow{Name: "ship", Commands: []string{"build"}}}); err != nil {
		t.Fatal(err)
	}
	wantErr(t, Apply(dir, Operation{Op: "workflow.create", Workflow: &mdl.Workflow{Name: "ship", Commands: []string{"build"}}}), "already exists")
	store.SaveLastWorkflow(dir, "ship")
	if err := Apply(dir, Operation{Op: "workflow.rename", Name: "ship", NewName: "release"}); err != nil {
		t.Fatal(err)
	}
	p := load(t, dir)
	if len(p.Workflows) != 1 || p.Workflows[0].Name != "release" {
		t.Fatalf("workflows after rename: %+v", p.Workflows)
	}
	if got := store.LoadLastWorkflow(dir); got != "release" {
		t.Fatalf("last workflow follows the rename: %q", got)
	}
}

func TestListOperations(t *testing.T) {
	dir := project(t, "deploy\t\t\tdeploy --env {env}\t\t\n")
	wantErr(t, Apply(dir, Operation{Op: "list.update", Name: "env"}), "entries is required")
	wantErr(t, Apply(dir, Operation{Op: "list.create", Name: "env", Entries: []mdl.ListEntry{{Value: " "}}}), "cannot be empty")
	if err := Apply(dir, Operation{Op: "list.create", Name: "env", Entries: []mdl.ListEntry{{Value: "staging"}, {Value: "prod", Label: "production"}}}); err != nil {
		t.Fatal(err)
	}
	wantErr(t, Apply(dir, Operation{Op: "list.delete", Name: "env"}), `referenced by command "deploy"`)
	if err := Apply(dir, Operation{Op: "list.update", Name: "env", Entries: []mdl.ListEntry{}}); err != nil {
		t.Fatal(err)
	}
	if entries, ok := load(t, dir).Lists["env"]; !ok || len(entries) != 0 {
		t.Fatalf("env after emptying: %v %v", entries, ok)
	}
	if err := Apply(dir, Operation{Op: "command.delete", Name: "deploy"}); err != nil {
		t.Fatal(err)
	}
	if err := Apply(dir, Operation{Op: "list.delete", Name: "env"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "lists", "env.tsv")); !os.IsNotExist(err) {
		t.Fatalf("list file after delete: %v", err)
	}
}

func TestVariableOperations(t *testing.T) {
	dir := project(t, "deploy\t\t\tdeploy --dir {$root}\t\t\n")
	value := `C:\work`
	wantErr(t, Apply(dir, Operation{Op: "variable.create", Name: "bad name", Value: &value}), "must match")
	wantErr(t, Apply(dir, Operation{Op: "variable.create", Name: "root"}), "value is required")
	wantErr(t, Apply(dir, Operation{Op: "variable.update", Name: "root", Value: &value}), "not found")
	if err := Apply(dir, Operation{Op: "variable.create", Name: "root", Value: &value}); err != nil {
		t.Fatal(err)
	}
	p := load(t, dir)
	if p.Vars["root"] != value || len(p.Warnings) > 0 {
		t.Fatalf("vars %v warnings %v", p.Vars, p.Warnings)
	}
	if err := Apply(dir, Operation{Op: "variable.delete", Name: "root"}); err != nil {
		t.Fatal(err)
	}
	if p := load(t, dir); len(p.Warnings) == 0 {
		t.Fatal("deleting a referenced variable left no warning")
	}
}

func TestApplyRejectsUnknownOperations(t *testing.T) {
	wantErr(t, Apply(project(t), Operation{Op: "command.explode"}), `unknown operation "command.explode"`)
}
