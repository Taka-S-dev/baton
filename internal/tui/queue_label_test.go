package tui

import (
	"strings"
	"testing"

	mdl "github.com/Taka-S-dev/baton/internal/model"
	"github.com/charmbracelet/x/ansi"
)

func TestQueueLabelShowsCommandInExecutionOrder(t *testing.T) {
	a := mdl.Command{Name: "first", Cmd: `echo "a  b" {@file}`, Dir: "."}
	b := mdl.Command{Name: "second", Cmd: "echo {$value}", Dir: "sub"}
	m := Model{vars: map[string]string{"value": "resolved"}, msItems: []msItem{{cmd: &a}, {cmd: &b}}, msSelected: []int{1, 0}}
	first := stripStyle(m.queueCommandLabel(0, 180))
	second := stripStyle(m.queueCommandLabel(1, 180))
	if !strings.HasPrefix(first, "1. $ echo resolved") || !strings.Contains(first, "In:") {
		t.Fatal(first)
	}
	if !strings.Contains(second, `2. $ echo "a  b" {@file}`) {
		t.Fatal(second)
	}
	if ansi.StringWidth(m.queueCommandLabel(0, 32)) > 32 {
		t.Fatal("overflow")
	}
}
