package tui

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Taka-S-dev/baton/internal/store"
)

func TestLoadProjectRecordsOpenOrder(t *testing.T) {
	base := t.TempDir()
	for _, name := range []string{"alpha", "beta"} {
		dir := filepath.Join(base, name)
		if err := os.Mkdir(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "commands.tsv"), []byte("name\tgroup\tworkdir\tcmd\tshell\tslots\nb\t\t\techo b\t\t\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	m := Model{projectsDir: base}
	for _, name := range []string{"alpha", "beta"} {
		if err := m.loadProject(filepath.Join(base, name)); err != nil {
			t.Fatal(err)
		}
	}
	if got := store.LoadProjectHistory(base); !reflect.DeepEqual(got, []string{"beta", "alpha"}) {
		t.Fatalf("history = %v", got)
	}
	if m.errMsg != "" {
		t.Fatalf("remembering a project must not report: %q", m.errMsg)
	}

	outside := filepath.Join(t.TempDir(), "elsewhere")
	if err := os.Mkdir(outside, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "commands.tsv"), []byte("name\tgroup\tworkdir\tcmd\tshell\tslots\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := m.loadProject(outside); err != nil {
		t.Fatal(err)
	}
	if got := store.LoadProjectHistory(base); !reflect.DeepEqual(got, []string{"beta", "alpha"}) {
		t.Fatalf("a project outside the projects directory must not be recorded: %v", got)
	}
}
