package vault

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// NamespacedVault provides namespace isolation for vault access
// Each application/pacakge can only access secrets in its own namespace
type NamespacedVault struct {
	namespace string
	osVault   *osVault
}

// NewNamespacedVault creates a vault for a specific application namespace
func NewNamespacedVault(appID string) *NamespacedVault {
	return &NamespacedVault{
		namespace: sanitizeNamespace(appID),
		osVault:   newOSVault(),
	}
}

// NewNamespacedVaultFromProcess automatically detects the namespace based on the process
// It uses the executable name as the namespace
func NewNamespacedVaultFromProcess() *NamespacedVault {
	exePath, err := os.Executable()
	if err != nil {
		// Fallback to a default namespace if we can't detect
		return NewNamespacedVault("unknown")
	}

	appName := filepath.Base(exePath)
	// Remove extension if present
	appName = strings.TrimSuffix(appName, filepath.Ext(appName))

	return NewNamespacedVault(appName)
}

// NewNamespacedVaultFromEnv creates a vault using namespace from environment variable
// The app can set COCKPIT_APP_ID to define its namespace
func NewNamespacedVaultFromEnv() *NamespacedVault {
	appID := os.Getenv("COCKPIT_APP_ID")
	if appID == "" {
		// Fallback to process detection
		return NewNamespacedVaultFromProcess()
	}

	return NewNamespacedVault(appID)
}

// Set stores a secret in the application's namespace
func (nv *NamespacedVault) Set(key string, value string) error {
	namespacedKey := nv.namespacedKey(key)
	return nv.osVault.Set(namespacedKey, value)
}

// Get retrieves a secret from the application's namespace
func (nv *NamespacedVault) Get(key string) (string, error) {
	namespacedKey := nv.namespacedKey(key)
	return nv.osVault.Get(namespacedKey)
}

// Delete removes a secret from the application's namespace
func (nv *NamespacedVault) Delete(key string) error {
	namespacedKey := nv.namespacedKey(key)
	return nv.osVault.Delete(namespacedKey)
}

// ListSecrets returns all keys in the application's namespace
// This is a convenience method for debugging and management
func (nv *NamespacedVault) ListSecrets() ([]string, error) {
	// Note: The underlying keyring doesn't support listing,
	// so this would need to be implemented with a caching mechanism
	// or by maintaining a separate index
	return nil, fmt.Errorf("listing secrets not supported by underlying keyring")
}

// GetNamespace returns the current namespace
func (nv *NamespacedVault) GetNamespace() string {
	return nv.namespace
}

// namespacedKey creates the full key with namespace prefix
func (nv *NamespacedVault) namespacedKey(key string) string {
	return fmt.Sprintf("%s:%s", nv.namespace, key)
}

// sanitizeNamespace ensures the namespace is safe and consistent
func sanitizeNamespace(appID string) string {
	// Remove any special characters and convert to lowercase
	appID = strings.ToLower(appID)
	appID = strings.ReplaceAll(appID, " ", "_")
	appID = strings.ReplaceAll(appID, "/", "_")
	appID = strings.ReplaceAll(appID, "\\", "_")

	// Limit length
	if len(appID) > 64 {
		appID = appID[:64]
	}

	// Ensure it's not empty
	if appID == "" {
		appID = "default"
	}

	return appID
}

// ClearAllSecrets removes all secrets from the vault (factory reset)
func (nv *NamespacedVault) ClearAllSecrets() error {
	return nv.osVault.ClearAllSecrets()
}
