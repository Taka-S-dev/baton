package store

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestProjectHistoryOrder(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"alpha", "beta", "gamma"} {
		if err := os.Mkdir(filepath.Join(dir, name), 0755); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"beta", "gamma", "beta"} {
		if err := RecordProjectOpen(dir, name); err != nil {
			t.Fatal(err)
		}
	}
	names := []string{"alpha", "beta", "gamma"}
	SortProjectsByRecent(dir, names)
	if !reflect.DeepEqual(names, []string{"beta", "gamma", "alpha"}) {
		t.Fatal(names)
	}
	if !reflect.DeepEqual(LoadProjectHistory(dir), []string{"beta", "gamma"}) {
		t.Fatal("history must deduplicate")
	}
	if err := os.Remove(filepath.Join(dir, "gamma")); err != nil {
		t.Fatal(err)
	}
	if err := RecordProjectOpen(dir, "alpha"); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(LoadProjectHistory(dir), []string{"alpha", "beta"}) {
		t.Fatal("deleted history not pruned")
	}
	if _, err := os.Stat(filepath.Join(dir, ".baton-recent", "projects.json")); err != nil {
		t.Fatal("history must live under .baton-recent: ", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".baton-recent", "projects.json"), []byte("broken"), 0644); err != nil {
		t.Fatal(err)
	}
	names = []string{"beta", "alpha"}
	SortProjectsByRecent(dir, names)
	if !reflect.DeepEqual(names, []string{"alpha", "beta"}) {
		t.Fatal("invalid history must fall back to name order")
	}
}
