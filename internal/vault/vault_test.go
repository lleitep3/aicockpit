package vault

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/zalando/go-keyring"
)

func TestNewOSVault(t *testing.T) {
	keyring.MockInit()

	manager := NewOSVault()
	if manager == nil {
		t.Fatal("expected Manager, got nil")
	}

	v, ok := manager.(*osVault)
	if !ok {
		t.Fatalf("expected *osVault, got %T", manager)
	}
	v.indexPath = filepath.Join(t.TempDir(), ".vault_index.json")

	key := "osvault-key"
	value := "osvault-value"
	if err := manager.Set(key, value); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err := manager.Get(key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != value {
		t.Errorf("expected %q, got %q", value, got)
	}

	if err := manager.Delete(key); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if err := manager.Set("k2", "v2"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := manager.ClearAllSecrets(); err != nil {
		t.Fatalf("ClearAllSecrets: %v", err)
	}
	if _, err := manager.Get("k2"); err == nil {
		t.Error("secret should be removed after ClearAllSecrets")
	}
}

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

	meta, err := v.loadIndex()
	if err != nil {
		t.Fatalf("loadIndex: %v", err)
	}
	if _, ok := meta["key1"]; !ok {
		t.Error("index should contain key1 after Set")
	}
	if _, ok := meta["key2"]; !ok {
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

	meta, err := v.loadIndex()
	if err != nil {
		t.Fatalf("loadIndex: %v", err)
	}
	if _, ok := meta["key1"]; ok {
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

func TestOSVault_SaveIndex_Error(t *testing.T) {
	keyring.MockInit()
	v := &osVault{indexPath: "/dev/null/.vault_index.json"}

	if err := v.Set("k", "v"); err == nil {
		t.Fatal("expected error when index path cannot be created")
	}
}

func TestOSVault_Delete_SaveIndexError(t *testing.T) {
	v := newTestVault(t)

	if err := v.Set("k", "v"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	v.indexPath = "/dev/null/.vault_index.json"
	if err := v.Delete("k"); err == nil {
		t.Fatal("expected error when index cannot be saved")
	}
}

func TestOSVault_ClearAllSecrets_WithDeleteFailure(t *testing.T) {
	v := newTestVault(t)

	if err := v.Set("stubborn", "value"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	// Remove the secret from the keyring while keeping it in the index so
	// ClearAllSecrets encounters a delete failure.
	if err := keyring.Delete(serviceName, "stubborn"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if err := v.ClearAllSecrets(); err == nil {
		t.Fatal("expected ClearAllSecrets to report delete failure")
	}
}

func TestOSVault_List(t *testing.T) {
	v := newTestVault(t)

	if err := v.Set("alpha", "val1"); err != nil {
		t.Fatalf("Set alpha: %v", err)
	}
	if err := v.Set("beta", "val2"); err != nil {
		t.Fatalf("Set beta: %v", err)
	}

	infos, err := v.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(infos) != 2 {
		t.Fatalf("expected 2 secrets, got %d", len(infos))
	}

	keys := make(map[string]struct{}, len(infos))
	for _, info := range infos {
		keys[info.Key] = struct{}{}
		if info.Created == "" || info.Updated == "" {
			t.Errorf("expected timestamps for %s, got created=%q updated=%q", info.Key, info.Created, info.Updated)
		}
	}
	for _, k := range []string{"alpha", "beta"} {
		if _, ok := keys[k]; !ok {
			t.Errorf("expected %q in list", k)
		}
	}
}

func TestOSVault_List_LegacyIndex(t *testing.T) {
	v := newTestVault(t)

	if err := os.MkdirAll(filepath.Dir(v.indexPath), 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	legacy, _ := json.Marshal([]string{"legacy-a", "legacy-b"})
	if err := os.WriteFile(v.indexPath, legacy, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	infos, err := v.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(infos) != 2 {
		t.Fatalf("expected 2 legacy secrets, got %d", len(infos))
	}
	for _, info := range infos {
		if info.Created == "" || info.Updated == "" {
			t.Errorf("expected migrated timestamps for %s", info.Key)
		}
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
