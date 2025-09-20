package security

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/kevindurb/media-converter/internal/utils"
)

type SecurityChecker struct {
	minOutputSizeRatio     float64
	minOutputSizeRatioAVIF float64
	minOutputSizeRatioWebP float64
}

func NewSecurityChecker(minOutputSizeRatio, minOutputSizeRatioAVIF, minOutputSizeRatioWebP float64) *SecurityChecker {
	return &SecurityChecker{
		minOutputSizeRatio:     minOutputSizeRatio,
		minOutputSizeRatioAVIF: minOutputSizeRatioAVIF,
		minOutputSizeRatioWebP: minOutputSizeRatioWebP,
	}
}

func (s *SecurityChecker) CheckDiskSpace(sourceDir, destDir string) error {
	sourceSize, err := getDirSize(sourceDir)
	if err != nil {
		return fmt.Errorf("failed to get source directory size: %w", err)
	}

	destAvailable, err := getAvailableSpace(destDir)
	if err != nil {
		return fmt.Errorf("failed to get available space: %w", err)
	}

	// Estimate needed space (50% of original for safety)
	estimatedNeeded := sourceSize / 2

	if destAvailable < estimatedNeeded {
		return fmt.Errorf("insufficient disk space! Available: %s, Estimated needed: %s",
			formatBytes(destAvailable), formatBytes(estimatedNeeded))
	}

	return nil
}

func (s *SecurityChecker) VerifyOutputFile(inputPath, outputPath, fileType, outputFormat string) error {
	// Check if file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		return fmt.Errorf("output file does not exist: %s", outputPath)
	}

	// Check if file is not empty
	outputInfo, err := os.Stat(outputPath)
	if err != nil {
		return fmt.Errorf("failed to stat output file: %w", err)
	}

	if outputInfo.Size() == 0 {
		os.Remove(outputPath) // Clean up empty file
		return fmt.Errorf("output file is empty: %s", outputPath)
	}

	// Check minimum size ratio
	inputInfo, err := os.Stat(inputPath)
	if err != nil {
		return fmt.Errorf("failed to stat input file: %w", err)
	}

	// Use format-specific ratio if available
	var ratio float64
	if fileType == "photo" {
		switch strings.ToLower(outputFormat) {
		case "avif":
			ratio = s.minOutputSizeRatioAVIF
		case "webp":
			ratio = s.minOutputSizeRatioWebP
		default:
			ratio = s.minOutputSizeRatio
		}
	} else {
		ratio = s.minOutputSizeRatio
	}

	minSize := int64(float64(inputInfo.Size()) * ratio)
	if outputInfo.Size() < minSize {
		os.Remove(outputPath) // Clean up potentially corrupted file
		return fmt.Errorf("output file too small (%d < %d bytes, ratio %.3f for %s): %s",
			outputInfo.Size(), minSize, ratio, outputFormat, outputPath)
	}

	// Verify file integrity based on type
	switch fileType {
	case "photo":
		return s.verifyImageIntegrity(outputPath)
	case "video":
		return s.verifyVideoIntegrity(outputPath)
	default:
		return fmt.Errorf("unknown file type: %s", fileType)
	}
}

func (s *SecurityChecker) verifyImageIntegrity(imagePath string) error {
	cmd := exec.Command("magick", "identify", imagePath)
	if err := cmd.Run(); err != nil {
		os.Remove(imagePath) // Clean up corrupted file
		return fmt.Errorf("image is corrupted: %s", imagePath)
	}
	return nil
}

func (s *SecurityChecker) verifyVideoIntegrity(videoPath string) error {
	cmd := exec.Command("ffprobe", videoPath)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		os.Remove(videoPath) // Clean up corrupted file
		return fmt.Errorf("video is corrupted: %s", videoPath)
	}
	return nil
}

func (s *SecurityChecker) SafeDelete(filePath, outputPath string) error {
	// Triple verification before deletion
	outputInfo, err := os.Stat(outputPath)
	if err != nil {
		return fmt.Errorf("cannot verify output file before deletion: %w", err)
	}

	// Ensure output file exists, is not empty, and has reasonable size
	if outputInfo.Size() < 1000 {
		return fmt.Errorf("deletion cancelled for safety: output file too small")
	}

	// Perform the deletion
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete original file: %w", err)
	}

	return nil
}

func getDirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			if utils.IsPermissionError(walkErr) {
				return nil
			}
			return walkErr
		}

		if info.IsDir() {
			if utils.ShouldSkipSystemEntry(info.Name(), true) {
				return filepath.SkipDir
			}
			return nil
		}

		if utils.ShouldSkipSystemEntry(info.Name(), false) {
			return nil
		}

		size += info.Size()
		return nil
	})
	return size, err
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// IsFileCorrupted checks if an existing file is corrupted or incomplete
func (s *SecurityChecker) IsFileCorrupted(filePath, fileType string) bool {
	// Check if file exists
	info, err := os.Stat(filePath)
	if err != nil {
		return true // File doesn't exist or can't be accessed
	}

	// Check if file is empty
	if info.Size() == 0 {
		return true
	}

	// Verify file integrity based on type
	switch fileType {
	case "photo":
		return s.verifyImageIntegrity(filePath) != nil
	case "video":
		return s.verifyVideoIntegrity(filePath) != nil
	default:
		return true // Unknown file type
	}
}

// CreateProcessingMarker creates a marker file to indicate conversion in progress
func (s *SecurityChecker) CreateProcessingMarker(filePath string) error {
	markerPath := filePath + ".processing"

	// Create marker with PID and timestamp
	content := fmt.Sprintf("PID:%d\nStarted:%s\nFile:%s\n",
		os.Getpid(),
		time.Now().Format(time.RFC3339),
		filePath)

	return os.WriteFile(markerPath, []byte(content), 0644)
}

// RemoveProcessingMarker removes the processing marker file
func (s *SecurityChecker) RemoveProcessingMarker(filePath string) error {
	markerPath := filePath + ".processing"
	if err := os.Remove(markerPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove processing marker: %w", err)
	}
	return nil
}

// FindAbandonedMarkers finds processing markers from previous runs
func (s *SecurityChecker) FindAbandonedMarkers(dir string) ([]string, error) {
	var abandoned []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(path, ".processing") {
			// Check if the process is still running
			if s.isMarkerAbandoned(path) {
				abandoned = append(abandoned, path)
			}
		}

		return nil
	})

	return abandoned, err
}

// isMarkerAbandoned checks if a processing marker is from a dead process
func (s *SecurityChecker) isMarkerAbandoned(markerPath string) bool {
	content, err := os.ReadFile(markerPath)
	if err != nil {
		return true
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "PID:") {
			pidStr := strings.TrimPrefix(line, "PID:")
			pid, err := strconv.Atoi(pidStr)
			if err != nil {
				return true
			}

			if !processExists(pid) {
				return true
			}

			return false // Process is still running
		}
	}

	return true // No PID found or invalid format
}

// CleanupAbandonedFiles removes temporary and abandoned files
func (s *SecurityChecker) CleanupAbandonedFiles(dir string) error {
	var errors []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			// Remove .tmp files
			if strings.HasSuffix(path, ".tmp") {
				if err := os.Remove(path); err != nil {
					errors = append(errors, fmt.Sprintf("failed to remove %s: %v", path, err))
				}
			}

			// Remove abandoned .processing markers
			if strings.HasSuffix(path, ".processing") && s.isMarkerAbandoned(path) {
				if err := os.Remove(path); err != nil {
					errors = append(errors, fmt.Sprintf("failed to remove %s: %v", path, err))
				}
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	if len(errors) > 0 {
		return fmt.Errorf("cleanup errors: %s", strings.Join(errors, "; "))
	}

	return nil
}

// VerifyFileIntegrity performs comprehensive integrity check
func (s *SecurityChecker) VerifyFileIntegrity(filePath, fileType string) error {
	// Check if file exists and is not empty
	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("file doesn't exist: %w", err)
	}

	if info.Size() == 0 {
		return fmt.Errorf("file is empty")
	}

	// Verify file can be opened and read
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("cannot open file: %w", err)
	}
	defer file.Close()

	// Read a small chunk to verify file accessibility
	buffer := make([]byte, 1024)
	_, err = file.Read(buffer)
	if err != nil && err.Error() != "EOF" {
		return fmt.Errorf("cannot read file: %w", err)
	}

	// Type-specific integrity checks
	switch fileType {
	case "photo":
		return s.verifyImageIntegrity(filePath)
	case "video":
		return s.verifyVideoIntegrity(filePath)
	default:
		return fmt.Errorf("unknown file type: %s", fileType)
	}
}
