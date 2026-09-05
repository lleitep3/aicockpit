package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lleitep3/aicockpit/internal/vault"
)

func TestVaultMigrateStateCommandConfirmed(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	statePath := vault.DefaultLockStatePath()
	if err := os.MkdirAll(filepath.Dir(statePath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(statePath, []byte(`{"version":"v1","data":"legacy","signature":"sig"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	log, cfg, tr := newTestDeps(t)
	cmd := NewVaultMigrateStateCommand(log, cfg, tr)
	cmd.SetArgs([]string{"--confirm"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("migrate-state: %v", err)
	}
	state, err := vault.NewLockManager("").GetStatusWithError()
	if err != nil || !state.IsLocked || state.GlobalUnlock {
		t.Fatalf("migration command did not leave vault blocked: status=%+v err=%v", state, err)
	}
}
