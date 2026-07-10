package vault

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zalando/go-keyring"
)

// newTestVault returns an osVault with an index file isolated to the test's
// temp directory so tests never share state.
func newTestVault(t *testing.T) *osVault {
	t.Helper()
	keyring.MockInit()
	return &osVault{indexPath: filepath.Join(t.TempDir(), ".vault_index.json")}
}

func TestOSVault(t *testing.T) {
	v := newTestVault(t)
	key := "test_api_key"
	value := "super_secret_value"

	// Test Set
	if err := v.Set(key, value); err != nil {
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
	if _, err = v.Get("non_existent_key"); err == nil {
		t.Error("Expected error when getting non-existent key, got nil")
	}

	// Test Delete
	if err = v.Delete(key); err != nil {
		t.Fatalf("Failed to delete value: %v", err)
	}

	// Test Get after Delete
	if _, err = v.Get(key); err == nil {
		t.Error("Expected error when getting deleted key, got nil")
	}

	// Test Delete Non-Existent
	if err = v.Delete("non_existent_key"); err == nil {
		t.Error("Expected error when deleting non-existent key, got nil")
	}
}

func TestOSVault_IndexTracksKeys(t *testing.T) {
	v := newTestVault(t)

	if err := v.Set("key1", "val1"); err != nil {
		t.Fatalf("Set key1: %v", err)
	}
	if err := v.Set("key2", "val2"); err != nil {
		t.Fatalf("Set key2: %v", err)
	}

	set, err := v.loadIndex()
	if err != nil {
		t.Fatalf("loadIndex: %v", err)
	}
	if _, ok := set["key1"]; !ok {
		t.Error("index should contain key1 after Set")
	}
	if _, ok := set["key2"]; !ok {
		t.Error("index should contain key2 after Set")
	}
}

func TestOSVault_DeleteRemovesFromIndex(t *testing.T) {
	v := newTestVault(t)

	if err := v.Set("key1", "val1"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := v.Delete("key1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	set, err := v.loadIndex()
	if err != nil {
		t.Fatalf("loadIndex: %v", err)
	}
	if _, ok := set["key1"]; ok {
		t.Error("index should NOT contain key1 after Delete")
	}
}

func TestOSVault_ClearAllSecrets(t *testing.T) {
	v := newTestVault(t)

	// Set several keys
	for _, k := range []string{"alpha", "beta", "gamma"} {
		if err := v.Set(k, "value-"+k); err != nil {
			t.Fatalf("Set %s: %v", k, err)
		}
	}

	// Clear all
	if err := v.ClearAllSecrets(); err != nil {
		t.Fatalf("ClearAllSecrets: %v", err)
	}

	// Each key must no longer exist in the keyring
	for _, k := range []string{"alpha", "beta", "gamma"} {
		if _, err := v.Get(k); err == nil {
			t.Errorf("key %q still retrievable after ClearAllSecrets", k)
		}
	}

	// Index file must be gone
	if _, err := os.Stat(v.indexPath); !os.IsNotExist(err) {
		t.Error("vault index file should be removed after ClearAllSecrets")
	}
}

func TestOSVault_ClearAllSecrets_EmptyVault(t *testing.T) {
	// Clearing an empty vault (no index file) must succeed without error.
	v := newTestVault(t)
	if err := v.ClearAllSecrets(); err != nil {
		t.Errorf("ClearAllSecrets on empty vault: %v", err)
	}
}

func TestOSVault_LoadIndex_CorruptFile(t *testing.T) {
	v := newTestVault(t)

	// Write garbage to the index
	if err := os.MkdirAll(filepath.Dir(v.indexPath), 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(v.indexPath, []byte("{not json}"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// loadIndex must return an empty map (not an error) on corrupt data
	set, err := v.loadIndex()
	if err != nil {
		t.Fatalf("loadIndex with corrupt file should not error, got: %v", err)
	}
	if len(set) != 0 {
		t.Errorf("expected empty set from corrupt index, got %d keys", len(set))
	}
}
