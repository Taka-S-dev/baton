package tui

import (
	"fmt"
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
	if !strings.HasPrefix(first, "1. second  $ echo resolved") || strings.Contains(first, "in sub") {
		t.Fatal(first)
	}
	if !strings.Contains(second, `2. first   $ echo "a  b" {@file}`) {
		t.Fatal(second)
	}
	if ansi.StringWidth(m.queueCommandLabel(0, 32)) > 32 {
		t.Fatal("overflow")
	}
}

func TestQueueCommandColumnsAlign(t *testing.T) {
	m := Model{}
	for i := 0; i < 12; i++ {
		name := fmt.Sprintf("c%d", i)
		if i == 3 {
			name = "日本語の長いコマンド名"
		}
		c := mdl.Command{Name: name, Cmd: "echo hi"}
		m.msItems = append(m.msItems, msItem{cmd: &c})
		m.msSelected = append(m.msSelected, i)
	}
	for _, w := range []int{24, 40, 80} {
		column := -1
		for i := range m.msSelected {
			row := ansi.Strip(m.queueCommandLabel(i, w))
			pos := strings.Index(row, "$ ")
			if pos < 0 {
				t.Fatal("command column missing")
			}
			col := ansi.StringWidth(row[:pos])
			if column < 0 {
				column = col
			}
			if col != column || ansi.StringWidth(row) > w {
				t.Fatal("misaligned or overflowing", row)
			}
		}
	}
}
