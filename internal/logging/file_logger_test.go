package logging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestNewFileLoggerInvalidDir verifies error when logs dir cannot be created.
func TestNewFileLoggerInvalidDir(t *testing.T) {
	tmpDir := t.TempDir()
	blocker := filepath.Join(tmpDir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	// blocker is a regular file; trying to create blocker/sub will fail
	_, err := NewFileLogger(filepath.Join(blocker, "sub"), true)
	if err == nil {
		t.Fatal("expected error creating logger in invalid dir, got nil")
	}
}

// TestFileLoggerCloseNoFile verifies Close on a logger with nil logFile returns nil.
func TestFileLoggerCloseNoFile(t *testing.T) {
	// Build a FileLogger manually with logFile == nil (never opened)
	fl := &FileLogger{
		logsDir:    t.TempDir(),
		jsonFormat: true,
	}
	if err := fl.Close(); err != nil {
		t.Fatalf("Close with nil logFile failed: %v", err)
	}
}

// TestFileLoggerCloseSuccess verifies Close on a normally-created logger succeeds.
func TestFileLoggerCloseSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	logsDir := filepath.Join(tmpDir, "logs")

	logger, err := NewFileLogger(logsDir, true)
	if err != nil {
		t.Fatalf("NewFileLogger failed: %v", err)
	}

	if err := logger.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

// TestFileLoggerRotation_DateChange simulates rotation by manipulating internal state.
func TestFileLoggerRotation_DateChange(t *testing.T) {
	tmpDir := t.TempDir()
	logsDir := filepath.Join(tmpDir, "logs")

	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		t.Fatalf("setup logsDir: %v", err)
	}

	// Create a fake "old" log file to represent yesterday's log.
	oldFile := filepath.Join(logsDir, "cockpit-1970-01-01.log")
	if err := os.WriteFile(oldFile, []byte("old entry\n"), 0o644); err != nil {
		t.Fatalf("create old log file: %v", err)
	}

	logger, err := NewFileLogger(logsDir, true)
	if err != nil {
		t.Fatalf("NewFileLogger failed: %v", err)
	}
	defer logger.Close()

	// Force the logger to believe it's currently 1970-01-01 so the next
	// Log call will see today != currentDay and trigger rotation.
	logger.mu.Lock()
	logger.currentDay = "1970-01-01"
	logger.mu.Unlock()

	// Log should trigger rotation to today's file
	if err := logger.Log("INFO", "after rotation", nil); err != nil {
		t.Fatalf("Log after forced rotation failed: %v", err)
	}

	// Both the old (1970) file and today's file should now exist
	logs, err := logger.GetAllLogs()
	if err != nil {
		t.Fatalf("GetAllLogs failed: %v", err)
	}

	found1970 := false
	foundToday := false
	today := time.Now().Format("2006-01-02")
	for _, l := range logs {
		if strings.Contains(l, "1970-01-01") {
			found1970 = true
		}
		if strings.Contains(l, today) {
			foundToday = true
		}
	}
	if !found1970 {
		t.Error("expected log file for 1970-01-01 (pre-rotation) to exist")
	}
	if !foundToday {
		t.Errorf("expected log file for today (%s) to exist", today)
	}
}

// TestFileLoggerTextFormatWithContext verifies text-format log includes context keys.
func TestFileLoggerTextFormatWithContext(t *testing.T) {
	tmpDir := t.TempDir()
	logsDir := filepath.Join(tmpDir, "logs")

	logger, err := NewFileLogger(logsDir, false)
	if err != nil {
		t.Fatalf("NewFileLogger failed: %v", err)
	}
	defer logger.Close()

	ctx := map[string]interface{}{"mykey": "myval"}
	if err := logger.Log("DEBUG", "ctx test", ctx); err != nil {
		t.Fatalf("Log failed: %v", err)
	}

	logFile := logger.GetLogsForDate(time.Now())
	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "mykey") {
		t.Error("expected context key 'mykey' in text log")
	}
	if !strings.Contains(content, "myval") {
		t.Error("expected context value 'myval' in text log")
	}
}

// TestFileLoggerGetAllLogsWithNonLogFiles ensures non-.log files are skipped.
func TestFileLoggerGetAllLogsWithNonLogFiles(t *testing.T) {
	tmpDir := t.TempDir()
	logsDir := filepath.Join(tmpDir, "logs")

	logger, err := NewFileLogger(logsDir, true)
	if err != nil {
		t.Fatalf("NewFileLogger failed: %v", err)
	}
	defer logger.Close()

	// Create a non-.log file in the logs dir
	dummy := filepath.Join(logsDir, "not-a-log.txt")
	if err := os.WriteFile(dummy, []byte("x"), 0o644); err != nil {
		t.Fatalf("create dummy file: %v", err)
	}

	logs, err := logger.GetAllLogs()
	if err != nil {
		t.Fatalf("GetAllLogs failed: %v", err)
	}

	for _, l := range logs {
		if strings.HasSuffix(l, ".txt") {
			t.Errorf("GetAllLogs returned non-.log file: %s", l)
		}
	}
	if len(logs) != 1 {
		t.Errorf("expected exactly 1 .log file, got %d", len(logs))
	}
}

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
