package converter

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/kevindurb/media-converter/internal/api"
	"github.com/kevindurb/media-converter/internal/utils"
)

func (c *Converter) convertFile(inputPath, fileType string) error {
	switch fileType {
	case "photo":
		return c.convertImage(inputPath)
	case "video":
		return c.convertVideo(inputPath)
	default:
		return fmt.Errorf("unknown file type: %s", fileType)
	}
}

func (c *Converter) convertImage(inputPath string) error {
	filename := filepath.Base(inputPath)
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	startTime := time.Now()

	// Get file size early for JSON event
	fileInfo, _ := os.Stat(inputPath)
	fileSize := int64(0)
	if fileInfo != nil {
		fileSize = fileInfo.Size()
	}

	// Emit file_start event in JSON mode
	if c.logger.IsJSONMode() {
		c.logger.GetJSONWriter().EmitFileStart(api.NewFileStartEvent(
			inputPath,
			filename,
			fileSize,
			"image",
			"convert",
		))
	}

	// Track success/failure for file_end event
	var convertSuccess bool
	var outputPath string
	var outputSize int64
	var compressionRatio float64
	var errorMsg string

	// Ensure file_end event is always emitted
	defer func() {
		if c.logger.IsJSONMode() {
			duration := time.Since(startTime)
			c.logger.GetJSONWriter().EmitFileEnd(api.NewFileEndEvent(
				inputPath,
				filename,
				convertSuccess,
				outputPath,
				outputSize,
				compressionRatio,
				duration,
				errorMsg,
				"",
			))
		}
	}()

	// CRITICAL: Extract date from original file BEFORE conversion
	// This prevents using the conversion timestamp instead of the original photo date
	fileDate, err := utils.GetFileDate(inputPath)
	if err != nil {
		errorMsg = fmt.Sprintf("unable to determine file date: %v", err)
		c.logger.Warn(fmt.Sprintf("Could not extract date from %s: %v - skipping file", filename, err))
		return fmt.Errorf("unable to determine file date: %w", err)
	}

	// Determine destination path
	destPath := utils.CreateDestinationPath(c.config.DestDir, fileDate, "image", c.config.OrganizeByDate, c.config.Language)
	if err := utils.EnsureDir(destPath); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Generate base filename and check if already converted
	baseName := utils.CleanFilename(name, c.config.PhotoFormat, fileDate, 1)
	baseOutputPath := filepath.Join(destPath, baseName)

	// Check if file already exists and is valid (idempotency check with integrity verification)
	if existingInfo, err := os.Stat(baseOutputPath); err == nil {
		// File exists, but verify it's not corrupted
		if !c.security.IsFileCorrupted(baseOutputPath, "photo") {
			c.logger.Info(fmt.Sprintf("📷 %s -> %s (already exists and valid, skipping)", filename, baseName))
			c.stats.mu.Lock()
			c.stats.skippedFiles++
			c.stats.mu.Unlock()

			// Mark as successful skip for JSON mode
			convertSuccess = true
			outputPath = baseOutputPath
			outputSize = existingInfo.Size()
			if fileSize > 0 {
				compressionRatio = 1.0 - float64(outputSize)/float64(fileSize)
			}
			return nil
		} else {
			// File is corrupted, remove it and proceed with conversion
			c.logger.Warn(fmt.Sprintf("📷 %s -> %s (corrupted file detected, re-converting)", filename, baseName))
			os.Remove(baseOutputPath)
			c.stats.mu.Lock()
			c.stats.recoveredFiles++
			c.stats.mu.Unlock()
		}
	}

	// Use the base name for conversion
	cleanName := baseName
	outputPath = baseOutputPath
	tempPath := outputPath + ".tmp"

	// Dry run mode
	if c.config.DryRun {
		c.logger.Info(fmt.Sprintf("[DRY-RUN] Would convert: %s → %s", filename, cleanName))
		return nil
	}

	// Get file size for progress tracking (fileInfo already declared above)
	fileSizeMB := float64(fileSize) / (1024 * 1024)

	// Show initial progress
	c.logger.Info(fmt.Sprintf("📷 %s (%.1f MB) -> %s", filename, fileSizeMB, c.config.PhotoFormat))

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

	// Convert to temporary file with timeout
	conversionStartTime := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), c.config.ConversionTimeoutPhoto)
	defer cancel()

	var cmd *exec.Cmd
	quality := c.config.PhotoQualityAVIF
	if c.config.PhotoFormat == "webp" {
		quality = c.config.PhotoQualityWebP
	}

	// Direct conversion for all image formats (including RAW)
	// Preserve EXIF metadata during conversion to maintain original dates
	cmd = exec.CommandContext(ctx, "magick", inputPath,
		"-auto-orient",
		"-quality", fmt.Sprintf("%d", quality),
		"-define", "heic:preserve-orientation=true",
		"-define", "avif:preserve-exif=true", // Preserve EXIF for AVIF
		"-define", "webp:preserve-exif=true", // Preserve EXIF for WebP
		fmt.Sprintf("%s:%s", c.config.PhotoFormat, tempPath))

	// Capture stderr for detailed error information
	var stderrBuf strings.Builder
	cmd.Stderr = &stderrBuf

	if err := cmd.Run(); err != nil {
		stderrOutput := stderrBuf.String()
		if stderrOutput != "" {
			return fmt.Errorf("conversion failed: %w - ImageMagick Error: %s", err, strings.TrimSpace(stderrOutput))
		}
		return fmt.Errorf("conversion failed: %w", err)
	}

	conversionTime := time.Since(conversionStartTime)

	// Verify temporary file integrity
	if err := c.security.VerifyOutputFile(inputPath, tempPath, "photo", c.config.PhotoFormat); err != nil {
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

	// Log successful conversion with enhanced info
	newFileSizeMB := float64(newInfo.Size()) / (1024 * 1024)
	logEntry := fmt.Sprintf("✅ %s -> %s | -%d%% (%.1f->%.1f MB) | %v",
		filename, cleanName, reduction, fileSizeMB, newFileSizeMB, c.formatDuration(conversionTime))
	c.logger.Success(logEntry)

	// Update size statistics
	c.updateSizeStats(fileSizeMB, newFileSizeMB)

	// Mark conversion as successful for JSON mode
	convertSuccess = true
	outputSize = newInfo.Size()
	compressionRatio = float64(reduction) / 100.0

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

func (c *Converter) calculateImageS3Cost(fileSizeMB float64) string {
	if fileSizeMB == 0 {
		return ""
	}

	// AWS S3 Standard storage cost (as of 2024): $0.023 per GB per month
	// Assuming files are stored for 1 year = 12 months
	storageGBPerMonth := fileSizeMB / 1024 * 12 // Convert MB to GB and multiply by 12 months
	storageCost := storageGBPerMonth * 0.023

	// PUT request cost: $0.0005 per 1,000 PUT requests
	putCost := 0.0005 / 1000

	// GET request cost estimate (assume 50 downloads for images): $0.0004 per 1,000 GET requests
	getCost := (50 * 0.0004) / 1000

	totalCost := storageCost + putCost + getCost

	return fmt.Sprintf(" | S3: $%.4f/year", totalCost)
}
