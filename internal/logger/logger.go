package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Logger wraps slog.Logger with additional functionality.
type Logger struct {
	*slog.Logger
	logFile *os.File
	logPath string
	mu      sync.Mutex
}

var (
	instance *Logger
	once     sync.Once
)

// New creates a new logger instance (singleton pattern).
func New() *Logger {
	once.Do(func() {
		instance = &Logger{}
		instance.init()
	})
	return instance
}

// init initializes the logger with file and console output.
func (l *Logger) init() {
	// Create logs directory if it doesn't exist
	logsDir := filepath.Join(os.ExpandEnv("$HOME"), ".cockpit", "logs")
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create logs directory: %v\n", err)
		return
	}

	// Create log file with timestamp
	logFileName := fmt.Sprintf("cockpit-%s.log", time.Now().Format("2006-01-02"))
	l.logPath = filepath.Join(logsDir, logFileName)

	// Open or create log file
	logFile, err := os.OpenFile(l.logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open log file: %v\n", err)
		return
	}
	l.logFile = logFile

	// Create multi-writer for both stdout and file
	multiWriter := io.MultiWriter(os.Stdout, logFile)

	// Create handler with text format
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	handler := slog.NewTextHandler(multiWriter, opts)
	l.Logger = slog.New(handler)
}

// Close closes the log file.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.logFile != nil {
		return l.logFile.Close()
	}
	return nil
}

// GetLogPath returns the path to the current log file.
func (l *Logger) GetLogPath() string {
	return l.logPath
}
