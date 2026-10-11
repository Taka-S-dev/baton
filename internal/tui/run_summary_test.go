package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	mdl "github.com/Taka-S-dev/baton/internal/model"
	"github.com/Taka-S-dev/baton/internal/runner"
	tea "github.com/charmbracelet/bubbletea"
)

func TestLauncherSuccessQuitsAfterSummary(t *testing.T) {
	c := mdl.Command{Name: "build", Cmd: "echo ok"}
	for _, launcher := range []bool{false, true} {
		m := Model{exitAfterRun: launcher, screen: ScreenRunning, running: &runningState{items: []mdl.RunItem{{Name: c.Name, Cmd: &c}}, runStart: time.Now(), stepStart: time.Now()}}
		nm, cmd := m.handleRunnerDone(runner.DoneMsg{})
		m = nm.(Model)
		if cmd == nil || !m.running.completed {
			t.Fatal("completion summary missing")
		}
		line := stripStyle(m.runCompletionLine())
		if !strings.Contains(line, "✓ Completed") || strings.Contains(line, "1/1") {
			t.Fatal(line)
		}
		if strings.Contains(m.viewRunning(80), "Done") || strings.Contains(m.viewRunning(80), "Completed") {
			t.Fatal("duplicate completion in view")
		}
		_, finish := m.Update(runSummaryPrintedMsg{})
		if launcher {
			if finish == nil {
				t.Fatal("launcher did not quit")
			}
			if _, ok := finish().(tea.QuitMsg); !ok {
				t.Fatal("wrong finish action")
			}
		} else if finish != nil {
			t.Fatal("normal mode quit")
		}
	}
}

func TestLauncherFailureKeepsRetry(t *testing.T) {
	m := Model{exitAfterRun: true, screen: ScreenRunning, running: &runningState{}}
	nm, cmd := m.handleRunnerDone(runner.DoneMsg{Err: errors.New("failed")})
	if nm.(Model).screen != ScreenRetry || cmd != nil {
		t.Fatal("failure must offer retry")
	}
}

func TestSingleStepHeaderHasNoProgressFraction(t *testing.T) {
	got := stepHeader(1, 1, "build", mdl.Command{Cmd: "go build ."})
	if strings.Contains(got, "1/1") || !strings.Contains(got, "── build") {
		t.Fatal(got)
	}
}

func TestCompletionOutputLeavesFullProgressBar(t *testing.T) {
	for _, count := range []int{1, 2, 5} {
		m := Model{running: &runningState{items: make([]mdl.RunItem, count), current: count}}
		got := stripStyle(m.runCompletionOutput())
		if strings.Count(got, "✓ Completed") != 1 {
			t.Fatal(got)
		}
		if count == 1 {
			if strings.Contains(got, "█") || strings.Contains(got, "100%") {
				t.Fatal(got)
			}
			continue
		}
		lines := strings.Split(got, "\n")
		if len(lines) != 2 || !strings.Contains(lines[0], strings.Repeat("█", 24)) || !strings.Contains(lines[0], "100%") || strings.Contains(lines[0], "─") {
			t.Fatal(got)
		}
		if !strings.Contains(lines[1], "✓ Completed") || strings.Contains(lines[1], "steps") {
			t.Fatal(got)
		}
	}
}

func TestPartialProgressLabelsCompletedSteps(t *testing.T) {
	got := stripStyle(progressBar(1, 2, 24))
	if !strings.Contains(got, strings.Repeat("█", 12)+strings.Repeat("─", 12)) || !strings.Contains(got, "1/2 steps") || !strings.Contains(got, "50%") {
		t.Fatal(got)
	}
}
