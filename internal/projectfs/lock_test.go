package projectfs

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestLockIsExclusiveWhileHeld(t *testing.T) {
	dir := t.TempDir()
	release, err := Lock(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Lock(dir); err == nil || !strings.Contains(err.Error(), "busy") {
		t.Fatalf("second lock while held: %v", err)
	}
	release()
	release, err = Lock(dir)
	if err != nil {
		t.Fatalf("lock after release: %v", err)
	}
	release()
}

func TestLockReplacesTheLockOfAFinishedProcess(t *testing.T) {
	// The test binary itself, run with no tests selected, is a process
	// that has certainly exited by the time its pid is written.
	cmd := exec.Command(os.Args[0], "-test.run=^$")
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, ".baton-lock")
	if err := os.WriteFile(path, []byte("pid="+strconv.Itoa(cmd.Process.Pid)+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	release, err := Lock(dir)
	if err != nil {
		t.Fatalf("lock left by a finished process: %v", err)
	}
	defer release()
	if pid, ok := lockOwner(path); !ok || pid != os.Getpid() {
		t.Fatalf("lock owner after takeover: %d %v", pid, ok)
	}
	if _, err := os.Stat(path + ".stale"); !os.IsNotExist(err) {
		t.Fatalf("stale copy left behind: %v", err)
	}
}

func TestLockKeepsAnUnreadableLock(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".baton-lock")
	if err := os.WriteFile(path, []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Lock(dir); err == nil || !strings.Contains(err.Error(), "busy") {
		t.Fatalf("lock without a pid: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("unreadable lock removed: %v", err)
	}
}
