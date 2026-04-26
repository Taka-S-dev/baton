package runner

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/Taka-S-dev/baton/internal/model"
	tea "github.com/charmbracelet/bubbletea"
)

// DoneMsg is sent to the bubbletea program when a command finishes.
type DoneMsg struct {
	Index int
	Err   error
}

// Exec returns a tea.Cmd that runs cmd via tea.ExecProcess (suspends TUI, shows live output).
func Exec(idx int, cmd model.Command, dryRun bool) tea.Cmd {
	if dryRun {
		return func() tea.Msg {
			fmt.Printf("\n  [dry-run] %s\n  $ %s\n", cmd.Name, cmd.Cmd)
			if cmd.Dir != "" {
				fmt.Printf("    dir: %s\n", cmd.Dir)
			}
			fmt.Println()
			return DoneMsg{Index: idx}
		}
	}

	c := buildExec(cmd)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return DoneMsg{Index: idx, Err: err}
	})
}

// BuildExecForTest exposes buildExec for use in tests.
func BuildExecForTest(cmd model.Command) *exec.Cmd { return buildExec(cmd) }

func buildExec(cmd model.Command) *exec.Cmd {
	var c *exec.Cmd
	switch {
	case cmd.Shell == "ps" && runtime.GOOS == "windows":
		c = exec.Command("powershell", "-Command", cmd.Cmd)
	case cmd.Shell == "ps":
		c = exec.Command("pwsh", "-Command", cmd.Cmd)
	case runtime.GOOS == "windows":
		c = exec.Command("cmd", "/C", cmd.Cmd)
	default:
		c = exec.Command("sh", "-c", cmd.Cmd)
	}
	if cmd.Dir != "" {
		c.Dir = cmd.Dir
	}
	return c
}
