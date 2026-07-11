package vault

import (
	"os"
	"path/filepath"
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

func TestLockManager_CanPackageAccess_UnlockedWithoutGlobal(t *testing.T) {
	lm := newTestLockManager(t)

	// Simulate an unlocked vault without a global unlock flag by mutating state directly.
	lm.state.IsLocked = false
	lm.state.GlobalUnlock = false

	if lm.IsVaultLocked() {
		t.Error("vault should report unlocked when IsLocked is false")
	}
	if !lm.CanPackageAccess("any-package") {
		t.Error("any package should be able to access an unlocked vault")
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
