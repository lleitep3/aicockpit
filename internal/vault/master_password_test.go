package vault

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"

	"github.com/zalando/go-keyring"
)

func newTestMasterPassword(t *testing.T) *MasterPassword {
	t.Helper()
	keyring.MockInit()
	mp := &MasterPassword{
		storagePath: filepath.Join(t.TempDir(), "master_password.dat"),
		enabled:     false,
	}
	_ = mp.load()
	return mp
}

func TestNewMasterPassword(t *testing.T) {
	keyring.MockInit()
	mp := NewMasterPassword()
	if mp == nil {
		t.Fatal("expected MasterPassword, got nil")
	}
}

func TestMasterPassword_SetAndValidate(t *testing.T) {
	mp := newTestMasterPassword(t)

	password := "super-secret-password"
	if err := mp.SetPassword(password); err != nil {
		t.Fatalf("SetPassword: %v", err)
	}
	if !mp.IsEnabled() {
		t.Error("password should be enabled after SetPassword")
	}
	if !mp.Validate(password) {
		t.Error("Validate should succeed with correct password")
	}
	if mp.Validate("wrong-password") {
		t.Error("Validate should fail with wrong password")
	}

	// Validation of a disabled password always succeeds.
	mp2 := newTestMasterPassword(t)
	if !mp2.Validate("anything") {
		t.Error("Validate should succeed when master password is disabled")
	}
}

func TestMasterPassword_EnableDisable(t *testing.T) {
	mp := newTestMasterPassword(t)

	if err := mp.Enable("enable-pass"); err != nil {
		t.Fatalf("Enable: %v", err)
	}
	if !mp.IsEnabled() {
		t.Error("expected enabled after Enable")
	}

	if err := mp.Disable(); err != nil {
		t.Fatalf("Disable: %v", err)
	}
	if mp.IsEnabled() {
		t.Error("expected disabled after Disable")
	}
}

func TestMasterPassword_ChangePassword(t *testing.T) {
	mp := newTestMasterPassword(t)

	oldPwd := "old-password"
	newPwd := "new-password"

	if err := mp.SetPassword(oldPwd); err != nil {
		t.Fatalf("SetPassword: %v", err)
	}

	if err := mp.ChangePassword("wrong", newPwd); err == nil {
		t.Error("ChangePassword should fail with invalid old password")
	}

	if err := mp.ChangePassword(oldPwd, newPwd); err != nil {
		t.Fatalf("ChangePassword: %v", err)
	}
	if !mp.Validate(newPwd) {
		t.Error("new password should validate after change")
	}
	if mp.Validate(oldPwd) {
		t.Error("old password should not validate after change")
	}
}

func TestMasterPassword_ForceSet(t *testing.T) {
	mp := newTestMasterPassword(t)

	if err := mp.SetPassword("first"); err != nil {
		t.Fatalf("SetPassword: %v", err)
	}
	if err := mp.ForceSet("second"); err != nil {
		t.Fatalf("ForceSet: %v", err)
	}
	if !mp.Validate("second") {
		t.Error("ForceSet should override existing password")
	}
}

func TestMasterPassword_Persistence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "master_password.dat")
	keyring.MockInit()

	mp1 := &MasterPassword{storagePath: path, enabled: false}
	_ = mp1.load()
	if err := mp1.SetPassword("persisted"); err != nil {
		t.Fatalf("SetPassword: %v", err)
	}

	mp2 := &MasterPassword{storagePath: path, enabled: false}
	_ = mp2.load()
	if !mp2.IsEnabled() {
		t.Error("enabled state should persist")
	}
	if !mp2.Validate("persisted") {
		t.Error("password should validate after reload")
	}
}

func TestMasterPassword_UsesVersionedPasswordKDF(t *testing.T) {
	mp := newTestMasterPassword(t)
	if err := mp.SetPassword("kdf-protected-password"); err != nil {
		t.Fatalf("SetPassword: %v", err)
	}

	ciphertext, err := os.ReadFile(mp.storagePath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	plaintext, err := decryptSystemData(ciphertext)
	if err != nil {
		t.Fatalf("decryptSystemData: %v", err)
	}
	parts := strings.Split(string(plaintext), "|")
	if len(parts) != 2 || !strings.HasPrefix(parts[1], masterPasswordVerifierVersion+"$") {
		t.Fatalf("expected versioned password verifier, got %q", string(plaintext))
	}
	if mp.Validate("kdf-protected-password") == false || mp.Validate("wrong-password") {
		t.Fatal("password verifier validation result is incorrect")
	}
}

func TestMasterPassword_RejectsLegacyFastVerifier(t *testing.T) {
	path := filepath.Join(t.TempDir(), "master_password.dat")
	ciphertext, err := encryptSystemData([]byte("true|legacy-sha256-verifier"))
	if err != nil {
		t.Fatalf("encryptSystemData: %v", err)
	}
	if err := os.WriteFile(path, ciphertext, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	mp := NewMasterPasswordAt(path)
	if mp.InitializationError() == nil {
		t.Fatal("legacy fast verifier must be rejected")
	}
	if mp.Validate("legacy-password") {
		t.Fatal("legacy fast verifier must fail closed")
	}
}

func TestMasterPassword_LoadCorruptFile(t *testing.T) {
	mp := newTestMasterPassword(t)

	if err := os.WriteFile(mp.storagePath, []byte("corrupt-data"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := mp.load(); err == nil {
		t.Fatal("corrupt file should return an error")
	}
	if mp.IsEnabled() {
		t.Error("corrupt file must not enable a password")
	}
}

func TestMasterPassword_LoadInvalidFormat(t *testing.T) {
	mp := newTestMasterPassword(t)

	// Encrypt something valid but with the wrong internal format.
	cipherData, err := encryptSystemData([]byte("no-pipe"))
	if err != nil {
		t.Fatalf("encryptSystemData: %v", err)
	}
	if err := os.WriteFile(mp.storagePath, cipherData, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := mp.load(); err == nil {
		t.Fatal("invalid format should return an error")
	}
	if mp.IsEnabled() {
		t.Error("invalid format must not enable a password")
	}
}

func TestMasterPassword_ConstructorFailsClosedOnCorruption(t *testing.T) {
	path := filepath.Join(t.TempDir(), "master_password.dat")
	if err := os.WriteFile(path, []byte("corrupt"), 0o600); err != nil {
		t.Fatal(err)
	}
	mp := NewMasterPasswordAt(path)
	if mp.InitializationError() == nil {
		t.Fatal("constructor must retain corruption error")
	}
	if mp.Validate("anything") {
		t.Fatal("corrupt master-password state must not validate")
	}
}

func TestMasterPassword_DeriveSystemKey(t *testing.T) {
	key1 := deriveSystemKey()
	key2 := deriveSystemKey()
	if len(key1) != 32 {
		t.Errorf("expected 32-byte key, got %d", len(key1))
	}
	if string(key1) != string(key2) {
		t.Error("deriveSystemKey should be deterministic within the same process")
	}
}

func TestMasterPassword_EncryptDecryptRoundTrip(t *testing.T) {
	plaintext := []byte("hello world")
	encrypted, err := encryptSystemData(plaintext)
	if err != nil {
		t.Fatalf("encryptSystemData: %v", err)
	}
	decrypted, err := decryptSystemData(encrypted)
	if err != nil {
		t.Fatalf("decryptSystemData: %v", err)
	}
	if string(decrypted) != string(plaintext) {
		t.Errorf("expected %q, got %q", plaintext, decrypted)
	}
}

func TestMasterPassword_DecryptSystemDataErrors(t *testing.T) {
	// Invalid base64 should fail.
	if _, err := decryptSystemData([]byte("!!!")); err == nil {
		t.Error("expected error for invalid base64")
	}

	// Too-short ciphertext should fail.
	short := make([]byte, 8)
	encoded := []byte(base64.URLEncoding.EncodeToString(short))
	if _, err := decryptSystemData(encoded); err == nil {
		t.Error("expected error for short ciphertext")
	}

	// Tampered ciphertext should fail.
	encrypted, err := encryptSystemData([]byte("secret"))
	if err != nil {
		t.Fatalf("encryptSystemData: %v", err)
	}
	encrypted[len(encrypted)/2] ^= 0xff
	if _, err := decryptSystemData(encrypted); err == nil {
		t.Error("expected error for tampered ciphertext")
	}
}

func TestPromptPassword_NonTerminal(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("stdin redirection test skipped on windows")
	}

	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Pipe: %v", err)
	}
	// Capture current fd 0 so we can restore it.
	oldFd0, err := syscall.Dup(0)
	if err != nil {
		t.Fatalf("Dup: %v", err)
	}
	if err := syscall.Dup2(int(r.Fd()), 0); err != nil {
		t.Fatalf("Dup2: %v", err)
	}
	os.Stdin = r

	defer func() {
		_ = syscall.Dup2(oldFd0, 0)
		_ = syscall.Close(oldFd0)
		os.Stdin = oldStdin
		_ = r.Close()
		_ = w.Close()
	}()

	if _, err := PromptPassword(); err == nil {
		t.Error("expected error when stdin is not a terminal")
	}
}

func TestPromptAndValidate_NonTerminal(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("stdin redirection test skipped on windows")
	}

	mp := newTestMasterPassword(t)
	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Pipe: %v", err)
	}
	oldFd0, err := syscall.Dup(0)
	if err != nil {
		t.Fatalf("Dup: %v", err)
	}
	if err := syscall.Dup2(int(r.Fd()), 0); err != nil {
		t.Fatalf("Dup2: %v", err)
	}
	os.Stdin = r

	defer func() {
		_ = syscall.Dup2(oldFd0, 0)
		_ = syscall.Close(oldFd0)
		os.Stdin = oldStdin
		_ = r.Close()
		_ = w.Close()
	}()

	// When reading the password fails, PromptAndValidate should return the error.
	if err := mp.PromptAndValidate(); err == nil {
		t.Error("expected error when stdin is not a terminal")
	}
}

func TestMasterPassword_RejectsUnsafeOverwriteAndSaveFailures(t *testing.T) {
	mp := newTestMasterPassword(t)
	if err := mp.SetPassword("first-password"); err != nil {
		t.Fatal(err)
	}
	if err := mp.SetPassword("second-password"); err == nil {
		t.Fatal("SetPassword must not overwrite an enabled password")
	}
	mp.storagePath = t.TempDir()
	if err := mp.ChangePassword("first-password", "second-password"); err == nil {
		t.Fatal("ChangePassword must report persistence failure")
	}
	if err := mp.ForceSet("recovery-password"); err == nil {
		t.Fatal("ForceSet must report persistence failure")
	}
	if err := mp.Disable(); err == nil {
		t.Fatal("Disable must report persistence failure")
	}
}
