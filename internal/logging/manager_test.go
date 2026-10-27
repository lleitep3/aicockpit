package logging

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	tmpDir := t.TempDir()

	mgr, err := NewManager(tmpDir)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer mgr.Close()

	// logs sub-dir must exist
	logsDir := filepath.Join(tmpDir, "logs")
	if _, err := os.Stat(logsDir); os.IsNotExist(err) {
		t.Error("logs directory was not created")
	}
}

func TestNewManagerFallsBackFromInvalidDir(t *testing.T) {
	// Point cockpitDir to a file so that the preferred logs path is invalid.
	tmpDir := t.TempDir()
	blocker := filepath.Join(tmpDir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	mgr, err := NewManager(filepath.Join(blocker, "sub"))
	if err != nil {
		t.Fatalf("expected temp fallback, got error: %v", err)
	}
	if mgr.GetFileLogger() == nil {
		t.Fatal("expected file logger after temp fallback")
	}
	defer mgr.Close()
}

func TestManagerGetters(t *testing.T) {
	tmpDir := t.TempDir()

	mgr, err := NewManager(tmpDir)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer mgr.Close()

	if mgr.GetMetrics() == nil {
		t.Error("GetMetrics returned nil")
	}
	if mgr.GetFileLogger() == nil {
		t.Error("GetFileLogger returned nil")
	}
}

func TestManagerClose(t *testing.T) {
	tmpDir := t.TempDir()

	mgr, err := NewManager(tmpDir)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	if err := mgr.Close(); err != nil {
		t.Errorf("Close returned error: %v", err)
	}
}

func TestManagerLogInfo(t *testing.T) {
	tmpDir := t.TempDir()

	mgr, err := NewManager(tmpDir)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer mgr.Close()

	if err := mgr.LogInfo("info message", map[string]interface{}{"k": "v"}); err != nil {
		t.Errorf("LogInfo failed: %v", err)
	}
	if err := mgr.LogInfo("info no context", nil); err != nil {
		t.Errorf("LogInfo (nil ctx) failed: %v", err)
	}
}

func TestManagerLogWarn(t *testing.T) {
	tmpDir := t.TempDir()

	mgr, err := NewManager(tmpDir)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer mgr.Close()

	if err := mgr.LogWarn("warn message", nil); err != nil {
		t.Errorf("LogWarn failed: %v", err)
	}
}

func TestManagerLogError(t *testing.T) {
	tmpDir := t.TempDir()

	mgr, err := NewManager(tmpDir)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer mgr.Close()

	if err := mgr.LogError("error message", map[string]interface{}{"reason": "test"}); err != nil {
		t.Errorf("LogError failed: %v", err)
	}
}

func TestManagerLogCommand(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		args     []string
		status   string
		exitCode int
		duration time.Duration
		output   string
		err      error
	}{
		{
			name:     "success",
			command:  "setup",
			args:     []string{"--flag"},
			status:   "success",
			exitCode: 0,
			duration: 150 * time.Millisecond,
			output:   "ok",
			err:      nil,
		},
		{
			name:     "error with err value",
			command:  "info",
			args:     nil,
			status:   "error",
			exitCode: 1,
			duration: 50 * time.Millisecond,
			output:   "",
			err:      errors.New("something failed"),
		},
		{
			name:     "error status no err value",
			command:  "doctor",
			args:     []string{},
			status:   "error",
			exitCode: 2,
			duration: 0,
			output:   "",
			err:      nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			mgr, err := NewManager(tmpDir)
			if err != nil {
				t.Fatalf("NewManager failed: %v", err)
			}
			defer mgr.Close()

			if err := mgr.LogCommand(tc.command, tc.args, tc.status, tc.exitCode, tc.duration, tc.output, tc.err); err != nil {
				t.Errorf("LogCommand returned unexpected error: %v", err)
			}

			// Verify metric was recorded
			metrics := mgr.GetMetrics().GetMetrics()
			if len(metrics) != 1 {
				t.Errorf("expected 1 metric, got %d", len(metrics))
				return
			}
			if metrics[0].Command != tc.command {
				t.Errorf("expected command %q, got %q", tc.command, metrics[0].Command)
			}
		})
	}
}

func TestManagerLogCommandEnvVars(t *testing.T) {
	tmpDir := t.TempDir()

	// Set custom env vars
	t.Setenv("COCKPIT_VERSION", "1.2.3")
	t.Setenv("COCKPIT_LANGUAGE", "pt-br")

	mgr, err := NewManager(tmpDir)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer mgr.Close()

	if err := mgr.LogCommand("setup", nil, "success", 0, 10*time.Millisecond, "", nil); err != nil {
		t.Errorf("LogCommand failed: %v", err)
	}

	metrics := mgr.GetMetrics().GetMetrics()
	if len(metrics) != 1 {
		t.Fatalf("expected 1 metric, got %d", len(metrics))
	}
	if metrics[0].Version != "1.2.3" {
		t.Errorf("expected version 1.2.3, got %s", metrics[0].Version)
	}
	if metrics[0].Language != "pt-br" {
		t.Errorf("expected language pt-br, got %s", metrics[0].Language)
	}
}

func TestManagerLogCommandDefaultEnvVars(t *testing.T) {
	tmpDir := t.TempDir()

	// Ensure env vars are empty
	t.Setenv("COCKPIT_VERSION", "")
	t.Setenv("COCKPIT_LANGUAGE", "")

	mgr, err := NewManager(tmpDir)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer mgr.Close()

	if err := mgr.LogCommand("setup", nil, "success", 0, 0, "", nil); err != nil {
		t.Errorf("LogCommand failed: %v", err)
	}

	metrics := mgr.GetMetrics().GetMetrics()
	if len(metrics) != 1 {
		t.Fatalf("expected 1 metric, got %d", len(metrics))
	}
	if metrics[0].Version != "0.1.0" {
		t.Errorf("expected default version 0.1.0, got %s", metrics[0].Version)
	}
	if metrics[0].Language != "en-us" {
		t.Errorf("expected default language en-us, got %s", metrics[0].Language)
	}
}
