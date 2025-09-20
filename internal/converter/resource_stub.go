//go:build !darwin && !linux

package converter

import "errors"

func cpuUsagePercent() (float64, error) {
	return 0, errors.New("cpu usage sampling not supported on this platform")
}

func memoryAvailablePercent() (float64, error) {
	return 0, errors.New("memory sampling not supported on this platform")
}
