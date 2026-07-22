// baton is a terminal-based command/workflow runner.
//
// Usage:
//
//	baton [--dry-run]
//	baton check [project|path]
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
	"runtime/debug"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Taka-S-dev/baton/internal/tui"
)

// version is stamped by the release build via
// -ldflags "-X main.version=v1.2.3". Plain `go build` falls back to the
// VCS-derived module version when Go recorded one.
var version = ""

func versionString() string {
	if version != "" {
		return version
	}
	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		return bi.Main.Version
	}
	return "dev"
}

const usage = `baton - terminal-based command/workflow runner

Usage:
  baton [--dry-run]
  baton check [project|path]

Commands:
  check        Validate projects without starting the TUI and print any
               warnings. With no argument every project is checked; pass a
               project name or a directory path to check one. Exits 1 when
               anything is wrong — usable from scripts, CI, and AI agents.

Flags:
  --dry-run    Print what would be executed without running any commands.
  --version    Print the version and exit.
  -h, --help   Show this help.

Environment variables:
  BATON_PROJECTS_DIR   Path to the projects/ directory (default: next to
                       the executable, then ~/.config/baton/projects/).
  BATON_DISPLAY_NAME   App name shown in the TUI logo (default "BATON").
`

func main() {
	if len(os.Args) > 1 && os.Args[1] == "check" {
		os.Exit(runCheck(os.Args[2:]))
	}

	dryRun := false
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--dry-run":
			dryRun = true
		case "-h", "--help":
			fmt.Print(usage)
			return
		case "--version":
			fmt.Println("baton " + versionString())
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
