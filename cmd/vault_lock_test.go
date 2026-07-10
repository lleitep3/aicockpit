package cmd

import (
	"testing"
)

func TestNewVaultLockCommand_Constructor(t *testing.T) {
	log, cfg, tr := newTestDeps(t)
	cmd := NewVaultLockCommand(log, cfg, tr)
	if cmd == nil {
		t.Fatal("NewVaultLockCommand() returned nil")
	}
	if cmd.Use != "lock [package]" {
		t.Errorf("Use = %q, want %q", cmd.Use, "lock [package]")
	}
}

func TestNewVaultUnlockCommand_Constructor(t *testing.T) {
	log, cfg, tr := newTestDeps(t)
	cmd := NewVaultUnlockCommand(log, cfg, tr)
	if cmd == nil {
		t.Fatal("NewVaultUnlockCommand() returned nil")
	}
}

func TestNewVaultStatusCommand_Run(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	log, cfg, tr := newTestDeps(t)
	cmd := NewVaultStatusCommand(log, cfg, tr)
	if cmd == nil {
		t.Fatal("NewVaultStatusCommand() returned nil")
	}
	// Must execute without error — status is always readable.
	if err := cmd.RunE(cmd, []string{}); err != nil {
		t.Errorf("vault status RunE error = %v", err)
	}
}

func TestNewVaultSetMasterPasswordCommand_Constructor(t *testing.T) {
	log, cfg, tr := newTestDeps(t)
	cmd := NewVaultSetMasterPasswordCommand(log, cfg, tr)
	if cmd == nil {
		t.Fatal("NewVaultSetMasterPasswordCommand() returned nil")
	}
}

func TestNewVaultDisableMasterPasswordCommand_Constructor(t *testing.T) {
	log, cfg, tr := newTestDeps(t)
	cmd := NewVaultDisableMasterPasswordCommand(log, cfg, tr)
	if cmd == nil {
		t.Fatal("NewVaultDisableMasterPasswordCommand() returned nil")
	}
}

func TestNewVaultChangeMasterPasswordCommand_Constructor(t *testing.T) {
	log, cfg, tr := newTestDeps(t)
	cmd := NewVaultChangeMasterPasswordCommand(log, cfg, tr)
	if cmd == nil {
		t.Fatal("NewVaultChangeMasterPasswordCommand() returned nil")
	}
}

func TestNewVaultFactoryResetCommand_Constructor(t *testing.T) {
	log, cfg, tr := newTestDeps(t)
	cmd := NewVaultFactoryResetCommand(log, cfg, tr)
	if cmd == nil {
		t.Fatal("NewVaultFactoryResetCommand() returned nil")
	}
}
