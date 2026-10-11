package slot

import (
	"strings"
	"testing"

	"github.com/Taka-S-dev/baton/internal/model"
)

func TestReservedPathSlotsStaySeparate(t *testing.T) {
	c := model.Command{Cmd: "echo {file} {$root} {@file:.go,.mod} {@file...:.go} {@dir}", Dir: "{project}"}
	slots := GetSlots(c)
	if len(slots) != 5 || slots[0].Name != "project" || slots[1].Name != "file" {
		t.Fatalf("workdir resolution order or legacy name changed: %+v", slots)
	}
	values := map[string]string{"project": "src", "file": "legacy"}
	for _, d := range slots {
		if d.PathKind != "" {
			if strings.Contains(d.Name, ".") {
				t.Fatal("scoped key contains dots")
			}
			values[d.Name] = "chosen"
		}
	}
	got := Apply(c, values)
	if got.Cmd != "echo legacy {$root} chosen chosen chosen" || got.Dir != "src" {
		t.Fatalf("bad expansion: %+v", got)
	}
	if !HasPlaceholders(model.Command{Cmd: "echo {@file}"}) {
		t.Fatal("path slot not a template")
	}
}

func TestPathSlotValuesRoundTrip(t *testing.T) {
	d := PathDefinition("{@file...:.go,.mod}")
	if !d.Variadic || d.PathKind != "file" || len(d.Extensions) != 2 || d.Placeholder() != "{@file...:.go,.mod}" {
		t.Fatalf("bad definition: %+v", d)
	}
	vars := map[string]string{}
	SetCommandValues(vars, "build.v1", map[string]string{d.Name: "test.go go.mod"})
	dir := t.TempDir()
	if err := SaveVars(dir, vars); err != nil {
		t.Fatal(err)
	}
	read, warnings := LoadVars(dir)
	if len(warnings) > 0 {
		t.Fatal(warnings)
	}
	values := CommandValues(read, "build.v1")
	got := Apply(model.Command{Cmd: "go build {@file...:.go,.mod}"}, values)
	if got.Cmd != "go build test.go go.mod" {
		t.Fatalf("saved values detached: %s", got.Cmd)
	}
}

func TestPathSyntaxValidation(t *testing.T) {
	for _, ref := range []string{"{@file}", "{@dir}", "{@file...:.go,.mod}", "{@file:.tar.gz}"} {
		if got := PathErrors(ref); len(got) > 0 {
			t.Fatal(got)
		}
	}
	for _, ref := range []string{"{@files}", "{@file:go}", "{@file:.go,}", "{@dir:.go}", "{@file:.go .mod}"} {
		if got := PathErrors(ref); len(got) == 0 {
			t.Fatalf("invalid syntax accepted: %s", ref)
		}
	}
	if len(PathErrors("{file} {$file} %PATH%")) != 0 {
		t.Fatal("legacy syntax flagged")
	}
}
