package utils

import (
	"debug/macho"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// ResolveFFmpegCommand returns the ffmpeg command (with optional prefix) to run.
// On Apple Silicon systems we try to prefer the native arm64 Homebrew binary
// because Rosetta builds are significantly slower for video encoding.
func ResolveFFmpegCommand() ([]string, string) {
	defaultCmd := []string{"ffmpeg"}

	// Special handling for Apple Silicon to pick the fastest available binary.
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		candidates := uniqueExistingPaths(findFFmpegInPath(),
			"/opt/homebrew/bin/ffmpeg",
			"/opt/homebrew/opt/ffmpeg/bin/ffmpeg",
		)

		var fallback string
		for _, candidate := range candidates {
			optimized, err := isAppleSiliconBinary(candidate)
			if err != nil {
				// If we cannot inspect the binary keep it as a fallback.
				if fallback == "" {
					fallback = candidate
				}
				continue
			}
			if optimized {
				message := fmt.Sprintf("Using Apple Silicon optimized ffmpeg at %s", candidate)
				return []string{candidate}, message
			}
			if fallback == "" {
				fallback = candidate
			}
		}

		if fallback != "" {
			message := fmt.Sprintf("Using ffmpeg at %s (Rosetta build detected; install the arm64 Homebrew package for best performance)", fallback)
			return []string{fallback}, message
		}

		return defaultCmd, "ffmpeg not found; install it with Homebrew: brew install ffmpeg"
	}

	if path := findFFmpegInPath(); path != "" {
		message := fmt.Sprintf("Using ffmpeg at %s", path)
		return []string{path}, message
	}

	return defaultCmd, "ffmpeg not found; install it to enable video conversion"
}

// CheckVideoAcceleration verifies whether hardware acceleration is available
// for the configured ffmpeg command.
func CheckVideoAcceleration(ffmpegCmd []string) (bool, string) {
	if runtime.GOOS == "darwin" {
		return checkVideoToolbox(ffmpegCmd)
	}
	return false, "Hardware acceleration not supported on this platform"
}

// checkVideoToolbox verifies that the VideoToolbox H.265 encoder is available.
func checkVideoToolbox(ffmpegCmd []string) (bool, string) {
	output, err := runFFmpegCommand(ffmpegCmd, "-hide_banner", "-encoders")
	if err != nil {
		return false, "FFmpeg not found or error checking encoders"
	}

	outputStr := string(output)
	if strings.Contains(outputStr, "hevc_videotoolbox") {
		return true, "VideoToolbox H.265 encoder available"
	}

	return false, "VideoToolbox H.265 encoder not available"
}

// runFFmpegCommand executes ffmpeg (with optional prefix command) and returns the output.
func runFFmpegCommand(ffmpegCmd []string, args ...string) ([]byte, error) {
	command := "ffmpeg"
	if len(ffmpegCmd) > 0 {
		command = ffmpegCmd[0]
	}

	cmdArgs := []string{}
	if len(ffmpegCmd) > 1 {
		cmdArgs = append(cmdArgs, ffmpegCmd[1:]...)
	}
	cmdArgs = append(cmdArgs, args...)

	return exec.Command(command, cmdArgs...).Output()
}

// findFFmpegInPath returns the first ffmpeg binary found in PATH.
func findFFmpegInPath() string {
	path, err := exec.LookPath("ffmpeg")
	if err != nil {
		return ""
	}
	return path
}

// uniqueExistingPaths filters the paths that actually exist and removes duplicates while preserving order.
func uniqueExistingPaths(paths ...string) []string {
	seen := make(map[string]struct{})
	var result []string
	for _, p := range paths {
		if p == "" {
			continue
		}
		if _, already := seen[p]; already {
			continue
		}
		if _, err := os.Stat(p); err != nil {
			continue
		}
		seen[p] = struct{}{}
		result = append(result, p)
	}
	return result
}

// isAppleSiliconBinary inspects the Mach-O headers to determine whether the binary contains an arm64 slice.
func isAppleSiliconBinary(path string) (bool, error) {
	fat, err := macho.OpenFat(path)
	if err == nil {
		defer fat.Close()
		for _, arch := range fat.Arches {
			if arch.Cpu == macho.CpuArm64 {
				return true, nil
			}
		}
		return false, nil
	}

	if !errors.Is(err, macho.ErrNotFat) {
		return false, err
	}

	file, err := macho.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()

	return file.Cpu == macho.CpuArm64, nil
}
