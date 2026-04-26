package runner_test

import (
	"runtime"
	"strings"
	"testing"

	"github.com/Taka-S-dev/baton/internal/model"
	"github.com/Taka-S-dev/baton/internal/runner"
)

// ── Shell selection (buildExec is tested via Exec dry-run) ────────────────────

func TestExec_DryRun_ReturnsCmd(t *testing.T) {
	cmd := model.Command{Name: "build", Cmd: "echo hello", Dir: "/tmp"}
	fn := runner.Exec(0, cmd, true)
	if fn == nil {
		t.Fatal("want non-nil tea.Cmd")
	}
	msg := fn()
	done, ok := msg.(runner.DoneMsg)
	if !ok {
		t.Fatalf("want DoneMsg, got %T", msg)
	}
	if done.Index != 0 {
		t.Errorf("want index=0, got %d", done.Index)
	}
	if done.Err != nil {
		t.Errorf("want no error, got %v", done.Err)
	}
}

func TestExec_DryRun_NoDir(t *testing.T) {
	cmd := model.Command{Name: "test", Cmd: "go test ./..."}
	fn := runner.Exec(2, cmd, true)
	msg := fn()
	done, ok := msg.(runner.DoneMsg)
	if !ok {
		t.Fatalf("want DoneMsg, got %T", msg)
	}
	if done.Index != 2 {
		t.Errorf("want index=2, got %d", done.Index)
	}
}

// ── Shell selection ───────────────────────────────────────────────────────────

func TestBuildExec_DefaultShell(t *testing.T) {
	c := runner.BuildExecForTest(model.Command{Cmd: "echo hello"})
	if runtime.GOOS == "windows" {
		if c.Path == "" || !strings.HasSuffix(strings.ToLower(c.Path), "cmd.exe") {
			t.Errorf("want cmd.exe on windows, got %q", c.Path)
		}
	} else {
		if !strings.HasSuffix(c.Path, "sh") {
			t.Errorf("want sh on unix, got %q", c.Path)
		}
	}
}

func TestBuildExec_PowerShellShell(t *testing.T) {
	c := runner.BuildExecForTest(model.Command{Cmd: "Write-Host hi", Shell: "ps"})
	if runtime.GOOS == "windows" {
		if !strings.Contains(strings.ToLower(c.Path), "powershell") {
			t.Errorf("want powershell on windows, got %q", c.Path)
		}
	} else {
		if !strings.Contains(c.Path, "pwsh") {
			t.Errorf("want pwsh on unix, got %q", c.Path)
		}
	}
}

func TestBuildExec_DirSet(t *testing.T) {
	c := runner.BuildExecForTest(model.Command{Cmd: "ls", Dir: "/tmp"})
	if c.Dir != "/tmp" {
		t.Errorf("want dir=/tmp, got %q", c.Dir)
	}
}

func TestBuildExec_NoDirEmpty(t *testing.T) {
	c := runner.BuildExecForTest(model.Command{Cmd: "ls"})
	if c.Dir != "" {
		t.Errorf("want empty dir, got %q", c.Dir)
	}
}
