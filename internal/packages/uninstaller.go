package packages

import (
	"fmt"
	"os"
	"path/filepath"
)

// Uninstaller handles the uninstallation of installed packages.
type Uninstaller struct {
	cockpitDir  string
	packagesDir string
	backup      *BackupManager
}

// NewUninstaller creates a new Uninstaller.
func NewUninstaller(cockpitDir, packagesDir string, backup *BackupManager) *Uninstaller {
	return &Uninstaller{
		cockpitDir:  cockpitDir,
		packagesDir: packagesDir,
		backup:      backup,
	}
}

// Uninstall uninstalls a package.
func (u *Uninstaller) Uninstall(packageName string) error {
	installedPath := filepath.Join(u.packagesDir, packageName)

	if _, err := os.Stat(installedPath); os.IsNotExist(err) {
		return fmt.Errorf("package not found: %s", packageName)
	}

	pkg, err := LoadPackage(installedPath)
	if err != nil {
		return fmt.Errorf("failed to load package manifest: %w", err)
	}

	if _, err := u.backup.Backup(installedPath, pkg.Name, pkg.Version); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	if err := os.RemoveAll(installedPath); err != nil {
		return fmt.Errorf("failed to remove package directory: %w", err)
	}

	return nil
}
