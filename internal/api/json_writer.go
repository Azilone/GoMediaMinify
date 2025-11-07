package api

import (
	"encoding/json"
	"io"
	"os"
	"sync"
	"time"
)

// JSONWriter handles writing JSON events to stdout
type JSONWriter struct {
	mu     sync.Mutex
	writer io.Writer
}

// NewJSONWriter creates a new JSON writer
func NewJSONWriter() *JSONWriter {
	return &JSONWriter{
		writer: os.Stdout,
	}
}

// NewJSONWriterWithOutput creates a JSON writer with custom output
func NewJSONWriterWithOutput(w io.Writer) *JSONWriter {
	return &JSONWriter{
		writer: w,
	}
}

// EmitEvent writes a JSON event to stdout
func (w *JSONWriter) EmitEvent(eventType EventType, data interface{}) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	event := JSONEvent{
		Type:      eventType,
		Timestamp: time.Now(),
		Data:      data,
	}

	encoder := json.NewEncoder(w.writer)
	if err := encoder.Encode(event); err != nil {
		return err
	}

	return nil
}

// EmitStarted emits a conversion started event
func (w *JSONWriter) EmitStarted(data StartedEvent) error {
	return w.EmitEvent(EventTypeStarted, data)
}

// EmitProgress emits a progress update
func (w *JSONWriter) EmitProgress(data ProgressEvent) error {
	return w.EmitEvent(EventTypeProgress, data)
}

// EmitFileStart emits a file processing start event
func (w *JSONWriter) EmitFileStart(data FileStartEvent) error {
	return w.EmitEvent(EventTypeFileStart, data)
}

// EmitFileEnd emits a file processing end event
func (w *JSONWriter) EmitFileEnd(data FileEndEvent) error {
	return w.EmitEvent(EventTypeFileEnd, data)
}

// EmitLog emits a log message
func (w *JSONWriter) EmitLog(level, message string) error {
	return w.EmitEvent(EventTypeLog, LogEvent{
		Level:   level,
		Message: message,
	})
}

// EmitError emits an error event
func (w *JSONWriter) EmitError(message, filePath string, fatal bool) error {
	return w.EmitEvent(EventTypeError, ErrorEvent{
		Message:  message,
		FilePath: filePath,
		Fatal:    fatal,
	})
}

// EmitComplete emits a completion event
func (w *JSONWriter) EmitComplete(data CompleteEvent) error {
	return w.EmitEvent(EventTypeComplete, data)
}

// EmitStatistics emits statistics
func (w *JSONWriter) EmitStatistics(data StatisticsEvent) error {
	return w.EmitEvent(EventTypeStatistics, data)
}

// Helper functions for creating event data

// NewStartedEvent creates a started event
func NewStartedEvent(sourceDir, destDir string, totalFiles int, mode string, dryRun, keepOriginals, organizeByDate bool) StartedEvent {
	return StartedEvent{
		SourceDir:      sourceDir,
		DestDir:        destDir,
		TotalFiles:     totalFiles,
		Mode:           mode,
		DryRun:         dryRun,
		KeepOriginals:  keepOriginals,
		OrganizeByDate: organizeByDate,
	}
}

// NewProgressEvent creates a progress event
func NewProgressEvent(processed, total int, eta string) ProgressEvent {
	progressPercent := 0.0
	if total > 0 {
		progressPercent = float64(processed) / float64(total) * 100.0
	}

	return ProgressEvent{
		ProcessedFiles:         processed,
		TotalFiles:             total,
		ProgressPercent:        progressPercent,
		EstimatedTimeRemaining: eta,
	}
}

// NewFileStartEvent creates a file start event
func NewFileStartEvent(filePath, fileName string, fileSize int64, fileType, operation string) FileStartEvent {
	return FileStartEvent{
		FilePath:  filePath,
		FileName:  fileName,
		FileSize:  fileSize,
		FileType:  fileType,
		Operation: operation,
	}
}

// NewFileEndEvent creates a file end event
func NewFileEndEvent(filePath, fileName string, success bool, outputPath string, outputSize int64, compressionRatio float64, duration time.Duration, errorMsg, checksum string) FileEndEvent {
	return FileEndEvent{
		FilePath:         filePath,
		FileName:         fileName,
		Success:          success,
		OutputPath:       outputPath,
		OutputSize:       outputSize,
		CompressionRatio: compressionRatio,
		Duration:         duration.String(),
		ErrorMessage:     errorMsg,
		Checksum:         checksum,
	}
}

// NewCompleteEvent creates a completion event
func NewCompleteEvent(success bool, total, processed, failed, skipped int, duration time.Duration, errorMsg string) CompleteEvent {
	return CompleteEvent{
		Success:        success,
		TotalFiles:     total,
		ProcessedFiles: processed,
		FailedFiles:    failed,
		SkippedFiles:   skipped,
		TotalDuration:  duration.String(),
		ErrorMessage:   errorMsg,
	}
}
