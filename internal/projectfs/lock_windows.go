package projectfs

import "syscall"

// alive reports whether a process with this pid still runs. A process
// whose handle cannot be opened for lack of rights exists, so it counts
// as alive; a handle whose exit code is set belongs to a finished one.
func alive(pid int) bool {
	const queryLimitedInformation = 0x1000
	const stillActive = 259
	h, err := syscall.OpenProcess(queryLimitedInformation, false, uint32(pid))
	if err != nil {
		return err == syscall.ERROR_ACCESS_DENIED
	}
	defer syscall.CloseHandle(h)
	var code uint32
	if err := syscall.GetExitCodeProcess(h, &code); err != nil {
		return true
	}
	return code == stillActive
}
