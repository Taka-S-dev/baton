package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Taka-S-dev/baton/internal/model"
)

func TestLoadTSVRejectsInvalidRows(t *testing.T) {
	for _, row := range []string{"build make . echo_hi", "build\tgroup\tdir", "\t\t\techo hi", "build\t\t\t", "build\t\t\techo hi\t\t\textra"} {
		t.Run(row, func(t *testing.T) {
			d := t.TempDir()
			p := filepath.Join(d, "commands.tsv")
			if err := os.WriteFile(p, []byte("name\tgroup\tworkdir\tcmd\tshell\tslots\n"+row+"\n"), 0644); err != nil {
				t.Fatal(err)
			}
			_, err := LoadProject(d)
			if err == nil || !strings.Contains(err.Error(), "line 2") {
				t.Fatalf("expected line diagnostic, got %v", err)
			}
		})
	}
}
func TestTSVMutationsStopOnReadFailure(t *testing.T) {
	for _, action := range []string{"append", "update", "delete"} {
		t.Run(action, func(t *testing.T) {
			d := t.TempDir()
			p := filepath.Join(d, "commands.tsv")
			if err := os.Mkdir(p, 0755); err != nil {
				t.Fatal(err)
			}
			if _, _, err := readTSVLines(p); err == nil {
				t.Fatal("read error swallowed")
			}
			var err error
			switch action {
			case "append":
				_, err = AppendCommandTSV(d, model.Command{Name: "new", Cmd: "echo hi"})
			case "update":
				_, err = UpdateCommandTSV(d, "old", model.Command{Name: "new", Cmd: "echo hi"})
			case "delete":
				_, err = DeleteCommandsTSV(d, []string{"old"})
			}
			if err == nil {
				t.Fatal("mutation accepted unreadable file")
			}
			if _, err := os.Stat(p + ".tmp"); !os.IsNotExist(err) {
				t.Fatalf("write attempted after read failure: %v", err)
			}
		})
	}
}
