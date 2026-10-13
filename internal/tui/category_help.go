package tui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

func categoryHelpContent(tab, w int) string {
	what, how := categoryHelp(tab)
	wrap := func(text string) string {
		return " " + strings.ReplaceAll(ansi.Wrap(text, max(1, w-2), ""), "\n", "\n ")
	}
	return gray(wrap(what)) + "\n\n" + dim(wrap("none yet · "+how))
}
