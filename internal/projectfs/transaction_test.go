package projectfs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRecoveryRollbackAndCompletedCommit(t *testing.T) {
	for _, completed := range []bool{false, true} {
		dir := t.TempDir()
		before := Files{"commands.tsv": []byte("old"), "vars.tsv": []byte("vars")}
		after := Files{"commands.tsv": []byte("new"), "workflows.json": []byte("[]")}
		for name, b := range before {
			if err := Write(dir, name, b); err != nil {
				t.Fatal(err)
			}
		}
		b, _ := json.Marshal(journal{before, after})
		if err := os.WriteFile(filepath.Join(dir, Journal), b, 0600); err != nil {
			t.Fatal(err)
		}
		if err := Write(dir, "commands.tsv", after["commands.tsv"]); err != nil {
			t.Fatal(err)
		}
		if completed {
			if err := restore(dir, before, after); err != nil {
				t.Fatal(err)
			}
		}
		if err := Recover(dir); err != nil {
			t.Fatal(err)
		}
		got, _ := Read(dir)
		want := before
		if completed {
			want = after
		}
		if Revision(got) != Revision(want) || Pending(dir) {
			t.Fatal("incorrect recovery")
		}
	}
}
func TestCommitConflictAndRecoveryConflict(t *testing.T) {
	dir := t.TempDir()
	before := Files{"commands.tsv": []byte("old")}
	after := Files{"commands.tsv": []byte("new")}
	if err := Write(dir, "commands.tsv", []byte("external")); err != nil {
		t.Fatal(err)
	}
	if err := Commit(dir, before, after); err == nil {
		t.Fatal("conflict accepted")
	}
	b, _ := json.Marshal(journal{before, after})
	_ = os.WriteFile(filepath.Join(dir, Journal), b, 0600)
	if err := Recover(dir); err == nil {
		t.Fatal("recovery overwrote external change")
	}
	b, _ = os.ReadFile(filepath.Join(dir, "commands.tsv"))
	if string(b) != "external" {
		t.Fatal("lost external edit")
	}
}

func TestRevisionIgnoresBookkeepingFiles(t *testing.T) {
	a := Files{"commands.tsv": []byte("x"), ".last_workflow": []byte("ship"), ".command_names": []byte("a")}
	b := Files{"commands.tsv": []byte("x"), ".last_workflow": []byte("build"), ".command_names": []byte("b")}
	c := Files{"commands.tsv": []byte("y"), ".last_workflow": []byte("ship")}
	if Revision(a) != Revision(b) {
		t.Fatal("bookkeeping files changed the revision")
	}
	if Revision(a) == Revision(c) {
		t.Fatal("a definition change kept the revision")
	}
}
