package vault

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"
)

// LockState represents the lock state of the vault.
type LockState struct {
	IsLocked         bool            `json:"is_locked"`
	LockedAt         time.Time       `json:"locked_at,omitempty"`
	LockedBy         string          `json:"locked_by,omitempty"`
	PackageLocks     map[string]bool `json:"package_locks"`
	GlobalUnlock     bool            `json:"global_unlock"`
	UnlockReason     string          `json:"unlock_reason,omitempty"`
	UnlockTime       time.Time       `json:"unlock_time,omitempty"`
	AutoLockExpireAt time.Time       `json:"auto_lock_expire_at,omitempty"`
	StoragePath      string          `json:"-"`
}

// LockManager manages lock states. initErr is retained so the legacy
// constructor cannot accidentally turn a load/keyring failure into access.
type LockManager struct {
	state       *LockState
	storagePath string
	mu          sync.RWMutex
	encryptor   *StateEncryptor
	initErr     error
}

// NewLockManager creates a manager. Callers that need to inspect failures can
// use InitializationError; all access checks fail closed while it is non-nil.
func NewLockManager(storagePath string) *LockManager {
	if storagePath == "" {
		storagePath = DefaultLockStatePath()
	}
	return NewLockManagerWithKeyStore(storagePath, nil)
}

func NewLockManagerWithKeyStore(storagePath string, keyStore KeyStore) *LockManager {
	if storagePath == "" {
		storagePath = DefaultLockStatePath()
	}
	lm := &LockManager{
		state:       defaultLockState(),
		storagePath: storagePath,
		encryptor:   NewStateEncryptorAt(storagePath, keyStore),
	}
	if err := lm.loadInitial(); err != nil {
		lm.initErr = err
	}
	return lm
}

func (lm *LockManager) InitializationError() error {
	return lm.initErr
}

// Lock locks the vault globally.
func (lm *LockManager) Lock(reason string) error {
	return lm.mutate(func(state *LockState) {
		state.IsLocked = true
		state.LockedAt = time.Now()
		state.LockedBy = getCurrentUser()
		state.UnlockReason = reason
		state.GlobalUnlock = false
		state.AutoLockExpireAt = time.Time{}
		state.PackageLocks = make(map[string]bool)
	})
}

// Unlock unlocks the vault globally.
func (lm *LockManager) Unlock(reason string) error {
	return lm.mutate(func(state *LockState) {
		state.IsLocked = false
		state.GlobalUnlock = true
		state.UnlockReason = reason
		state.UnlockTime = time.Now()
		state.LockedBy = ""
	})
}

// UnlockPackage grants access for one package while the global state remains locked.
func (lm *LockManager) UnlockPackage(packageName string, reason string) error {
	if packageName == "" {
		return fmt.Errorf("package name cannot be empty")
	}
	return lm.mutate(func(state *LockState) {
		state.PackageLocks[packageName] = true
		state.UnlockReason = reason
	})
}

// LockPackage revokes access for one package.
func (lm *LockManager) LockPackage(packageName string) error {
	if packageName == "" {
		return fmt.Errorf("package name cannot be empty")
	}
	return lm.mutate(func(state *LockState) {
		state.PackageLocks[packageName] = false
	})
}

// IsVaultLocked returns whether global access is currently locked.
func (lm *LockManager) IsVaultLocked() bool {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	if err := lm.refreshUnlocked(); err != nil {
		return true
	}
	return lm.state.IsLocked || !lm.state.GlobalUnlock
}

// IsPackageUnlocked returns whether a package currently has access.
func (lm *LockManager) IsPackageUnlocked(packageName string) bool {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	if err := lm.refreshUnlocked(); err != nil {
		return false
	}
	if lm.state.GlobalUnlock {
		return true
	}
	return lm.state.PackageLocks[packageName]
}

// CanPackageAccess checks current state and fails closed on any read/lock error.
func (lm *LockManager) CanPackageAccess(packageName string) bool {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	if err := lm.refreshUnlocked(); err != nil {
		return false
	}
	if lm.state.GlobalUnlock {
		return true
	}
	if lm.state.IsLocked {
		return lm.state.PackageLocks[packageName]
	}
	return false
}

// GetStatus preserves the old API and returns a locked snapshot on failure.
func (lm *LockManager) GetStatus() LockStatus {
	status, _ := lm.GetStatusWithError()
	return status
}

func (lm *LockManager) GetStatusWithError() (LockStatus, error) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	if err := lm.refreshUnlocked(); err != nil {
		return lockedStatus(err), err
	}
	return statusFromState(lm.state), nil
}

// LockStatus represents a safe snapshot of lock status.
type LockStatus struct {
	IsLocked         bool            `json:"is_locked"`
	LockedAt         time.Time       `json:"locked_at,omitempty"`
	LockedBy         string          `json:"locked_by,omitempty"`
	GlobalUnlock     bool            `json:"global_unlock"`
	UnlockReason     string          `json:"unlock_reason,omitempty"`
	PackageLocks     map[string]bool `json:"package_locks"`
	UnlockedPackages []string        `json:"unlocked_packages"`
	Error            string          `json:"error,omitempty"`
}

func (lm *LockManager) SetAutoLockTimeout(duration time.Duration) error {
	if duration <= 0 {
		return fmt.Errorf("auto-lock duration must be positive")
	}
	return lm.mutate(func(state *LockState) {
		state.AutoLockExpireAt = time.Now().Add(duration)
	})
}

func (lm *LockManager) mutate(change func(*LockState)) error {
	if lm.initErr != nil {
		return fmt.Errorf("vault lock state unavailable: %w", lm.initErr)
	}
	lm.mu.Lock()
	defer lm.mu.Unlock()
	return withStateFileLock(lm.storagePath, func() error {
		previous := cloneLockState(lm.state)
		if err := lm.refreshUnlocked(); err != nil {
			lm.state = previous
			return fmt.Errorf("failed to reload vault lock state: %w", err)
		}
		next := cloneLockState(lm.state)
		change(next)
		next.StoragePath = lm.storagePath
		if err := validateLockState(next); err != nil {
			lm.state = previous
			return err
		}
		if err := lm.encryptor.encryptAndSignUnlocked(next, false); err != nil {
			lm.state = previous
			return err
		}
		lm.state = next
		return nil
	})
}

func (lm *LockManager) loadInitial() error {
	return withStateFileLock(lm.storagePath, func() error {
		return lm.refreshUnlocked()
	})
}

// refreshUnlocked must be called while lm.mu is held. The process lock is
// acquired by callers that need a stable read-modify-write section.
func (lm *LockManager) refreshUnlocked() error {
	state, err := lm.encryptor.DecryptAndVerify()
	if err != nil {
		return err
	}
	if state.PackageLocks == nil {
		return errors.New("lock state has no package-lock map")
	}
	state.StoragePath = lm.storagePath
	if expireState(state, time.Now()) {
		// Expired access is denied locally even if persisting the revocation fails.
		// A caller performing an access check therefore never receives access.
		if err := lm.encryptor.encryptAndSignUnlocked(state, false); err != nil {
			return fmt.Errorf("failed to persist expired lock state: %w", err)
		}
	}
	lm.state = state
	return nil
}

func expireState(state *LockState, now time.Time) bool {
	if state.AutoLockExpireAt.IsZero() || state.IsLocked || now.Before(state.AutoLockExpireAt) {
		return false
	}
	state.IsLocked = true
	state.GlobalUnlock = false
	state.UnlockReason = ""
	state.AutoLockExpireAt = time.Time{}
	return true
}

func cloneLockState(state *LockState) *LockState {
	clone := *state
	clone.PackageLocks = make(map[string]bool, len(state.PackageLocks))
	for packageName, unlocked := range state.PackageLocks {
		clone.PackageLocks[packageName] = unlocked
	}
	return &clone
}

func statusFromState(state *LockState) LockStatus {
	packageLocks := make(map[string]bool, len(state.PackageLocks))
	var unlocked []string
	for packageName, isUnlocked := range state.PackageLocks {
		packageLocks[packageName] = isUnlocked
		if isUnlocked {
			unlocked = append(unlocked, packageName)
		}
	}
	return LockStatus{
		IsLocked:         state.IsLocked,
		LockedAt:         state.LockedAt,
		LockedBy:         state.LockedBy,
		GlobalUnlock:     state.GlobalUnlock,
		UnlockReason:     state.UnlockReason,
		PackageLocks:     packageLocks,
		UnlockedPackages: unlocked,
	}
}

// getUnlockedPackages is retained for package-local callers; it returns a
// snapshot and never exposes the manager's map.
func (lm *LockManager) getUnlockedPackages() []string {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	var unlocked []string
	for packageName, isUnlocked := range lm.state.PackageLocks {
		if isUnlocked {
			unlocked = append(unlocked, packageName)
		}
	}
	return unlocked
}

func lockedStatus(err error) LockStatus {
	return LockStatus{IsLocked: true, PackageLocks: make(map[string]bool), Error: err.Error()}
}

func getCurrentUser() string {
	if user := os.Getenv("USER"); user != "" {
		return user
	}
	if user := os.Getenv("USERNAME"); user != "" {
		return user
	}
	return fmt.Sprintf("uid-%d", os.Getuid())
}
