package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// SecureVault provides enhanced security with encryption and identity verification
// This is the recommended way to access vault in production environments
type SecureVault struct {
	namespace     string
	osVault       *osVault
	encryptionKey []byte
	verifiedAppID string
}

// NewSecureVault creates a vault with enhanced security features
func NewSecureVault(appID string) (*SecureVault, error) {
	// Verify process identity
	verifiedID, err := verifyProcessIdentity(appID)
	if err != nil {
		return nil, fmt.Errorf("identity verification failed: %w", err)
	}

	// Generate encryption key based on app ID and system
	encryptionKey := generateEncryptionKey(verifiedID)

	return &SecureVault{
		namespace:     sanitizeNamespace(appID),
		osVault:       newOSVault(),
		encryptionKey: encryptionKey,
		verifiedAppID: verifiedID,
	}, nil
}

// Set stores a secret with encryption
func (sv *SecureVault) Set(key string, value string) error {
	// Encrypt the value before storing
	encrypted, err := sv.encrypt(value)
	if err != nil {
		return fmt.Errorf("encryption failed: %w", err)
	}

	namespacedKey := sv.namespacedKey(key)
	return sv.osVault.Set(namespacedKey, encrypted)
}

// Get retrieves and decrypts a secret
func (sv *SecureVault) Get(key string) (string, error) {
	namespacedKey := sv.namespacedKey(key)

	encrypted, err := sv.osVault.Get(namespacedKey)
	if err != nil {
		return "", err
	}

	// Try to decrypt
	decrypted, err := sv.decrypt(encrypted)
	if err != nil {
		// If decryption fails, it might be stored without encryption (backward compatibility)
		// Try returning the raw value
		return encrypted, nil
	}

	return decrypted, nil
}

// Delete removes a secret
func (sv *SecureVault) Delete(key string) error {
	namespacedKey := sv.namespacedKey(key)
	return sv.osVault.Delete(namespacedKey)
}

// encrypt encrypts a value using AES-GCM
func (sv *SecureVault) encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(sv.encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.URLEncoding.EncodeToString(ciphertext), nil
}

// decrypt decrypts a value using AES-GCM
func (sv *SecureVault) decrypt(ciphertext string) (string, error) {
	data, err := base64.URLEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(sv.encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, cipherData := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, cipherData, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// namespacedKey creates the full key with namespace and version
func (sv *SecureVault) namespacedKey(key string) string {
	return fmt.Sprintf("v2:%s:%s", sv.namespace, key)
}

// verifyProcessIdentity verifies that the caller's identity matches the claimed app ID
func verifyProcessIdentity(claimedAppID string) (string, error) {
	// Get actual process information
	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}

	// Extract actual app ID from executable path
	actualAppID := extractAppIDFromPath(exePath)

	// Normalize both IDs for comparison
	claimedID := sanitizeNamespace(claimedAppID)
	actualID := sanitizeNamespace(actualAppID)

	// If they don't match, check if there's an environment variable override
	// (for development/testing purposes)
	if claimedID != actualID {
		envOverride := os.Getenv("COCKPIT_DEV_MODE")
		if envOverride != "true" {
			return "", fmt.Errorf("identity mismatch: claimed=%s, actual=%s", claimedID, actualID)
		}
		// In dev mode, allow the mismatch but log a warning
		fmt.Printf("[WARNING] Identity mismatch in dev mode: claimed=%s, actual=%s\n", claimedID, actualID)
	}

	// Return the verified ID
	return actualID, nil
}

// extractAppIDFromPath extracts app ID from executable path
func extractAppIDFromPath(exePath string) string {
	// Get just the executable name
	exeName := filepath.Base(exePath)

	// Remove extension
	exeName = strings.TrimSuffix(exeName, filepath.Ext(exeName))

	// Remove common prefixes/suffixes
	exeName = strings.TrimPrefix(exeName, "cockpit-")
	exeName = strings.TrimPrefix(exeName, "aicockpit-")
	exeName = strings.TrimSuffix(exeName, "-bin")
	exeName = strings.TrimSuffix(exeName, "_bin")

	return exeName
}

// generateEncryptionKey generates an encryption key based on app ID and system
func generateEncryptionKey(appID string) []byte {
	// Create a deterministic but unique key based on:
	// - App ID
	// - System hostname
	// - OS type
	// - User ID

	hostname := getHostname()
	osType := runtime.GOOS
	userID := os.Getuid()

	// Combine all factors
	combined := fmt.Sprintf("%s|%s|%s|%d", appID, hostname, osType, userID)

	// Hash to create a 32-byte key (for AES-256)
	hash := sha256.Sum256([]byte(combined))
	return hash[:]
}

// getHostname returns the system hostname
func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}

// GetVerifiedAppID returns the verified application ID
func (sv *SecureVault) GetVerifiedAppID() string {
	return sv.verifiedAppID
}
