package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func GetFileDate(filePath string) (time.Time, error) {
	// Try multiple methods to extract date

	// 1. Try macOS mdls (most reliable for RAW files)
	if date, err := getMacOSMetadataDate(filePath); err == nil && isValidDate(date) {
		return date, nil
	}

	// 2. Try to get creation date from image metadata
	if date, err := getImageMetadataDate(filePath); err == nil && isValidDate(date) {
		return date, nil
	}

	// 3. Try to get creation date from video metadata
	if date, err := getVideoMetadataDate(filePath); err == nil && isValidDate(date) {
		return date, nil
	}

	// 4. Fall back to file modification time, but validate it
	info, err := os.Stat(filePath)
	if err != nil {
		return time.Time{}, err
	}

	modTime := info.ModTime()
	if isValidDate(modTime) {
		return modTime, nil
	}

	// 5. Last resort: return an error instead of current time
	return time.Time{}, fmt.Errorf("no valid date found for file: %s", filepath.Base(filePath))
}

func getMacOSMetadataDate(filePath string) (time.Time, error) {
	// Only works on macOS
	if runtime.GOOS != "darwin" {
		return time.Time{}, fmt.Errorf("mdls only available on macOS")
	}

	// Use mdls to get metadata (same as original bash script)
	cmd := exec.Command("mdls", "-name", "kMDItemContentCreationDate", "-raw", filePath)
	output, err := cmd.Output()
	if err != nil {
		return time.Time{}, fmt.Errorf("mdls command failed: %w", err)
	}

	dateStr := strings.TrimSpace(string(output))
	if dateStr == "" || dateStr == "(null)" {
		return time.Time{}, fmt.Errorf("no creation date found in metadata")
	}

	// Parse the macOS metadata date format
	// Format: "2024-01-15 10:30:45 +0000"
	formats := []string{
		"2006-01-02 15:04:05 -0700", // macOS mdls format
		"2006-01-02 15:04:05 +0000", // macOS mdls UTC format
		"2006-01-02 15:04:05",       // Without timezone
	}

	for _, format := range formats {
		if date, err := time.Parse(format, dateStr); err == nil {
			return date, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse mdls date: %s", dateStr)
}

func getImageMetadataDate(filePath string) (time.Time, error) {
	// Check if it's an image file
	if !HasExtension(filePath, []string{"jpg", "jpeg", "heic", "heif", "cr2", "arw", "nef", "dng", "tiff", "tif", "png", "raw", "bmp", "gif", "webp"}) {
		return time.Time{}, fmt.Errorf("not an image file")
	}

	// Try multiple EXIF fields in order of preference
	exifFields := []string{
		"%[EXIF:DateTimeOriginal]", // Camera capture time (preferred)
		"%[EXIF:DateTime]",         // File save time
		"%[date:create]",           // General creation time
		"%[date:modify]",           // Modification time
	}

	for _, field := range exifFields {
		cmd := exec.Command("magick", "identify", "-format", field, filePath)
		output, err := cmd.Output()
		if err != nil {
			continue // Try next field
		}

		dateStr := strings.TrimSpace(string(output))
		if dateStr == "" || dateStr == "(null)" {
			continue // Try next field
		}

		if date, err := parseDateTime(dateStr); err == nil {
			return date, nil
		}
	}

	return time.Time{}, fmt.Errorf("no valid date found in image metadata")
}

func getVideoMetadataDate(filePath string) (time.Time, error) {
	// Check if it's a video file
	if !HasExtension(filePath, []string{"mov", "mp4", "avi", "mkv", "m4v", "mts", "m2ts", "mpg", "mpeg", "wmv", "flv", "3gp", "3gpp"}) {
		return time.Time{}, fmt.Errorf("not a video file")
	}

	// Use ffprobe to extract creation time
	cmd := exec.Command("ffprobe", "-v", "quiet", "-show_entries", "format_tags=creation_time", "-of", "csv=p=0", filePath)
	output, err := cmd.Output()
	if err != nil {
		return time.Time{}, fmt.Errorf("ffprobe failed: %w", err)
	}

	dateStr := strings.TrimSpace(string(output))
	if dateStr == "" {
		return time.Time{}, fmt.Errorf("no creation time found in video metadata")
	}

	return parseDateTime(dateStr)
}

func parseDateTime(dateStr string) (time.Time, error) {
	// Parse common date formats found in metadata
	formats := []string{
		"2006:01:02 15:04:05",         // EXIF format
		"2006-01-02T15:04:05.000000Z", // ISO with microseconds
		"2006-01-02T15:04:05Z",        // ISO basic
		"2006-01-02T15:04:05-07:00",   // ISO with timezone
		"2006-01-02 15:04:05",         // Standard format
		time.RFC3339,                  // RFC3339
		time.RFC3339Nano,              // RFC3339 with nanoseconds
	}

	for _, format := range formats {
		if date, err := time.Parse(format, dateStr); err == nil {
			return date, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}

func isValidDate(date time.Time) bool {
	// Check if date is reasonable for media files
	now := time.Now()

	// Reject dates in the future
	if date.After(now) {
		return false
	}

	// Reject dates before digital photography era (1990)
	minDate := time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)
	if date.Before(minDate) {
		return false
	}

	return true
}

func CleanFilename(filename, extension string, date time.Time, counter int) string {
	// Remove special characters and normalize
	reg := regexp.MustCompile(`[^a-zA-Z0-9._-]`)
	cleanName := reg.ReplaceAllString(filename, "_")

	// Remove multiple underscores
	reg = regexp.MustCompile(`_+`)
	cleanName = reg.ReplaceAllString(cleanName, "_")

	// Remove leading/trailing underscores
	cleanName = strings.Trim(cleanName, "_")

	// Format date
	dateStr := date.Format("2006-01-02")

	return fmt.Sprintf("%s_%s_%03d.%s", dateStr, cleanName, counter, extension)
}

func GetMonthName(month int, language string) string {
	language = strings.ToLower(language)
	monthNames := map[string][]string{
		"fr": {"Janvier", "Fevrier", "Mars", "Avril", "Mai", "Juin",
			"Juillet", "Aout", "Septembre", "Octobre", "Novembre", "Decembre"},
		"en": {"January", "February", "March", "April", "May", "June",
			"July", "August", "September", "October", "November", "December"},
		"es": {"Enero", "Febrero", "Marzo", "Abril", "Mayo", "Junio",
			"Julio", "Agosto", "Septiembre", "Octubre", "Noviembre", "Diciembre"},
		"de": {"Januar", "Februar", "Maerz", "April", "Mai", "Juni",
			"Juli", "August", "September", "Oktober", "November", "Dezember"},
	}

	names, exists := monthNames[language]
	if !exists {
		names = monthNames["en"] // Default to English
	}

	if month < 1 || month > 12 {
		return "Unknown"
	}

	return fmt.Sprintf("%02d-%s", month, names[month-1])
}

func CreateDestinationPath(baseDir string, fileDate time.Time, mediaType string, organizeByDate bool, language string) string {
	if !organizeByDate {
		return filepath.Join(baseDir, mediaType+"s")
	}

	year := strconv.Itoa(fileDate.Year())
	monthName := GetMonthName(int(fileDate.Month()), language)
	dayStr := fileDate.Format("2006-01-02")

	return filepath.Join(baseDir, year, monthName, dayStr, mediaType+"s")
}

func EnsureDir(dirPath string) error {
	return os.MkdirAll(dirPath, 0755)
}

func GetUniqueFilename(basePath, filename, extension string) (string, error) {
	counter := 1
	for {
		fullPath := filepath.Join(basePath, filename)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			return fullPath, nil
		}

		// Extract base name without extension
		baseName := strings.TrimSuffix(filename, filepath.Ext(filename))

		// Create new filename with counter
		newFilename := fmt.Sprintf("%s_%03d%s", baseName, counter, filepath.Ext(filename))
		counter++

		if counter > 9999 { // Prevent infinite loop
			return "", fmt.Errorf("unable to create unique filename after 9999 attempts")
		}

		filename = newFilename
	}
}

func GetVideoDuration(filePath string) (time.Duration, error) {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		filePath,
	)

	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("ffprobe duration query failed: %w", err)
	}

	durationStr := strings.TrimSpace(string(output))
	if durationStr == "" || strings.EqualFold(durationStr, "N/A") {
		return 0, fmt.Errorf("video duration unavailable")
	}

	seconds, err := strconv.ParseFloat(durationStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid duration value %q: %w", durationStr, err)
	}

	return time.Duration(seconds * float64(time.Second)), nil
}

func CheckDependencies() error {
	dependencies := []string{"ffmpeg", "ffprobe", "magick"}

	var missing []string
	for _, dep := range dependencies {
		if _, err := exec.LookPath(dep); err != nil {
			missing = append(missing, dep)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing dependencies: %s", strings.Join(missing, ", "))
	}

	return nil
}

func HasExtension(filename string, extensions []string) bool {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filename), "."))
	for _, validExt := range extensions {
		if ext == strings.ToLower(validExt) {
			return true
		}
	}
	return false
}
