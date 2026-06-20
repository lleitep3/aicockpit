package logger

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNew(t *testing.T) {
	log := New()
	if log == nil {
		t.Fatal("Logger should not be nil")
	}

	if log.Logger == nil {
		t.Fatal("Internal logger should not be nil")
	}
}

func TestGetLogPath(t *testing.T) {
	log := New()
	logPath := log.GetLogPath()

	if logPath == "" {
		t.Fatal("Log path should not be empty")
	}

	// Verify log file was created
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("Log file was not created")
	}
}

func TestLogFileLocation(t *testing.T) {
	log := New()
	logPath := log.GetLogPath()

	expectedDir := filepath.Join(os.ExpandEnv("$HOME"), ".cockpit", "logs")
	actualDir := filepath.Dir(logPath)

	if actualDir != expectedDir {
		t.Errorf("Expected log directory %s, got %s", expectedDir, actualDir)
	}
}
