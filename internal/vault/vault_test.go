package vault

import (
	"testing"

	"github.com/zalando/go-keyring"
)

func TestOSVault(t *testing.T) {
	// Enable mock keyring for testing to avoid OS prompts or failures in CI
	keyring.MockInit()

	v := newOSVault()
	key := "test_api_key"
	value := "super_secret_value"

	// Test Set
	err := v.Set(key, value)
	if err != nil {
		t.Fatalf("Failed to set value: %v", err)
	}

	// Test Get
	retrieved, err := v.Get(key)
	if err != nil {
		t.Fatalf("Failed to get value: %v", err)
	}
	if retrieved != value {
		t.Errorf("Expected %q, got %q", value, retrieved)
	}

	// Test Get Non-Existent
	_, err = v.Get("non_existent_key")
	if err == nil {
		t.Errorf("Expected error when getting non-existent key, got nil")
	}

	// Test Delete
	err = v.Delete(key)
	if err != nil {
		t.Fatalf("Failed to delete value: %v", err)
	}

	// Test Get after Delete
	_, err = v.Get(key)
	if err == nil {
		t.Errorf("Expected error when getting deleted key, got nil")
	}

	// Test Delete Non-Existent
	err = v.Delete("non_existent_key")
	if err == nil {
		t.Errorf("Expected error when deleting non-existent key, got nil")
	}
}
