package tui

import (
	"strings"

	mdl "github.com/Taka-S-dev/baton/internal/model"
	"github.com/charmbracelet/x/ansi"
)

// commandMatchReason explains only matches absent from the visible label,
// on the same row so filtering never changes checkbox or cursor geometry.
func commandMatchReason(c *mdl.Command, label string, terms []string, budget int) string {
	visible := strings.ToLower(ansi.Strip(label))
	for _, term := range terms {
		term = strings.ToLower(term)
		if strings.Contains(visible, term) {
			continue
		}
		fields := [][2]string{{"cmd", c.Cmd}}
		for _, key := range sortedKeys(c.Values) {
			fields = append(fields, [2]string{"fixed " + key, c.Values[key]})
		}
		for _, field := range fields {
			if !strings.Contains(strings.ToLower(field[1]), term) {
				continue
			}
			text := []rune(field[1])
			n := len([]rune(term))
			start := 0
			for i := 0; i+n <= len(text); i++ {
				if strings.EqualFold(string(text[i:i+n]), term) {
					start = max(0, i-5)
					break
				}
			}
			prefix := field[0] + ": "
			if start > 0 {
				prefix += "…"
			}
			return prefix + truncate(string(text[start:]), max(1, budget-len(prefix)))
		}
	}
	return ""
}
