package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Taka-S-dev/baton/internal/model"
)

// TestFingerprint_IgnoresName checks a pure rename keeps the fingerprint
// stable while any content change breaks it.
func TestFingerprint_IgnoresName(t *testing.T) {
	a := model.Command{Name: "old", Cmd: "go run .", Dir: "./api"}
	b := a
	b.Name = "new"
	if Fingerprint(a) != Fingerprint(b) {
		t.Fatal("renaming must not change the fingerprint")
	}
	b.Cmd = "go run ./cmd"
	if Fingerprint(a) == Fingerprint(b) {
		t.Fatal("a content change must change the fingerprint")
	}
}

// TestSnapshotRoundTrip checks save/load preserve the entries and a
// missing file loads as an empty map.
func TestSnapshotRoundTrip(t *testing.T) {
	dir := t.TempDir()
	if got := LoadSnapshot(dir); len(got) != 0 {
		t.Fatalf("missing snapshot must load empty, got %v", got)
	}
	entries := map[string]string{"build": "aaaa", "deploy": "bbbb"}
	if err := SaveSnapshot(dir, entries); err != nil {
		t.Fatal(err)
	}
	if got := LoadSnapshot(dir); got["build"] != "aaaa" || got["deploy"] != "bbbb" || len(got) != 2 {
		t.Fatalf("round trip = %v", got)
	}
}

// TestDetectRenames_Simple checks the core case: a referenced name
// vanished, a command with the same fingerprint appeared under a name
// the snapshot does not know, and the counts describe the references.
func TestDetectRenames_Simple(t *testing.T) {
	cfg := model.Config{Base: []model.Command{{Name: "api-start", Cmd: "go run ."}}}
	workflows := []model.Workflow{{Name: "wf", Commands: []string{"apistart", "apistart", "other"}}}
	vars := map[string]string{"apistart.env": "dev", "root": "C:\\x"}
	snap := map[string]string{"apistart": Fingerprint(model.Command{Cmd: "go run ."})}

	got := DetectRenames(cfg, workflows, vars, snap)
	if len(got) != 1 {
		t.Fatalf("renames = %+v, want one", got)
	}
	r := got[0]
	if r.Old != "apistart" || r.New != "api-start" || r.WfSteps != 2 || r.VarKeys != 1 || r.TplRefs != 0 {
		t.Fatalf("rename = %+v", r)
	}
}

// TestDetectRenames_AmbiguityProposesNothing checks the guards: several
// candidates sharing the fingerprint, several missing names sharing it,
// or a content change alongside the rename all stay silent.
func TestDetectRenames_AmbiguityProposesNothing(t *testing.T) {
	fp := Fingerprint(model.Command{Cmd: "make"})
	workflows := []model.Workflow{{Name: "wf", Commands: []string{"old"}}}

	// Two unknown commands with the same content.
	cfg := model.Config{Base: []model.Command{{Name: "a", Cmd: "make"}, {Name: "b", Cmd: "make"}}}
	if got := DetectRenames(cfg, workflows, nil, map[string]string{"old": fp}); got != nil {
		t.Fatalf("two candidates must propose nothing, got %+v", got)
	}

	// Two missing names sharing one fingerprint.
	cfg = model.Config{Base: []model.Command{{Name: "a", Cmd: "make"}}}
	wfs := []model.Workflow{{Name: "wf", Commands: []string{"old", "older"}}}
	snap := map[string]string{"old": fp, "older": fp}
	if got := DetectRenames(cfg, wfs, nil, snap); got != nil {
		t.Fatalf("two missing names on one fingerprint must propose nothing, got %+v", got)
	}

	// Renamed and edited in the same session: fingerprints no longer match.
	cfg = model.Config{Base: []model.Command{{Name: "a", Cmd: "make all"}}}
	if got := DetectRenames(cfg, workflows, nil, map[string]string{"old": fp}); got != nil {
		t.Fatalf("a content change must prevent the match, got %+v", got)
	}
}

// TestDetectRenames_TemplateRefs checks a renamed template is detected
// through the template field of its derived commands.
func TestDetectRenames_TemplateRefs(t *testing.T) {
	cfg := model.Config{
		Base:     []model.Command{{Name: "build2", Cmd: "make {x}"}},
		Commands: []model.Command{{Name: "as", Template: "build"}},
	}
	snap := map[string]string{
		"build": Fingerprint(model.Command{Cmd: "make {x}"}),
		"as":    Fingerprint(model.Command{Template: "build"}),
	}
	got := DetectRenames(cfg, nil, nil, snap)
	if len(got) != 1 || got[0].Old != "build" || got[0].New != "build2" || got[0].TplRefs != 1 {
		t.Fatalf("renames = %+v, want the template rename with one template ref", got)
	}
}

// TestDanglingCommandRefs checks the reference sweep: workflow steps,
// template fields and scoped vars keys count, globals and resolvable
// names do not.
func TestDanglingCommandRefs(t *testing.T) {
	cfg := model.Config{
		Base:     []model.Command{{Name: "build", Cmd: "make"}},
		Commands: []model.Command{{Name: "as", Template: "ghost-tpl"}},
	}
	workflows := []model.Workflow{{Name: "wf", Commands: []string{"build", "ghost-step"}}}
	vars := map[string]string{"ghost-vars.x": "v", "as.y": "v", "root": "C:\\x"}

	got := DanglingCommandRefs(cfg, workflows, vars)
	want := []string{"ghost-step", "ghost-tpl", "ghost-vars"}
	if len(got) != len(want) {
		t.Fatalf("dangling = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("dangling = %v, want %v", got, want)
		}
	}
}

// TestSnapshotFileFormat pins the on-disk shape: one name<TAB>fingerprint
// line per command, sorted, so the file diffs cleanly.
func TestSnapshotFileFormat(t *testing.T) {
	dir := t.TempDir()
	if err := SaveSnapshot(dir, map[string]string{"b": "2222", "a": "1111"}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, ".command_names"))
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "a\t1111\nb\t2222\n" {
		t.Fatalf("snapshot file = %q", raw)
	}
}
