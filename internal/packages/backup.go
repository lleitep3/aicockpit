package packages

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// BackupManager creates backups of installed packages.
type BackupManager struct {
	cockpitDir string
}

// NewBackupManager creates a new BackupManager.
func NewBackupManager(cockpitDir string) *BackupManager {
	return &BackupManager{cockpitDir: cockpitDir}
}

// Backup creates a backup of src for the given package and version.
// It returns the path of the newly created backup directory.
func (b *BackupManager) Backup(src, packageName, version string) (string, error) {
	backupPath := filepath.Join(b.cockpitDir, "backups", fmt.Sprintf("%s_%s_%s",
		packageName, version, time.Now().Format("2006-01-02T15:04:05Z")))

	if err := backupPackage(src, backupPath); err != nil {
		return "", err
	}

	return backupPath, nil
}

// backupPackage creates a backup of src into dst. It is a package-level helper
// so existing tests and callers can keep using a destination path directly.
func backupPackage(src, dst string) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	if err := copyPackageFiles(src, dst); err != nil {
		return fmt.Errorf("failed to copy package files to backup: %w", err)
	}

	return nil
}
