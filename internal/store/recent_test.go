package store

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestRecentIsolationDedupAndOrder(t *testing.T) {
	root := t.TempDir()
	cwd := filepath.Join(root, "work")
	other := filepath.Join(root, "other")
	p := filepath.Join(root, "p")
	q := filepath.Join(root, "q")
	now := time.Now()
	for i, ref := range []struct{ cwd, p, name string }{{cwd, p, "build"}, {cwd, p, "test"}, {cwd, q, "build"}, {other, p, "other"}, {cwd, p, "build"}} {
		if err := RecordRecent(root, ref.cwd, ref.p, ref.name, now.Add(time.Duration(i)*time.Second)); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := LoadRecent(root, cwd)
	if err != nil || len(entries) != 3 || entries[0].Command != "build" || entries[0].Project != p || entries[1].Project != q {
		t.Fatalf("%+v %v", entries, err)
	}
	entries, err = LoadRecent(root, other)
	if err != nil || len(entries) != 1 || entries[0].Command != "other" {
		t.Fatal("directory histories mixed")
	}
	entries, err = LoadRecent(root, filepath.Join(root, "empty"))
	if err != nil || len(entries) != 0 {
		t.Fatal("empty history failed")
	}
}
func TestRecentConcurrentReferencesAndCorruption(t *testing.T) {
	root := t.TempDir()
	var wg sync.WaitGroup
	for _, name := range []string{"a", "b", "c"} {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			if err := RecordRecent(root, root, filepath.Join(root, "project"), name, time.Now()); err != nil {
				t.Error(err)
			}
		}(name)
	}
	wg.Wait()
	dir := recentDirectory(root, root)
	if err := os.WriteFile(filepath.Join(dir, "broken.json"), []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
	entries, err := LoadRecent(root, root)
	if len(entries) != 3 || err == nil {
		t.Fatalf("valid history should survive corruption: %v %v", entries, err)
	}
}
