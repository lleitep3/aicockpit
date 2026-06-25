package vault

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

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
type osVault struct{}

// newOSVault creates a new osVault instance (internal use only).
func newOSVault() *osVault {
	return &osVault{}
}

// Set securely stores the value in the OS keychain.
func (v *osVault) Set(key string, value string) error {
	err := keyring.Set(serviceName, key, value)
	if err != nil {
		return fmt.Errorf("failed to save secret to vault: %w", err)
	}
	return nil
}

// Get retrieves the value from the OS keychain.
func (v *osVault) Get(key string) (string, error) {
	val, err := keyring.Get(serviceName, key)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve secret from vault: %w", err)
	}
	return val, nil
}

// Delete removes the value from the OS keychain.
func (v *osVault) Delete(key string) error {
	err := keyring.Delete(serviceName, key)
	if err != nil {
		return fmt.Errorf("failed to delete secret from vault: %w", err)
	}
	return nil
}

// NewOSVault creates a new OSVault instance.
// DEPRECATED: Use NewNamespacedVault() instead for better security.
// This method is maintained for backward compatibility but should be avoided in new code.
// Direct access to OSVault allows bypassing namespace isolation and security controls.
func NewOSVault() Manager {
	return &osVault{}
}

// ClearAllSecrets removes all secrets from the vault (factory reset)
func (v *osVault) ClearAllSecrets() error {
	// Note: go-keyring doesn't provide a way to list all keys
	// This is a limitation of the underlying keyring systems
	// For a true factory reset, we would need platform-specific implementations

	// For Linux with gnome-keyring, we could use secret-tool
	// For macOS with Keychain, we could use security command
	// For Windows with Credential Manager, we could use cmdkey

	// For now, we'll provide a platform-specific implementation
	return clearAllSecretsPlatform()
}

// clearAllSecretsPlatform provides platform-specific implementation
func clearAllSecretsPlatform() error {
	if runtime.GOOS == "linux" {
		return clearAllSecretsLinux()
	} else if runtime.GOOS == "darwin" {
		return clearAllSecretsMacOS()
	} else if runtime.GOOS == "windows" {
		return clearAllSecretsWindows()
	}

	return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
}

// clearAllSecretsLinux clears secrets using secret-tool (gnome-keyring)
func clearAllSecretsLinux() error {
	// Try to use secret-tool to list and delete all aicockpit secrets
	cmd := exec.Command("secret-tool", "search", "service", "aicockpit")
	output, err := cmd.Output()
	if err != nil {
		// secret-tool might not be available
		return fmt.Errorf("secret-tool not available: %w", err)
	}

	// Parse output and delete each secret
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "aicockpit") {
			// Extract the label/attribute to delete
			// This is simplified - in production you'd need better parsing
			parts := strings.Fields(line)
			if len(parts) > 0 {
				label := parts[len(parts)-1]
				deleteCmd := exec.Command("secret-tool", "clear", "service", "aicockpit", "label", label)
				deleteCmd.Run()
			}
		}
	}

	return nil
}

// clearAllSecretsMacOS clears secrets using security command
func clearAllSecretsMacOS() error {
	// Use security command to delete all aicockpit entries
	cmd := exec.Command("security", "delete-generic-password", "-s", "aicockpit")
	return cmd.Run()
}

// clearAllSecretsWindows clears secrets using cmdkey
func clearAllSecretsWindows() error {
	// Use cmdkey to delete all aicockpit entries
	cmd := exec.Command("cmdkey", "/delete:aicockpit")
	return cmd.Run()
}
