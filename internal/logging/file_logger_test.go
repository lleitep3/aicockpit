package logging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFileLoggerCreation(t *testing.T) {
	tmpDir := t.TempDir()
	logsDir := filepath.Join(tmpDir, "logs")

	logger, err := NewFileLogger(logsDir, true)
	if err != nil {
		t.Fatalf("NewFileLogger failed: %v", err)
	}
	defer logger.Close()

	// Verify logs directory was created
	if _, err := os.Stat(logsDir); os.IsNotExist(err) {
		t.Error("Logs directory was not created")
	}
}

func TestFileLoggerJSON(t *testing.T) {
	tmpDir := t.TempDir()
	logsDir := filepath.Join(tmpDir, "logs")

	logger, err := NewFileLogger(logsDir, true)
	if err != nil {
		t.Fatalf("NewFileLogger failed: %v", err)
	}
	defer logger.Close()

	// Log a message
	context := map[string]interface{}{
		"command": "setup",
		"status":  "success",
	}

	if err := logger.Log("INFO", "Test message", context); err != nil {
		t.Fatalf("Log failed: %v", err)
	}

	// Verify log file was created
	logFile := logger.GetLogsForDate(time.Now())
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Error("Log file was not created")
	}

	// Verify log content
	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "Test message") {
		t.Error("Log message not found in file")
	}

	if !strings.Contains(content, "INFO") {
		t.Error("Log level not found in file")
	}
}

func TestFileLoggerText(t *testing.T) {
	tmpDir := t.TempDir()
	logsDir := filepath.Join(tmpDir, "logs")

	logger, err := NewFileLogger(logsDir, false)
	if err != nil {
		t.Fatalf("NewFileLogger failed: %v", err)
	}
	defer logger.Close()

	// Log a message
	if err := logger.Log("WARN", "Warning message", nil); err != nil {
		t.Fatalf("Log failed: %v", err)
	}

	// Verify log file content
	logFile := logger.GetLogsForDate(time.Now())
	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "Warning message") {
		t.Error("Log message not found in file")
	}

	if !strings.Contains(content, "WARN") {
		t.Error("Log level not found in file")
	}
}

func TestFileLoggerRotation(t *testing.T) {
	tmpDir := t.TempDir()
	logsDir := filepath.Join(tmpDir, "logs")

	logger, err := NewFileLogger(logsDir, true)
	if err != nil {
		t.Fatalf("NewFileLogger failed: %v", err)
	}
	defer logger.Close()

	// Log a message today
	if err := logger.Log("INFO", "Today message", nil); err != nil {
		t.Fatalf("Log failed: %v", err)
	}

	todayFile := logger.GetLogsForDate(time.Now())

	// Verify today's file exists
	if _, err := os.Stat(todayFile); os.IsNotExist(err) {
		t.Error("Today's log file was not created")
	}

	// Get all logs
	logs, err := logger.GetAllLogs()
	if err != nil {
		t.Fatalf("GetAllLogs failed: %v", err)
	}

	if len(logs) != 1 {
		t.Errorf("Expected 1 log file, got %d", len(logs))
	}
}

func TestFileLoggerGetAllLogs(t *testing.T) {
	tmpDir := t.TempDir()
	logsDir := filepath.Join(tmpDir, "logs")

	logger, err := NewFileLogger(logsDir, true)
	if err != nil {
		t.Fatalf("NewFileLogger failed: %v", err)
	}
	defer logger.Close()

	// Log multiple messages
	for i := 0; i < 5; i++ {
		if err := logger.Log("INFO", "Test message", nil); err != nil {
			t.Fatalf("Log failed: %v", err)
		}
	}

	// Get all logs
	logs, err := logger.GetAllLogs()
	if err != nil {
		t.Fatalf("GetAllLogs failed: %v", err)
	}

	if len(logs) != 1 {
		t.Errorf("Expected 1 log file, got %d", len(logs))
	}
}
