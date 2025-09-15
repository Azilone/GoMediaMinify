//go:build !windows

package security

import "syscall"

func processExists(pid int) bool {
	if pid <= 0 {
		return false
	}

	err := syscall.Kill(pid, 0)
	if err == nil {
		return true
	}

	// ESRCH means the process does not exist. EPERM indicates it exists but we lack permission.
	return err != syscall.ESRCH
}
