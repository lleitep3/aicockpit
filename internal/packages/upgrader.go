package packages

import (
	"fmt"
	"os"
	"path/filepath"
)

// Upgrader handles upgrading an installed package from a new source directory.
type Upgrader struct {
	cockpitDir  string
	packagesDir string
	installer   *Installer
	hookRunner  *HookRunner
	syncer      *AssetSyncer
	backup      *BackupManager
}

// NewUpgrader creates a new Upgrader.
func NewUpgrader(cockpitDir, packagesDir string, installer *Installer, hookRunner *HookRunner, syncer *AssetSyncer, backup *BackupManager) *Upgrader {
	return &Upgrader{
		cockpitDir:  cockpitDir,
		packagesDir: packagesDir,
		installer:   installer,
		hookRunner:  hookRunner,
		syncer:      syncer,
		backup:      backup,
	}
}

// Upgrade validates and stages a candidate before touching the installed tree.
// Rollback covers managed local files, not external side effects of package hooks.
func (u *Upgrader) Upgrade(packageName, sourcePath string) (result error) {
	if !safeUpgradeName(packageName) {
		return fmt.Errorf("invalid package name")
	}
	installPath := filepath.Join(u.packagesDir, packageName)
	oldPkg, err := LoadPackage(installPath)
	if err != nil {
		return fmt.Errorf("failed to get old package info: %w", err)
	}
	newPkg, err := LoadPackage(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to load new package manifest: %w", err)
	}
	if oldPkg.Name != packageName || newPkg.Name != packageName {
		return fmt.Errorf("package identity mismatch")
	}
	if err := validateUpgradePackage(newPkg, sourcePath); err != nil {
		return err
	}
	if err := os.MkdirAll(u.cockpitDir, 0700); err != nil {
		return err
	}
	lockPath := filepath.Join(u.cockpitDir, ".package-upgrade.lock")
	lock, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("package upgrade busy; reconcile %s after confirming no updater is running", lockPath)
	}
	keepLock := false
	defer func() {
		if !keepLock {
			_ = os.Remove(lockPath)
		}
	}()
	if err := lock.Close(); err != nil {
		return err
	}
	tx, err := prepareUpgrade(u.cockpitDir, installPath, sourcePath, oldPkg, newPkg)
	if err != nil {
		return err
	}
	// A prepared complete snapshot exists before any hook is executed.
	defer func() {
		if result != nil {
			if err := tx.restore(); err != nil {
				keepLock = true
				result = fmt.Errorf("%w; rollback failed: %v; recover from %s", result, err, tx.dir)
			} else {
				result = fmt.Errorf("%w; managed files restored; recovery snapshot: %s (external hook effects are not rolled back)", result, tx.dir)
			}
		}
	}()
	if err := u.hookRunner.Run(installPath, oldPkg.Installation.PreUninstall); err != nil {
		return fmt.Errorf("pre-uninstall hook failed during upgrade: %w", err)
	}
	if err := u.hookRunner.Run(tx.stage, newPkg.Installation.PreInstall); err != nil {
		return fmt.Errorf("pre-install hook failed during upgrade: %w", err)
	}
	// Hooks may change staged files; verify again before activation.
	staged, err := LoadPackage(tx.stage)
	if err != nil {
		return fmt.Errorf("staged manifest unavailable: %w", err)
	}
	if staged.Name != newPkg.Name || staged.Version != newPkg.Version {
		return fmt.Errorf("hook changed candidate identity/version")
	}
	if err := validateUpgradePackage(staged, tx.stage); err != nil {
		return err
	}
	// Use the reviewed manifest; hook edits cannot broaden the asset mutation set.
	if err := SavePackage(tx.stage, newPkg); err != nil {
		return err
	}
	if err := validateUpgradePackage(newPkg, tx.stage); err != nil {
		return err
	}
	if err := os.Rename(installPath, filepath.Join(tx.dir, "previous")); err != nil {
		return fmt.Errorf("failed to move old package: %w", err)
	}
	if err := os.Rename(tx.stage, installPath); err != nil {
		return fmt.Errorf("failed to activate candidate: %w", err)
	}
	if err := u.hookRunner.Run(installPath, newPkg.Installation.PostInstall); err != nil {
		return fmt.Errorf("post-install hook failed during upgrade: %w", err)
	}
	// Do not follow paths/symlinks introduced by a post-install hook.
	if err := validateUpgradePackage(newPkg, installPath); err != nil {
		return err
	}
	if err := SavePackage(installPath, newPkg); err != nil {
		return err
	}
	for _, snapshot := range tx.snapshots {
		if err := checkUpgradeParents(u.cockpitDir, snapshot.target); err != nil {
			return err
		}
	}
	if err := u.syncer.Remove(oldPkg); err != nil {
		return fmt.Errorf("failed to remove old assets: %w", err)
	}
	if err := u.syncer.Sync(newPkg, installPath); err != nil {
		return fmt.Errorf("failed to sync new assets: %w", err)
	}
	fmt.Printf("Recovery snapshot: %s\n", tx.dir)
	return nil
}
