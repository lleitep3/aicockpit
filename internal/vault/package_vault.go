package vault

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// PackageVault provides vault access for packages with automatic namespace isolation
type PackageVault struct {
	namespace string
}

// NewPackageVault creates a vault instance for a package
// The package name is sanitized to create a safe namespace
func NewPackageVault(packageName string) *PackageVault {
	return &PackageVault{
		namespace: sanitizeNamespace(packageName),
	}
}

// Get retrieves a secret from the package's namespace
// This bypasses lock checks since namespace provides isolation
func (pv *PackageVault) Get(key string) (string, error) {
	cmd := exec.Command("cockpit", "vault", "get", "--namespace", pv.namespace, key)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get secret '%s': %w", key, err)
	}
	return strings.TrimSpace(string(output)), nil
}

// Set stores a secret in the package's namespace
// This bypasses lock checks since namespace provides isolation
func (pv *PackageVault) Set(key, value string) error {
	cmd := exec.Command("cockpit", "vault", "set", "--namespace", pv.namespace, "--value", value, key)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to set secret '%s': %w: %s", key, err, string(output))
	}
	return nil
}

// SetInteractive stores a secret with interactive input (more secure)
// This bypasses lock checks since namespace provides isolation
func (pv *PackageVault) SetInteractive(key string) error {
	cmd := exec.Command("cockpit", "vault", "set", "--namespace", pv.namespace, key)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Remove removes a secret from the package's namespace
// This bypasses lock checks since namespace provides isolation
func (pv *PackageVault) Remove(key string) error {
	cmd := exec.Command("cockpit", "vault", "remove", "--namespace", pv.namespace, key)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to remove secret '%s': %w: %s", key, err, string(output))
	}
	return nil
}

// GetWithDefault retrieves a secret, returning a default if not found
func (pv *PackageVault) GetWithDefault(key, defaultValue string) string {
	value, err := pv.Get(key)
	if err != nil {
		return defaultValue
	}
	return value
}
