package utils

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestHasExtension tests the file extension checking
func TestHasExtension(t *testing.T) {
	tests := []struct {
		filename   string
		extensions []string
		expected   bool
	}{
		{"photo.jpg", []string{"jpg", "png"}, true},
		{"photo.JPG", []string{"jpg", "png"}, true},
		{"photo.jpeg", []string{"jpg", "jpeg"}, true},
		{"video.mp4", []string{"mp4", "mov"}, true},
		{"document.txt", []string{"jpg", "png"}, false},
		{"photo.jpg", []string{"png", "gif"}, false},
		{"file", []string{"jpg"}, false},
	}

	for _, tt := range tests {
		result := HasExtension(tt.filename, tt.extensions)
		if result != tt.expected {
			t.Errorf("HasExtension(%q, %v) = %v; want %v",
				tt.filename, tt.extensions, result, tt.expected)
		}
	}
}

// TestCleanFilename tests filename cleaning and formatting
func TestCleanFilename(t *testing.T) {
	date := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		filename  string
		extension string
		counter   int
		want      string
	}{
		{"my photo", "avif", 1, "2024-01-15_my_photo_001.avif"},
		{"test@#$%file", "webp", 1, "2024-01-15_test_file_001.webp"},
		{"multiple   spaces", "jpg", 2, "2024-01-15_multiple_spaces_002.jpg"},
		{"_leading_trailing_", "png", 1, "2024-01-15_leading_trailing_001.png"},
	}

	for _, tt := range tests {
		got := CleanFilename(tt.filename, tt.extension, date, tt.counter)
		if got != tt.want {
			t.Errorf("CleanFilename(%q, %q, date, %d) = %q; want %q",
				tt.filename, tt.extension, tt.counter, got, tt.want)
		}
	}
}

// TestGetMonthName tests month name localization
func TestGetMonthName(t *testing.T) {
	tests := []struct {
		month    int
		language string
		want     string
	}{
		{1, "en", "01-January"},
		{1, "fr", "01-Janvier"},
		{1, "es", "01-Enero"},
		{1, "de", "01-Januar"},
		{12, "en", "12-December"},
		{12, "fr", "12-Decembre"},
		{6, "en", "06-June"},
		{1, "unknown", "01-January"}, // Fallback to English
	}

	for _, tt := range tests {
		got := GetMonthName(tt.month, tt.language)
		if got != tt.want {
			t.Errorf("GetMonthName(%d, %q) = %q; want %q",
				tt.month, tt.language, got, tt.want)
		}
	}
}

// TestShouldSkipSystemEntry tests system file/directory filtering
func TestShouldSkipSystemEntry(t *testing.T) {
	tests := []struct {
		name   string
		isDir  bool
		should bool
	}{
		{".DS_Store", false, true},
		{"Thumbs.db", false, true},
		{"desktop.ini", false, true},
		{"._metadata", false, true},
		{".Spotlight-V100", true, true},
		{".fseventsd", true, true},
		{"__MACOSX", true, true},
		{".Trash", true, true},
		{"photo.jpg", false, false},
		{"Videos", true, false},
		{"normal_file.txt", false, false},
	}

	for _, tt := range tests {
		got := ShouldSkipSystemEntry(tt.name, tt.isDir)
		if got != tt.should {
			t.Errorf("ShouldSkipSystemEntry(%q, %v) = %v; want %v",
				tt.name, tt.isDir, got, tt.should)
		}
	}
}

// TestCreateDestinationPath tests path creation logic
func TestCreateDestinationPath(t *testing.T) {
	baseDir := "/dest"
	date := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name          string
		mediaType     string
		organizeByDate bool
		language      string
		want          string
	}{
		{"organized", "image", true, "en", "/dest/2024/01-January/2024-01-15/images"},
		{"organized_fr", "video", true, "fr", "/dest/2024/01-Janvier/2024-01-15/videos"},
		{"flat", "image", false, "en", "/dest/images"},
		{"flat_video", "video", false, "en", "/dest/videos"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CreateDestinationPath(baseDir, date, tt.mediaType, tt.organizeByDate, tt.language)
			if got != tt.want {
				t.Errorf("CreateDestinationPath() = %q; want %q", got, tt.want)
			}
		})
	}
}

// TestIsValidDate tests date validation logic
func TestIsValidDate(t *testing.T) {
	tests := []struct {
		name  string
		date  time.Time
		valid bool
	}{
		{"current date", time.Now(), true},
		{"past date", time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), true},
		{"1990", time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC), true},
		{"too old", time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC), false},
		{"future", time.Now().Add(24 * time.Hour), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidDate(tt.date)
			if got != tt.valid {
				t.Errorf("isValidDate(%v) = %v; want %v", tt.date, got, tt.valid)
			}
		})
	}
}

// TestEnsureDir tests directory creation
func TestEnsureDir(t *testing.T) {
	tempDir := t.TempDir()
	testPath := filepath.Join(tempDir, "test", "nested", "dir")

	err := EnsureDir(testPath)
	if err != nil {
		t.Fatalf("EnsureDir failed: %v", err)
	}

	// Verify directory exists
	info, err := os.Stat(testPath)
	if err != nil {
		t.Fatalf("Directory was not created: %v", err)
	}

	if !info.IsDir() {
		t.Error("Created path is not a directory")
	}
}

// TestGetUniqueFilename tests unique filename generation
func TestGetUniqueFilename(t *testing.T) {
	tempDir := t.TempDir()

	// Create first file
	filename1, err := GetUniqueFilename(tempDir, "test.txt", ".txt")
	if err != nil {
		t.Fatalf("GetUniqueFilename failed: %v", err)
	}

	// Create the file
	if err := os.WriteFile(filename1, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Get second unique filename (should be different)
	filename2, err := GetUniqueFilename(tempDir, "test.txt", ".txt")
	if err != nil {
		t.Fatalf("GetUniqueFilename failed for second file: %v", err)
	}

	if filename1 == filename2 {
		t.Error("GetUniqueFilename returned the same filename twice")
	}

	// Verify the first filename exists
	if _, err := os.Stat(filename1); err != nil {
		t.Error("First file doesn't exist")
	}

	// Verify the second filename doesn't exist yet
	if _, err := os.Stat(filename2); !os.IsNotExist(err) {
		t.Error("Second file already exists")
	}
}

// TestCheckDependencies tests dependency checking
func TestCheckDependencies(t *testing.T) {
	// This test will pass if all dependencies are installed
	// It's more of an integration test
	err := CheckDependencies()
	if err != nil {
		t.Skipf("Dependencies not installed (this is expected in some environments): %v", err)
	}
}
