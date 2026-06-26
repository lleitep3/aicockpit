package vault

import (
	"os"
	"testing"

	"github.com/zalando/go-keyring"
)

func TestNamespacedVault(t *testing.T) {
	// Enable mock keyring for testing
	keyring.MockInit()

	// Create vaults for different applications
	app1Vault := NewNamespacedVault("app1")
	app2Vault := NewNamespacedVault("app2")

	t.Run("Test Namespace Isolation", func(t *testing.T) {
		key := "api_key"
		value1 := "app1_secret_value"
		value2 := "app2_secret_value"

		// Set same key in different namespaces
		err := app1Vault.Set(key, value1)
		if err != nil {
			t.Fatalf("Failed to set key in app1 namespace: %v", err)
		}

		err = app2Vault.Set(key, value2)
		if err != nil {
			t.Fatalf("Failed to set key in app2 namespace: %v", err)
		}

		// Verify isolation - each app should get its own value
		retrieved1, err := app1Vault.Get(key)
		if err != nil {
			t.Fatalf("Failed to get key from app1 namespace: %v", err)
		}
		if retrieved1 != value1 {
			t.Errorf("Expected %q, got %q for app1", value1, retrieved1)
		}

		retrieved2, err := app2Vault.Get(key)
		if err != nil {
			t.Fatalf("Failed to get key from app2 namespace: %v", err)
		}
		if retrieved2 != value2 {
			t.Errorf("Expected %q, got %q for app2", value2, retrieved2)
		}

		// Cleanup
		app1Vault.Delete(key)
		app2Vault.Delete(key)
	})

	t.Run("Test Cross-Namespace Access Prevention", func(t *testing.T) {
		key := "secret_key"
		value := "app1_value"

		// Set in app1
		err := app1Vault.Set(key, value)
		if err != nil {
			t.Fatalf("Failed to set key: %v", err)
		}

		// Try to access from app2 - should fail
		_, err = app2Vault.Get(key)
		if err == nil {
			t.Error("Expected error when accessing cross-namespace key, got nil")
		}

		// Cleanup
		app1Vault.Delete(key)
	})

	t.Run("Test Namespace Sanitization", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected string
		}{
			{"My App", "my_app"},
			{"my/app", "my_app"},
			{"my\\app", "my_app"},
			{"MYAPP", "myapp"},
			{"", "default"},
		}

		for _, tc := range testCases {
			result := sanitizeNamespace(tc.input)
			if result != tc.expected {
				t.Errorf("sanitizeNamespace(%q) = %q, expected %q", tc.input, result, tc.expected)
			}
		}
	})

	t.Run("Test Delete in Namespace", func(t *testing.T) {
		key := "temp_key"
		value := "temp_value"

		// Set and then delete
		err := app1Vault.Set(key, value)
		if err != nil {
			t.Fatalf("Failed to set key: %v", err)
		}

		err = app1Vault.Delete(key)
		if err != nil {
			t.Fatalf("Failed to delete key: %v", err)
		}

		// Verify it's gone
		_, err = app1Vault.Get(key)
		if err == nil {
			t.Error("Expected error when getting deleted key, got nil")
		}
	})
}

func TestNamespacedVaultFromProcess(t *testing.T) {
	// Test that process detection works
	vault := NewNamespacedVaultFromProcess()
	if vault == nil {
		t.Fatal("Expected vault, got nil")
	}

	namespace := vault.GetNamespace()
	if namespace == "" {
		t.Error("Expected non-empty namespace")
	}

	t.Logf("Detected namespace from process: %s", namespace)
}

func TestNamespacedVaultFromEnv(t *testing.T) {
	// Test environment variable detection
	testAppID := "test_app_from_env"

	// Set environment variable
	oldValue := os.Getenv("COCKPIT_APP_ID")
	os.Setenv("COCKPIT_APP_ID", testAppID)
	defer func() {
		if oldValue != "" {
			os.Setenv("COCKPIT_APP_ID", oldValue)
		} else {
			os.Unsetenv("COCKPIT_APP_ID")
		}
	}()

	vault := NewNamespacedVaultFromEnv()
	if vault == nil {
		t.Fatal("Expected vault, got nil")
	}

	namespace := vault.GetNamespace()
	if namespace != testAppID {
		t.Errorf("Expected namespace %q, got %q", testAppID, namespace)
	}
}
