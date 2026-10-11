package store

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Taka-S-dev/baton/internal/model"
	"github.com/Taka-S-dev/baton/internal/slot"
)

func TestSaveConfigWithVarsRollback(t *testing.T) {
	for _, existing := range []bool{false, true} {
		t.Run(map[bool]string{false: "new", true: "existing"}[existing], func(t *testing.T) {
			d := t.TempDir()
			old := model.Config{Commands: []model.Command{{Name: "old", Cmd: "echo old"}}}
			if err := SaveConfig(d, old); err != nil {
				t.Fatal(err)
			}
			var oldVars []byte
			if existing {
				if err := slot.SaveVars(d, map[string]string{"root": "old"}); err != nil {
					t.Fatal(err)
				}
				oldVars, _ = os.ReadFile(filepath.Join(d, "vars.tsv"))
			}
			oldConfig, _ := os.ReadFile(filepath.Join(d, "commands.local.json"))
			injected := errors.New("injected commit failure")
			err := saveConfigWithVars(d, model.Config{Commands: []model.Command{{Name: "new", Cmd: "echo new"}}}, map[string]string{"root": "new"}, func(from, to string) error {
				if filepath.Base(from) == "commands.local.json" {
					return injected
				}
				return os.Rename(from, to)
			})
			if !errors.Is(err, injected) {
				t.Fatalf("lost save error: %v", err)
			}
			got, _ := os.ReadFile(filepath.Join(d, "commands.local.json"))
			if string(got) != string(oldConfig) {
				t.Fatal("config changed on failed save")
			}
			got, readErr := os.ReadFile(filepath.Join(d, "vars.tsv"))
			if existing {
				if readErr != nil || string(got) != string(oldVars) {
					t.Fatal("vars not restored")
				}
			} else if !os.IsNotExist(readErr) {
				t.Fatal("new vars file not removed")
			}
			entries, _ := os.ReadDir(d)
			for _, e := range entries {
				if strings.HasPrefix(e.Name(), ".baton-save-") {
					t.Fatal("staging directory leaked")
				}
			}
		})
	}
}
func TestSaveConfigWithVarsSuccessAndEmptyVars(t *testing.T) {
	d := t.TempDir()
	cfg := model.Config{Commands: []model.Command{{Name: "new", Cmd: "echo hi"}}}
	if err := SaveConfigWithVars(d, cfg, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(d, "vars.tsv")); !os.IsNotExist(err) {
		t.Fatal("created unused vars file")
	}
	if err := SaveConfigWithVars(d, cfg, map[string]string{"root": "new"}); err != nil {
		t.Fatal(err)
	}
	vars, _ := slot.LoadVars(d)
	if vars["root"] != "new" {
		t.Fatal("vars not saved")
	}
	if err := SaveConfigWithVars(d, cfg, nil); err != nil {
		t.Fatal(err)
	}
	vars, warnings := slot.LoadVars(d)
	if len(vars) != 0 || len(warnings) != 0 {
		t.Fatalf("failed to empty vars: %v %v", vars, warnings)
	}
}
func TestSaveConfigWithVarsRejectsUnreadableTarget(t *testing.T) {
	d := t.TempDir()
	if err := slot.SaveVars(d, map[string]string{"root": "old"}); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(d, "commands.local.json"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := SaveConfigWithVars(d, model.Config{}, map[string]string{"root": "new"}); err == nil {
		t.Fatal("accepted directory target")
	}
	vars, _ := slot.LoadVars(d)
	if vars["root"] != "old" {
		t.Fatal("changed vars before preparation completed")
	}
}

func TestSaveConfigWithVarsRetainsRecoveryOnRollbackFailure(t *testing.T) {
	d := t.TempDir()
	if err := slot.SaveVars(d, map[string]string{"root": "old"}); err != nil {
		t.Fatal(err)
	}
	injected := errors.New("injected failure")
	err := saveConfigWithVars(d, model.Config{}, map[string]string{"root": "new"}, func(from, to string) error {
		if filepath.Base(from) == "commands.local.json" || strings.HasSuffix(from, ".backup") {
			return injected
		}
		return os.Rename(from, to)
	})
	if err == nil || !strings.Contains(err.Error(), "recovery files kept") {
		t.Fatalf("recovery location not reported: %v", err)
	}
	backups, e := filepath.Glob(filepath.Join(d, ".baton-save-*", "vars.tsv.backup"))
	if e != nil || len(backups) != 1 {
		t.Fatalf("backup missing: %v %v", backups, e)
	}
	data, e := os.ReadFile(backups[0])
	if e != nil || !strings.Contains(string(data), "old") {
		t.Fatal("original contents not retained")
	}
}
