package logger

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/kevindurb/media-converter/internal/api"
	"github.com/sirupsen/logrus"
)

type Logger struct {
	log        *logrus.Logger
	colors     struct {
		red    *color.Color
		green  *color.Color
		yellow *color.Color
		blue   *color.Color
		purple *color.Color
		cyan   *color.Color
		bold   *color.Color
	}
	logFile    *os.File
	jsonWriter *api.JSONWriter
	jsonMode   bool
}

func NewLogger(logPath string) (*Logger, error) {
	return NewLoggerWithMode(logPath, false)
}

func NewLoggerWithMode(logPath string, jsonMode bool) (*Logger, error) {
	l := &Logger{
		log:      logrus.New(),
		jsonMode: jsonMode,
	}

	// Initialize JSON writer if in JSON mode
	if jsonMode {
		l.jsonWriter = api.NewJSONWriter()
	}

	// Initialize colors
	l.colors.red = color.New(color.FgRed)
	l.colors.green = color.New(color.FgGreen)
	l.colors.yellow = color.New(color.FgYellow)
	l.colors.blue = color.New(color.FgBlue)
	l.colors.purple = color.New(color.FgMagenta)
	l.colors.cyan = color.New(color.FgCyan)
	l.colors.bold = color.New(color.Bold)

	// Create log file if path provided
	if logPath != "" && !jsonMode {
		var err error
		l.logFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}

		// Set up multi-writer for both console and file
		l.log.SetOutput(io.MultiWriter(os.Stdout, l.logFile))
	} else if logPath != "" && jsonMode {
		// In JSON mode, only write to log file (not stdout)
		var err error
		l.logFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		l.log.SetOutput(l.logFile)
	}

	l.log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "15:04:05",
		DisableColors:   jsonMode, // Disable colors in JSON mode
	})

	return l, nil
}

func (l *Logger) Close() error {
	if l.logFile != nil {
		return l.logFile.Close()
	}
	return nil
}

func (l *Logger) Log(message string) {
	if l.jsonMode {
		l.jsonWriter.EmitLog("info", message)
		return
	}

	timestamp := time.Now().Format("15:04:05")
	formatted := fmt.Sprintf("[%s] %s", l.colors.blue.Sprint(timestamp), message)
	fmt.Println(formatted)

	if l.logFile != nil {
		l.logFile.WriteString(fmt.Sprintf("[%s] %s\n", timestamp, message))
	}
}

func (l *Logger) Error(message string) {
	if l.jsonMode {
		l.jsonWriter.EmitLog("error", message)
		return
	}

	timestamp := time.Now().Format("15:04:05")
	formatted := fmt.Sprintf("[ERROR %s] %s", l.colors.red.Sprint(timestamp), message)
	fmt.Println(formatted)

	if l.logFile != nil {
		l.logFile.WriteString(fmt.Sprintf("[ERROR %s] %s\n", timestamp, message))
	}
}

func (l *Logger) Success(message string) {
	if l.jsonMode {
		l.jsonWriter.EmitLog("info", message)
		return
	}

	formatted := fmt.Sprintf("[%s] %s", l.colors.green.Sprint("✓"), message)
	fmt.Println(formatted)

	if l.logFile != nil {
		l.logFile.WriteString(fmt.Sprintf("[SUCCESS] %s\n", message))
	}
}

func (l *Logger) Warn(message string) {
	if l.jsonMode {
		l.jsonWriter.EmitLog("warn", message)
		return
	}

	formatted := fmt.Sprintf("[%s] %s", l.colors.yellow.Sprint("⚠"), message)
	fmt.Println(formatted)

	if l.logFile != nil {
		l.logFile.WriteString(fmt.Sprintf("[WARN] %s\n", message))
	}
}

func (l *Logger) Info(message string) {
	if l.jsonMode {
		l.jsonWriter.EmitLog("info", message)
		return
	}

	formatted := fmt.Sprintf("[%s] %s", l.colors.cyan.Sprint("i"), message)
	fmt.Println(formatted)

	if l.logFile != nil {
		l.logFile.WriteString(fmt.Sprintf("[INFO] %s\n", message))
	}
}

func (l *Logger) Security(message string) {
	if l.jsonMode {
		l.jsonWriter.EmitLog("warn", "SECURITY: "+message)
		return
	}

	formatted := fmt.Sprintf("[%s] %s",
		l.colors.red.Add(color.Bold).Sprint("🔒 SECURITY"),
		message)
	fmt.Println(formatted)

	if l.logFile != nil {
		l.logFile.WriteString(fmt.Sprintf("[SECURITY] %s\n", message))
	}
}

func (l *Logger) ShowHeader(keepOriginals bool) {
	// Skip header in JSON mode
	if l.jsonMode {
		return
	}

	// Clear screen
	fmt.Print("\033[H\033[2J")

	header := `
╔══════════════════════════════════════════════════════════════╗
║        ` + l.colors.bold.Sprint("Media Converter SECURE v1.0") + `               ║
╚══════════════════════════════════════════════════════════════╝
`
	l.colors.purple.Print(header)
	fmt.Println()

	if !keepOriginals {
		l.colors.red.Add(color.Bold).Println("⚠️  WARNING: Deletion mode activated!")
		l.colors.red.Println("Original files will be deleted after conversion")
		l.colors.yellow.Println("To keep originals: --keep-originals")
		fmt.Println()
	} else {
		l.colors.green.Println("🔒 Secure mode: Originals will be preserved")
		fmt.Println()
	}
}

// GetJSONWriter returns the JSON writer (if in JSON mode)
func (l *Logger) GetJSONWriter() *api.JSONWriter {
	return l.jsonWriter
}

// IsJSONMode returns true if logger is in JSON mode
func (l *Logger) IsJSONMode() bool {
	return l.jsonMode
}
