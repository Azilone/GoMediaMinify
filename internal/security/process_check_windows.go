//go:build windows

package security

import "syscall"

func processExists(pid int) bool {
	if pid <= 0 {
		return false
	}

	h, err := syscall.OpenProcess(syscall.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer syscall.CloseHandle(h)
	return true
}
