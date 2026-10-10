package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"strings"
)

// Match rune ranges, preserving the original spelling and display width.
func searchMatchMask(text string, terms []string) []bool {
	runes := []rune(text)
	mask := make([]bool, len(runes))
	for _, term := range terms {
		n := len([]rune(term))
		if n == 0 {
			continue
		}
		for i := 0; i+n <= len(runes); i++ {
			if strings.EqualFold(string(runes[i:i+n]), term) {
				for j := i; j < i+n; j++ {
					mask[j] = true
				}
			}
		}
	}
	return mask
}
func searchHighlight(line string, terms []string, selected, focused bool) string {
	text := ansi.Strip(line)
	runes := []rune(text)
	mask := searchMatchMask(text, terms)
	any := false
	for _, hit := range mask {
		any = any || hit
	}
	if !any {
		return line
	}
	base := sWhite
	if selected {
		base = sRowInactive
		if focused {
			base = sRowActive
		}
	}
	match := base.Foreground(lipgloss.Color("220")).Bold(true)
	var b strings.Builder
	for start := 0; start < len(runes); {
		end := start + 1
		for end < len(runes) && mask[end] == mask[start] {
			end++
		}
		style := base
		if mask[start] {
			style = match
		}
		b.WriteString(style.Render(string(runes[start:end])))
		start = end
	}
	return b.String()
}
