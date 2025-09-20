//go:build linux

package converter

import (
	"bufio"
	"errors"
	"os"
	"runtime"
	"strconv"
	"strings"
)

func cpuUsagePercent() (float64, error) {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return 0, err
	}

	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return 0, errors.New("unexpected /proc/loadavg format")
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
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var totalKB, availableKB float64

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			if value, parseErr := parseMeminfoValue(line); parseErr == nil {
				totalKB = value
			}
		}
		if strings.HasPrefix(line, "MemAvailable:") {
			if value, parseErr := parseMeminfoValue(line); parseErr == nil {
				availableKB = value
			}
		}
		if totalKB > 0 && availableKB > 0 {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, err
	}

	if totalKB <= 0 {
		return 0, errors.New("failed to read MemTotal")
	}

	percent := (availableKB / totalKB) * 100
	if percent < 0 {
		percent = 0
	}

	return percent, nil
}

func parseMeminfoValue(line string) (float64, error) {
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return 0, errors.New("unexpected meminfo line")
	}

	value, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return 0, err
	}

	return value, nil
}
