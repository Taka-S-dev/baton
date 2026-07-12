// baton is a terminal-based command/workflow runner.
//
// Usage:
//
//	baton [--dry-run]
//
// Flags:
//
//	--dry-run  Print what would be executed without running any commands.
//
// Environment variables:
//
//	BATON_PROJECTS_DIR   Path to the projects/ directory. When unset, baton
//	                     looks next to the executable, then in
//	                     ~/.config/baton/projects/.
//	BATON_DISPLAY_NAME   Overrides the app name shown in the TUI logo
//	                     (default "BATON"). Useful when running the tool
//	                     under a neutral name. ASCII recommended.
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Taka-S-dev/baton/internal/tui"
)

const usage = `baton - terminal-based command/workflow runner

Usage:
  baton [--dry-run]

Flags:
  --dry-run    Print what would be executed without running any commands.
  -h, --help   Show this help.

Environment variables:
  BATON_PROJECTS_DIR   Path to the projects/ directory (default: next to
                       the executable, then ~/.config/baton/projects/).
  BATON_DISPLAY_NAME   App name shown in the TUI logo (default "BATON").
`

func main() {
	dryRun := false
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--dry-run":
			dryRun = true
		case "-h", "--help":
			fmt.Print(usage)
			return
		}
	}

	m, err := tui.New(dryRun)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
