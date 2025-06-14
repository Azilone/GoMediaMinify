//go:build windows

package security

import (
	"syscall"
	"unsafe"
)

func getAvailableSpace(path string) (int64, error) {
	h := syscall.MustLoadDLL("kernel32.dll")
	c := h.MustFindProc("GetDiskFreeSpaceExW")

	var freeBytesAvailable int64
	var totalNumberOfBytes int64
	var totalNumberOfFreeBytes int64

	pathPtr, _ := syscall.UTF16PtrFromString(path)

	r1, _, e1 := c.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		uintptr(unsafe.Pointer(&freeBytesAvailable)),
		uintptr(unsafe.Pointer(&totalNumberOfBytes)),
		uintptr(unsafe.Pointer(&totalNumberOfFreeBytes)))

	if r1 == 0 {
		return 0, e1
	}

	return freeBytesAvailable, nil
}
