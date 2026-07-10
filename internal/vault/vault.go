package vault

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/zalando/go-keyring"
)

const serviceName = "aicockpit"

// Manager defines the interface for interacting with the Vault.
type Manager interface {
	// Set stores a secret value for a given key.
	Set(key string, value string) error

	// Get retrieves the secret value for a given key.
	Get(key string) (string, error)

	// Delete removes the secret for a given key.
	Delete(key string) error

	// ClearAllSecrets removes all secrets (factory reset)
	ClearAllSecrets() error
}

// osVault is the internal implementation that uses the operating system's native keychain.
// This is intentionally unexported (lowercase) to prevent direct access from external packages.
// External packages should use NamespacedVault or CommandHandler for security.
//
// Because go-keyring provides no enumeration API, osVault maintains a lightweight
// JSON index of all keys it has ever written.  The index lives in the user's
// cockpit directory and is the source-of-truth for ClearAllSecrets.
type osVault struct {
	indexPath string
	mu        sync.Mutex
}

// newOSVault creates a new osVault instance (internal use only).
func newOSVault() *osVault {
	return &osVault{indexPath: defaultIndexPath()}
}

func defaultIndexPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), ".cockpit_vault_index.json")
	}
	return filepath.Join(home, ".cockpit", ".vault_index.json")
}

// loadIndex reads the persisted key set from disk.
// Returns an empty set when the file does not exist yet.
func (v *osVault) loadIndex() (map[string]struct{}, error) {
	data, err := os.ReadFile(v.indexPath)
	if os.IsNotExist(err) {
		return make(map[string]struct{}), nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read vault index: %w", err)
	}
	var keys []string
	if err := json.Unmarshal(data, &keys); err != nil {
		// Corrupt index — start fresh rather than blocking the vault entirely.
		return make(map[string]struct{}), nil
	}
	set := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		set[k] = struct{}{}
	}
	return set, nil
}

// saveIndex persists the key set to disk atomically.
func (v *osVault) saveIndex(set map[string]struct{}) error {
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	data, err := json.Marshal(keys)
	if err != nil {
		return fmt.Errorf("failed to marshal vault index: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(v.indexPath), 0o700); err != nil {
		return fmt.Errorf("failed to create vault index directory: %w", err)
	}
	tmp := v.indexPath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("failed to write vault index: %w", err)
	}
	if err := os.Rename(tmp, v.indexPath); err != nil {
		os.Remove(tmp) //nolint:errcheck
		return fmt.Errorf("failed to commit vault index: %w", err)
	}
	return nil
}

// Set securely stores the value in the OS keychain and records the key in the index.
func (v *osVault) Set(key string, value string) error {
	if err := keyring.Set(serviceName, key, value); err != nil {
		return fmt.Errorf("failed to save secret to vault: %w", err)
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	set, err := v.loadIndex()
	if err != nil {
		return err
	}
	set[key] = struct{}{}
	return v.saveIndex(set)
}

// Get retrieves the value from the OS keychain.
func (v *osVault) Get(key string) (string, error) {
	val, err := keyring.Get(serviceName, key)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve secret from vault: %w", err)
	}
	return val, nil
}

// Delete removes the value from the OS keychain and strikes the key from the index.
func (v *osVault) Delete(key string) error {
	if err := keyring.Delete(serviceName, key); err != nil {
		return fmt.Errorf("failed to delete secret from vault: %w", err)
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	set, err := v.loadIndex()
	if err != nil {
		return err
	}
	delete(set, key)
	return v.saveIndex(set)
}

// ClearAllSecrets deletes every key recorded in the index, then removes the
// index file itself.  Any individual delete failure is collected and returned
// as a combined error so the caller knows which keys could not be removed.
func (v *osVault) ClearAllSecrets() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	set, err := v.loadIndex()
	if err != nil {
		return err
	}

	var errs []string
	for key := range set {
		if err := keyring.Delete(serviceName, key); err != nil {
			errs = append(errs, fmt.Sprintf("delete %q: %v", key, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("ClearAllSecrets: %d key(s) could not be deleted: %v", len(errs), errs)
	}

	// Remove the index file — vault is now empty.
	if err := os.Remove(v.indexPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove vault index: %w", err)
	}
	return nil
}

// NewOSVault creates a new OSVault instance.
// DEPRECATED: Use NewNamespacedVault() instead for better security.
// This method is maintained for backward compatibility but should be avoided in new code.
// Direct access to OSVault allows bypassing namespace isolation and security controls.
func NewOSVault() Manager {
	return &osVault{indexPath: defaultIndexPath()}
}
