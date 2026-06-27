package vault

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// LockState represents the lock state of the vault
type LockState struct {
	IsLocked         bool            `json:"is_locked"`
	LockedAt         time.Time       `json:"locked_at,omitempty"`
	LockedBy         string          `json:"locked_by,omitempty"` // user/process
	PackageLocks     map[string]bool `json:"package_locks"`       // package -> is_unlocked
	GlobalUnlock     bool            `json:"global_unlock"`       // if true, all packages unlocked
	UnlockReason     string          `json:"unlock_reason,omitempty"`
	UnlockTime       time.Time       `json:"unlock_time,omitempty"`
	AutoLockExpireAt time.Time       `json:"auto_lock_expire_at,omitempty"` // When auto-lock should happen
	StoragePath      string          `json:"-"`
}

// NewLockManager creates a new lock manager
func NewLockManager(storagePath string) *LockManager {
	if storagePath == "" {
		storagePath = "/home/lleite/.cockpit/vault/lock_state.json"
	}

	lm := &LockManager{
		state: &LockState{
			IsLocked:     true, // Default to locked for security
			PackageLocks: make(map[string]bool),
			GlobalUnlock: false,
		},
		storagePath: storagePath,
		encryptor:   NewStateEncryptor(),
	}

	lm.load()
	return lm
}

// LockManager manages vault lock states
type LockManager struct {
	state       *LockState
	storagePath string
	mu          sync.RWMutex
	encryptor   *StateEncryptor
}

// Lock locks the vault globally
func (lm *LockManager) Lock(reason string) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	lm.state.IsLocked = true
	lm.state.LockedAt = time.Now()
	lm.state.LockedBy = getCurrentUser()
	lm.state.UnlockReason = reason
	lm.state.GlobalUnlock = false
	// Clear package unlocks when locking globally
	lm.state.PackageLocks = make(map[string]bool)

	return lm.save()
}

// Unlock unlocks the vault globally
func (lm *LockManager) Unlock(reason string) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	lm.state.IsLocked = false
	lm.state.GlobalUnlock = true
	lm.state.UnlockReason = reason
	lm.state.UnlockTime = time.Now()
	lm.state.LockedBy = ""

	return lm.save()
}

// UnlockPackage unlocks access for a specific package
func (lm *LockManager) UnlockPackage(packageName string, reason string) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	lm.state.PackageLocks[packageName] = true
	lm.state.UnlockReason = reason

	// Don't change global lock state - just allow this specific package
	// The CanPackageAccess check will handle the access control

	return lm.save()
}

// LockPackage locks access for a specific package
func (lm *LockManager) LockPackage(packageName string) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	lm.state.PackageLocks[packageName] = false

	return lm.save()
}

// IsVaultLocked returns if the vault is currently locked
func (lm *LockManager) IsVaultLocked() bool {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	// If globally unlocked, not locked
	if lm.state.GlobalUnlock {
		return false
	}

	return lm.state.IsLocked
}

// IsPackageUnlocked returns if a specific package has unlock access
func (lm *LockManager) IsPackageUnlocked(packageName string) bool {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	// If globally unlocked, all packages are unlocked
	if lm.state.GlobalUnlock {
		return true
	}

	// Check if specific package is unlocked
	return lm.state.PackageLocks[packageName]
}

// CanPackageAccess checks if a package can access the vault
func (lm *LockManager) CanPackageAccess(packageName string) bool {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	// Check for auto-lock expiration first
	lm.checkAutoLockInternal()

	// If globally unlocked, yes
	if lm.state.GlobalUnlock {
		return true
	}

	// If vault is locked and package not specifically unlocked, no
	if lm.state.IsLocked {
		return lm.state.PackageLocks[packageName]
	}

	// If vault is not locked, yes
	return true
}

// GetStatus returns the current lock status
func (lm *LockManager) GetStatus() LockStatus {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	return LockStatus{
		IsLocked:         lm.state.IsLocked,
		LockedAt:         lm.state.LockedAt,
		LockedBy:         lm.state.LockedBy,
		GlobalUnlock:     lm.state.GlobalUnlock,
		UnlockReason:     lm.state.UnlockReason,
		PackageLocks:     lm.state.PackageLocks,
		UnlockedPackages: lm.getUnlockedPackages(),
	}
}

// LockStatus represents the current lock status
type LockStatus struct {
	IsLocked         bool            `json:"is_locked"`
	LockedAt         time.Time       `json:"locked_at,omitempty"`
	LockedBy         string          `json:"locked_by,omitempty"`
	GlobalUnlock     bool            `json:"global_unlock"`
	UnlockReason     string          `json:"unlock_reason,omitempty"`
	PackageLocks     map[string]bool `json:"package_locks"`
	UnlockedPackages []string        `json:"unlocked_packages"`
}

func (lm *LockManager) getUnlockedPackages() []string {
	var unlocked []string

	lm.mu.RLock()
	defer lm.mu.RUnlock()

	for pkg, isUnlocked := range lm.state.PackageLocks {
		if isUnlocked {
			unlocked = append(unlocked, pkg)
		}
	}

	return unlocked
}

// save saves the lock state to disk (encrypted)
func (lm *LockManager) save() error {
	lm.state.StoragePath = lm.storagePath

	// Encrypt and sign the state
	return lm.encryptor.EncryptAndSign(lm.state)
}

// load loads the lock state from disk (decrypted)
func (lm *LockManager) load() error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	// Decrypt and verify the state
	state, err := lm.encryptor.DecryptAndVerify()
	if err != nil {
		// If decryption fails (file corrupted or tampered), use defaults
		lm.state = &LockState{
			IsLocked:     true,
			PackageLocks: make(map[string]bool),
			GlobalUnlock: false,
		}
		return nil
	}

	lm.state = state

	// Check for auto-lock expiration
	lm.checkAutoLockInternal()

	return nil
}

// getCurrentUser returns the current username
func getCurrentUser() string {
	// Try to get username from environment
	if user := os.Getenv("USER"); user != "" {
		return user
	}
	if user := os.Getenv("USERNAME"); user != "" {
		return user
	}

	// Fallback to current user ID
	return fmt.Sprintf("uid-%d", os.Getuid())
}

// SetAutoLockTimeout sets automatic lock after a duration
func (lm *LockManager) SetAutoLockTimeout(duration time.Duration) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	// Calculate expiration time
	lm.state.AutoLockExpireAt = time.Now().Add(duration)
	return lm.save()
}

// checkAutoLockInternal checks if auto-lock timeout has expired and locks if needed
// Assumes caller holds the lock
func (lm *LockManager) checkAutoLockInternal() {
	// If no auto-lock configured, do nothing
	if lm.state.AutoLockExpireAt.IsZero() {
		return
	}

	// If vault is already locked, do nothing
	if lm.state.IsLocked {
		return
	}

	// Check if current time is past expiration
	if time.Now().After(lm.state.AutoLockExpireAt) {
		// Auto-lock!
		lm.state.IsLocked = true
		lm.state.GlobalUnlock = false
		lm.state.UnlockReason = ""
		lm.state.AutoLockExpireAt = time.Time{} // Clear expiration
		// Don't save here to avoid deadlock - state will be saved on next operation
	}
}
