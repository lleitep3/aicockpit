package vault

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/zalando/go-keyring"
)

func newTestLockManager(t *testing.T) *LockManager {
	t.Helper()
	keyring.MockInit()
	dir := t.TempDir()
	return NewLockManager(filepath.Join(dir, "lock_state.json"))
}

func TestNewLockManager(t *testing.T) {
	lm := newTestLockManager(t)
	if lm == nil {
		t.Fatal("expected LockManager, got nil")
	}
	if !lm.IsVaultLocked() {
		t.Error("new lock manager should be locked by default")
	}
}

func TestNewLockManager_DefaultPath(t *testing.T) {
	keyring.MockInit()
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".cockpit", "vault"), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	lm := NewLockManager("")
	if lm == nil {
		t.Fatal("expected LockManager, got nil")
	}
	expected := filepath.Join(home, ".cockpit", "vault", "lock_state.json")
	if lm.storagePath != expected {
		t.Errorf("expected storage path %q, got %q", expected, lm.storagePath)
	}
}

func TestLockManager_LockAndUnlock(t *testing.T) {
	lm := newTestLockManager(t)

	if err := lm.Unlock("manual unlock"); err != nil {
		t.Fatalf("Unlock: %v", err)
	}
	if lm.IsVaultLocked() {
		t.Error("vault should be unlocked after Unlock")
	}

	status := lm.GetStatus()
	if !status.GlobalUnlock {
		t.Error("GlobalUnlock should be true after Unlock")
	}
	if status.UnlockReason != "manual unlock" {
		t.Errorf("expected unlock reason %q, got %q", "manual unlock", status.UnlockReason)
	}

	if err := lm.Lock("maintenance"); err != nil {
		t.Fatalf("Lock: %v", err)
	}
	if !lm.IsVaultLocked() {
		t.Error("vault should be locked after Lock")
	}

	status = lm.GetStatus()
	if status.GlobalUnlock {
		t.Error("GlobalUnlock should be false after Lock")
	}
	if status.UnlockReason != "maintenance" {
		t.Errorf("expected lock reason %q, got %q", "maintenance", status.UnlockReason)
	}
}

func TestLockManager_PackageLocks(t *testing.T) {
	lm := newTestLockManager(t)

	pkg := "test-package"
	if err := lm.UnlockPackage(pkg, "package access"); err != nil {
		t.Fatalf("UnlockPackage: %v", err)
	}
	if !lm.IsPackageUnlocked(pkg) {
		t.Error("package should be unlocked")
	}
	if !lm.CanPackageAccess(pkg) {
		t.Error("package should be able to access vault")
	}

	other := "other-package"
	if lm.CanPackageAccess(other) {
		t.Error("other package should not be able to access locked vault")
	}

	if err := lm.LockPackage(pkg); err != nil {
		t.Fatalf("LockPackage: %v", err)
	}
	if lm.IsPackageUnlocked(pkg) {
		t.Error("package should be locked after LockPackage")
	}
	if lm.CanPackageAccess(pkg) {
		t.Error("locked package should not access vault")
	}

	// Global lock should clear package locks.
	if err := lm.UnlockPackage(pkg, "reopen"); err != nil {
		t.Fatalf("UnlockPackage: %v", err)
	}
	if err := lm.Lock("reset"); err != nil {
		t.Fatalf("Lock: %v", err)
	}
	if lm.IsPackageUnlocked(pkg) {
		t.Error("global Lock should clear package unlocks")
	}
}

func TestLockManager_CanPackageAccess_RejectsIncoherentState(t *testing.T) {
	lm := newTestLockManager(t)

	// Simulate an unlocked vault without a global unlock flag by mutating state directly.
	lm.state.IsLocked = false
	lm.state.GlobalUnlock = false

	if !lm.IsVaultLocked() {
		t.Error("incoherent state must fail closed as locked")
	}
	if lm.CanPackageAccess("any-package") {
		t.Error("incoherent state must not grant access")
	}
}

func TestLockManager_GlobalUnlock(t *testing.T) {
	lm := newTestLockManager(t)

	if err := lm.Unlock("global"); err != nil {
		t.Fatalf("Unlock: %v", err)
	}

	for _, pkg := range []string{"any", "package"} {
		if !lm.CanPackageAccess(pkg) {
			t.Errorf("package %q should be accessible during global unlock", pkg)
		}
		if !lm.IsPackageUnlocked(pkg) {
			t.Errorf("package %q should report unlocked during global unlock", pkg)
		}
	}
}

func TestLockManager_GetStatus(t *testing.T) {
	lm := newTestLockManager(t)

	if err := lm.UnlockPackage("p1", "test"); err != nil {
		t.Fatalf("UnlockPackage: %v", err)
	}
	if err := lm.UnlockPackage("p2", "test"); err != nil {
		t.Fatalf("UnlockPackage: %v", err)
	}

	status := lm.GetStatus()
	if !status.IsLocked {
		t.Error("status should reflect that the vault remains locked when only packages are unlocked")
	}
	if len(status.UnlockedPackages) != 2 {
		t.Errorf("expected 2 unlocked packages, got %d", len(status.UnlockedPackages))
	}
	if status.LockedBy == "" && os.Getenv("USER") == "" && os.Getenv("USERNAME") == "" {
		t.Log("no user env set, LockedBy may be uid-based")
	}
}

func TestLockManager_StatusIsSnapshot(t *testing.T) {
	lm := newTestLockManager(t)
	if err := lm.UnlockPackage("snapshot-pkg", "test"); err != nil {
		t.Fatal(err)
	}
	status := lm.GetStatus()
	status.PackageLocks["injected"] = true
	status.UnlockedPackages[0] = "injected"
	fresh := lm.GetStatus()
	if fresh.PackageLocks["injected"] || fresh.UnlockedPackages[0] == "injected" {
		t.Fatal("GetStatus must not expose mutable internal state")
	}
}

func TestLockManager_ConcurrentManagersPreserveTransitions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lock_state.json")
	store := newFakeKeyStore()
	lm1 := NewLockManagerWithKeyStore(path, store)
	lm2 := NewLockManagerWithKeyStore(path, store)
	items := []struct {
		manager     *LockManager
		packageName string
	}{{lm1, "one"}, {lm2, "two"}}
	var wg sync.WaitGroup
	for _, item := range items {
		item := item
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := item.manager.UnlockPackage(item.packageName, "concurrent"); err != nil {
				t.Errorf("UnlockPackage(%s): %v", item.packageName, err)
			}
		}()
	}
	wg.Wait()
	final := NewLockManagerWithKeyStore(path, store)
	if !final.IsPackageUnlocked("one") || !final.IsPackageUnlocked("two") {
		t.Fatalf("concurrent transitions lost a grant: %+v", final.GetStatus())
	}
}

func TestLockManager_FailClosedAndRejectsInvalidOperations(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lock_state.json")
	store := newFakeKeyStore()
	writer := NewLockManagerWithKeyStore(path, store)
	if err := writer.Unlock("seed"); err != nil {
		t.Fatal(err)
	}
	store.getErr = os.ErrPermission
	reader := NewLockManagerWithKeyStore(path, store)
	if reader.InitializationError() == nil {
		t.Fatal("keyring read failure must be retained")
	}
	if !reader.IsVaultLocked() || reader.CanPackageAccess("any") {
		t.Fatal("unavailable state must fail closed")
	}
	if _, err := reader.GetStatusWithError(); err == nil {
		t.Fatal("status must report unavailable state")
	}
	if err := writer.UnlockPackage("", "invalid"); err == nil {
		t.Error("empty package must fail")
	}
	if err := writer.LockPackage(""); err == nil {
		t.Error("empty package must fail")
	}
	if err := writer.SetAutoLockTimeout(0); err == nil {
		t.Error("non-positive timeout must fail")
	}
}

func TestLockManager_AutoLockTimeout(t *testing.T) {
	lm := newTestLockManager(t)

	if err := lm.Unlock("auto test"); err != nil {
		t.Fatalf("Unlock: %v", err)
	}
	if err := lm.SetAutoLockTimeout(50 * time.Millisecond); err != nil {
		t.Fatalf("SetAutoLockTimeout: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Trigger the auto-lock check through a package access check.
	if lm.CanPackageAccess("any") {
		t.Error("access should be denied after auto-lock")
	}
	if !lm.IsVaultLocked() {
		t.Error("vault should auto-lock after timeout")
	}
}

func TestLockManager_SetAutoLockTimeoutWhileLocked(t *testing.T) {
	lm := newTestLockManager(t)

	if err := lm.SetAutoLockTimeout(5 * time.Minute); err != nil {
		t.Fatalf("SetAutoLockTimeout: %v", err)
	}
	if !lm.IsVaultLocked() {
		t.Error("setting timeout should not unlock the vault")
	}
}

func TestLockManager_Persistence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lock_state.json")
	keyring.MockInit()

	lm1 := NewLockManager(path)
	if err := lm1.UnlockPackage("persist-pkg", "persist"); err != nil {
		t.Fatalf("UnlockPackage: %v", err)
	}

	lm2 := NewLockManager(path)
	if !lm2.IsPackageUnlocked("persist-pkg") {
		t.Error("package unlock should persist across LockManager instances")
	}
}

func TestLockManager_LoadDefaultsWhenMissing(t *testing.T) {
	lm := newTestLockManager(t)
	if !lm.IsVaultLocked() {
		t.Error("missing state should default to locked")
	}
}

func TestLockManager_LoadCorruptState(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lock_state.json")
	if err := os.WriteFile(path, []byte("not-valid-encrypted-state"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	keyring.MockInit()

	lm := NewLockManager(path)
	if !lm.IsVaultLocked() {
		t.Error("corrupt state should default to locked")
	}
}

func TestLockManager_GetUnlockedPackages(t *testing.T) {
	lm := newTestLockManager(t)

	lm.state.PackageLocks["a"] = true
	lm.state.PackageLocks["b"] = false
	lm.state.PackageLocks["c"] = true

	unlocked := lm.getUnlockedPackages()
	if len(unlocked) != 2 {
		t.Errorf("expected 2 unlocked packages, got %d", len(unlocked))
	}
	seen := make(map[string]bool)
	for _, p := range unlocked {
		seen[p] = true
	}
	if !seen["a"] || !seen["c"] {
		t.Errorf("unexpected set of unlocked packages: %v", unlocked)
	}
}

func TestGetCurrentUser(t *testing.T) {
	t.Run("from USER env", func(t *testing.T) {
		t.Setenv("USER", "test-user")
		t.Setenv("USERNAME", "")
		if got := getCurrentUser(); got != "test-user" {
			t.Errorf("expected %q, got %q", "test-user", got)
		}
	})

	t.Run("from USERNAME env", func(t *testing.T) {
		t.Setenv("USER", "")
		t.Setenv("USERNAME", "test-username")
		if got := getCurrentUser(); got != "test-username" {
			t.Errorf("expected %q, got %q", "test-username", got)
		}
	})

	t.Run("fallback", func(t *testing.T) {
		t.Setenv("USER", "")
		t.Setenv("USERNAME", "")
		user := getCurrentUser()
		if user == "" {
			t.Error("getCurrentUser should return a non-empty identifier")
		}
	})
}
