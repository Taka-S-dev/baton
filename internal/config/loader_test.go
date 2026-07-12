package config

import (
	"os"
	"path/filepath"
	"testing"
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
