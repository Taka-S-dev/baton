package slot_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Taka-S-dev/baton/internal/model"
	"github.com/Taka-S-dev/baton/internal/slot"
)

// ── HasPlaceholders ───────────────────────────────────────────────────────────

func TestHasPlaceholders(t *testing.T) {
	tests := []struct {
		name string
		cmd  model.Command
		want bool
	}{
		{"no placeholder", model.Command{Cmd: "echo hello"}, false},
		{"placeholder in cmd", model.Command{Cmd: "echo {name}"}, true},
		{"placeholder in dir", model.Command{Cmd: "ls", Dir: "{project}"}, true},
		{"both fields", model.Command{Cmd: "echo {env}", Dir: "{project}"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := slot.HasPlaceholders(tt.cmd); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// ── GetSlots ──────────────────────────────────────────────────────────────────

func TestGetSlots_Order(t *testing.T) {
	cmd := model.Command{Cmd: "deploy {env} {region}", Dir: "{project}"}
	slots := slot.GetSlots(cmd)
	if len(slots) != 3 {
		t.Fatalf("want 3 slots, got %d", len(slots))
	}
	if slots[0].Name != "env" || slots[1].Name != "region" || slots[2].Name != "project" {
		t.Errorf("unexpected order: %+v", slots)
	}
}

func TestGetSlots_Dedup(t *testing.T) {
	cmd := model.Command{Cmd: "echo {project}", Dir: "{project}"}
	slots := slot.GetSlots(cmd)
	if len(slots) != 1 {
		t.Fatalf("want 1 slot (deduped), got %d", len(slots))
	}
}

func TestGetSlots_VarsMapping(t *testing.T) {
	cmd := model.Command{
		Cmd:  "build {projDir}",
		Vars: map[string]string{"projDir": "project"},
	}
	slots := slot.GetSlots(cmd)
	if len(slots) != 1 {
		t.Fatalf("want 1 slot, got %d", len(slots))
	}
	if slots[0].ListName != "project" {
		t.Errorf("want ListName=project, got %s", slots[0].ListName)
	}
}

func TestGetSlots_DefaultListName(t *testing.T) {
	cmd := model.Command{Cmd: "echo {env}"}
	slots := slot.GetSlots(cmd)
	if slots[0].ListName != "env" {
		t.Errorf("want ListName=env, got %s", slots[0].ListName)
	}
}

func TestGetSlots_NoPlaceholders(t *testing.T) {
	cmd := model.Command{Cmd: "echo hello"}
	if got := slot.GetSlots(cmd); len(got) != 0 {
		t.Errorf("want empty, got %+v", got)
	}
}

// ── Apply ─────────────────────────────────────────────────────────────────────

func TestApply_ReplacesCmd(t *testing.T) {
	cmd := model.Command{Cmd: "deploy {env}", Dir: "{project}/bin"}
	result := slot.Apply(cmd, map[string]string{"env": "production", "project": "/app"})
	if result.Cmd != "deploy production" {
		t.Errorf("Cmd: got %q", result.Cmd)
	}
	if result.Dir != "/app/bin" {
		t.Errorf("Dir: got %q", result.Dir)
	}
}

func TestApply_MultipleOccurrences(t *testing.T) {
	cmd := model.Command{Cmd: "echo {x} and {x}"}
	result := slot.Apply(cmd, map[string]string{"x": "hello"})
	if result.Cmd != "echo hello and hello" {
		t.Errorf("got %q", result.Cmd)
	}
}

func TestApply_DoesNotMutateOriginal(t *testing.T) {
	cmd := model.Command{Cmd: "echo {name}"}
	_ = slot.Apply(cmd, map[string]string{"name": "world"})
	if cmd.Cmd != "echo {name}" {
		t.Error("original command was mutated")
	}
}

func TestApply_UnknownKeyIgnored(t *testing.T) {
	cmd := model.Command{Cmd: "echo {name}"}
	result := slot.Apply(cmd, map[string]string{"other": "value"})
	if result.Cmd != "echo {name}" {
		t.Errorf("got %q", result.Cmd)
	}
}

// ── HighlightSlot ─────────────────────────────────────────────────────────────

func TestHighlightSlot_ResolvedReplaced(t *testing.T) {
	out := slot.HighlightSlot("deploy {env}", "env", map[string]string{"env": "prod"})
	if out != "deploy prod" {
		t.Errorf("got %q", out)
	}
}

func TestHighlightSlot_CurrentSlotHighlighted(t *testing.T) {
	out := slot.HighlightSlot("deploy {env}", "env", map[string]string{})
	if out == "deploy {env}" {
		t.Error("current slot should be highlighted (ANSI), not plain")
	}
}

func TestHighlightSlot_UnresolvedOtherSlotUnchanged(t *testing.T) {
	out := slot.HighlightSlot("{a} and {b}", "a", map[string]string{})
	// {b} is unresolved and not current — should remain as-is
	if out == "... and ..." {
		t.Error("unexpected replacement")
	}
	// {a} gets ANSI, {b} stays literal
	if len(out) <= len("{a} and {b}") {
		t.Error("expected ANSI escape in output")
	}
}

// ── LoadLists / SaveList ──────────────────────────────────────────────────────

func TestSaveAndLoadList(t *testing.T) {
	dir := t.TempDir()
	entries := []model.ListEntry{
		{Value: "/home/user/api", Label: "api"},
		{Value: "/home/user/web", Label: "web"},
		{Value: "/home/user/worker"},
	}
	if err := slot.SaveList(dir, "project", entries); err != nil {
		t.Fatalf("SaveList: %v", err)
	}
	lists := slot.LoadLists(dir)
	got, ok := lists["project"]
	if !ok {
		t.Fatal("list 'project' not found")
	}
	if len(got) != len(entries) {
		t.Fatalf("want %d entries, got %d", len(entries), len(got))
	}
	for i, e := range entries {
		if got[i].Value != e.Value || got[i].Label != e.Label {
			t.Errorf("entry %d: want %+v, got %+v", i, e, got[i])
		}
	}
}

func TestLoadLists_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	lists := slot.LoadLists(dir)
	if len(lists) != 0 {
		t.Errorf("want empty map, got %+v", lists)
	}
}

func TestLoadLists_NonexistentDir(t *testing.T) {
	lists := slot.LoadLists("/nonexistent/path/xyz")
	if len(lists) != 0 {
		t.Errorf("want empty map, got %+v", lists)
	}
}

func TestSaveList_EmptyEntries(t *testing.T) {
	dir := t.TempDir()
	if err := slot.SaveList(dir, "empty", nil); err != nil {
		t.Fatalf("SaveList: %v", err)
	}
	// empty list should not appear in LoadLists (no entries)
	lists := slot.LoadLists(dir)
	if _, ok := lists["empty"]; ok {
		t.Error("empty list should not be returned by LoadLists")
	}
}

func TestLoadLists_SkipsNonTSV(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("hello"), 0644)
	_ = slot.SaveList(dir, "valid", []model.ListEntry{{Value: "x"}})
	lists := slot.LoadLists(dir)
	if _, ok := lists["notes"]; ok {
		t.Error("non-.tsv file should be ignored")
	}
	if _, ok := lists["valid"]; !ok {
		t.Error("valid .tsv not loaded")
	}
}
