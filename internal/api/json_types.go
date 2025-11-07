package api

import "time"

// JSONMode enables structured JSON output for Tauri integration
type JSONMode struct {
	Enabled bool
	Writer  *JSONWriter
}

// EventType represents different types of JSON events
type EventType string

const (
	EventTypeStarted    EventType = "started"
	EventTypeProgress   EventType = "progress"
	EventTypeFileStart  EventType = "file_start"
	EventTypeFileEnd    EventType = "file_end"
	EventTypeLog        EventType = "log"
	EventTypeError      EventType = "error"
	EventTypeComplete   EventType = "complete"
	EventTypeStatistics EventType = "statistics"
)

// JSONEvent is the base structure for all JSON events
type JSONEvent struct {
	Type      EventType   `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// StartedEvent is emitted when conversion begins
type StartedEvent struct {
	SourceDir      string `json:"source_dir"`
	DestDir        string `json:"dest_dir"`
	TotalFiles     int    `json:"total_files"`
	Mode           string `json:"mode"` // "conversion" or "copy-only"
	DryRun         bool   `json:"dry_run"`
	KeepOriginals  bool   `json:"keep_originals"`
	OrganizeByDate bool   `json:"organize_by_date"`
}

// ProgressEvent is emitted for overall progress
type ProgressEvent struct {
	ProcessedFiles int     `json:"processed_files"`
	TotalFiles     int     `json:"total_files"`
	ProgressPercent float64 `json:"progress_percent"`
	EstimatedTimeRemaining string `json:"eta,omitempty"`
}

// FileStartEvent is emitted when a file processing starts
type FileStartEvent struct {
	FilePath     string `json:"file_path"`
	FileName     string `json:"file_name"`
	FileSize     int64  `json:"file_size"`
	FileType     string `json:"file_type"` // "image" or "video"
	Operation    string `json:"operation"` // "convert" or "copy"
}

// FileEndEvent is emitted when a file processing completes
type FileEndEvent struct {
	FilePath       string  `json:"file_path"`
	FileName       string  `json:"file_name"`
	Success        bool    `json:"success"`
	OutputPath     string  `json:"output_path,omitempty"`
	OutputSize     int64   `json:"output_size,omitempty"`
	CompressionRatio float64 `json:"compression_ratio,omitempty"`
	Duration       string  `json:"duration"`
	ErrorMessage   string  `json:"error_message,omitempty"`
	Checksum       string  `json:"checksum,omitempty"`
}

// LogEvent is emitted for general log messages
type LogEvent struct {
	Level   string `json:"level"` // "info", "warn", "error", "debug"
	Message string `json:"message"`
}

// ErrorEvent is emitted when an error occurs
type ErrorEvent struct {
	Message  string `json:"message"`
	FilePath string `json:"file_path,omitempty"`
	Fatal    bool   `json:"fatal"`
}

// CompleteEvent is emitted when conversion finishes
type CompleteEvent struct {
	Success      bool   `json:"success"`
	TotalFiles   int    `json:"total_files"`
	ProcessedFiles int  `json:"processed_files"`
	FailedFiles  int    `json:"failed_files"`
	SkippedFiles int    `json:"skipped_files"`
	TotalDuration string `json:"total_duration"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// StatisticsEvent contains detailed statistics
type StatisticsEvent struct {
	ImagesConverted int    `json:"images_converted"`
	VideosConverted int    `json:"videos_converted"`
	FilesCopied     int    `json:"files_copied"`
	FilesSkipped    int    `json:"files_skipped"`
	DuplicatesFound int    `json:"duplicates_found"`
	TotalInputSize  int64  `json:"total_input_size"`
	TotalOutputSize int64  `json:"total_output_size"`
	SpaceSaved      int64  `json:"space_saved"`
	CompressionRatio float64 `json:"compression_ratio"`
	AverageSpeed    string `json:"average_speed"`
	TotalDuration   string `json:"total_duration"`
}
