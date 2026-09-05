package vault

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestMigrateStateDiscardsLegacyGrantsAndKeepsBackup(t *testing.T) {
	store := newFakeKeyStore()
	path := filepath.Join(t.TempDir(), "lock_state.json")
	legacy := []byte(`{"version":"v1","data":"legacy-ciphertext","signature":"forged","salt":"public"}`)
	if err := os.WriteFile(path, legacy, 0o600); err != nil {
		t.Fatal(err)
	}
	backup, err := MigrateState(path, store, true)
	if err != nil {
		t.Fatalf("MigrateState: %v", err)
	}
	if backup == "" {
		t.Fatal("expected an exclusive legacy backup")
	}
	backupData, err := os.ReadFile(backup)
	if err != nil || string(backupData) != string(legacy) {
		t.Fatalf("backup mismatch: data=%q err=%v", backupData, err)
	}
	se := NewStateEncryptorAt(path, store)
	state, err := se.DecryptAndVerify()
	if err != nil || !state.IsLocked || state.GlobalUnlock || len(state.PackageLocks) != 0 {
		t.Fatalf("migration must produce blocked empty state: state=%+v err=%v", state, err)
	}
}

func TestMigrateStateRequiresConfirmationOnlyWhenNeeded(t *testing.T) {
	store := newFakeKeyStore()
	path := filepath.Join(t.TempDir(), "lock_state.json")
	if _, err := MigrateState(path, store, false); !errors.Is(err, ErrMigrationConfirmation) {
		t.Fatalf("missing-state initialization should require confirmation, got %v", err)
	}
	if _, err := MigrateState(path, store, true); err != nil {
		t.Fatalf("initialization: %v", err)
	}
	sets := store.sets
	if _, err := MigrateState(path, store, false); err != nil {
		t.Fatalf("valid v2 should be a no-op without confirmation: %v", err)
	}
	if store.sets != sets {
		t.Fatalf("valid v2 no-op must not create another key: before=%d after=%d", sets, store.sets)
	}
}

func TestMigrateStateRejectsUnknownAndPreservesBytes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lock_state.json")
	original := []byte(`{"version":"v9","data":"unknown"}`)
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := MigrateState(path, newFakeKeyStore(), true); err == nil {
		t.Fatal("unknown version must fail")
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != string(original) {
		t.Fatalf("unknown version changed state: %q err=%v", got, err)
	}
}

func TestMigrateStateLegacyWithoutConfirmationPreservesBytes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lock_state.json")
	original := []byte(`{"version":"v1","data":"legacy","signature":"sig"}`)
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := MigrateState(path, newFakeKeyStore(), false); !errors.Is(err, ErrMigrationConfirmation) {
		t.Fatalf("expected confirmation error, got %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != string(original) {
		t.Fatalf("confirmation refusal changed legacy state: %q err=%v", got, err)
	}
}
