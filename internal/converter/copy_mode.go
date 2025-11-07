package converter

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/kevindurb/media-converter/internal/api"
	"github.com/kevindurb/media-converter/internal/checksum"
	"github.com/kevindurb/media-converter/internal/utils"
)

type copyAction string

const (
	copyActionCopied          copyAction = "copied"
	copyActionDuplicate       copyAction = "duplicate"
	copyActionDuplicateQueued copyAction = "duplicate-session"
	copyActionDryRun          copyAction = "dry-run"
)

type copyResult struct {
	action           copyAction
	destPath         string
	duplicateOf      string
	checksum         uint64
	checksumDuration time.Duration
	sourceSize       int64
}

type copyStats struct {
	mu               sync.Mutex
	copied           int
	duplicates       int
	dryRunCopies     int
	failed           int
	totalBytes       int64
	copiedBytes      int64
	checksumDuration time.Duration
}

func (s *copyStats) addResult(res copyResult, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err != nil {
		s.failed++
		return
	}

	s.totalBytes += res.sourceSize
	s.checksumDuration += res.checksumDuration

	switch res.action {
	case copyActionCopied:
		s.copied++
		s.copiedBytes += res.sourceSize
	case copyActionDryRun:
		s.dryRunCopies++
	case copyActionDuplicate, copyActionDuplicateQueued:
		s.duplicates++
	}
}

func (c *Converter) runCopyMode() error {
	c.logger.Log("🗂️  Starting copy-only archive mode")
	c.logger.Info(fmt.Sprintf("Source: %s", c.config.SourceDir))
	c.logger.Info(fmt.Sprintf("Destination: %s", c.config.DestDir))

	if c.config.DryRun {
		c.logger.Info("DRY RUN MODE - Files will not be copied")
	}

	c.stats = &ConversionStats{
		startTime: time.Now(),
	}
	c.tracker = checksum.NewTracker()

	if c.config.VerifyChecksum {
		indexed, err := c.loadExistingChecksums()
		if err != nil {
			return fmt.Errorf("failed to index destination: %w", err)
		}
		if indexed > 0 {
			c.logger.Info(fmt.Sprintf("Indexed %d existing files for duplicate detection", indexed))
		}
	}

	if err := c.security.CheckDiskSpace(c.config.SourceDir, c.config.DestDir); err != nil {
		return fmt.Errorf("disk space check failed: %w", err)
	}
	c.logger.Success("Disk space check passed")

	photoFiles, videoFiles, err := c.findFiles()
	if err != nil {
		return fmt.Errorf("failed to find files: %w", err)
	}

	files := append(photoFiles, videoFiles...)
	c.stats.totalFiles = len(files)

	c.logger.Info(fmt.Sprintf("📸 Photos queued: %d", len(photoFiles)))
	c.logger.Info(fmt.Sprintf("🎬 Videos queued: %d", len(videoFiles)))
	c.logger.Info(fmt.Sprintf("📁 Total files: %d", len(files)))

	// Emit started event in JSON mode
	if c.logger.IsJSONMode() {
		c.logger.GetJSONWriter().EmitStarted(api.NewStartedEvent(
			c.config.SourceDir,
			c.config.DestDir,
			len(files),
			"copy-only",
			c.config.DryRun,
			c.config.KeepOriginals,
			c.config.OrganizeByDate,
		))
	}

	if len(files) == 0 {
		c.logger.Warn("No media files found to copy")
		return nil
	}

	copyStats := &copyStats{}
	maxJobs := c.config.MaxJobs
	if maxJobs < 1 {
		maxJobs = 1
	}

	jobs := make(chan string)
	var wg sync.WaitGroup

	worker := func() {
		defer wg.Done()
		for path := range jobs {
			res, err := c.copyFile(path)
			copyStats.addResult(res, err)

			if err != nil {
				c.logger.Error(fmt.Sprintf("Failed to copy %s: %v", filepath.Base(path), err))
				continue
			}

			c.stats.mu.Lock()
			c.stats.processedFiles++
			processed := c.stats.processedFiles
			c.stats.mu.Unlock()

			if processed%10 == 0 {
				c.showOverallProgress()
			}
		}
	}

	for i := 0; i < maxJobs; i++ {
		wg.Add(1)
		go worker()
	}

	for _, file := range files {
		jobs <- file
	}
	close(jobs)

	wg.Wait()

	c.showCopySummary(copyStats)
	return nil
}

func (c *Converter) copyFile(srcPath string) (copyResult, error) {
	result := copyResult{}

	info, err := os.Stat(srcPath)
	if err != nil {
		return result, fmt.Errorf("stat source: %w", err)
	}
	result.sourceSize = info.Size()

	mediaType := detectMediaType(srcPath, c.config.PhotoFormats)
	if mediaType == "" {
		mediaType = "video"
	}

	copyStart := time.Now()
	srcChecksum, err := c.hasher.Calculate(srcPath)
	if err != nil {
		return result, fmt.Errorf("checksum source: %w", err)
	}
	result.checksumDuration += time.Since(copyStart)
	result.checksum = srcChecksum

	if c.tracker.IsDuplicate(srcChecksum) {
		if original, ok := c.tracker.FirstPath(srcChecksum); ok {
			result.action = copyActionDuplicateQueued
			result.duplicateOf = original
			c.logger.Info(fmt.Sprintf("⏭️  Skipped duplicate: %s (same as %s, checksum: %s)",
				filepath.Base(srcPath),
				filepath.Base(original),
				checksum.FormatChecksum(srcChecksum)))
			return result, nil
		}
	}

	fileDate, err := utils.GetFileDate(srcPath)
	if err != nil {
		c.logger.Warn(fmt.Sprintf("Failed to extract date for %s, using file modification time", filepath.Base(srcPath)))
		fileDate = info.ModTime()
	}

	destDir := utils.CreateDestinationPath(c.config.DestDir, fileDate, mediaType, c.config.OrganizeByDate, c.config.Language)
	if err := utils.EnsureDir(destDir); err != nil {
		return result, fmt.Errorf("create destination: %w", err)
	}

	destPath := filepath.Join(destDir, filepath.Base(srcPath))
	result.destPath = destPath

	if existingInfo, err := os.Stat(destPath); err == nil && !existingInfo.IsDir() {
		checkStart := time.Now()
		existingChecksum, checksumErr := c.hasher.Calculate(destPath)
		result.checksumDuration += time.Since(checkStart)

		if checksumErr == nil && existingChecksum == srcChecksum {
			c.tracker.Register(srcChecksum, destPath)
			result.action = copyActionDuplicate
			result.duplicateOf = destPath
			c.logger.Info(fmt.Sprintf("⏭️  Skipped duplicate: %s (matches existing %s, checksum: %s)",
				filepath.Base(srcPath),
				filepath.Base(destPath),
				checksum.FormatChecksum(srcChecksum)))
			return result, nil
		}

		uniquePath, uniqueErr := utils.GetUniqueFilename(destDir, filepath.Base(srcPath), filepath.Ext(srcPath))
		if uniqueErr != nil {
			return result, fmt.Errorf("resolve unique filename: %w", uniqueErr)
		}
		destPath = uniquePath
		result.destPath = destPath
	} else if err != nil && !os.IsNotExist(err) {
		return result, fmt.Errorf("inspect destination: %w", err)
	}

	if c.config.DryRun {
		result.action = copyActionDryRun
		c.tracker.Register(srcChecksum, destPath)
		c.logger.Info(fmt.Sprintf("🔎 DRY RUN: would copy %s → %s",
			filepath.Base(srcPath),
			destPath))
		return result, nil
	}

	tmpPath := destPath + ".tmp"
	if err := copyFileData(srcPath, tmpPath); err != nil {
		os.Remove(tmpPath)
		return result, fmt.Errorf("copy data: %w", err)
	}

	verifyStart := time.Now()
	copiedChecksum, err := c.hasher.Calculate(tmpPath)
	result.checksumDuration += time.Since(verifyStart)
	if err != nil {
		os.Remove(tmpPath)
		return result, fmt.Errorf("verify copy: %w", err)
	}

	if copiedChecksum != srcChecksum {
		os.Remove(tmpPath)
		return result, fmt.Errorf("checksum mismatch: source=%s copied=%s",
			checksum.FormatChecksum(srcChecksum),
			checksum.FormatChecksum(copiedChecksum))
	}

	if err := os.Rename(tmpPath, destPath); err != nil {
		os.Remove(tmpPath)
		return result, fmt.Errorf("finalize copy: %w", err)
	}

	result.action = copyActionCopied
	result.destPath = destPath
	c.tracker.Register(srcChecksum, destPath)

	c.logger.Success(fmt.Sprintf("✅ Copied: %s → %s (checksum: %s)",
		filepath.Base(srcPath),
		destPath,
		checksum.FormatChecksum(srcChecksum)))

	return result, nil
}

func copyFileData(src, dst string) error {
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

	buffer := make([]byte, 1024*1024) // 1MB buffer keeps IO efficient
	_, err = io.CopyBuffer(destFile, sourceFile, buffer)
	return err
}

func detectMediaType(path string, photoFormats []string) string {
	if utils.HasExtension(path, photoFormats) {
		return "image"
	}
	return "video"
}

func (c *Converter) loadExistingChecksums() (int, error) {
	if _, err := os.Stat(c.config.DestDir); os.IsNotExist(err) {
		return 0, nil
	} else if err != nil {
		return 0, fmt.Errorf("stat destination: %w", err)
	}

	var indexed int
	err := filepath.Walk(c.config.DestDir, func(path string, info os.FileInfo, walkErr error) error {
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

		if !info.Mode().IsRegular() {
			return nil
		}

		lowerName := strings.ToLower(info.Name())
		if strings.HasSuffix(lowerName, ".tmp") || strings.HasSuffix(lowerName, ".processing") {
			return nil
		}

		if !(utils.HasExtension(path, c.config.PhotoFormats) || utils.HasExtension(path, c.config.VideoFormats)) {
			return nil
		}

		sum, err := c.hasher.Calculate(path)
		if err != nil {
			c.logger.Warn(fmt.Sprintf("Failed to index existing file %s: %v", filepath.Base(path), err))
			return nil
		}

		c.tracker.Register(sum, path)
		indexed++
		return nil
	})

	return indexed, err
}

func (c *Converter) showCopySummary(stats *copyStats) {
	duration := time.Since(c.stats.startTime)

	// Skip ASCII banner in JSON mode
	if !c.logger.IsJSONMode() {
		fmt.Println()
		fmt.Println("╔══════════════════════════════════════════════════════════════╗")
		fmt.Println("║                 Copy-Only Mode Complete                      ║")
		fmt.Println("╚══════════════════════════════════════════════════════════════╝")
		fmt.Println()
	}

	c.logger.Success(fmt.Sprintf("✅ Files processed: %d/%d", c.stats.processedFiles, c.stats.totalFiles))
	if stats.copied > 0 {
		c.logger.Success(fmt.Sprintf("📦 Files copied: %d", stats.copied))
	}
	if stats.dryRunCopies > 0 {
		c.logger.Info(fmt.Sprintf("🔎 Dry-run copies simulated: %d", stats.dryRunCopies))
	}
	if stats.duplicates > 0 {
		c.logger.Info(fmt.Sprintf("⏭️  Duplicates skipped: %d", stats.duplicates))
	}
	if stats.failed > 0 {
		c.logger.Warn(fmt.Sprintf("⚠️  Failed copies: %d", stats.failed))
	}

	if stats.copiedBytes > 0 {
		c.logger.Info(fmt.Sprintf("💾 Data copied: %.1f GB", float64(stats.copiedBytes)/(1024*1024*1024)))
	}
	c.logger.Info(fmt.Sprintf("⏱️  Total time: %v", duration.Round(time.Second)))
	if stats.checksumDuration > 0 {
		c.logger.Info(fmt.Sprintf("🔐 Time spent on checksum verification: %v", stats.checksumDuration.Round(time.Second)))
	}

	if !c.logger.IsJSONMode() {
		fmt.Println()
	}
	c.logger.Info(fmt.Sprintf("📁 Archived files in: %s", c.config.DestDir))
	c.logger.Info(fmt.Sprintf("📄 Detailed logs: %s/conversion.log", c.config.DestDir))

	// Emit complete and statistics events in JSON mode
	if c.logger.IsJSONMode() {
		jsonWriter := c.logger.GetJSONWriter()

		// Complete event
		jsonWriter.EmitComplete(api.NewCompleteEvent(
			stats.failed == 0,
			c.stats.totalFiles,
			c.stats.processedFiles,
			stats.failed,
			c.stats.skippedFiles+stats.duplicates,
			duration,
			"",
		))

		// Statistics event
		jsonWriter.EmitStatistics(api.StatisticsEvent{
			ImagesConverted:  0,
			VideosConverted:  0,
			FilesCopied:      stats.copied,
			FilesSkipped:     c.stats.skippedFiles + stats.duplicates,
			DuplicatesFound:  stats.duplicates,
			TotalInputSize:   stats.totalBytes,
			TotalOutputSize:  stats.copiedBytes,
			SpaceSaved:       0,
			CompressionRatio: 0,
			AverageSpeed:     "",
			TotalDuration:    duration.String(),
		})
	}
}
