package vault

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeKeyStore struct {
	mu     sync.Mutex
	values map[string]string
	getErr error
	setErr error
	gets   int
	sets   int
}

func newFakeKeyStore() *fakeKeyStore { return &fakeKeyStore{values: make(map[string]string)} }

func (s *fakeKeyStore) Get(service, key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gets++
	if s.getErr != nil {
		return "", s.getErr
	}
	value, ok := s.values[service+"\x00"+key]
	if !ok {
		return "", errors.New("secret not found in keyring")
	}
	return value, nil
}

func (s *fakeKeyStore) Set(service, key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sets++
	if s.setErr != nil {
		return s.setErr
	}
	s.values[service+"\x00"+key] = value
	return nil
}

func newTestStateEncryptor(t *testing.T) (*StateEncryptor, *fakeKeyStore) {
	t.Helper()
	store := newFakeKeyStore()
	return NewStateEncryptorAt(filepath.Join(t.TempDir(), "lock_state.json"), store), store
}

func readEncryptedState(t *testing.T, path string) EncryptedState {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var envelope EncryptedState
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	return envelope
}

func testState(global bool) *LockState {
	return &LockState{IsLocked: !global, GlobalUnlock: global, PackageLocks: map[string]bool{"pkg": true}}
}

func TestStateEncryptorV2RoundTripAndRandomizedCiphertext(t *testing.T) {
	se, store := newTestStateEncryptor(t)
	state := testState(true)
	if err := se.EncryptAndSign(state); err != nil {
		t.Fatalf("EncryptAndSign: %v", err)
	}
	envelope := readEncryptedState(t, se.path)
	if envelope.Version != stateVersionV2 || envelope.KeyID == "" || envelope.Data == "" {
		t.Fatalf("invalid v2 envelope: %+v", envelope)
	}
	if envelope.Signature != "" || envelope.Nonce != "" || envelope.Salt != "" {
		t.Fatalf("v2 must not contain legacy authentication fields: %+v", envelope)
	}
	if store.sets != 1 {
		t.Fatalf("expected one key creation, got %d", store.sets)
	}
	loaded, err := se.DecryptAndVerify()
	if err != nil || loaded.GlobalUnlock != state.GlobalUnlock || loaded.UnlockReason != state.UnlockReason {
		t.Fatalf("round trip failed: state=%+v err=%v", loaded, err)
	}
	second := NewStateEncryptorAt(filepath.Join(filepath.Dir(se.path), "second.json"), store)
	if err := second.EncryptAndSign(state); err != nil {
		t.Fatalf("second EncryptAndSign: %v", err)
	}
	if readEncryptedState(t, se.path).Data == readEncryptedState(t, second.path).Data {
		t.Error("AES-GCM ciphertext should use a fresh random nonce")
	}
}

func TestStateEncryptorMissingFileDoesNotCreateKey(t *testing.T) {
	se, store := newTestStateEncryptor(t)
	state, err := se.DecryptAndVerify()
	if err != nil || !state.IsLocked || state.GlobalUnlock || store.gets != 0 || store.sets != 0 {
		t.Fatalf("missing state must be blocked without key creation: state=%+v err=%v gets=%d sets=%d", state, err, store.gets, store.sets)
	}
}

func TestStateEncryptorRejectsTamperingAndWrongKey(t *testing.T) {
	se, store := newTestStateEncryptor(t)
	if err := se.EncryptAndSign(testState(true)); err != nil {
		t.Fatal(err)
	}
	envelope := readEncryptedState(t, se.path)
	ciphertext, err := base64.RawURLEncoding.DecodeString(envelope.Data)
	if err != nil {
		t.Fatal(err)
	}
	ciphertext[len(ciphertext)-1] ^= 1
	envelope.Data = base64.RawURLEncoding.EncodeToString(ciphertext)
	writeEnvelope(t, se.path, envelope)
	if _, err := se.DecryptAndVerify(); err == nil {
		t.Error("tampered ciphertext must fail")
	}

	if err := se.EncryptAndSign(testState(true)); err != nil {
		t.Fatal(err)
	}
	envelope = readEncryptedState(t, se.path)
	keyName := stateKeyService + "\x00" + stateKeyPrefix + envelope.KeyID
	store.mu.Lock()
	store.values[keyName] = base64.RawURLEncoding.EncodeToString(make([]byte, stateKeySize))
	store.mu.Unlock()
	if _, err := se.DecryptAndVerify(); err == nil {
		t.Error("wrong key must fail authentication")
	}
}

func TestStateEncryptorRejectsMalformedEnvelope(t *testing.T) {
	se, _ := newTestStateEncryptor(t)
	cases := []struct {
		name string
		env  EncryptedState
	}{
		{name: "unknown version", env: EncryptedState{Version: "v99", KeyID: "id", Data: "AA"}},
		{name: "bad base64", env: EncryptedState{Version: stateVersionV2, KeyID: "id", Data: "!"}},
		{name: "truncated", env: EncryptedState{Version: stateVersionV2, KeyID: "id", Data: base64.RawURLEncoding.EncodeToString([]byte("short"))}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			writeEnvelope(t, se.path, tc.env)
			if _, err := se.DecryptAndVerify(); err == nil {
				t.Fatal("expected malformed envelope error")
			}
		})
	}
	if err := os.WriteFile(se.path, []byte(strings.Repeat("x", maxStateFileSize+1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := se.DecryptAndVerify(); err == nil {
		t.Error("oversized envelope must fail before parsing")
	}
}

func TestStateEncryptorKeyringFailuresDenyAccess(t *testing.T) {
	se, store := newTestStateEncryptor(t)
	store.setErr = errors.New("keyring unavailable")
	if err := se.EncryptAndSign(testState(true)); err == nil {
		t.Fatal("key creation failure must be returned")
	}
	if _, err := se.DecryptAndVerify(); err != nil {
		t.Fatalf("missing file should remain safely blocked: %v", err)
	}
	store.setErr = nil
	if err := se.EncryptAndSign(testState(true)); err != nil {
		t.Fatal(err)
	}
	store.getErr = errors.New("keyring locked")
	if _, err := se.DecryptAndVerify(); !errors.Is(err, ErrStateKeyMissing) {
		t.Fatalf("expected unavailable key error, got %v", err)
	}
}

func TestStateEncryptorLegacyRequiresMigration(t *testing.T) {
	se, _ := newTestStateEncryptor(t)
	writeEnvelope(t, se.path, EncryptedState{Version: "v1", Data: "legacy", Signature: "sig", Salt: "public"})
	if _, err := se.DecryptAndVerify(); !errors.Is(err, ErrMigrationRequired) {
		t.Fatalf("expected migration error, got %v", err)
	}
	if err := se.EncryptAndSign(testState(true)); !errors.Is(err, ErrMigrationRequired) {
		t.Fatalf("must not overwrite legacy state implicitly, got %v", err)
	}
}

func TestStateEncryptorRejectsV2LegacyFieldsAndBadKeyID(t *testing.T) {
	se, _ := newTestStateEncryptor(t)
	for _, envelope := range []EncryptedState{
		{Version: stateVersionV2, KeyID: "not-base64", Data: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"},
		{Version: stateVersionV2, KeyID: base64.RawURLEncoding.EncodeToString(make([]byte, stateKeyIDSize)), Data: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", Salt: "old"},
	} {
		writeEnvelope(t, se.path, envelope)
		if _, err := se.DecryptAndVerify(); err == nil {
			t.Fatal("invalid v2 envelope must be rejected")
		}
	}
}

func TestStateEncryptorAtomicWriteFailurePreservesPreviousFile(t *testing.T) {
	se, _ := newTestStateEncryptor(t)
	if err := se.EncryptAndSign(testState(true)); err != nil {
		t.Fatal(err)
	}
	target := t.TempDir()
	se.path = target
	if err := se.EncryptAndSign(testState(true)); err == nil {
		t.Fatal("directory target must fail")
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatal(err)
	}
}

func writeEnvelope(t *testing.T, path string, envelope EncryptedState) {
	t.Helper()
	data, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestValidateLockState(t *testing.T) {
	if err := validateLockState(nil); err == nil {
		t.Error("nil state must fail")
	}
	if err := validateLockState(&LockState{IsLocked: false, PackageLocks: map[string]bool{}}); err == nil {
		t.Error("unlocked state without global grant must fail")
	}
	if err := validateLockState(&LockState{IsLocked: true, GlobalUnlock: true, PackageLocks: map[string]bool{}}); err == nil {
		t.Error("contradictory state must fail")
	}
	_ = time.Time{}
}
