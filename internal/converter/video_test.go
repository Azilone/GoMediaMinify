package converter

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kevindurb/media-converter/internal/config"
	"github.com/kevindurb/media-converter/internal/logger"
)

// TestVideoConversion tests the complete video conversion workflow
func TestVideoConversion(t *testing.T) {
	// Skip if FFmpeg is not available
	if !isFFmpegAvailable() {
		t.Skip("FFmpeg not available, skipping video conversion tests")
	}

	// Create temporary directories for test
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	// Create a simple test video
	testVideoPath := filepath.Join(sourceDir, "test.mp4")
	if err := createTestVideo(testVideoPath); err != nil {
		t.Skipf("Failed to create test video: %v (FFmpeg might not support all codecs)", err)
	}

	// Configure converter
	cfg := &config.Config{
		SourceDir:             sourceDir,
		DestDir:               destDir,
		VideoCodec:            "h264", // Use h264 for faster tests
		VideoCRF:              23,
		VideoAcceleration:     false, // Disable hardware acceleration for consistent tests
		KeepOriginals:         true,
		OrganizeByDate:        false,
		DryRun:                false,
		ConversionTimeoutVideo: 120,
		PhotoFormats:          []string{"jpg", "jpeg", "png"},
		VideoFormats:          []string{"mp4", "mov", "avi"},
		MaxJobs:               1,
	}

	// Create logger
	log, err := logger.NewLogger(filepath.Join(destDir, "conversion.log"))
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Close()

	// Create converter
	converter := NewConverter(cfg, log)

	// Run conversion
	err = converter.convertVideo(testVideoPath)
	if err != nil {
		t.Fatalf("Video conversion failed: %v", err)
	}

	// Verify output file exists
	outputFiles, err := filepath.Glob(filepath.Join(destDir, "*.mp4"))
	if err != nil {
		t.Fatalf("Failed to search for output files: %v", err)
	}

	if len(outputFiles) == 0 {
		t.Error("No output video file was created")
	}

	// Verify original file still exists (KeepOriginals=true)
	if _, err := os.Stat(testVideoPath); os.IsNotExist(err) {
		t.Error("Original video file was deleted despite KeepOriginals=true")
	}
}

// TestVideoConversionH265 tests conversion to H.265
func TestVideoConversionH265(t *testing.T) {
	if !isFFmpegAvailable() {
		t.Skip("FFmpeg not available, skipping video conversion tests")
	}

	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	testVideoPath := filepath.Join(sourceDir, "test.mp4")
	if err := createTestVideo(testVideoPath); err != nil {
		t.Skipf("Failed to create test video: %v", err)
	}

	cfg := &config.Config{
		SourceDir:              sourceDir,
		DestDir:                destDir,
		VideoCodec:             "h265",
		VideoCRF:               28,
		VideoAcceleration:      false,
		KeepOriginals:          true,
		OrganizeByDate:         false,
		DryRun:                 false,
		ConversionTimeoutVideo: 120,
		PhotoFormats:           []string{"jpg", "jpeg", "png"},
		VideoFormats:           []string{"mp4", "mov", "avi"},
		MaxJobs:                1,
	}

	log, err := logger.NewLogger(filepath.Join(destDir, "conversion.log"))
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Close()
	converter := NewConverter(cfg, log)

	err = converter.convertVideo(testVideoPath)
	if err != nil {
		t.Fatalf("H.265 video conversion failed: %v", err)
	}

	// Verify output file exists
	outputFiles, err := filepath.Glob(filepath.Join(destDir, "*.mp4"))
	if err != nil {
		t.Fatalf("Failed to search for output files: %v", err)
	}

	if len(outputFiles) == 0 {
		t.Error("No output video file was created")
	}
}

// TestVideoDryRun tests that dry-run mode doesn't actually convert videos
func TestVideoDryRun(t *testing.T) {
	if !isFFmpegAvailable() {
		t.Skip("FFmpeg not available, skipping tests")
	}

	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	testVideoPath := filepath.Join(sourceDir, "test.mp4")
	if err := createTestVideo(testVideoPath); err != nil {
		t.Skipf("Failed to create test video: %v", err)
	}

	cfg := &config.Config{
		SourceDir:              sourceDir,
		DestDir:                destDir,
		VideoCodec:             "h264",
		VideoCRF:               23,
		VideoAcceleration:      false,
		KeepOriginals:          true,
		OrganizeByDate:         false,
		DryRun:                 true, // DRY RUN MODE
		ConversionTimeoutVideo: 120,
		PhotoFormats:           []string{"jpg", "jpeg", "png"},
		VideoFormats:           []string{"mp4", "mov", "avi"},
		MaxJobs:                1,
	}

	log, err := logger.NewLogger(filepath.Join(destDir, "conversion.log"))
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Close()
	converter := NewConverter(cfg, log)

	// Run conversion in dry-run mode
	err = converter.convertVideo(testVideoPath)
	if err != nil {
		t.Fatalf("Dry run failed: %v", err)
	}

	// Verify NO output file was created
	outputFiles, err := filepath.Glob(filepath.Join(destDir, "*.mp4"))
	if err != nil {
		t.Fatalf("Failed to search for output files: %v", err)
	}

	// Count only converted video files (not the original test file)
	convertedCount := 0
	for _, file := range outputFiles {
		if file != testVideoPath {
			convertedCount++
		}
	}

	if convertedCount > 0 {
		t.Error("Output video file was created in dry-run mode")
	}
}

// TestVideoIdempotency tests that running video conversion twice skips the second run
func TestVideoIdempotency(t *testing.T) {
	if !isFFmpegAvailable() {
		t.Skip("FFmpeg not available, skipping tests")
	}

	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	testVideoPath := filepath.Join(sourceDir, "test.mp4")
	if err := createTestVideo(testVideoPath); err != nil {
		t.Skipf("Failed to create test video: %v", err)
	}

	cfg := &config.Config{
		SourceDir:              sourceDir,
		DestDir:                destDir,
		VideoCodec:             "h264",
		VideoCRF:               23,
		VideoAcceleration:      false,
		KeepOriginals:          true,
		OrganizeByDate:         false,
		DryRun:                 false,
		ConversionTimeoutVideo: 120,
		PhotoFormats:           []string{"jpg", "jpeg", "png"},
		VideoFormats:           []string{"mp4", "mov", "avi"},
		MaxJobs:                1,
	}

	log, err := logger.NewLogger(filepath.Join(destDir, "conversion.log"))
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Close()
	converter := NewConverter(cfg, log)

	// First conversion
	err = converter.convertVideo(testVideoPath)
	if err != nil {
		t.Fatalf("First conversion failed: %v", err)
	}

	// Get the output file modification time
	outputFiles, _ := filepath.Glob(filepath.Join(destDir, "*.mp4"))
	if len(outputFiles) == 0 {
		t.Fatal("No output file created")
	}
	firstStat, _ := os.Stat(outputFiles[0])
	firstModTime := firstStat.ModTime()

	// Second conversion (should skip)
	err = converter.convertVideo(testVideoPath)
	if err != nil {
		t.Fatalf("Second conversion failed: %v", err)
	}

	// Verify the file wasn't re-converted (modification time should be the same)
	secondStat, _ := os.Stat(outputFiles[0])
	secondModTime := secondStat.ModTime()

	if !firstModTime.Equal(secondModTime) {
		t.Error("Video was re-converted on second run, idempotency not working")
	}
}
