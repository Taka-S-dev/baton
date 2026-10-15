//go:build !windows

package projectfs

import (
	"errors"
	"os"
	"syscall"
)

// alive reports whether a process with this pid still runs. A process
// that exists but belongs to another user answers EPERM, which counts
// as alive.
func alive(pid int) bool {
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = p.Signal(syscall.Signal(0))
	return err == nil || errors.Is(err, syscall.EPERM)
}
