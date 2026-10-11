package tui

import (
	"strings"
	"testing"

	mdl "github.com/Taka-S-dev/baton/internal/model"
)

func TestCompactConfirmKeepsButtonsFixed(t *testing.T) {
	for _, h := range []int{7, 10, 17} {
		for _, mixed := range []bool{false, true} {
			for _, n := range []int{1, 3, 20} {
				m := Model{height: h, width: 100}
				for i := 0; i < n; i++ {
					dir := "."
					if mixed && i%2 == 1 {
						dir = "sub"
					}
					c := mdl.Command{Cmd: "echo ok", Dir: dir}
					m.confirmRunItems = append(m.confirmRunItems, mdl.RunItem{Name: "test", Cmd: &c})
				}
				lines := strings.Split(stripStyle(m.viewConfirmRun(100)), "\n")
				if len(lines) != h || !strings.Contains(lines[h-2], "Run") || !strings.Contains(lines[h-1], "Enter Select") {
					t.Fatal(strings.Join(lines, "\n"))
				}
				if n > m.confirmRunPerPage() && !strings.Contains(lines[h-3], "Scroll") {
					t.Fatal("missing scroll hint")
				}
			}
		}
	}
}
