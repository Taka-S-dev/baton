package store_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Taka-S-dev/baton/internal/config"
	"github.com/Taka-S-dev/baton/internal/model"
	"github.com/Taka-S-dev/baton/internal/store"
)

// ── Workflows ─────────────────────────────────────────────────────────────────

func TestSaveAndLoadWorkflows(t *testing.T) {
	dir := t.TempDir()
	workflows := []model.Workflow{
		{Name: "build-all", Commands: []string{"build", "test"}},
		{Name: "deploy", Commands: []string{"build", "deploy"}},
	}
	if err := store.SaveWorkflows(dir, workflows); err != nil {
		t.Fatalf("SaveWorkflows: %v", err)
	}
	got, err := store.LoadWorkflows(dir)
	if err != nil {
		t.Fatalf("LoadWorkflows: %v", err)
	}
	if len(got) != len(workflows) {
		t.Fatalf("want %d workflows, got %d", len(workflows), len(got))
	}
	if got[0].Name != "build-all" {
		t.Errorf("want name=build-all, got %s", got[0].Name)
	}
	if len(got[1].Commands) != 2 || got[1].Commands[1] != "deploy" {
		t.Errorf("commands not preserved: %+v", got[1].Commands)
	}
}

func TestLoadWorkflows_NoFile(t *testing.T) {
	dir := t.TempDir()
	got, err := store.LoadWorkflows(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("want empty slice, got %+v", got)
	}
}

func TestLoadWorkflows_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "workflows.json"), []byte("not json"), 0644)
	_, err := store.LoadWorkflows(dir)
	if err == nil {
		t.Error("want error for invalid JSON, got nil")
	}
}

func TestSaveWorkflows_Empty(t *testing.T) {
	dir := t.TempDir()
	if err := store.SaveWorkflows(dir, []model.Workflow{}); err != nil {
		t.Fatalf("SaveWorkflows: %v", err)
	}
	got, err := store.LoadWorkflows(dir)
	if err != nil {
		t.Fatalf("LoadWorkflows: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("want empty slice, got %+v", got)
	}
}

// ── LastWorkflow ──────────────────────────────────────────────────────────────

func TestSaveAndLoadLastWorkflow(t *testing.T) {
	dir := t.TempDir()
	store.SaveLastWorkflow(dir, "build-all")
	got := store.LoadLastWorkflow(dir)
	if got != "build-all" {
		t.Errorf("want build-all, got %q", got)
	}
}

func TestLoadLastWorkflow_NoFile(t *testing.T) {
	dir := t.TempDir()
	got := store.LoadLastWorkflow(dir)
	if got != "" {
		t.Errorf("want empty string, got %q", got)
	}
}

func TestSaveLastWorkflow_Overwrite(t *testing.T) {
	dir := t.TempDir()
	store.SaveLastWorkflow(dir, "first")
	store.SaveLastWorkflow(dir, "second")
	got := store.LoadLastWorkflow(dir)
	if got != "second" {
		t.Errorf("want second, got %q", got)
	}
}

// ── Config ────────────────────────────────────────────────────────────────────

// TestSaveConfigRoundtrip ensures saving preserves both concrete and
// template-derived commands, and that templates stay in templates.json only.
func TestSaveConfigRoundtrip(t *testing.T) {
	dir := t.TempDir()

	writeJSON(t, filepath.Join(dir, "templates.json"), model.Config{
		Commands: []model.Command{
			{Name: "build", Cmd: "make {target}", Slots: map[string]string{"target": "targets"}},
		},
	})

	cfg := model.Config{
		Base: []model.Command{{Name: "should-not-be-written", Cmd: "x"}},
		Commands: []model.Command{
			{Name: "list-files", Cmd: "ls -la"},
			{Name: "build-src", Template: "build", Values: map[string]string{"target": "src"}},
		},
	}
	if err := store.SaveConfig(dir, cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	loaded, err := config.LoadConfig(dir)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if len(loaded.Base) != 1 || loaded.Base[0].Name != "build" {
		t.Errorf("templates not loaded from templates.json: %+v", loaded.Base)
	}
	if len(loaded.Commands) != 2 {
		t.Fatalf("commands lost on save: got %d, want 2", len(loaded.Commands))
	}
	if loaded.Commands[1].Template != "build" || loaded.Commands[1].Values["target"] != "src" {
		t.Errorf("template-derived command not roundtripped: %+v", loaded.Commands[1])
	}
	if loaded.Commands[1].Cmd != "make src" {
		t.Errorf("template-derived command not baked on load: got %q, want %q", loaded.Commands[1].Cmd, "make src")
	}

	// The hand-written layer must never leak into commands.local.json.
	raw, err := os.ReadFile(filepath.Join(dir, "commands.local.json"))
	if err != nil {
		t.Fatalf("read commands.local.json: %v", err)
	}
	var onDisk model.Config
	if err := json.Unmarshal(raw, &onDisk); err != nil {
		t.Fatalf("commands.local.json is not valid JSON: %v", err)
	}
	for _, c := range onDisk.Commands {
		if c.Name == "should-not-be-written" {
			t.Error("hand-written command leaked into commands.local.json")
		}
	}
}

// TestLegacySavedCommandsMigrate ensures the old "saved_commands" section is
// converted to unified commands on load, and disappears after the next save.
func TestLegacySavedCommandsMigrate(t *testing.T) {
	dir := t.TempDir()
	legacy := `{
  "commands": [{"name": "pwd", "cmd": "pwd"}],
  "saved_commands": [{"name": "build-src", "template_ref": "build", "values": {"workdir": "src"}}]
}`
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(legacy), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.LoadConfig(dir)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if len(cfg.Commands) != 2 {
		t.Fatalf("want 2 commands (pwd + migrated build-src), got %d", len(cfg.Commands))
	}
	migrated := cfg.Commands[1]
	if migrated.Name != "build-src" || migrated.Template != "build" || migrated.Values["workdir"] != "src" {
		t.Fatalf("legacy saved_command not converted: %+v", migrated)
	}

	// Resave and confirm the file migrated to commands.local.json in the
	// unified format, with the legacy config.json removed.
	if err := store.SaveConfig(dir, cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "config.json")); !os.IsNotExist(err) {
		t.Error("legacy config.json still present after migration save")
	}
	raw, _ := os.ReadFile(filepath.Join(dir, "commands.local.json"))
	var onDisk map[string]json.RawMessage
	if err := json.Unmarshal(raw, &onDisk); err != nil {
		t.Fatal(err)
	}
	if _, ok := onDisk["saved_commands"]; ok {
		t.Error("saved_commands section still present after migration save")
	}

	reloaded, err := config.LoadConfig(dir)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if len(reloaded.Commands) != 2 {
		t.Fatalf("commands lost after migration save: %+v", reloaded.Commands)
	}
}

// TestLegacyVarsKeyReadable ensures the old "vars" key is read as Slots.
func TestLegacyVarsKeyReadable(t *testing.T) {
	dir := t.TempDir()
	legacy := `{"commands": [{"name": "build", "cmd": "make {target}", "vars": {"target": "targets"}}]}`
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(legacy), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadConfig(dir)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if len(cfg.Commands) != 1 || cfg.Commands[0].Slots["target"] != "targets" {
		t.Fatalf("legacy vars key not read as slots: %+v", cfg.Commands)
	}
}

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}
