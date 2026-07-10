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

// Upgrade upgrades an installed package from a new source directory.
func (u *Upgrader) Upgrade(packageName, sourcePath string) error {
	installPath := filepath.Join(u.packagesDir, packageName)

	if !packageExists(u.packagesDir, packageName) {
		return fmt.Errorf("package not found: %s", packageName)
	}

	oldPkg, err := LoadPackage(installPath)
	if err != nil {
		return fmt.Errorf("failed to get old package info: %w", err)
	}

	// Run pre_uninstall hooks
	if len(oldPkg.Installation.PreUninstall) > 0 {
		if err := u.hookRunner.Run(installPath, oldPkg.Installation.PreUninstall); err != nil {
			return fmt.Errorf("pre-uninstall hook failed during upgrade: %w", err)
		}
	}

	// Remove old assets
	if err := u.syncer.Remove(oldPkg); err != nil {
		return fmt.Errorf("failed to remove old assets: %w", err)
	}

	// Create backup
	if _, err := u.backup.Backup(installPath, oldPkg.Name, oldPkg.Version); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Remove old package directory
	if err := os.RemoveAll(installPath); err != nil {
		return fmt.Errorf("failed to remove old package directory: %w", err)
	}

	// Load new package manifest
	newPkg, err := LoadPackage(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to load new package manifest: %w", err)
	}

	// Run pre_install hooks from source
	if len(newPkg.Installation.PreInstall) > 0 {
		if err := u.hookRunner.Run(sourcePath, newPkg.Installation.PreInstall); err != nil {
			return fmt.Errorf("pre-install hook failed during upgrade: %w", err)
		}
	}

	// Create install directory
	if err := os.MkdirAll(installPath, 0o755); err != nil {
		return fmt.Errorf("failed to create install directory: %w", err)
	}

	// Copy new files
	if err := copyPackageFiles(sourcePath, installPath); err != nil {
		return fmt.Errorf("failed to copy new package files: %w", err)
	}

	// Save new manifest
	if err := SavePackage(installPath, newPkg); err != nil {
		return fmt.Errorf("failed to save new manifest: %w", err)
	}

	// Run post_install hooks from install path
	if len(newPkg.Installation.PostInstall) > 0 {
		if err := u.hookRunner.Run(installPath, newPkg.Installation.PostInstall); err != nil {
			return fmt.Errorf("post-install hook failed during upgrade: %w", err)
		}
	}

	// Sync new assets
	if err := u.syncer.Sync(newPkg, installPath); err != nil {
		return fmt.Errorf("failed to sync new assets: %w", err)
	}

	return nil
}
