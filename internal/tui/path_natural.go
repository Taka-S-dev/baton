package tui

import "strings"

// naturalPathLess orders paths with digit runs compared by magnitude,
// without integer overflow.
func naturalPathLess(a, b string) bool {
	originalA, originalB := a, b
	a, b = strings.ToLower(a), strings.ToLower(b)
	for len(a) > 0 && len(b) > 0 {
		if a[0] >= '0' && a[0] <= '9' && b[0] >= '0' && b[0] <= '9' {
			i, j := 0, 0
			for i < len(a) && a[i] >= '0' && a[i] <= '9' {
				i++
			}
			for j < len(b) && b[j] >= '0' && b[j] <= '9' {
				j++
			}
			x, y := strings.TrimLeft(a[:i], "0"), strings.TrimLeft(b[:j], "0")
			if len(x) != len(y) {
				return len(x) < len(y)
			}
			if x != y {
				return x < y
			}
			if i != j {
				return i < j
			}
			a, b = a[i:], b[j:]
		} else {
			if a[0] != b[0] {
				return a[0] < b[0]
			}
			a, b = a[1:], b[1:]
		}
	}
	if len(a) != len(b) {
		return len(a) < len(b)
	}
	return originalA < originalB
}
