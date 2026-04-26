package store_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Taka-S-dev/baton/internal/model"
	"github.com/Taka-S-dev/baton/internal/store"
)

// ── Workflows ─────────────────────────────────────────────────────────────────

func TestSaveAndLoadWorkflows(t *testing.T) {
	dir := t.TempDir()
	workflows := []model.Workflow{
		{Name: "build-all", Commands: []string{"build", "test"}},
		{
			Name:     "deploy",
			Commands: []string{"build", "deploy"},
			Vars:     map[string]map[string]string{"deploy": {"env": "production"}},
		},
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
	if got[1].Vars["deploy"]["env"] != "production" {
		t.Errorf("vars not preserved: %+v", got[1].Vars)
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

// ── Aliases ───────────────────────────────────────────────────────────────────

func TestSaveAndLoadAliases(t *testing.T) {
	dir := t.TempDir()
	aliases := []model.Alias{
		{Name: "clean-build", Steps: []string{"clean", "build"}},
		{
			Name:  "full-deploy",
			Steps: []string{"build", "test", "deploy"},
			Vars:  map[string]map[string]string{"deploy": {"env": "staging"}},
		},
	}
	if err := store.SaveAliases(dir, aliases); err != nil {
		t.Fatalf("SaveAliases: %v", err)
	}
	got, err := store.LoadAliases(dir)
	if err != nil {
		t.Fatalf("LoadAliases: %v", err)
	}
	if len(got) != len(aliases) {
		t.Fatalf("want %d aliases, got %d", len(aliases), len(got))
	}
	if got[0].Name != "clean-build" {
		t.Errorf("want name=clean-build, got %s", got[0].Name)
	}
	if got[1].Vars["deploy"]["env"] != "staging" {
		t.Errorf("vars not preserved: %+v", got[1].Vars)
	}
}

func TestLoadAliases_NoFile(t *testing.T) {
	dir := t.TempDir()
	got, err := store.LoadAliases(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("want empty slice, got %+v", got)
	}
}

func TestLoadAliases_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "aliases.json"), []byte("not json"), 0644)
	_, err := store.LoadAliases(dir)
	if err == nil {
		t.Error("want error for invalid JSON, got nil")
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
