package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunTabPreference(t *testing.T) {
	dir := t.TempDir()
	if LoadRunTab(dir) != "" {
		t.Fatal("missing preference")
	}
	for _, tab := range []string{"commands", "workflows"} {
		if err := SaveRunTab(dir, tab); err != nil {
			t.Fatal(err)
		}
		if LoadRunTab(dir) != tab {
			t.Fatal("round trip")
		}
	}
	if err := os.WriteFile(filepath.Join(dir, ".last_run_tab"), []byte("stale"), 0600); err != nil {
		t.Fatal(err)
	}
	if LoadRunTab(dir) != "" {
		t.Fatal("unknown tab must fall back")
	}
	if LoadRunTab(t.TempDir()) != "" {
		t.Fatal("project preference leaked")
	}
}
