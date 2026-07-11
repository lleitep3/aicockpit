package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewDoctorCommand(t *testing.T) {
	log, cfg, tr := newTestDeps(t)
	cmd := NewDoctorCommand(log, cfg, tr)
	if cmd == nil {
		t.Fatal("NewDoctorCommand() returned nil")
	}
	if cmd.Use != "doctor" {
		t.Errorf("Use = %q, want %q", cmd.Use, "doctor")
	}
}

func TestRunDoctor_AllPresent(t *testing.T) {
	log, cfg, tr := newTestDeps(t)

	// Redirect HOME so GetCockpitDir() points to our temp dir.
	tmpDir := t.TempDir()
	cockpitDir := filepath.Join(tmpDir, ".cockpit")
	t.Setenv("HOME", tmpDir)

	// Create all checked paths.
	for _, sub := range []string{"", "vault", "logs", "packages", "cache"} {
		if err := os.MkdirAll(filepath.Join(cockpitDir, sub), 0o755); err != nil {
			t.Fatalf("MkdirAll %s: %v", sub, err)
		}
	}
	// Create the config file too.
	configPath := filepath.Join(cockpitDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("version: 0.1.0\n"), 0o644); err != nil {
		t.Fatalf("WriteFile config: %v", err)
	}

	if err := runDoctor(log, cfg, tr); err != nil {
		t.Errorf("runDoctor() error = %v", err)
	}
}

func TestRunDoctor_MissingDirs(t *testing.T) {
	log, cfg, tr := newTestDeps(t)

	// Use a completely empty HOME — no .cockpit dir at all.
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// runDoctor must NOT return an error even when checks fail —
	// it just prints failures and returns nil.
	if err := runDoctor(log, cfg, tr); err != nil {
		t.Errorf("runDoctor() with missing dirs should return nil, got %v", err)
	}
}

func TestNewDoctorCommand_Execute(t *testing.T) {
	log, cfg, tr := newTestDeps(t)
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cmd := NewDoctorCommand(log, cfg, tr)
	// Execute via cobra — exercises the RunE lambda
	if err := cmd.Execute(); err != nil {
		t.Errorf("doctor Execute error = %v", err)
	}
}
