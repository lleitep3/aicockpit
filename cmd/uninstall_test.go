package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewUninstallCommand(t *testing.T) {
	log, cfg, tr := newTestDeps(t)
	cmd := NewUninstallCommand(log, cfg, tr)
	if cmd == nil {
		t.Fatal("NewUninstallCommand() returned nil")
	}
	if cmd.Use != "uninstall" {
		t.Errorf("Use = %q, want %q", cmd.Use, "uninstall")
	}
}

func TestRunUninstall_Cancel(t *testing.T) {
	log, cfg, tr := newTestDeps(t)

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	withStdin(t, "n", func() {
		if err := runUninstall(log, cfg, tr); err != nil {
			t.Errorf("runUninstall (cancel) error = %v", err)
		}
	})
}

func TestRunUninstall_Confirm(t *testing.T) {
	log, cfg, tr := newTestDeps(t)

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create cockpit dir so RemoveAll actually removes something
	cockpitDir := filepath.Join(tmpDir, ".cockpit")
	if err := os.MkdirAll(cockpitDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	withStdin(t, "y", func() {
		if err := runUninstall(log, cfg, tr); err != nil {
			t.Errorf("runUninstall (confirm) error = %v", err)
		}
	})

	// Verify the dir was removed
	if _, err := os.Stat(cockpitDir); !os.IsNotExist(err) {
		t.Error("expected cockpit dir to be removed after uninstall")
	}
}

func TestRunUninstall_ConfirmPortuguese(t *testing.T) {
	log, cfg, tr := newTestDeps(t)

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	withStdin(t, "sim", func() {
		if err := runUninstall(log, cfg, tr); err != nil {
			t.Errorf("runUninstall (sim) error = %v", err)
		}
	})
}

func TestRunUninstall_ConfirmS(t *testing.T) {
	log, cfg, tr := newTestDeps(t)

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	withStdin(t, "s", func() {
		if err := runUninstall(log, cfg, tr); err != nil {
			t.Errorf("runUninstall (s) error = %v", err)
		}
	})
}

func TestRunUninstall_ConfirmYes(t *testing.T) {
	log, cfg, tr := newTestDeps(t)

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	withStdin(t, "yes", func() {
		if err := runUninstall(log, cfg, tr); err != nil {
			t.Errorf("runUninstall (yes) error = %v", err)
		}
	})
}

func TestNewUninstallCommand_Execute(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	log, cfg, tr := newTestDeps(t)
	cmd := NewUninstallCommand(log, cfg, tr)

	// Provide "n" to cancel — exercises the RunE lambda
	withStdin(t, "n", func() {
		if err := cmd.Execute(); err != nil {
			t.Errorf("uninstall Execute error = %v", err)
		}
	})
}

func TestRunUninstall_RemoveAllFails(t *testing.T) {
	log, cfg, tr := newTestDeps(t)

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create cockpit dir with read-only parent to prevent removal
	cockpitDir := filepath.Join(tmpDir, ".cockpit")
	os.MkdirAll(filepath.Join(cockpitDir, "subdir"), 0o755)

	// Make the subdir unremovable by removing write perm on parent
	os.Chmod(cockpitDir, 0o555)
	t.Cleanup(func() { os.Chmod(cockpitDir, 0o755) })

	withStdin(t, "y", func() {
		err := runUninstall(log, cfg, tr)
		if err == nil {
			// Some OS/filesystems allow root to bypass permissions
			t.Log("RemoveAll succeeded despite read-only parent (running as root?)")
		}
	})
}

// ── getCurrentProcessName ─────────────────────────────────────────────────

func TestGetCurrentProcessName(t *testing.T) {
	name := getCurrentProcessName()
	// Must return a non-empty string in all cases.
	if name == "" {
		t.Error("getCurrentProcessName() returned empty string")
	}
}
