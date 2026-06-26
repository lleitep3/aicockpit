package vault

import (
	"os"
	"testing"

	"github.com/zalando/go-keyring"
)

func TestSecureVault(t *testing.T) {
	// Enable mock keyring for testing
	keyring.MockInit()

	// Set dev mode for testing
	os.Setenv("COCKPIT_DEV_MODE", "true")
	defer os.Unsetenv("COCKPIT_DEV_MODE")

	t.Run("Test Secure Vault Creation", func(t *testing.T) {
		sv, err := NewSecureVault("test-app")
		if err != nil {
			t.Fatalf("Failed to create secure vault: %v", err)
		}

		if sv == nil {
			t.Fatal("Expected vault, got nil")
		}

		if sv.GetVerifiedAppID() == "" {
			t.Error("Expected verified app ID to be non-empty")
		}
	})

	t.Run("Test Encrypted Set and Get", func(t *testing.T) {
		sv, err := NewSecureVault("secure-test")
		if err != nil {
			t.Fatalf("Failed to create secure vault: %v", err)
		}

		key := "encrypted_secret"
		value := "super_sensitive_value_12345"

		// Set encrypted
		err = sv.Set(key, value)
		if err != nil {
			t.Fatalf("Failed to set encrypted secret: %v", err)
		}

		// Get and decrypt
		retrieved, err := sv.Get(key)
		if err != nil {
			t.Fatalf("Failed to get encrypted secret: %v", err)
		}

		if retrieved != value {
			t.Errorf("Expected %q, got %q", value, retrieved)
		}

		// Cleanup
		sv.Delete(key)
	})

	t.Run("Test Backward Compatibility", func(t *testing.T) {
		sv, err := NewSecureVault("compat-test")
		if err != nil {
			t.Fatalf("Failed to create secure vault: %v", err)
		}

		key := "legacy_secret"
		value := "legacy_value"

		// Set using plain OSVault (simulating legacy data)
		plainVault := newOSVault()
		plainKey := sv.namespacedKey(key)
		err = plainVault.Set(plainKey, value)
		if err != nil {
			t.Fatalf("Failed to set legacy secret: %v", err)
		}

		// Get using SecureVault (should handle decryption failure gracefully)
		retrieved, err := sv.Get(key)
		if err != nil {
			t.Fatalf("Failed to get legacy secret: %v", err)
		}

		if retrieved != value {
			t.Errorf("Expected %q, got %q", value, retrieved)
		}

		// Cleanup
		sv.Delete(key)
	})

	t.Run("Test Namespace Isolation", func(t *testing.T) {
		sv1, _ := NewSecureVault("app1")
		sv2, _ := NewSecureVault("app2")

		key := "shared_key"
		value1 := "app1_value"
		value2 := "app2_value"

		// Set same key in different namespaces
		sv1.Set(key, value1)
		sv2.Set(key, value2)

		// Verify isolation
		retrieved1, _ := sv1.Get(key)
		retrieved2, _ := sv2.Get(key)

		if retrieved1 == retrieved2 {
			t.Error("Expected different values for different namespaces")
		}

		if retrieved1 != value1 {
			t.Errorf("Expected %q, got %q for app1", value1, retrieved1)
		}

		if retrieved2 != value2 {
			t.Errorf("Expected %q, got %q for app2", value2, retrieved2)
		}

		// Cleanup
		sv1.Delete(key)
		sv2.Delete(key)
	})

	t.Run("Test Identity Verification", func(t *testing.T) {
		// Test with mismatched identity (should fail in non-dev mode)
		os.Unsetenv("COCKPIT_DEV_MODE")

		_, err := NewSecureVault("fake-malicious-app")
		if err == nil {
			t.Error("Expected error for identity mismatch, got nil")
		}

		// Re-enable dev mode for other tests
		os.Setenv("COCKPIT_DEV_MODE", "true")
	})
}

func TestExtractAppIDFromPath(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"/usr/bin/cockpit-myapp", "myapp"},
		{"/home/user/bin/aicockpit-test", "test"},
		{"/opt/app/myapp-bin", "myapp"},
		{"/usr/local/bin/simple", "simple"},
		// Skip Windows path test on non-Windows systems
	}

	for _, tc := range testCases {
		result := extractAppIDFromPath(tc.input)
		if result != tc.expected {
			t.Errorf("extractAppIDFromPath(%q) = %q, expected %q", tc.input, result, tc.expected)
		}
	}
}
