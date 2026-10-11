package tui

import (
	"os"
	"path/filepath"
	"testing"

	mdl "github.com/Taka-S-dev/baton/internal/model"
	"github.com/Taka-S-dev/baton/internal/slot"
)

func TestSaveConfigFailurePreservesVars(t *testing.T) {
	d := t.TempDir()
	vars := map[string]string{"old.x": "A", "root": "original"}
	if err := slot.SaveVars(d, vars); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(d, "commands.local.json"), 0755); err != nil {
		t.Fatal(err)
	}
	m := Model{projectDir: d, vars: vars, config: mdl.Config{Commands: []mdl.Command{{Name: "new", Template: "tpl", Values: map[string]string{"x": "B"}}}}}
	if err := m.saveConfig(); err == nil {
		t.Fatal("expected save failure")
	}
	if m.vars["old.x"] != "A" || m.vars["new.x"] != "" {
		t.Fatalf("failed save changed in-memory vars: %v", m.vars)
	}
	disk, _ := slot.LoadVars(d)
	if disk["old.x"] != "A" || disk["new.x"] != "" {
		t.Fatalf("failed save changed disk vars: %v", disk)
	}
}
