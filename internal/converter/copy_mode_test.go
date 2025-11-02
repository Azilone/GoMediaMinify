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
