package vault

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestStateEncryptorConstructorAndWriteErrorPaths(t *testing.T) {
	if NewStateEncryptor() == nil {
		t.Fatal("default constructor returned nil")
	}
	se, _ := newTestStateEncryptor(t)
	parent := filepath.Join(t.TempDir(), "parent")
	if err := os.WriteFile(parent, []byte("file"), 0o600); err != nil {
		t.Fatal(err)
	}
	se.path = filepath.Join(parent, "state.json")
	if err := se.EncryptAndSign(testState(true)); err == nil {
		t.Fatal("file parent must make state write fail")
	}
	if err := se.EncryptAndSign(nil); err == nil {
		t.Fatal("nil state must be rejected")
	}
	if err := withStateFileLock(filepath.Join(t.TempDir(), "state.json"), func() error {
		return errors.New("callback failure")
	}); err == nil || err.Error() != "callback failure" {
		t.Fatalf("lock callback error was not returned: %v", err)
	}
}

func TestMasterPasswordRejectsMalformedFlags(t *testing.T) {
	mp := newTestMasterPassword(t)
	for _, payload := range []string{"maybe|hash", "true|"} {
		ciphertext, err := encryptSystemData([]byte(payload))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(mp.storagePath, ciphertext, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := mp.load(); err == nil {
			t.Fatalf("payload %q must be rejected", payload)
		}
	}
}

func TestMigrateStateReadFailure(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "parent")
	if err := os.WriteFile(parent, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := MigrateState(filepath.Join(parent, "state.json"), newFakeKeyStore(), true); err == nil {
		t.Fatal("migration must report a state read failure")
	}
}

func TestOSVaultIndexReadFailure(t *testing.T) {
	v := newTestVault(t)
	indexDir := filepath.Join(t.TempDir(), "index")
	if err := os.Mkdir(indexDir, 0o700); err != nil {
		t.Fatal(err)
	}
	v.indexPath = indexDir
	if err := v.Set("key", "value"); err == nil {
		t.Fatal("set must report index read failure")
	}

	// A malformed but readable index is intentionally treated as empty.
	indexFile := filepath.Join(t.TempDir(), "index.json")
	v.indexPath = indexFile
	data, _ := json.Marshal([]string{"key"})
	if err := os.WriteFile(indexFile, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := v.loadIndex(); err != nil {
		t.Fatal(err)
	}
}

func TestSecureVaultCryptoFailures(t *testing.T) {
	sv := &SecureVault{encryptionKey: []byte("short")}
	if _, err := sv.encrypt("secret"); err == nil {
		t.Error("invalid encryption key must fail")
	}
	if _, err := sv.decrypt("not-base64"); err == nil {
		t.Error("invalid ciphertext encoding must fail")
	}
	sv.encryptionKey = make([]byte, 32)
	if _, err := sv.decrypt("YQ=="); err == nil {
		t.Error("short ciphertext must fail")
	}
}

func TestStateEncryptorRejectsInvalidKeyMaterial(t *testing.T) {
	se, store := newTestStateEncryptor(t)
	if err := se.EncryptAndSign(testState(true)); err != nil {
		t.Fatal(err)
	}
	envelope := readEncryptedState(t, se.path)
	key := stateKeyService + "\x00" + stateKeyPrefix + envelope.KeyID
	store.mu.Lock()
	store.values[key] = "not-base64"
	store.mu.Unlock()
	if _, err := se.DecryptAndVerify(); !errors.Is(err, ErrStateKeyMissing) {
		t.Fatalf("invalid base64 key must fail closed: %v", err)
	}
	store.mu.Lock()
	store.values[key] = "AA"
	store.mu.Unlock()
	if _, err := se.DecryptAndVerify(); !errors.Is(err, ErrStateKeyMissing) {
		t.Fatalf("short key must fail closed: %v", err)
	}
}

func TestLockManagerDoesNotPublishFailedMutation(t *testing.T) {
	lm := newTestLockManager(t)
	if err := lm.Unlock("initial"); err != nil {
		t.Fatal(err)
	}
	before := lm.GetStatus()
	originalPath := lm.encryptor.path
	lm.encryptor.path = t.TempDir()
	if err := lm.Lock("failed"); err == nil {
		t.Fatal("mutation with an invalid encryptor path must fail")
	}
	lm.encryptor.path = originalPath
	after := lm.GetStatus()
	if after.GlobalUnlock != before.GlobalUnlock || after.UnlockReason != before.UnlockReason {
		t.Fatalf("failed mutation was published: before=%+v after=%+v", before, after)
	}
}
