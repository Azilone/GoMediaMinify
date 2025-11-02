package converter

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kevindurb/media-converter/internal/config"
	"github.com/kevindurb/media-converter/internal/logger"
)

// TestImageConversion tests the complete image conversion workflow
func TestImageConversion(t *testing.T) {
	// Skip if ImageMagick is not available
	if !isImageMagickAvailable() {
		t.Skip("ImageMagick not available, skipping image conversion tests")
	}

	// Create temporary directories for test
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	// Create a simple test image
	testImagePath := filepath.Join(sourceDir, "test.jpg")
	if err := createTestImage(testImagePath); err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}

	// Configure converter
	cfg := &config.Config{
		SourceDir:              sourceDir,
		DestDir:                destDir,
		PhotoFormat:            "avif",
		PhotoQualityAVIF:       80,
		KeepOriginals:          true,
		OrganizeByDate:         false,
		DryRun:                 false,
		ConversionTimeoutPhoto: 60,
		PhotoFormats:           []string{"jpg", "jpeg", "png"},
		VideoFormats:           []string{"mp4", "mov"},
		MaxJobs:                1,
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
	err = converter.convertImage(testImagePath)
	if err != nil {
		t.Fatalf("Image conversion failed: %v", err)
	}

	// Verify output file exists
	outputFiles, err := filepath.Glob(filepath.Join(destDir, "*.avif"))
	if err != nil {
		t.Fatalf("Failed to search for output files: %v", err)
	}

	if len(outputFiles) == 0 {
		t.Error("No output file was created")
	}

	// Verify original file still exists (KeepOriginals=true)
	if _, err := os.Stat(testImagePath); os.IsNotExist(err) {
		t.Error("Original file was deleted despite KeepOriginals=true")
	}
}

// TestImageConversionWithDelete tests image conversion with deletion
func TestImageConversionWithDelete(t *testing.T) {
	if !isImageMagickAvailable() {
		t.Skip("ImageMagick not available, skipping image conversion tests")
	}

	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	testImagePath := filepath.Join(sourceDir, "test.jpg")
	if err := createTestImage(testImagePath); err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}

	cfg := &config.Config{
		SourceDir:              sourceDir,
		DestDir:                destDir,
		PhotoFormat:            "webp",
		PhotoQualityWebP:       85,
		KeepOriginals:          false, // Delete originals
		OrganizeByDate:         false,
		DryRun:                 false,
		ConversionTimeoutPhoto: 60,
		PhotoFormats:           []string{"jpg", "jpeg", "png"},
		VideoFormats:           []string{"mp4", "mov"},
		MaxJobs:                1,
	}

	log, err := logger.NewLogger(filepath.Join(destDir, "conversion.log"))
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Close()
	converter := NewConverter(cfg, log)

	// Run conversion
	err = converter.convertImage(testImagePath)
	if err != nil {
		t.Fatalf("Image conversion failed: %v", err)
	}

	// Verify output file exists
	outputFiles, err := filepath.Glob(filepath.Join(destDir, "*.webp"))
	if err != nil {
		t.Fatalf("Failed to search for output files: %v", err)
	}

	if len(outputFiles) == 0 {
		t.Error("No output file was created")
	}
}

// TestDryRunMode tests that dry-run mode doesn't actually convert files
func TestDryRunMode(t *testing.T) {
	if !isImageMagickAvailable() {
		t.Skip("ImageMagick not available, skipping tests")
	}

	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	testImagePath := filepath.Join(sourceDir, "test.jpg")
	if err := createTestImage(testImagePath); err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}

	cfg := &config.Config{
		SourceDir:              sourceDir,
		DestDir:                destDir,
		PhotoFormat:            "avif",
		PhotoQualityAVIF:       80,
		KeepOriginals:          true,
		OrganizeByDate:         false,
		DryRun:                 true, // DRY RUN MODE
		ConversionTimeoutPhoto: 60,
		PhotoFormats:           []string{"jpg", "jpeg", "png"},
		VideoFormats:           []string{"mp4", "mov"},
		MaxJobs:                1,
	}

	log, err := logger.NewLogger(filepath.Join(destDir, "conversion.log"))
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Close()
	converter := NewConverter(cfg, log)

	// Run conversion in dry-run mode
	err = converter.convertImage(testImagePath)
	if err != nil {
		t.Fatalf("Dry run failed: %v", err)
	}

	// Verify NO output file was created
	outputFiles, err := filepath.Glob(filepath.Join(destDir, "*"))
	if err != nil {
		t.Fatalf("Failed to search for output files: %v", err)
	}

	// In dry-run, only log file might exist, but no converted images
	for _, file := range outputFiles {
		if filepath.Ext(file) == ".avif" || filepath.Ext(file) == ".webp" {
			t.Error("Output file was created in dry-run mode")
		}
	}
}

// TestFindFiles tests the file discovery functionality
func TestFindFiles(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")

	// Create test directory structure
	if err := os.MkdirAll(filepath.Join(sourceDir, "subdir"), 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Create test files
	testFiles := []string{
		"photo1.jpg",
		"photo2.png",
		"video1.mp4",
		"video2.mov",
		"subdir/photo3.jpeg",
		"subdir/video3.avi",
		"document.txt", // Should be ignored
	}

	for _, file := range testFiles {
		filePath := filepath.Join(sourceDir, file)
		if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", file, err)
		}
	}

	destDir := filepath.Join(tempDir, "dest")
	cfg := &config.Config{
		SourceDir:    sourceDir,
		DestDir:      destDir,
		PhotoFormats: []string{"jpg", "jpeg", "png", "heic"},
		VideoFormats: []string{"mp4", "mov", "avi", "mkv"},
	}

	// Create destination directory for logger
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatalf("Failed to create destination directory: %v", err)
	}

	log, err := logger.NewLogger(filepath.Join(cfg.DestDir, "conversion.log"))
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Close()
	converter := NewConverter(cfg, log)

	photoFiles, videoFiles, err := converter.findFiles()
	if err != nil {
		t.Fatalf("findFiles failed: %v", err)
	}

	// Verify correct number of files found
	expectedPhotos := 3 // photo1.jpg, photo2.png, subdir/photo3.jpeg
	expectedVideos := 3 // video1.mp4, video2.mov, subdir/video3.avi

	if len(photoFiles) != expectedPhotos {
		t.Errorf("Expected %d photo files, got %d", expectedPhotos, len(photoFiles))
	}

	if len(videoFiles) != expectedVideos {
		t.Errorf("Expected %d video files, got %d", expectedVideos, len(videoFiles))
	}
}

// TestIdempotency tests that running conversion twice on the same file skips the second run
func TestIdempotency(t *testing.T) {
	if !isImageMagickAvailable() {
		t.Skip("ImageMagick not available, skipping tests")
	}

	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	testImagePath := filepath.Join(sourceDir, "test.jpg")
	if err := createTestImage(testImagePath); err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}

	cfg := &config.Config{
		SourceDir:              sourceDir,
		DestDir:                destDir,
		PhotoFormat:            "avif",
		PhotoQualityAVIF:       80,
		KeepOriginals:          true,
		OrganizeByDate:         false,
		DryRun:                 false,
		ConversionTimeoutPhoto: 60,
		PhotoFormats:           []string{"jpg", "jpeg", "png"},
		VideoFormats:           []string{"mp4", "mov"},
		MaxJobs:                1,
	}

	log, err := logger.NewLogger(filepath.Join(destDir, "conversion.log"))
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Close()
	converter := NewConverter(cfg, log)

	// First conversion
	err = converter.convertImage(testImagePath)
	if err != nil {
		t.Fatalf("First conversion failed: %v", err)
	}

	// Get the output file modification time
	outputFiles, _ := filepath.Glob(filepath.Join(destDir, "*.avif"))
	if len(outputFiles) == 0 {
		t.Fatal("No output file created")
	}
	firstStat, _ := os.Stat(outputFiles[0])
	firstModTime := firstStat.ModTime()

	// Second conversion (should skip)
	err = converter.convertImage(testImagePath)
	if err != nil {
		t.Fatalf("Second conversion failed: %v", err)
	}

	// Verify the file wasn't re-converted (modification time should be the same)
	secondStat, _ := os.Stat(outputFiles[0])
	secondModTime := secondStat.ModTime()

	if !firstModTime.Equal(secondModTime) {
		t.Error("File was re-converted on second run, idempotency not working")
	}
}
