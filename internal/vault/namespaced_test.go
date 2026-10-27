package vault

import (
	"os"
	"path/filepath"
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

func newTestNamespacedVault(t *testing.T, appID string) *NamespacedVault {
	t.Helper()
	keyring.MockInit()
	vault := NewNamespacedVault(appID)
	vault.osVault.indexPath = filepath.Join(t.TempDir(), ".vault_index.json")
	return vault
}

func TestNamespacedVault_ListSecrets(t *testing.T) {
	vault := newTestNamespacedVault(t, "list-test")

	if err := vault.Set("secret-key", "secret-value"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := vault.Set("another-key", "another-value"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	secrets, err := vault.ListSecrets()
	if err != nil {
		t.Fatalf("ListSecrets: %v", err)
	}
	if len(secrets) != 2 {
		t.Fatalf("expected 2 secrets, got %d", len(secrets))
	}
	for _, s := range secrets {
		if s.Created == "" || s.Updated == "" {
			t.Errorf("expected timestamps for %s", s.Key)
		}
	}
}

func TestNamespacedVault_ClearAllSecrets(t *testing.T) {
	vault := newTestNamespacedVault(t, "clear-test")

	key := "secret-key"
	value := "secret-value"
	if err := vault.Set(key, value); err != nil {
		t.Fatalf("Set: %v", err)
	}

	if err := vault.ClearAllSecrets(); err != nil {
		t.Fatalf("ClearAllSecrets: %v", err)
	}

	if _, err := vault.Get(key); err == nil {
		t.Error("secret should be removed after ClearAllSecrets")
	}
}

func TestNamespacedVaultFromEnv_Fallback(t *testing.T) {
	oldValue := os.Getenv("COCKPIT_APP_ID")
	os.Unsetenv("COCKPIT_APP_ID")
	defer func() {
		if oldValue != "" {
			os.Setenv("COCKPIT_APP_ID", oldValue)
		}
	}()

	vault := NewNamespacedVaultFromEnv()
	if vault == nil {
		t.Fatal("Expected vault, got nil")
	}
	if vault.GetNamespace() == "" {
		t.Error("Expected non-empty namespace from process fallback")
	}
}

func TestSanitizeNamespace_LongName(t *testing.T) {
	long := make([]byte, 100)
	for i := range long {
		long[i] = 'a'
	}
	result := sanitizeNamespace(string(long))
	if len(result) != 64 {
		t.Errorf("expected length 64, got %d", len(result))
	}
}
