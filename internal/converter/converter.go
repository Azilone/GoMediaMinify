package converter

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"

	"github.com/kevindurb/media-converter/internal/config"
	"github.com/kevindurb/media-converter/internal/logger"
	"github.com/kevindurb/media-converter/internal/security"
	"github.com/kevindurb/media-converter/internal/utils"
)

type Converter struct {
	config   *config.Config
	logger   *logger.Logger
	security *security.SecurityChecker
	stats    *ConversionStats
}

type ConversionStats struct {
	mu              sync.Mutex
	totalFiles      int
	processedFiles  int
	failedFiles     int
	skippedFiles    int
	recoveredFiles  int
	cleanedFiles    int
	verifiedFiles   int
	startTime       time.Time
	totalSizeMB     float64
	processedSizeMB float64
	outputSizeMB    float64
	totalS3Cost     float64
	savedSizeMB     float64
}

func NewConverter(cfg *config.Config, log *logger.Logger) *Converter {
	return &Converter{
		config:   cfg,
		logger:   log,
		security: security.NewSecurityChecker(cfg.MinOutputSizeRatio, cfg.MinOutputSizeRatioAVIF, cfg.MinOutputSizeRatioWebP),
		stats: &ConversionStats{
			startTime: time.Now(),
		},
	}
}

func (c *Converter) Convert() error {
	c.logger.Log("Starting secure media conversion")
	c.logger.Info(fmt.Sprintf("Source: %s", c.config.SourceDir))
	c.logger.Info(fmt.Sprintf("Destination: %s", c.config.DestDir))

	if c.config.DryRun {
		c.logger.Info("DRY RUN MODE - No files will be converted")
	}

	c.logger.Info(fmt.Sprintf("Keep originals: %v", c.config.KeepOriginals))
	fmt.Println()

	// Recovery phase: cleanup abandoned files and recover incomplete conversions
	if err := c.performRecovery(); err != nil {
		c.logger.Warn(fmt.Sprintf("Recovery issues detected: %v", err))
	}

	// Check disk space
	if err := c.security.CheckDiskSpace(c.config.SourceDir, c.config.DestDir); err != nil {
		return fmt.Errorf("disk space check failed: %w", err)
	}
	c.logger.Success("Disk space check passed")

	// Run safety test if not in dry-run mode
	if !c.config.DryRun {
		if err := c.runSafetyTest(); err != nil {
			return fmt.Errorf("safety test failed: %w", err)
		}
	}

	// Find files to convert
	photoFiles, videoFiles, err := c.findFiles()
	if err != nil {
		return fmt.Errorf("failed to find files: %w", err)
	}

	c.stats.totalFiles = len(photoFiles) + len(videoFiles)
	c.logger.Info(fmt.Sprintf("📸 Photos found: %d", len(photoFiles)))
	c.logger.Info(fmt.Sprintf("🎬 Videos found: %d", len(videoFiles)))
	c.logger.Info(fmt.Sprintf("📁 Total files: %d", c.stats.totalFiles))
	fmt.Println()

	// Calculate total file size
	c.calculateTotalSize(append(photoFiles, videoFiles...))

	// Convert files
	if len(photoFiles) > 0 {
		c.logger.Log("Converting photos...")
		if err := c.convertFiles(photoFiles, "photo"); err != nil {
			c.logger.Error(fmt.Sprintf("Photo conversion failed: %v", err))
		}
	}

	if len(videoFiles) > 0 {
		fmt.Println()
		c.logger.Log("Converting videos...")
		if err := c.convertFiles(videoFiles, "video"); err != nil {
			c.logger.Error(fmt.Sprintf("Video conversion failed: %v", err))
		}
	}

	// Show final report
	c.showFinalReport()

	return nil
}

func (c *Converter) findFiles() ([]string, []string, error) {
	var photoFiles, videoFiles []string

	err := filepath.Walk(c.config.SourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if utils.HasExtension(path, c.config.PhotoFormats) {
			photoFiles = append(photoFiles, path)
		} else if utils.HasExtension(path, c.config.VideoFormats) {
			videoFiles = append(videoFiles, path)
		}

		return nil
	})

	return photoFiles, videoFiles, err
}

func (c *Converter) convertFiles(files []string, fileType string) error {
	ctx := context.Background()
	sem := semaphore.NewWeighted(int64(c.config.MaxJobs))

	var wg sync.WaitGroup

	for _, file := range files {
		wg.Add(1)

		go func(filePath string) {
			defer wg.Done()

			if err := sem.Acquire(ctx, 1); err != nil {
				c.logger.Error(fmt.Sprintf("Failed to acquire semaphore: %v", err))
				return
			}
			defer sem.Release(1)

			if err := c.convertFile(filePath, fileType); err != nil {
				c.logger.Error(fmt.Sprintf("Failed to convert %s: %v", filepath.Base(filePath), err))
				c.stats.mu.Lock()
				c.stats.failedFiles++
				c.stats.mu.Unlock()
			} else {
				c.stats.mu.Lock()
				c.stats.processedFiles++
				c.stats.mu.Unlock()
				// Show overall progress every 10 files
				if c.stats.processedFiles%10 == 0 {
					c.showOverallProgress()
				}
			}
		}(file)
	}

	wg.Wait()
	return nil
}

func (c *Converter) runSafetyTest() error {
	c.logger.Info("Running safety test...")

	// Find a test file (prefer smaller files like JPG over large RAW files)
	var testFile string
	var preferredFile string

	err := filepath.Walk(c.config.SourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		if utils.HasExtension(path, c.config.PhotoFormats) {
			// Prefer JPG/JPEG files for safety test (smaller and faster)
			if utils.HasExtension(path, []string{"jpg", "jpeg"}) {
				preferredFile = path
				return filepath.SkipDir // Stop after finding preferred file type
			}
			// Keep first photo file as fallback
			if testFile == "" {
				testFile = path
			}
		}

		return nil
	})

	// Use preferred file if found, otherwise use any photo file
	if preferredFile != "" {
		testFile = preferredFile
	}

	if err != nil {
		return err
	}

	if testFile == "" {
		c.logger.Warn("No test file found, skipping safety test")
		return nil
	}

	// Create test directory
	testDir := filepath.Join(c.config.DestDir, ".safety_test")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		return err
	}
	defer os.RemoveAll(testDir)

	// Copy test file
	testCopy := filepath.Join(testDir, filepath.Base(testFile))
	if err := c.copyFile(testFile, testCopy); err != nil {
		return err
	}

	c.logger.Info(fmt.Sprintf("Testing conversion on: %s", filepath.Base(testFile)))

	// Test conversion with temporary settings to avoid polluting destination
	originalKeepSetting := c.config.KeepOriginals
	originalOrganizeByDate := c.config.OrganizeByDate
	originalDestDir := c.config.DestDir

	c.config.KeepOriginals = true   // Force keep originals for test
	c.config.OrganizeByDate = false // Don't organize by date for test
	c.config.DestDir = testDir      // Use test directory

	err = c.convertFile(testCopy, "photo")

	// Restore original settings
	c.config.KeepOriginals = originalKeepSetting
	c.config.OrganizeByDate = originalOrganizeByDate
	c.config.DestDir = originalDestDir

	if err != nil {
		return fmt.Errorf("safety test failed: %w", err)
	}

	c.logger.Success("Safety test passed ✅")
	return nil
}

func (c *Converter) calculateTotalSize(files []string) {
	c.stats.mu.Lock()
	defer c.stats.mu.Unlock()

	for _, file := range files {
		if info, err := os.Stat(file); err == nil {
			c.stats.totalSizeMB += float64(info.Size()) / (1024 * 1024)
		}
	}
}

func (c *Converter) showOverallProgress() {
	c.stats.mu.Lock()
	defer c.stats.mu.Unlock()

	if c.stats.totalFiles == 0 {
		return
	}

	progressPercent := float64(c.stats.processedFiles) / float64(c.stats.totalFiles) * 100
	barWidth := 25
	filledWidth := int(progressPercent / 100 * float64(barWidth))
	bar := strings.Repeat("█", filledWidth) + strings.Repeat("░", barWidth-filledWidth)

	elapsed := time.Since(c.stats.startTime)
	var eta string
	if c.stats.processedFiles > 0 {
		avgTimePerFile := elapsed / time.Duration(c.stats.processedFiles)
		remaining := time.Duration(c.stats.totalFiles-c.stats.processedFiles) * avgTimePerFile
		eta = fmt.Sprintf("ETA: %v", c.formatDuration(remaining))
	} else {
		eta = "ETA: --:--"
	}

	c.logger.Info(fmt.Sprintf("📈 Progress: [%s] %d/%d (%.1f%%) | %s",
		bar, c.stats.processedFiles, c.stats.totalFiles, progressPercent, eta))
}

func (c *Converter) formatDuration(d time.Duration) string {
	if d < 0 {
		return "--:--"
	}

	totalSeconds := int(d.Seconds())
	if totalSeconds < 60 {
		return fmt.Sprintf("%ds", totalSeconds)
	}

	minutes := totalSeconds / 60
	seconds := totalSeconds % 60

	if minutes < 60 {
		return fmt.Sprintf("%dm%ds", minutes, seconds)
	}

	hours := minutes / 60
	minutes = minutes % 60
	return fmt.Sprintf("%dh%dm", hours, minutes)
}

func (c *Converter) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = destFile.ReadFrom(sourceFile)
	return err
}

func (c *Converter) showFinalReport() {
	duration := time.Since(c.stats.startTime)

	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                 Conversion Complete                          ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	c.logger.Success(fmt.Sprintf("✅ Files processed: %d/%d", c.stats.processedFiles, c.stats.totalFiles))

	if c.stats.skippedFiles > 0 {
		c.logger.Info(fmt.Sprintf("⏭️  Files skipped (already exist): %d", c.stats.skippedFiles))
	}

	if c.stats.recoveredFiles > 0 {
		c.logger.Info(fmt.Sprintf("🔄 Files recovered from corruption: %d", c.stats.recoveredFiles))
	}

	if c.stats.cleanedFiles > 0 {
		c.logger.Info(fmt.Sprintf("🧹 Abandoned files cleaned: %d", c.stats.cleanedFiles))
	}

	if c.stats.verifiedFiles > 0 {
		c.logger.Info(fmt.Sprintf("🔍 Files verified for integrity: %d", c.stats.verifiedFiles))
	}

	if c.stats.failedFiles > 0 {
		c.logger.Warn(fmt.Sprintf("⚠️  Failed conversions: %d", c.stats.failedFiles))
	}

	c.logger.Info(fmt.Sprintf("⏱️  Total time: %v", duration.Round(time.Second)))

	// Show comprehensive size and savings statistics
	if c.stats.processedSizeMB > 0 {
		reductionPercent := (c.stats.savedSizeMB / c.stats.processedSizeMB) * 100

		c.logger.Info(fmt.Sprintf("📊 Original size: %.1f MB", c.stats.processedSizeMB))
		c.logger.Info(fmt.Sprintf("📦 Compressed size: %.1f MB", c.stats.outputSizeMB))
		c.logger.Success(fmt.Sprintf("💾 Space saved: %.1f MB (%.1f%% reduction)", c.stats.savedSizeMB, reductionPercent))

		// Calculate S3 storage cost savings
		if c.stats.savedSizeMB > 0 {
			// AWS S3 Standard: $0.023 per GB per month
			monthlySavings := (c.stats.savedSizeMB / 1024) * 0.023
			yearlySavings := monthlySavings * 12
			c.logger.Success(fmt.Sprintf("💰 Estimated S3 savings: $%.2f/month ($%.2f/year)", monthlySavings, yearlySavings))

			// Calculate total storage cost for compressed files
			totalStorageGB := c.stats.outputSizeMB / 1024
			monthlyStorageCost := totalStorageGB * 0.023
			yearlyStorageCost := monthlyStorageCost * 12
			c.logger.Info(fmt.Sprintf("☁️  Total S3 storage cost: $%.2f/month ($%.2f/year)", monthlyStorageCost, yearlyStorageCost))
		}
	}

	fmt.Println()
	c.logger.Info(fmt.Sprintf("📁 Converted files in: %s", c.config.DestDir))
	c.logger.Info(fmt.Sprintf("📄 Detailed logs: %s/conversion.log", c.config.DestDir))

	if c.config.KeepOriginals {
		c.logger.Success("🔒 Original files have been preserved")
	}
}

func (c *Converter) updateSizeStats(inputSizeMB, outputSizeMB float64) {
	c.stats.mu.Lock()
	defer c.stats.mu.Unlock()

	c.stats.processedSizeMB += inputSizeMB
	c.stats.outputSizeMB += outputSizeMB
	c.stats.savedSizeMB += (inputSizeMB - outputSizeMB)
}

// performRecovery handles cleanup and recovery of incomplete conversions
func (c *Converter) performRecovery() error {
	c.logger.Info("🔍 Performing recovery check...")

	// Cleanup abandoned files in destination directory
	if err := c.security.CleanupAbandonedFiles(c.config.DestDir); err != nil {
		return fmt.Errorf("cleanup failed: %w", err)
	}

	// Find and process abandoned markers
	abandonedMarkers, err := c.security.FindAbandonedMarkers(c.config.DestDir)
	if err != nil {
		return fmt.Errorf("failed to find abandoned markers: %w", err)
	}

	if len(abandonedMarkers) > 0 {
		c.logger.Info(fmt.Sprintf("🔄 Found %d abandoned conversion markers", len(abandonedMarkers)))
		for _, marker := range abandonedMarkers {
			if err := os.Remove(marker); err != nil {
				c.logger.Warn(fmt.Sprintf("Failed to remove marker %s: %v", marker, err))
			} else {
				c.stats.mu.Lock()
				c.stats.cleanedFiles++
				c.stats.mu.Unlock()
			}
		}
	}

	// Verify existing converted files and mark corrupted ones for re-conversion
	if err := c.verifyExistingFiles(); err != nil {
		return fmt.Errorf("verification failed: %w", err)
	}

	c.logger.Success("✅ Recovery check completed")
	return nil
}

// verifyExistingFiles checks integrity of existing converted files
func (c *Converter) verifyExistingFiles() error {
	return filepath.Walk(c.config.DestDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Check converted image files
		if strings.HasSuffix(strings.ToLower(path), ".avif") ||
			strings.HasSuffix(strings.ToLower(path), ".webp") {
			if c.security.IsFileCorrupted(path, "photo") {
				c.logger.Warn(fmt.Sprintf("🔍 Corrupted image detected: %s (will be re-converted)", filepath.Base(path)))
				os.Remove(path) // Remove corrupted file
				c.stats.mu.Lock()
				c.stats.recoveredFiles++
				c.stats.mu.Unlock()
			} else {
				c.stats.mu.Lock()
				c.stats.verifiedFiles++
				c.stats.mu.Unlock()
			}
		}

		// Check converted video files (mp4)
		if strings.HasSuffix(strings.ToLower(path), ".mp4") &&
			strings.Contains(strings.ToLower(path), "_") { // Only check converted files (with date prefix)
			if c.security.IsFileCorrupted(path, "video") {
				c.logger.Warn(fmt.Sprintf("🔍 Corrupted video detected: %s (will be re-converted)", filepath.Base(path)))
				os.Remove(path) // Remove corrupted file
				c.stats.mu.Lock()
				c.stats.recoveredFiles++
				c.stats.mu.Unlock()
			} else {
				c.stats.mu.Lock()
				c.stats.verifiedFiles++
				c.stats.mu.Unlock()
			}
		}

		return nil
	})
}
