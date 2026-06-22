package vault

import (
	"fmt"

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
}

// OSVault is a Vault Manager that uses the operating system's native keychain.
type OSVault struct{}

// NewOSVault creates a new OSVault instance.
func NewOSVault() *OSVault {
	return &OSVault{}
}

// Set securely stores the value in the OS keychain.
func (v *OSVault) Set(key string, value string) error {
	err := keyring.Set(serviceName, key, value)
	if err != nil {
		return fmt.Errorf("failed to save secret to vault: %w", err)
	}
	return nil
}

// Get retrieves the value from the OS keychain.
func (v *OSVault) Get(key string) (string, error) {
	val, err := keyring.Get(serviceName, key)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve secret from vault: %w", err)
	}
	return val, nil
}

// Delete removes the value from the OS keychain.
func (v *OSVault) Delete(key string) error {
	err := keyring.Delete(serviceName, key)
	if err != nil {
		return fmt.Errorf("failed to delete secret from vault: %w", err)
	}
	return nil
}
