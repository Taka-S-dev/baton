package slot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyVars(t *testing.T) {
	vars := map[string]string{"root": `C:\work\Phase2`}

	cases := []struct {
		in, want string
	}{
		{`{$root}\api`, `C:\work\Phase2\api`},
		{`build {$root} twice {$root}`, `build C:\work\Phase2 twice C:\work\Phase2`},
		{`{root}`, `{root}`},         // interactive slots are never touched
		{`{$missing}`, `{$missing}`}, // undefined refs stay literal
		{``, ``},
	}
	for _, c := range cases {
		if got := ApplyVars(c.in, vars); got != c.want {
			t.Errorf("ApplyVars(%q) = %q, want %q", c.in, got, c.want)
		}
	}

	// Single pass: a value containing {$other} must not expand recursively.
	rec := map[string]string{"a": "{$b}", "b": "x"}
	if got := ApplyVars("{$a}", rec); got != "{$b}" {
		t.Errorf("recursive expansion: got %q, want literal {$b}", got)
	}
}

func TestUndefinedVars(t *testing.T) {
	vars := map[string]string{"root": "x"}
	got := UndefinedVars(`{$root} {$phase} {env} {$phase}`, vars)
	if len(got) != 2 || got[0] != "phase" || got[1] != "phase" {
		t.Errorf("UndefinedVars = %v, want [phase phase]", got)
	}
}

func TestLoadVars(t *testing.T) {
	dir := t.TempDir()
	tsv := "command\tname\tvalue\n" +
		"*\troot\tC:\\work\\Phase2\n" +
		"*\troot\tduplicate\n" +
		"as\tworkdir\t./src\n" +
		"broken-line\n"
	if err := os.WriteFile(filepath.Join(dir, "vars.tsv"), []byte(tsv), 0o644); err != nil {
		t.Fatal(err)
	}

	vars, warnings := LoadVars(dir)
	if vars["root"] != `C:\work\Phase2` || vars["as.workdir"] != "./src" {
		t.Fatalf("vars = %v", vars)
	}
	joined := strings.Join(warnings, "; ")
	if !strings.Contains(joined, "duplicate entry root") {
		t.Errorf("warnings must report the duplicate, got %q", joined)
	}
	if !strings.Contains(joined, "missing value") {
		t.Errorf("warnings must report the broken line, got %q", joined)
	}
}

func TestLoadVars_LegacyTwoColumnRows(t *testing.T) {
	dir := t.TempDir()
	tsv := "name\tvalue\nroot\tC:\\x\nas.workdir\t./src\n"
	if err := os.WriteFile(filepath.Join(dir, "vars.tsv"), []byte(tsv), 0o644); err != nil {
		t.Fatal(err)
	}

	vars, warnings := LoadVars(dir)
	if vars["root"] != `C:\x` || vars["as.workdir"] != "./src" || len(warnings) != 0 {
		t.Fatalf("legacy rows must load: vars=%v warnings=%v", vars, warnings)
	}
}

func TestLoadVars_Missing(t *testing.T) {
	vars, warnings := LoadVars(t.TempDir())
	if len(vars) != 0 || len(warnings) != 0 {
		t.Fatalf("no file must mean no vars and no warnings, got %v / %v", vars, warnings)
	}
}

func TestSaveVars_Roundtrip(t *testing.T) {
	dir := t.TempDir()
	in := map[string]string{"root": `C:\x`, "as.workdir": `{$root}\src`}
	if err := SaveVars(dir, in); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "vars.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	want := "command\tname\tvalue\n*\troot\tC:\\x\nas\tworkdir\t{$root}\\src\n"
	if string(raw) != want {
		t.Fatalf("file = %q, want %q", raw, want)
	}
	out, warnings := LoadVars(dir)
	if len(warnings) != 0 {
		t.Fatalf("roundtrip warnings: %v", warnings)
	}
	if out["root"] != `C:\x` || out["as.workdir"] != `{$root}\src` || len(out) != 2 {
		t.Fatalf("roundtrip vars = %v", out)
	}

	// An empty map must not create a file in a fresh project...
	empty := t.TempDir()
	if err := SaveVars(empty, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(empty, "vars.tsv")); err == nil {
		t.Fatal("empty vars must not create vars.tsv")
	}
	// ...but must empty an existing one.
	if err := SaveVars(dir, nil); err != nil {
		t.Fatal(err)
	}
	if out, _ := LoadVars(dir); len(out) != 0 {
		t.Fatalf("existing file must be emptied, got %v", out)
	}
}

func TestCommandValues_SetGetPrune(t *testing.T) {
	vars := map[string]string{"root": `C:\x`}

	SetCommandValues(vars, "as", map[string]string{"workdir": "./src", "env": "dev"})
	SetCommandValues(vars, "v1.2-build", map[string]string{"workdir": "./v12"})

	if got := CommandValues(vars, "as"); got["workdir"] != "./src" || got["env"] != "dev" || len(got) != 2 {
		t.Fatalf("CommandValues(as) = %v", got)
	}
	// Dotted command names split at the LAST dot.
	if got := CommandValues(vars, "v1.2-build"); got["workdir"] != "./v12" || len(got) != 1 {
		t.Fatalf("CommandValues(v1.2-build) = %v", got)
	}
	// "v1" must not capture "v1.2-build.workdir" (rest contains a dot).
	if got := CommandValues(vars, "v1"); got != nil {
		t.Fatalf("CommandValues(v1) = %v, want nil", got)
	}

	// Replacing values drops stale slots.
	SetCommandValues(vars, "as", map[string]string{"workdir": "./out"})
	if got := CommandValues(vars, "as"); got["workdir"] != "./out" || len(got) != 1 {
		t.Fatalf("after replace: %v", got)
	}

	// Pruning drops entries for deleted commands, keeps globals.
	PruneCommandValues(vars, func(name string) bool { return name == "as" })
	if _, ok := vars["v1.2-build.workdir"]; ok {
		t.Fatal("pruning must drop entries for deleted commands")
	}
	if vars["root"] != `C:\x` || CommandValues(vars, "as") == nil {
		t.Fatalf("pruning must keep globals and live commands, got %v", vars)
	}
}
