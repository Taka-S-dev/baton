package runner

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"time"

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
			// The step header already echoes the name, command, and workdir.
			fmt.Printf("   [dry-run] skipped\n")
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

// RunCaptured uses the same shell selection as the TUI, without an interactive stdin.
func RunCaptured(cmd model.Command, timeout time.Duration, stdout, stderr io.Writer) (int, error) {
	base := buildExec(cmd)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	c := exec.CommandContext(ctx, base.Path, base.Args[1:]...)
	c.Dir, c.Stdout, c.Stderr = base.Dir, stdout, stderr
	c.WaitDelay = time.Second
	err := c.Run()
	if err == nil {
		return 0, nil
	}
	if ctx.Err() != nil {
		return -1, ctx.Err()
	}
	if c.ProcessState != nil {
		return c.ProcessState.ExitCode(), err
	}
	return -1, err
}
