package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewInfoCommand(t *testing.T) {
	log, cfg, tr := newTestDeps(t)
	cmd := NewInfoCommand(log, cfg, tr)
	if cmd == nil {
		t.Fatal("NewInfoCommand() returned nil")
	}
	if cmd.Use != "info" {
		t.Errorf("Use = %q, want %q", cmd.Use, "info")
	}
}

func TestRunInfo_WithPackages(t *testing.T) {
	log, cfg, tr := newTestDeps(t)

	tmpDir := t.TempDir()
	cockpitDir := filepath.Join(tmpDir, ".cockpit")
	t.Setenv("HOME", tmpDir)

	// Create packages and logs dirs with some entries.
	pkgsDir := filepath.Join(cockpitDir, "packages", "my-pkg")
	if err := os.MkdirAll(pkgsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll packages: %v", err)
	}
	logsDir := filepath.Join(cockpitDir, "logs")
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll logs: %v", err)
	}
	// Write a dummy log file so the log path branch is covered.
	if err := os.WriteFile(filepath.Join(logsDir, "2006-01-02.log"), []byte(""), 0o644); err != nil {
		t.Fatalf("WriteFile log: %v", err)
	}

	if err := runInfo(log, cfg, tr); err != nil {
		t.Errorf("runInfo() error = %v", err)
	}
}

func TestRunInfo_NoPackages(t *testing.T) {
	log, cfg, tr := newTestDeps(t)

	// Empty HOME — packages dir will not exist, triggering the "no packages" branch.
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	if err := runInfo(log, cfg, tr); err != nil {
		t.Errorf("runInfo() with empty cockpit dir error = %v", err)
	}
}
