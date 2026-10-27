package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zalando/go-keyring"
)

const (
	stateVersionV2       = "v2"
	stateKeyService      = "aicockpit-internal"
	stateKeyPrefix       = "vault-state-v2/"
	maxStateFileSize     = 1 << 20
	stateKeySize         = 32
	stateKeyIDSize       = 16
	stateAADPrefix       = "aicockpit-vault-state\x00"
	defaultLockStatePath = ".cockpit/vault/lock_state.json"
)

var (
	// ErrMigrationRequired is returned for the forgeable v1 state format.
	ErrMigrationRequired = errors.New("vault lock state migration required")
	ErrStateKeyMissing   = errors.New("vault state key is unavailable")
)

// KeyStore is the small keyring surface needed by the lock-state protector.
// The vault credential index deliberately does not see these entries.
type KeyStore interface {
	Get(service, key string) (string, error)
	Set(service, key, value string) error
}

type osKeyStore struct{}

func (osKeyStore) Get(service, key string) (string, error) {
	return keyring.Get(service, key)
}

func (osKeyStore) Set(service, key, value string) error {
	return keyring.Set(service, key, value)
}

// EncryptedState is the on-disk envelope. The legacy fields remain only so
// callers can inspect old fixtures; v1 is never accepted for access.
type EncryptedState struct {
	Version   string `json:"version"`
	KeyID     string `json:"key_id"`
	Data      string `json:"data"`
	Signature string `json:"signature,omitempty"`
	Nonce     string `json:"nonce,omitempty"`
	Salt      string `json:"salt,omitempty"`
}

// DefaultLockStatePath returns the user-scoped state path without embedding a
// particular user's home directory in the binary.
func DefaultLockStatePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), defaultLockStatePath)
	}
	return filepath.Join(home, defaultLockStatePath)
}

// StateEncryptor protects lock state with a random key stored in the OS
// keyring. It has no dependency on the master password: those are separate
// authorization layers with different purposes.
type StateEncryptor struct {
	keyStore KeyStore
	path     string
}

func NewStateEncryptor() *StateEncryptor {
	return NewStateEncryptorAt(DefaultLockStatePath(), osKeyStore{})
}

func NewStateEncryptorAt(path string, keyStore KeyStore) *StateEncryptor {
	if keyStore == nil {
		keyStore = osKeyStore{}
	}
	return &StateEncryptor{keyStore: keyStore, path: path}
}

func (se *StateEncryptor) EncryptAndSign(state *LockState) error {
	if err := validateLockState(state); err != nil {
		return err
	}
	return withStateFileLock(se.path, func() error {
		return se.encryptAndSignUnlocked(state, false)
	})
}

// initializeLockedState creates a new v2 key and blocked envelope. It is used
// only for explicit initialization or migration while the state lock is held.
func (se *StateEncryptor) initializeLockedState() error {
	return se.encryptAndSignUnlocked(defaultLockState(), true)
}

func (se *StateEncryptor) encryptAndSignUnlocked(state *LockState, replaceExisting bool) error {
	var keyID string
	var key []byte
	existing, err := se.readEnvelope()
	if err == nil {
		if existing.Version != stateVersionV2 && !replaceExisting {
			return fmt.Errorf("%w: found %q", ErrMigrationRequired, existing.Version)
		}
		if existing.Version == stateVersionV2 {
			keyID = existing.KeyID
			key, err = se.loadKey(keyID)
			if err != nil {
				return err
			}
		}
	} else if !errors.Is(err, os.ErrNotExist) && !replaceExisting {
		return err
	}
	if len(key) == 0 {
		keyID, key, err = se.createKey()
		if err != nil {
			return err
		}
	}

	plaintext, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal lock state: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("failed to create lock-state cipher: %w", err)
	}
	aead, err := cipher.NewGCMWithRandomNonce(block)
	if err != nil {
		return fmt.Errorf("failed to create lock-state AEAD: %w", err)
	}
	ciphertext := aead.Seal(nil, nil, plaintext, stateAAD(keyID))
	envelope := EncryptedState{
		Version: stateVersionV2,
		KeyID:   keyID,
		Data:    base64.RawURLEncoding.EncodeToString(ciphertext),
	}
	encoded, err := json.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal lock-state envelope: %w", err)
	}
	if err := atomicWriteFile(se.path, encoded, 0o600); err != nil {
		return fmt.Errorf("failed to persist lock state: %w", err)
	}
	return nil
}

func (se *StateEncryptor) DecryptAndVerify() (*LockState, error) {
	envelope, err := se.readEnvelope()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return defaultLockState(), nil
		}
		return nil, err
	}
	if envelope.Version != stateVersionV2 {
		if envelope.Version == "v1" || envelope.Salt != "" || envelope.Signature != "" {
			return nil, fmt.Errorf("%w: %q", ErrMigrationRequired, envelope.Version)
		}
		return nil, fmt.Errorf("unsupported vault lock state version %q", envelope.Version)
	}
	if err := validateEnvelope(envelope); err != nil {
		return nil, err
	}
	key, err := se.loadKey(envelope.KeyID)
	if err != nil {
		return nil, err
	}
	ciphertext, err := base64.RawURLEncoding.DecodeString(envelope.Data)
	if err != nil {
		return nil, fmt.Errorf("invalid lock-state data encoding: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create lock-state cipher: %w", err)
	}
	aead, err := cipher.NewGCMWithRandomNonce(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create lock-state AEAD: %w", err)
	}
	plaintext, err := aead.Open(nil, nil, ciphertext, stateAAD(envelope.KeyID))
	if err != nil {
		return nil, fmt.Errorf("lock-state authentication failed: %w", err)
	}
	var state LockState
	if err := json.Unmarshal(plaintext, &state); err != nil {
		return nil, fmt.Errorf("invalid decrypted lock state: %w", err)
	}
	if err := validateLockState(&state); err != nil {
		return nil, fmt.Errorf("invalid decrypted lock state: %w", err)
	}
	return &state, nil
}

func (se *StateEncryptor) readEnvelope() (EncryptedState, error) {
	data, err := os.ReadFile(se.path)
	if err != nil {
		return EncryptedState{}, fmt.Errorf("failed to read lock state: %w", err)
	}
	if len(data) > maxStateFileSize {
		return EncryptedState{}, fmt.Errorf("lock state exceeds maximum size")
	}
	var envelope EncryptedState
	if err := json.Unmarshal(data, &envelope); err != nil {
		return EncryptedState{}, fmt.Errorf("failed to parse lock-state envelope: %w", err)
	}
	return envelope, nil
}

func (se *StateEncryptor) loadKey(keyID string) ([]byte, error) {
	encoded, err := se.keyStore.Get(stateKeyService, stateKeyPrefix+keyID)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return nil, fmt.Errorf("%w: key id %q not found", ErrStateKeyMissing, keyID)
		}
		return nil, fmt.Errorf("%w: keyring read failed", ErrStateKeyMissing)
	}
	key, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil || len(key) != stateKeySize {
		return nil, fmt.Errorf("%w: invalid key material for id %q", ErrStateKeyMissing, keyID)
	}
	return key, nil
}

func (se *StateEncryptor) createKey() (string, []byte, error) {
	key := make([]byte, stateKeySize)
	if _, err := rand.Read(key); err != nil {
		return "", nil, fmt.Errorf("failed to generate lock-state key: %w", err)
	}
	idBytes := make([]byte, stateKeyIDSize)
	if _, err := rand.Read(idBytes); err != nil {
		return "", nil, fmt.Errorf("failed to generate lock-state key id: %w", err)
	}
	keyID := base64.RawURLEncoding.EncodeToString(idBytes)
	if err := se.keyStore.Set(stateKeyService, stateKeyPrefix+keyID, base64.RawURLEncoding.EncodeToString(key)); err != nil {
		return "", nil, fmt.Errorf("failed to create lock-state key: %w", err)
	}
	return keyID, key, nil
}

func stateAAD(keyID string) []byte {
	return []byte(stateAADPrefix + stateVersionV2 + "\x00" + keyID)
}

func validateEnvelope(envelope EncryptedState) error {
	if envelope.KeyID == "" || strings.ContainsAny(envelope.KeyID, "\r\n") {
		return fmt.Errorf("invalid vault lock state key id")
	}
	keyID, err := base64.RawURLEncoding.DecodeString(envelope.KeyID)
	if err != nil || len(keyID) != stateKeyIDSize {
		return fmt.Errorf("invalid vault lock state key id")
	}
	if envelope.Signature != "" || envelope.Nonce != "" || envelope.Salt != "" {
		return fmt.Errorf("legacy fields are not valid in a v2 lock state")
	}
	data, err := base64.RawURLEncoding.DecodeString(envelope.Data)
	if err != nil {
		return fmt.Errorf("invalid lock-state data encoding: %w", err)
	}
	if len(data) < 28 {
		return fmt.Errorf("lock-state ciphertext is truncated")
	}
	return nil
}

func validateLockState(state *LockState) error {
	if state == nil || state.PackageLocks == nil {
		return fmt.Errorf("lock state must contain a package-lock map")
	}
	if state.IsLocked && state.GlobalUnlock {
		return fmt.Errorf("lock state cannot be locked and globally unlocked")
	}
	if !state.IsLocked && !state.GlobalUnlock {
		return fmt.Errorf("unlocked state must grant global access")
	}
	return nil
}

func defaultLockState() *LockState {
	return &LockState{IsLocked: true, PackageLocks: make(map[string]bool)}
}

// atomicWriteFile preserves the previous state until the complete new file is
// synced and renamed into place. Temp files are unique and never a shared .tmp.
func atomicWriteFile(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".lock-state-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	removeTemp := true
	defer func() {
		if removeTemp {
			_ = os.Remove(tmpName)
		}
	}()
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	removeTemp = false
	if dirFile, err := os.Open(dir); err == nil {
		_ = dirFile.Sync()
		_ = dirFile.Close()
	}
	return nil
}

func withStateFileLock(path string, fn func() error) error {
	lock, err := acquireStateFileLock(path)
	if err != nil {
		return err
	}
	defer lock.Close()
	return fn()
}
