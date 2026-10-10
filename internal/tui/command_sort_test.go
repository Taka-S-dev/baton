package tui

import (
	"reflect"
	"testing"

	mdl "github.com/Taka-S-dev/baton/internal/model"
)

func TestCandidateSortPreservesQueueAndCursor(t *testing.T) {
	a := mdl.Command{Name: "zeta", Group: "a"}
	b := mdl.Command{Name: "alpha", Group: "z"}
	m := msModel([]msItem{{cmd: &a}, {cmd: &b}})
	m.projectDir = t.TempDir()
	m.loadCommandSort()
	m.msSelected = []int{0, 1}
	m.msCursor = 0
	if !reflect.DeepEqual(m.msFiltered(), []int{1, 0}) {
		t.Fatal("default not name order")
	}
	m.cycleCommandSort()
	if m.commandSort != "group" || m.msFiltered()[m.msCursor] != 1 {
		t.Fatal("cursor identity changed")
	}
	if !reflect.DeepEqual(m.msSelected, []int{0, 1}) {
		t.Fatal("queue reordered")
	}
	reloaded := Model{projectDir: m.projectDir}
	reloaded.loadCommandSort()
	if reloaded.commandSort != "group" {
		t.Fatal("preference not remembered")
	}
	m.msSearchTI.SetValue("alpha")
	m.cycleCommandSort()
	if len(m.msFiltered()) != 1 || m.msFiltered()[m.msCursor] != 1 {
		t.Fatal("filtered selection changed")
	}
	m.cycleCommandSort()
	m.msSearchTI.SetValue("")
	if !reflect.DeepEqual(m.msFiltered(), []int{0, 1}) {
		t.Fatal("definition order changed")
	}
}

func TestRunSortKeepsUnselectedLastAndQueueUnchanged(t *testing.T) {
	a := mdl.Command{Name: "alpha"}
	b := mdl.Command{Name: "beta"}
	c := mdl.Command{Name: "charlie"}
	m := msModel([]msItem{{cmd: &a}, {cmd: &b}, {cmd: &c}})
	m.msSelected = []int{2, 0}
	m.setCommandSort("run")
	if !reflect.DeepEqual(m.msFiltered(), []int{2, 0, 1}) {
		t.Fatal(m.msFiltered())
	}
	m.setCommandSort("run-desc")
	if !reflect.DeepEqual(m.msFiltered(), []int{0, 2, 1}) {
		t.Fatal(m.msFiltered())
	}
	if !reflect.DeepEqual(m.msSelected, []int{2, 0}) {
		t.Fatal("queue changed")
	}
	m.setCommandSort("name-desc")
	if !reflect.DeepEqual(m.msFiltered(), []int{2, 1, 0}) {
		t.Fatal(m.msFiltered())
	}
}
