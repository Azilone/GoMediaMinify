package converter

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/kevindurb/media-converter/internal/utils"
)

func (c *Converter) convertVideo(inputPath string) error {
	filename := filepath.Base(inputPath)
	name := strings.TrimSuffix(filename, filepath.Ext(filename))

	fileDate, err := utils.GetFileDate(inputPath)
	if err != nil {
		c.logger.Warn(fmt.Sprintf("Could not extract date from %s: %v - skipping file", filename, err))
		return fmt.Errorf("unable to determine file date: %w", err)
	}

	// Determine destination path
	destPath := utils.CreateDestinationPath(c.config.DestDir, fileDate, "video", c.config.OrganizeByDate)
	if err := utils.EnsureDir(destPath); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Generate base filename and check if already converted (always use mp4 for output)
	baseName := utils.CleanFilename(name, "mp4", fileDate, 1)
	baseOutputPath := filepath.Join(destPath, baseName)

	// Check if file already exists and is valid (idempotency check with integrity verification)
	if _, err := os.Stat(baseOutputPath); err == nil {
		// File exists, but verify it's not corrupted
		if !c.security.IsFileCorrupted(baseOutputPath, "video") {
			c.logger.Info(fmt.Sprintf("📹 %s -> %s (already exists and valid, skipping)", filename, baseName))
			c.stats.mu.Lock()
			c.stats.skippedFiles++
			c.stats.mu.Unlock()
			return nil
		} else {
			// File is corrupted, remove it and proceed with conversion
			c.logger.Warn(fmt.Sprintf("📹 %s -> %s (corrupted file detected, re-converting)", filename, baseName))
			os.Remove(baseOutputPath)
			c.stats.mu.Lock()
			c.stats.recoveredFiles++
			c.stats.mu.Unlock()
		}
	}

	// Use the base name for conversion
	cleanName := baseName
	outputPath := baseOutputPath
	tempPath := outputPath + ".tmp"

	// Dry run mode
	if c.config.DryRun {
		c.logger.Info(fmt.Sprintf("[DRY-RUN] Would convert: %s → %s", filename, cleanName))
		return nil
	}

	// Create processing marker
	if err := c.security.CreateProcessingMarker(outputPath); err != nil {
		c.logger.Warn(fmt.Sprintf("Failed to create processing marker: %v", err))
	}

	// Ensure cleanup of marker and temp file on any exit
	defer func() {
		c.security.RemoveProcessingMarker(outputPath)
		if _, err := os.Stat(tempPath); err == nil {
			os.Remove(tempPath) // Clean up temp file if conversion fails
		}
	}()

	// Determine video codec
	var videoCodec string
	switch c.config.VideoCodec {
	case "h265":
		videoCodec = "libx265"
	case "av1":
		videoCodec = "libaom-av1"
	default:
		videoCodec = "libx264"
	}

	// Convert to temporary file with timeout
	ctx, cancel := context.WithTimeout(context.Background(), c.config.ConversionTimeoutVideo)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffmpeg", "-i", inputPath,
		"-c:v", videoCodec,
		"-crf", fmt.Sprintf("%d", c.config.VideoCRF),
		"-preset", "medium",
		"-c:a", "aac", "-b:a", "128k",
		"-movflags", "+faststart",
		"-map_metadata", "0",
		"-f", "mp4", // Force MP4 format
		"-progress", "pipe:2", // Enable progress reporting
		"-y", tempPath) // Convert to temporary file

	// Start the command and monitor progress
	if err := c.runVideoConversionWithProgress(cmd, inputPath, filename); err != nil {
		return fmt.Errorf("conversion failed: %w", err)
	}

	// Verify temporary file integrity
	if err := c.security.VerifyOutputFile(inputPath, tempPath, "video", "mp4"); err != nil {
		return fmt.Errorf("output verification failed: %w", err)
	}

	// Atomic move: rename temp file to final destination
	if err := os.Rename(tempPath, outputPath); err != nil {
		return fmt.Errorf("failed to finalize conversion: %w", err)
	}

	// Calculate file size reduction
	originalInfo, _ := os.Stat(inputPath)
	newInfo, _ := os.Stat(outputPath)
	var reduction int
	if originalInfo.Size() > 0 {
		reduction = int((originalInfo.Size() - newInfo.Size()) * 100 / originalInfo.Size())
	}

	// Calculate file sizes for stats
	originalSizeMB := float64(originalInfo.Size()) / (1024 * 1024)
	newSizeMB := float64(newInfo.Size()) / (1024 * 1024)

	// Log successful conversion
	logEntry := fmt.Sprintf("✅ %s → %s | -%d%% (%.1f->%.1f MB)", filename, cleanName, reduction, originalSizeMB, newSizeMB)
	c.logger.Success(logEntry)

	// Update size statistics
	c.updateSizeStats(originalSizeMB, newSizeMB)

	// Safe deletion if requested
	if !c.config.KeepOriginals {
		if err := c.security.SafeDelete(inputPath, outputPath); err != nil {
			c.logger.Warn(fmt.Sprintf("Deletion cancelled for safety: %s (%v)", filename, err))
		} else {
			c.logger.Security(fmt.Sprintf("Safe deletion: %s", filename))
		}
	}

	return nil
}

func (c *Converter) runVideoConversionWithProgress(cmd *exec.Cmd, inputPath, filename string) error {
	// Create pipes for stderr (where ffmpeg sends progress)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	// Monitor progress in a goroutine and capture stderr output
	done := make(chan error, 1)
	var stderrOutput strings.Builder
	go func() {
		defer stderr.Close()
		c.monitorVideoProgress(stderr, inputPath, filename, &stderrOutput)
		done <- cmd.Wait()
	}()

	// Wait for completion
	err = <-done
	if err != nil {
		// Include stderr output in error message
		stderrText := stderrOutput.String()
		if stderrText != "" {
			return fmt.Errorf("%w - FFmpeg Error: %s", err, strings.TrimSpace(stderrText))
		}
		return err
	}
	return nil
}

type VideoProgress struct {
	filename      string
	startTime     time.Time
	totalDuration time.Duration
	currentTime   time.Duration
	speed         float64
	fileSizeMB    float64
	lastUpdate    time.Time
	progressShown bool
}

func (c *Converter) monitorVideoProgress(reader io.Reader, inputPath, filename string, stderrBuffer *strings.Builder) {
	scanner := bufio.NewScanner(reader)

	// Get file size for cost calculation
	var fileSizeMB float64
	if info, err := os.Stat(inputPath); err == nil && info != nil {
		fileSizeMB = float64(info.Size()) / (1024 * 1024)
	} else {
		fileSizeMB = 0 // Default to 0 if we can't get file size
	}

	progress := &VideoProgress{
		filename:      filename,
		startTime:     time.Now(),
		totalDuration: 0, // Will be extracted from ffmpeg output
		fileSizeMB:    fileSizeMB,
		lastUpdate:    time.Now(),
	}

	// Regex to parse ffmpeg progress output
	timeRegex := regexp.MustCompile(`time=(\d{2}):(\d{2}):(\d{2}\.\d{2})`)
	speedRegex := regexp.MustCompile(`speed=\s*([0-9.]+)x`)
	durationRegex := regexp.MustCompile(`Duration: (\d{2}):(\d{2}):(\d{2}\.\d{2})`)

	for scanner.Scan() {
		line := scanner.Text()

		// Capture all stderr output for error reporting
		stderrBuffer.WriteString(line + "\n")

		// Parse total duration from ffmpeg output (appears early in the output)
		if progress.totalDuration == 0 {
			if durationMatch := durationRegex.FindStringSubmatch(line); len(durationMatch) >= 4 {
				hours, _ := strconv.Atoi(durationMatch[1])
				minutes, _ := strconv.Atoi(durationMatch[2])
				seconds, _ := strconv.ParseFloat(durationMatch[3], 64)
				progress.totalDuration = time.Duration(float64(hours*3600+minutes*60)+seconds) * time.Second
			}
		}

		// Parse current time
		if timeMatch := timeRegex.FindStringSubmatch(line); len(timeMatch) >= 4 {
			hours, _ := strconv.Atoi(timeMatch[1])
			minutes, _ := strconv.Atoi(timeMatch[2])
			seconds, _ := strconv.ParseFloat(timeMatch[3], 64)
			progress.currentTime = time.Duration(float64(hours*3600+minutes*60)+seconds) * time.Second
		}

		// Parse speed
		if speedMatch := speedRegex.FindStringSubmatch(line); len(speedMatch) >= 2 {
			progress.speed, _ = strconv.ParseFloat(speedMatch[1], 64)
		}

		// Show progress update every 30 seconds (reduced frequency for parallel processing)
		if time.Since(progress.lastUpdate) > 30*time.Second {
			c.showVideoProgress(progress)
			progress.lastUpdate = time.Now()
		}

		// Check for completion
		if strings.Contains(line, "progress=end") {
			c.showVideoProgress(progress) // Final progress update
			c.showVideoCompletion(progress)
			return
		}
	}
}

// getVideoInfo function removed - duration is now extracted from ffmpeg output stream
// This eliminates the preliminary ffprobe call for better performance

func (c *Converter) showVideoProgress(progress *VideoProgress) {
	if progress.totalDuration == 0 {
		return
	}

	// Calculate progress percentage
	progressPercent := float64(progress.currentTime) / float64(progress.totalDuration) * 100
	if progressPercent > 100 {
		progressPercent = 100
	}

	// Create progress bar
	barWidth := 30
	filledWidth := int(progressPercent / 100 * float64(barWidth))
	bar := strings.Repeat("█", filledWidth) + strings.Repeat("░", barWidth-filledWidth)

	// Calculate ETA
	eta := "--:--"
	if progress.speed > 0 && progressPercent > 5 {
		elapsed := time.Since(progress.startTime)
		totalEstimated := time.Duration(float64(elapsed) / (progressPercent / 100))
		remaining := totalEstimated - elapsed
		if remaining > 0 {
			eta = c.formatDuration(remaining)
		}
	}

	// Show compact progress line with clear file identification
	fileName := filepath.Base(progress.filename)
	if !progress.progressShown {
		c.logger.Info(fmt.Sprintf("📹 %s (%.1f MB) - converting...", fileName, progress.fileSizeMB))
		progress.progressShown = true
	}

	progressLine := fmt.Sprintf("   %s: [%s] %.1f%% (%.1fx, ETA: %s)",
		fileName, bar, progressPercent, progress.speed, eta)

	c.logger.Info(progressLine)
}

func (c *Converter) showVideoCompletion(progress *VideoProgress) {
	duration := time.Since(progress.startTime)

	c.logger.Success(fmt.Sprintf("✅ %s completed in %s",
		filepath.Base(progress.filename), c.formatDuration(duration)))
}

func (c *Converter) calculateS3Cost(fileSizeMB float64, progressPercent float64) string {
	if fileSizeMB == 0 {
		return ""
	}

	// AWS S3 Standard storage cost (as of 2024): $0.023 per GB per month
	// Assuming files are stored for 1 year = 12 months
	storageGBPerMonth := fileSizeMB / 1024 * 12 // Convert MB to GB and multiply by 12 months
	storageCost := storageGBPerMonth * 0.023

	// PUT request cost: $0.0005 per 1,000 PUT requests
	putCost := 0.0005 / 1000

	// GET request cost estimate (assume 100 downloads): $0.0004 per 1,000 GET requests
	getCost := (100 * 0.0004) / 1000

	totalCost := storageCost + putCost + getCost

	if progressPercent < 100 {
		return fmt.Sprintf(" | Est. S3 cost: $%.4f/year", totalCost)
	}
	return fmt.Sprintf(" | S3 cost: $%.4f/year", totalCost)
}
