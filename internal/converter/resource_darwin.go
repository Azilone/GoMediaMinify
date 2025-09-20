//go:build darwin

package converter

import (
	"errors"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

func cpuUsagePercent() (float64, error) {
	output, err := exec.Command("sysctl", "-n", "vm.loadavg").Output()
	if err != nil {
		return 0, err
	}

	cleaned := strings.TrimSpace(string(output))
	cleaned = strings.TrimPrefix(cleaned, "{")
	cleaned = strings.TrimSuffix(cleaned, "}")
	fields := strings.Fields(cleaned)
	if len(fields) == 0 {
		return 0, errors.New("unexpected load average format")
	}

	load, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, err
	}

	if load < 0 {
		load = 0
	}

	percent := load / float64(runtime.NumCPU()) * 100
	return percent, nil
}

func memoryAvailablePercent() (float64, error) {
	pageSize := 4096
	vmOutput, err := exec.Command("vm_stat").Output()
	if err != nil {
		return 0, err
	}

	lines := strings.Split(string(vmOutput), "\n")
	var freePages, inactivePages, speculativePages uint64

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "page size of") {
			// Example: "Mach Virtual Memory Statistics: (page size of 16384 bytes)"
			fields := strings.Fields(line)
			for i := 0; i < len(fields); i++ {
				if fields[i] == "of" && i+1 < len(fields) {
					size, parseErr := strconv.Atoi(fields[i+1])
					if parseErr == nil {
						pageSize = size
					}
					break
				}
			}
			continue
		}

		if strings.HasPrefix(line, "Pages free:") {
			if value, parseErr := parseVMStatValue(line); parseErr == nil {
				freePages = value
			}
		}
		if strings.HasPrefix(line, "Pages inactive:") {
			if value, parseErr := parseVMStatValue(line); parseErr == nil {
				inactivePages = value
			}
		}
		if strings.HasPrefix(line, "Pages speculative:") {
			if value, parseErr := parseVMStatValue(line); parseErr == nil {
				speculativePages = value
			}
		}
	}

	totalBytesOutput, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
	if err != nil {
		return 0, err
	}
	totalBytes, err := strconv.ParseUint(strings.TrimSpace(string(totalBytesOutput)), 10, 64)
	if err != nil {
		return 0, err
	}

	availableBytes := (freePages + inactivePages + speculativePages) * uint64(pageSize)
	if totalBytes == 0 {
		return 0, errors.New("total memory reported as zero")
	}

	percent := float64(availableBytes) / float64(totalBytes) * 100
	if percent < 0 {
		percent = 0
	}

	return percent, nil
}

func parseVMStatValue(line string) (uint64, error) {
	parts := strings.Split(line, ":")
	if len(parts) != 2 {
		return 0, errors.New("invalid vm_stat line")
	}

	numeric := strings.TrimSpace(parts[1])
	numeric = strings.TrimSuffix(numeric, ".")
	numeric = strings.ReplaceAll(numeric, ".", "")

	value, err := strconv.ParseUint(numeric, 10, 64)
	if err != nil {
		return 0, err
	}

	return value, nil
}
