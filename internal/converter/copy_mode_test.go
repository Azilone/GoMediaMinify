package converter

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kevindurb/media-converter/internal/config"
	"github.com/kevindurb/media-converter/internal/logger"
)

func TestCopyModeCopiesAndSkipsDuplicates(t *testing.T) {
	sourceDir := t.TempDir()
	destDir := t.TempDir()

	content := []byte("duplicate-content")
	if err := os.WriteFile(filepath.Join(sourceDir, "alpha.bin"), content, 0644); err != nil {
		t.Fatalf("write source file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "beta.bin"), content, 0644); err != nil {
		t.Fatalf("write duplicate file: %v", err)
	}

	cfg := &config.Config{
		SourceDir:              sourceDir,
		DestDir:                destDir,
		MaxJobs:                1,
		DryRun:                 false,
		CopyOnly:               true,
		VerifyChecksum:         true,
		KeepOriginals:          true,
		OrganizeByDate:         false,
		Language:               "en",
		PhotoFormats:           []string{"bin"},
		VideoFormats:           []string{},
		MinOutputSizeRatio:     0.005,
		MinOutputSizeRatioAVIF: 0.001,
		MinOutputSizeRatioWebP: 0.003,
	}
	log, err := logger.NewLogger("")
	if err != nil {
		t.Fatalf("init logger: %v", err)
	}
	defer log.Close()

	conv := NewConverter(cfg, log)

	if err := conv.runCopyMode(); err != nil {
		t.Fatalf("runCopyMode: %v", err)
	}

	destImagesDir := filepath.Join(destDir, "images")
	files, err := os.ReadDir(destImagesDir)
	if err != nil {
		t.Fatalf("read destination dir: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 copied file, got %d", len(files))
	}

	copiedPath := filepath.Join(destImagesDir, files[0].Name())
	copiedContent, err := os.ReadFile(copiedPath)
	if err != nil {
		t.Fatalf("read copied file: %v", err)
	}
	if string(copiedContent) != string(content) {
		t.Fatalf("copied content mismatch")
	}

	unique, duplicates := conv.tracker.Stats()
	if unique == 0 {
		t.Fatalf("expected at least one checksum to be tracked")
	}
	if duplicates != 0 {
		t.Fatalf("expected duplicate tracker to remain zero, got %d", duplicates)
	}
}

func TestCopyModeDryRun(t *testing.T) {
	sourceDir := t.TempDir()
	destDir := t.TempDir()

	content := []byte("dryrun-content")
	if err := os.WriteFile(filepath.Join(sourceDir, "sample.bin"), content, 0644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	cfg := &config.Config{
		SourceDir:              sourceDir,
		DestDir:                destDir,
		MaxJobs:                1,
		DryRun:                 true,
		CopyOnly:               true,
		VerifyChecksum:         true,
		KeepOriginals:          true,
		OrganizeByDate:         false,
		Language:               "en",
		PhotoFormats:           []string{"bin"},
		VideoFormats:           []string{},
		MinOutputSizeRatio:     0.005,
		MinOutputSizeRatioAVIF: 0.001,
		MinOutputSizeRatioWebP: 0.003,
	}
	log, err := logger.NewLogger("")
	if err != nil {
		t.Fatalf("init logger: %v", err)
	}
	defer log.Close()

	conv := NewConverter(cfg, log)

	if err := conv.runCopyMode(); err != nil {
		t.Fatalf("runCopyMode dry-run: %v", err)
	}

	if _, err := os.Stat(filepath.Join(destDir, "images", "sample.bin")); err == nil {
		t.Fatalf("dry-run should not create copied file")
	}
}

func TestCopyModeSkipsExistingDestinationChecksum(t *testing.T) {
	sourceDir := t.TempDir()
	destDir := t.TempDir()

	content := []byte("same-content")
	if err := os.WriteFile(filepath.Join(sourceDir, "duplicate.bin"), content, 0644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	previousDir := filepath.Join(destDir, "2024", "10-October", "2024-10-10", "images")
	if err := os.MkdirAll(previousDir, 0755); err != nil {
		t.Fatalf("create existing dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(previousDir, "duplicate.bin"), content, 0644); err != nil {
		t.Fatalf("write existing file: %v", err)
	}

	cfg := &config.Config{
		SourceDir:              sourceDir,
		DestDir:                destDir,
		MaxJobs:                1,
		DryRun:                 false,
		CopyOnly:               true,
		VerifyChecksum:         true,
		KeepOriginals:          true,
		OrganizeByDate:         false,
		Language:               "en",
		PhotoFormats:           []string{"bin"},
		VideoFormats:           []string{},
		MinOutputSizeRatio:     0.005,
		MinOutputSizeRatioAVIF: 0.001,
		MinOutputSizeRatioWebP: 0.003,
	}
	log, err := logger.NewLogger("")
	if err != nil {
		t.Fatalf("init logger: %v", err)
	}
	defer log.Close()

	conv := NewConverter(cfg, log)

	if err := conv.runCopyMode(); err != nil {
		t.Fatalf("runCopyMode: %v", err)
	}

	targetDir := filepath.Join(destDir, "images")
	if entries, err := os.ReadDir(targetDir); err == nil && len(entries) > 0 {
		t.Fatalf("expected no new file copy, found %d entries", len(entries))
	}
}

func TestCopyModeSourceContainsDestination(t *testing.T) {
	root := t.TempDir()
	sourceDir := filepath.Join(root, "source")
	if err := os.Mkdir(sourceDir, 0755); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}

	destDir := filepath.Join(sourceDir, "destination")
	if err := os.Mkdir(destDir, 0755); err != nil {
		t.Fatalf("mkdir dest: %v", err)
	}

	content := []byte("original")
	if err := os.WriteFile(filepath.Join(sourceDir, "photo.bin"), content, 0644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	baseCfg := config.Config{
		SourceDir:              sourceDir,
		DestDir:                destDir,
		MaxJobs:                1,
		CopyOnly:               true,
		VerifyChecksum:         true,
		KeepOriginals:          true,
		OrganizeByDate:         true,
		Language:               "en",
		PhotoFormats:           []string{"bin"},
		VideoFormats:           []string{},
		MinOutputSizeRatio:     0.005,
		MinOutputSizeRatioAVIF: 0.001,
		MinOutputSizeRatioWebP: 0.003,
	}

	run := func() *Converter {
		cfg := baseCfg
		log, err := logger.NewLogger("")
		if err != nil {
			t.Fatalf("init logger: %v", err)
		}
		defer log.Close()

		conv := NewConverter(&cfg, log)
		if err := conv.runCopyMode(); err != nil {
			t.Fatalf("runCopyMode: %v", err)
		}
		return conv
	}

	conv1 := run()
	if conv1.stats.totalFiles != 1 {
		t.Fatalf("expected first run to process 1 file, got %d", conv1.stats.totalFiles)
	}

	conv2 := run()
	if conv2.stats.totalFiles != 1 {
		t.Fatalf("expected second run to process 1 file, got %d", conv2.stats.totalFiles)
	}
}
