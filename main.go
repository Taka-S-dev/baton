// baton is a terminal-based command/workflow runner. Run baton --help
// for its commands, flags and environment variables.
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
  baton recent [--dry-run]
  baton r [--dry-run]
  baton check [project|path]

Commands:
  recent, r    Choose a recent command for the current directory, then
               fill its inputs and review before running.
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

	opts, err := parseLaunchArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(2)
	}
	if opts.help {
		fmt.Print(usage)
		return
	}
	if opts.version {
		fmt.Println("baton " + versionString())
		return
	}
	var m tui.Model
	if opts.recent {
		m, err = tui.NewRecent(opts.dryRun)
	} else {
		m, err = tui.New(opts.dryRun)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	// The overview is pointable: a tree is the one shape where reaching
	// for the mouse is the obvious move. Cell motion rather than all
	// motion — clicks and the wheel, no event per pixel of travel.
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
