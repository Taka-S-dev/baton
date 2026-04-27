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
	if len(cfg.Commands) != 1 {
		t.Fatalf("want 1 command, got %d", len(cfg.Commands))
	}
	cmd := cfg.Commands[0]
	if cmd.Vars["projDir"] != "project" {
		t.Errorf("projDir listName: want project, got %q", cmd.Vars["projDir"])
	}
	if cmd.Dir != "{projDir}" {
		t.Errorf("Dir: want {projDir}, got %q", cmd.Dir)
	}
}
