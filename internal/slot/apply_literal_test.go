package slot

import (
	"testing"

	"github.com/Taka-S-dev/baton/internal/model"
)

func TestApplyKeepsInsertedPlaceholdersLiteral(t *testing.T) {
	cmd := model.Command{Cmd: "echo {a} {b} {a...} {unknown} {empty}", Dir: "/{a}/{b}"}
	values := map[string]string{"a": "{b}", "b": "B", "empty": ""}
	for i := 0; i < 1000; i++ {
		got := Apply(cmd, values)
		if got.Cmd != "echo {b} B {b} {unknown} " || got.Dir != "/{b}/B" {
			t.Fatalf("unexpected recursive expansion: %+v", got)
		}
	}
	if cmd.Dir != "/{a}/{b}" {
		t.Fatal("mutated input")
	}
}
