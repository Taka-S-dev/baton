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

	writeJSON(t, filepath.Join(dir, "commands.json"), model.Config{
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
		t.Errorf("templates not loaded from commands.json: %+v", loaded.Base)
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
