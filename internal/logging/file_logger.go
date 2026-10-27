package logging

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Context   map[string]interface{} `json:"context,omitempty"`
}

// FileLogger handles file-based logging with daily rotation
type FileLogger struct {
	logsDir    string
	currentDay string
	logFile    *os.File
	mu         sync.Mutex
	jsonFormat bool
}

// NewFileLogger creates a new file logger
func NewFileLogger(logsDir string, jsonFormat bool) (*FileLogger, error) {
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create logs directory: %w", err)
	}

	fl := &FileLogger{
		logsDir:    logsDir,
		jsonFormat: jsonFormat,
	}

	// Open initial log file
	if err := fl.rotateIfNeeded(); err != nil {
		return nil, err
	}

	return fl, nil
}

// rotateIfNeeded checks if we need to rotate to a new log file
func (fl *FileLogger) rotateIfNeeded() error {
	today := time.Now().Format("2006-01-02")

	if fl.currentDay == today && fl.logFile != nil {
		return nil // No rotation needed
	}

	// Close old file if it exists
	if fl.logFile != nil {
		fl.logFile.Close()
	}

	// Open new file
	logFileName := fmt.Sprintf("cockpit-%s.log", today)
	logPath := filepath.Join(fl.logsDir, logFileName)

	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	fl.logFile = file
	fl.currentDay = today

	return nil
}

// Log logs a message
func (fl *FileLogger) Log(level string, message string, context map[string]interface{}) error {
	fl.mu.Lock()
	defer fl.mu.Unlock()

	// Check if rotation is needed
	if err := fl.rotateIfNeeded(); err != nil {
		return err
	}

	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Context:   context,
	}

	var line string
	if fl.jsonFormat {
		data, err := json.Marshal(entry)
		if err != nil {
			return fmt.Errorf("failed to marshal log entry: %w", err)
		}
		line = string(data) + "\n"
	} else {
		// Text format
		contextStr := ""
		if len(context) > 0 {
			for k, v := range context {
				contextStr += fmt.Sprintf(" %s=%v", k, v)
			}
		}
		line = fmt.Sprintf("[%s] %s: %s%s\n",
			entry.Timestamp.Format("2006-01-02 15:04:05"),
			level,
			message,
			contextStr,
		)
	}

	if _, err := fl.logFile.WriteString(line); err != nil {
		return fmt.Errorf("failed to write log: %w", err)
	}

	return nil
}

// Close closes the log file
func (fl *FileLogger) Close() error {
	fl.mu.Lock()
	defer fl.mu.Unlock()

	if fl.logFile != nil {
		return fl.logFile.Close()
	}
	return nil
}

// GetLogsForDate returns log file path for a specific date
func (fl *FileLogger) GetLogsForDate(date time.Time) string {
	logFileName := fmt.Sprintf("cockpit-%s.log", date.Format("2006-01-02"))
	return filepath.Join(fl.logsDir, logFileName)
}

// GetAllLogs returns all log files
func (fl *FileLogger) GetAllLogs() ([]string, error) {
	entries, err := os.ReadDir(fl.logsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read logs directory: %w", err)
	}

	var logs []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".log" {
			logs = append(logs, filepath.Join(fl.logsDir, entry.Name()))
		}
	}

	return logs, nil
}
