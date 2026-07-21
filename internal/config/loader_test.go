package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Taka-S-dev/baton/internal/model"
)

func TestParseVarsStr_SingleVar(t *testing.T) {
	v := parseVarsStr("projDir=project")
	if v["projDir"] != "project" {
		t.Errorf("want project, got %q", v["projDir"])
	}
}

func TestParseVarsStr_MultipleVars(t *testing.T) {
	v := parseVarsStr("projDir=project,projCmd=project")
	if v["projDir"] != "project" {
		t.Errorf("projDir: want project, got %q", v["projDir"])
	}
	if v["projCmd"] != "project" {
		t.Errorf("projCmd: want project, got %q", v["projCmd"])
	}
}

func TestLoadTSV_UnusedVarDoesNotAffectListName(t *testing.T) {
	dir := t.TempDir()
	tsv := "name\tgroup\tworkdir\tcmd\tshell\tvars\n" +
		"build\tmake\t{projDir}\techo building\t\tprojDir=project,projCmd=project\n"
	if err := os.WriteFile(filepath.Join(dir, "config.tsv"), []byte(tsv), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	// Legacy TSV commands are treated as templates.
	if len(cfg.Base) != 1 {
		t.Fatalf("want 1 template, got %d", len(cfg.Base))
	}
	cmd := cfg.Base[0]
	if cmd.Slots["projDir"] != "project" {
		t.Errorf("projDir listName: want project, got %q", cmd.Slots["projDir"])
	}
	if cmd.Dir != "{projDir}" {
		t.Errorf("Dir: want {projDir}, got %q", cmd.Dir)
	}
}

// TestLoadTSV_ExcelQuotedVars ensures Excel-style quoted fields are
// unquoted: Excel wraps cells containing commas in double quotes when
// saving as TSV.
func TestLoadTSV_ExcelQuotedVars(t *testing.T) {
	dir := t.TempDir()
	tsv := "name\tgroup\tworkdir\tcmd\tshell\tvars\n" +
		"say2\tutility\t\techo {message} {env}\t\t\"message=messages,env=environments\"\n"
	if err := os.WriteFile(filepath.Join(dir, "commands.tsv"), []byte(tsv), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Base) != 1 {
		t.Fatalf("want 1 command, got %d", len(cfg.Base))
	}
	slots := cfg.Base[0].Slots
	if slots["message"] != "messages" || slots["env"] != "environments" {
		t.Errorf("quoted vars field not parsed: %+v", slots)
	}
}

// TestLoadConfig_TSVAndJSONCoexist guards the TSV-main workflow: creating
// config.json via the TUI (saved commands) must not hide TSV commands.
func TestLoadConfig_TSVAndJSONCoexist(t *testing.T) {
	dir := t.TempDir()
	tsv := "name\tgroup\tworkdir\tcmd\tshell\tvars\n" +
		"build\tmake\t{projDir}\techo building\t\tprojDir=project\n"
	if err := os.WriteFile(filepath.Join(dir, "config.tsv"), []byte(tsv), 0644); err != nil {
		t.Fatal(err)
	}
	cfgJSON := `{"saved_commands":[{"name":"build-proj","template_ref":"build","values":{"projDir":"proj"}}]}`
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(cfgJSON), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Base) != 1 || cfg.Base[0].Name != "build" {
		t.Fatalf("TSV commands hidden by config.json: %+v", cfg.Base)
	}
	if len(cfg.Commands) != 1 || cfg.Commands[0].Template != "build" {
		t.Fatalf("legacy saved command not migrated: %+v", cfg.Commands)
	}
	if _, ok := cfg.FindCommand("build"); !ok {
		t.Error("template target from TSV not resolvable")
	}
}

// TestTemplatesTSVPreferred ensures templates.tsv is read, and wins over
// legacy config.tsv when both exist.
func TestTemplatesTSVPreferred(t *testing.T) {
	dir := t.TempDir()
	newTSV := "name\tgroup\tworkdir\tcmd\tshell\tvars\nnew-cmd\t\t\techo new\t\t\n"
	oldTSV := "name\tgroup\tworkdir\tcmd\tshell\tvars\nold-cmd\t\t\techo old\t\t\n"
	if err := os.WriteFile(filepath.Join(dir, "templates.tsv"), []byte(newTSV), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.tsv"), []byte(oldTSV), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Base) != 1 || cfg.Base[0].Name != "new-cmd" {
		t.Fatalf("templates.tsv not preferred: %+v", cfg.Base)
	}
}

// TestTemplateEditPropagates ensures editing a template updates derived
// commands on the next load (the baked cmd in config.json is a cache).
func TestTemplateEditPropagates(t *testing.T) {
	dir := t.TempDir()
	templates := `{"commands": [{"name": "build", "cmd": "gmake {target}"}]}`
	if err := os.WriteFile(filepath.Join(dir, "templates.json"), []byte(templates), 0644); err != nil {
		t.Fatal(err)
	}
	// Baked cmd is stale (from before the template was edited).
	cfgJSON := `{"commands": [{"name": "build-src", "cmd": "make src", "template": "build", "values": {"target": "src"}}]}`
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(cfgJSON), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Commands[0].Cmd != "gmake src" {
		t.Errorf("template edit not propagated: got %q, want %q", cfg.Commands[0].Cmd, "gmake src")
	}
}

// TestMissingTemplateFallsBackToBakedCmd ensures a derived command still
// runs from its baked cmd when the template has been deleted.
func TestMissingTemplateFallsBackToBakedCmd(t *testing.T) {
	dir := t.TempDir()
	cfgJSON := `{"commands": [{"name": "build-src", "cmd": "make src", "template": "gone", "values": {"target": "src"}}]}`
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(cfgJSON), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Commands[0].Cmd != "make src" {
		t.Errorf("baked cmd fallback lost: got %q", cfg.Commands[0].Cmd)
	}
}

// TestAppendCommandTSV checks the append-only TSV writer: it creates the
// file with a header when absent, appends to whatever TSV the loader
// reads (legacy names included), and re-reads the file at call time so
// rows saved by an editor between load and save survive.
func TestAppendCommandTSV(t *testing.T) {
	dir := t.TempDir()

	file, err := AppendCommandTSV(dir, model.Command{Name: "a", Cmd: "echo a"})
	if err != nil || file != "commands.tsv" {
		t.Fatalf("file=%q err=%v", file, err)
	}

	// Simulate a hand edit saved after the project was loaded.
	path := filepath.Join(dir, "commands.tsv")
	data, _ := os.ReadFile(path)
	if err := os.WriteFile(path, append(data, []byte("hand\t\t\techo hand\t\t\n")...), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := AppendCommandTSV(dir, model.Command{Name: "b", Cmd: "echo {x}", Slots: map[string]string{"x": "xs"}}); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Base) != 3 {
		t.Fatalf("want 3 commands (a, hand, b), got %+v", cfg.Base)
	}
	if _, ok := cfg.FindCommand("hand"); !ok {
		t.Fatal("hand-edited row lost by append")
	}
	if b, _ := cfg.FindCommand("b"); b.Slots["x"] != "xs" {
		t.Fatalf("slots column not written: %+v", b)
	}

	// Legacy TSV name: append must target the file the loader reads.
	legacy := t.TempDir()
	if err := os.WriteFile(filepath.Join(legacy, "config.tsv"), []byte("name\tgroup\tworkdir\tcmd\tshell\tslots\nold\t\t\techo old\t\t\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	file, err = AppendCommandTSV(legacy, model.Command{Name: "new", Cmd: "echo new"})
	if err != nil || file != "config.tsv" {
		t.Fatalf("file=%q err=%v — must append to the loaded legacy file", file, err)
	}
	if _, err := os.Stat(filepath.Join(legacy, "commands.tsv")); err == nil {
		t.Fatal("must not create commands.tsv beside a loaded legacy TSV")
	}
}

// TestUpdateAndDeleteCommandTSV checks the row-targeted writers: update
// rewrites exactly the named row (preserving CRLF endings and every
// other line), errors when the row vanished, and delete drops rows
// while ignoring names already gone.
func TestUpdateAndDeleteCommandTSV(t *testing.T) {
	dir := t.TempDir()
	tsv := "name\tgroup\tworkdir\tcmd\tshell\tslots\r\n" +
		"a\t\t\techo a\t\t\r\n" +
		"b\t\t\techo b\t\t\r\n"
	if err := os.WriteFile(filepath.Join(dir, "commands.tsv"), []byte(tsv), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := UpdateCommandTSV(dir, "b", model.Command{Name: "b2", Cmd: "echo b2"}); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(filepath.Join(dir, "commands.tsv"))
	got := string(raw)
	if !strings.Contains(got, "b2\t\t\techo b2\t\t\r\n") {
		t.Fatalf("row not replaced: %q", got)
	}
	if !strings.Contains(got, "a\t\t\techo a\t\t\r\n") {
		t.Fatalf("other rows must be byte-identical (incl. CRLF): %q", got)
	}

	if _, err := UpdateCommandTSV(dir, "vanished", model.Command{Name: "x"}); err == nil {
		t.Fatal("updating a missing row must error")
	}

	if _, err := DeleteCommandsTSV(dir, []string{"a", "already-gone"}); err != nil {
		t.Fatal(err)
	}
	raw, _ = os.ReadFile(filepath.Join(dir, "commands.tsv"))
	if strings.Contains(string(raw), "echo a") || !strings.Contains(string(raw), "b2") {
		t.Fatalf("delete result wrong: %q", raw)
	}
}

// TestExampleProjectsLoad guards the shipped sample projects against rot:
// they must load with zero warnings and contain the documented commands.
func TestExampleProjectsLoad(t *testing.T) {
	for _, dir := range []string{"example-json", "example-tsv"} {
		p, err := LoadProject(filepath.Join("..", "..", "projects.example", dir))
		if err != nil {
			t.Fatalf("%s: %v", dir, err)
		}
		if len(p.Warnings) != 0 {
			t.Errorf("%s: samples must be warning-free, got %v", dir, p.Warnings)
		}
		if len(p.Config.Base) < 6 {
			t.Errorf("%s: only %d commands loaded", dir, len(p.Config.Base))
		}
		found, ok := p.Config.FindCommand("build")
		if !ok || found.Slots["projDir"] != "project" {
			t.Errorf("%s: build command slots not loaded: %+v", dir, found)
		}
	}
}

// TestLoadProject_Warnings checks the diagnostics behind `baton check`:
// a project wired with known mistakes reports each of them.
func TestLoadProject_Warnings(t *testing.T) {
	dir := t.TempDir()
	tsv := "name\tgroup\tworkdir\tcmd\tshell\tslots\n" +
		"build\t\t\techo {$root}\tbash\t\n"
	if err := os.WriteFile(filepath.Join(dir, "commands.tsv"), []byte(tsv), 0o644); err != nil {
		t.Fatal(err)
	}
	wf := `[{"name":"wf1","commands":["build","ghost"]}]`
	if err := os.WriteFile(filepath.Join(dir, "workflows.json"), []byte(wf), 0o644); err != nil {
		t.Fatal(err)
	}

	p, err := LoadProject(dir)
	if err != nil {
		t.Fatal(err)
	}
	joined := ""
	for _, w := range p.Warnings {
		joined += w + "; "
	}
	for _, want := range []string{
		`unknown shell "bash"`,
		"undefined var {$root}",
		`workflow "wf1" references unknown command "ghost"`,
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("warnings missing %q, got %q", want, joined)
		}
	}
}
