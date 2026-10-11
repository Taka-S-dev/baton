package tui

import (
	"fmt"
	"strings"
	"testing"

	mdl "github.com/Taka-S-dev/baton/internal/model"
	"github.com/charmbracelet/x/ansi"
)

func TestCompactCandidatesKeepSelectionAndHitRows(t *testing.T) {
	for _, h := range []int{5, 9, 17} {
		for _, w := range []int{32, 80} {
			m := Model{height: h, width: w, msCursor: 12, msSelected: []int{12}}
			for i := 0; i < 20; i++ {
				c := mdl.Command{Name: fmt.Sprintf("command-%02d", i), Cmd: "echo ok"}
				m.msItems = append(m.msItems, msItem{cmd: &c})
			}
			hits := map[int]int{}
			view := stripStyle(m.viewCompactCandidates(w, func(i, y int) { hits[i] = y }))
			lines := strings.Split(view, "\n")
			if len(lines) != h || !strings.Contains(lines[0], "1 selected") {
				t.Fatal(view)
			}
			if !strings.Contains(lines[hits[12]], "▶ [1] command-12") {
				t.Fatal(view)
			}
			if strings.Contains(view, "Enter Review") || strings.Contains(view, "Selected(") {
				t.Fatal(view)
			}
			for _, line := range lines {
				if ansi.StringWidth(line) > w {
					t.Fatal(view)
				}
			}
		}
	}
}
