package logger

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/sirupsen/logrus"
)

type Logger struct {
	log    *logrus.Logger
	colors struct {
		red    *color.Color
		green  *color.Color
		yellow *color.Color
		blue   *color.Color
		purple *color.Color
		cyan   *color.Color
		bold   *color.Color
	}
	logFile *os.File
}

func NewLogger(logPath string) (*Logger, error) {
	l := &Logger{
		log: logrus.New(),
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
	if logPath != "" {
		var err error
		l.logFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}

		// Set up multi-writer for both console and file
		l.log.SetOutput(io.MultiWriter(os.Stdout, l.logFile))
	}

	l.log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "15:04:05",
		DisableColors:   false,
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
	timestamp := time.Now().Format("15:04:05")
	formatted := fmt.Sprintf("[%s] %s", l.colors.blue.Sprint(timestamp), message)
	fmt.Println(formatted)

	if l.logFile != nil {
		l.logFile.WriteString(fmt.Sprintf("[%s] %s\n", timestamp, message))
	}
}

func (l *Logger) Error(message string) {
	timestamp := time.Now().Format("15:04:05")
	formatted := fmt.Sprintf("[ERROR %s] %s", l.colors.red.Sprint(timestamp), message)
	fmt.Println(formatted)

	if l.logFile != nil {
		l.logFile.WriteString(fmt.Sprintf("[ERROR %s] %s\n", timestamp, message))
	}
}

func (l *Logger) Success(message string) {
	formatted := fmt.Sprintf("[%s] %s", l.colors.green.Sprint("✓"), message)
	fmt.Println(formatted)

	if l.logFile != nil {
		l.logFile.WriteString(fmt.Sprintf("[SUCCESS] %s\n", message))
	}
}

func (l *Logger) Warn(message string) {
	formatted := fmt.Sprintf("[%s] %s", l.colors.yellow.Sprint("⚠"), message)
	fmt.Println(formatted)

	if l.logFile != nil {
		l.logFile.WriteString(fmt.Sprintf("[WARN] %s\n", message))
	}
}

func (l *Logger) Info(message string) {
	formatted := fmt.Sprintf("[%s] %s", l.colors.cyan.Sprint("i"), message)
	fmt.Println(formatted)

	if l.logFile != nil {
		l.logFile.WriteString(fmt.Sprintf("[INFO] %s\n", message))
	}
}

func (l *Logger) Security(message string) {
	formatted := fmt.Sprintf("[%s] %s",
		l.colors.red.Add(color.Bold).Sprint("🔒 SÉCURITÉ"),
		message)
	fmt.Println(formatted)

	if l.logFile != nil {
		l.logFile.WriteString(fmt.Sprintf("[SECURITY] %s\n", message))
	}
}

func (l *Logger) ShowHeader(keepOriginals bool) {
	// Clear screen
	fmt.Print("\033[H\033[2J")

	header := `
╔══════════════════════════════════════════════════════════════╗
║        ` + l.colors.bold.Sprint("Media Converter SÉCURISÉ v1.0") + `               ║
╚══════════════════════════════════════════════════════════════╝
`
	l.colors.purple.Print(header)
	fmt.Println()

	if !keepOriginals {
		l.colors.red.Add(color.Bold).Println("⚠️  ATTENTION: Mode suppression activé !")
		l.colors.red.Println("Les fichiers originaux seront supprimés après conversion")
		l.colors.yellow.Println("Pour garder les originaux: --keep-originals")
		fmt.Println()
	} else {
		l.colors.green.Println("🔒 Mode sécurisé: Les originaux seront conservés")
		fmt.Println()
	}
}
