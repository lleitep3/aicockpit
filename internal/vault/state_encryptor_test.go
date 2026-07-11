package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zalando/go-keyring"
)

func newTestStateEncryptor(t *testing.T) *StateEncryptor {
	t.Helper()
	keyring.MockInit()
	se := NewStateEncryptor()
	se.path = filepath.Join(t.TempDir(), "lock_state.json")
	return se
}

func readEncryptedState(t *testing.T, path string) *EncryptedState {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var es EncryptedState
	if err := json.Unmarshal(data, &es); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	return &es
}

func TestStateEncryptor_New(t *testing.T) {
	se := newTestStateEncryptor(t)
	if se == nil {
		t.Fatal("expected StateEncryptor, got nil")
	}
	if se.path == "" {
		t.Error("expected non-empty default path")
	}
}

func TestStateEncryptor_EncryptAndSign(t *testing.T) {
	se := newTestStateEncryptor(t)
	state := &LockState{
		IsLocked:     true,
		LockedAt:     time.Now(),
		PackageLocks: map[string]bool{"pkg": true},
		GlobalUnlock: false,
	}

	if err := se.EncryptAndSign(state); err != nil {
		t.Fatalf("EncryptAndSign: %v", err)
	}

	if _, err := os.Stat(se.path); err != nil {
		t.Errorf("expected lock state file to exist: %v", err)
	}

	es := readEncryptedState(t, se.path)
	if es.Version != "v1" {
		t.Errorf("expected version v1, got %q", es.Version)
	}
	if es.Salt == "" {
		t.Error("expected salt to be set")
	}
}

func TestStateEncryptor_DecryptAndVerify_RoundTrip(t *testing.T) {
	se := newTestStateEncryptor(t)
	state := &LockState{
		IsLocked:     false,
		GlobalUnlock: true,
		UnlockReason: "roundtrip",
		PackageLocks: map[string]bool{"a": true, "b": false},
	}

	if err := se.EncryptAndSign(state); err != nil {
		t.Fatalf("EncryptAndSign: %v", err)
	}

	loaded, err := se.DecryptAndVerify()
	if err != nil {
		t.Fatalf("DecryptAndVerify: %v", err)
	}
	if loaded.IsLocked != state.IsLocked {
		t.Errorf("IsLocked mismatch: expected %v, got %v", state.IsLocked, loaded.IsLocked)
	}
	if loaded.GlobalUnlock != state.GlobalUnlock {
		t.Errorf("GlobalUnlock mismatch: expected %v, got %v", state.GlobalUnlock, loaded.GlobalUnlock)
	}
	if loaded.UnlockReason != state.UnlockReason {
		t.Errorf("UnlockReason mismatch: expected %q, got %q", state.UnlockReason, loaded.UnlockReason)
	}
	if !loaded.PackageLocks["a"] || loaded.PackageLocks["b"] {
		t.Errorf("PackageLocks mismatch: got %v", loaded.PackageLocks)
	}

	if salt := se.loadSalt(); salt == "" {
		t.Error("loadSalt should return existing salt after encryption")
	}
}

func TestStateEncryptor_DecryptAndVerify_MissingFile(t *testing.T) {
	se := newTestStateEncryptor(t)

	state, err := se.DecryptAndVerify()
	if err != nil {
		t.Fatalf("DecryptAndVerify: %v", err)
	}
	if !state.IsLocked {
		t.Error("missing file should default to locked state")
	}
	if state.GlobalUnlock {
		t.Error("missing file should default to no global unlock")
	}
}

func TestStateEncryptor_DecryptAndVerify_InvalidJSON(t *testing.T) {
	se := newTestStateEncryptor(t)
	if err := os.WriteFile(se.path, []byte("not-json"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if _, err := se.DecryptAndVerify(); err == nil {
		t.Error("expected error for invalid JSON lock state")
	}
}

func TestStateEncryptor_DecryptAndVerify_TamperedSignature(t *testing.T) {
	se := newTestStateEncryptor(t)
	state := &LockState{IsLocked: true}
	if err := se.EncryptAndSign(state); err != nil {
		t.Fatalf("EncryptAndSign: %v", err)
	}

	es := readEncryptedState(t, se.path)
	sig, err := base64.URLEncoding.DecodeString(es.Signature)
	if err != nil {
		t.Fatalf("DecodeString signature: %v", err)
	}
	sig[0] ^= 0xff
	es.Signature = base64.URLEncoding.EncodeToString(sig)
	data, _ := json.Marshal(es)
	if err := os.WriteFile(se.path, data, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if _, err := se.DecryptAndVerify(); err == nil {
		t.Error("expected error when signature is tampered")
	}
}

func TestStateEncryptor_DecryptAndVerify_TamperedData(t *testing.T) {
	se := newTestStateEncryptor(t)
	state := &LockState{IsLocked: true}
	if err := se.EncryptAndSign(state); err != nil {
		t.Fatalf("EncryptAndSign: %v", err)
	}

	es := readEncryptedState(t, se.path)
	cipherData, err := base64.URLEncoding.DecodeString(es.Data)
	if err != nil {
		t.Fatalf("DecodeString data: %v", err)
	}
	cipherData[0] ^= 0xff
	es.Data = base64.URLEncoding.EncodeToString(cipherData)
	data, _ := json.Marshal(es)
	if err := os.WriteFile(se.path, data, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if _, err := se.DecryptAndVerify(); err == nil {
		t.Error("expected error when encrypted data is tampered")
	}
}

func TestStateEncryptor_DecryptAndVerify_BadDataBase64(t *testing.T) {
	se := newTestStateEncryptor(t)
	es := EncryptedState{Data: "!!!", Signature: "", Nonce: "", Version: "v1", Salt: "s"}
	data, _ := json.Marshal(es)
	if err := os.WriteFile(se.path, data, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if _, err := se.DecryptAndVerify(); err == nil {
		t.Error("expected error for invalid data base64")
	}
}

func TestStateEncryptor_DecryptAndVerify_BadSignatureBase64(t *testing.T) {
	se := newTestStateEncryptor(t)
	es := EncryptedState{Data: base64.URLEncoding.EncodeToString([]byte("x")), Signature: "!!!", Nonce: "", Version: "v1", Salt: "s"}
	data, _ := json.Marshal(es)
	if err := os.WriteFile(se.path, data, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if _, err := se.DecryptAndVerify(); err == nil {
		t.Error("expected error for invalid signature base64")
	}
}

func TestStateEncryptor_DecryptAndVerify_ShortCiphertext(t *testing.T) {
	se := newTestStateEncryptor(t)
	salt := "testsalt"
	shortData := []byte("short")
	sig := se.signData(shortData, salt)
	es := EncryptedState{
		Data:      base64.URLEncoding.EncodeToString(shortData),
		Signature: base64.URLEncoding.EncodeToString(sig),
		Nonce:     base64.URLEncoding.EncodeToString([]byte("nonce")),
		Version:   "v1",
		Salt:      salt,
	}
	data, _ := json.Marshal(es)
	if err := os.WriteFile(se.path, data, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if _, err := se.DecryptAndVerify(); err == nil {
		t.Error("expected error for ciphertext too short")
	}
}

func TestStateEncryptor_GetEncryptionKey(t *testing.T) {
	se := newTestStateEncryptor(t)

	key := se.getEncryptionKey("salt1")
	if len(key) != 32 {
		t.Errorf("expected 32-byte key, got %d", len(key))
	}

	// With master password enabled the code path is the same (deriveFixedKey),
	// but this exercises the IsEnabled branch.
	se.masterPassword.enabled = true
	key2 := se.getEncryptionKey("salt2")
	if len(key2) != 32 {
		t.Errorf("expected 32-byte key, got %d", len(key2))
	}
}

func TestStateEncryptor_DeriveFixedKey(t *testing.T) {
	se := newTestStateEncryptor(t)

	k1 := se.deriveFixedKey("salt-a")
	k2 := se.deriveFixedKey("salt-a")
	k3 := se.deriveFixedKey("salt-b")
	if len(k1) != 32 {
		t.Errorf("expected 32-byte key, got %d", len(k1))
	}
	if string(k1) != string(k2) {
		t.Error("deriveFixedKey should be deterministic for the same salt")
	}
	if string(k1) == string(k3) {
		t.Error("deriveFixedKey should produce different keys for different salts")
	}
}

func TestStateEncryptor_SignAndVerify(t *testing.T) {
	se := newTestStateEncryptor(t)
	data := []byte("data-to-sign")
	salt := "sign-salt"

	sig := se.signData(data, salt)
	if !se.verifySignature(data, sig, salt) {
		t.Error("signature should verify for original data")
	}

	tampered := make([]byte, len(data))
	copy(tampered, data)
	tampered[0] ^= 0xff
	if se.verifySignature(tampered, sig, salt) {
		t.Error("signature should not verify for tampered data")
	}

	tamperedSig := make([]byte, len(sig))
	copy(tamperedSig, sig)
	tamperedSig[0] ^= 0xff
	if se.verifySignature(data, tamperedSig, salt) {
		t.Error("signature should not verify with wrong signature")
	}

	if se.verifySignature(data, sig, "other-salt") {
		t.Error("signature should not verify with wrong salt")
	}
}

func TestStateEncryptor_LoadSalt_InvalidJSON(t *testing.T) {
	se := newTestStateEncryptor(t)
	if err := os.WriteFile(se.path, []byte("bad json"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if salt := se.loadSalt(); salt != "" {
		t.Errorf("expected empty salt for invalid JSON, got %q", salt)
	}
}

func TestStateEncryptor_EncryptAndSign_WriteError(t *testing.T) {
	se := newTestStateEncryptor(t)
	// Point the encryptor at a directory so WriteFile fails.
	se.path = t.TempDir()

	state := &LockState{IsLocked: true}
	if err := se.EncryptAndSign(state); err == nil {
		t.Error("expected error when writing to a directory")
	}
}

func TestStateEncryptor_DecryptAndVerify_ReadFileError(t *testing.T) {
	se := newTestStateEncryptor(t)
	se.path = t.TempDir()

	if _, err := se.DecryptAndVerify(); err == nil {
		t.Error("expected error when lock state path is a directory")
	} else if !strings.Contains(err.Error(), "failed to read encrypted state") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestStateEncryptor_DecryptAndVerify_InvalidPlaintext(t *testing.T) {
	se := newTestStateEncryptor(t)
	salt := "badplaintextsalt"
	key := se.getEncryptionKey(salt)

	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatalf("NewCipher: %v", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatalf("NewGCM: %v", err)
	}

	nonce := make([]byte, 12)
	if _, err := rand.Read(nonce); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}

	plaintext := []byte("this is not valid json")
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	sig := se.signData(ciphertext, salt)

	es := EncryptedState{
		Data:      base64.URLEncoding.EncodeToString(ciphertext),
		Signature: base64.URLEncoding.EncodeToString(sig),
		Nonce:     base64.URLEncoding.EncodeToString(nonce),
		Version:   "v1",
		Salt:      salt,
	}
	data, _ := json.Marshal(es)
	if err := os.WriteFile(se.path, data, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if _, err := se.DecryptAndVerify(); err == nil {
		t.Fatal("expected error for non-JSON plaintext")
	} else if !strings.Contains(err.Error(), "failed to parse decrypted state") {
		t.Errorf("unexpected error: %v", err)
	}
}
