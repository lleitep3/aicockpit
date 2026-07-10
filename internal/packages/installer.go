package packages

import (
	"fmt"
	"os"
	"path/filepath"
)

// Installer handles the installation of a package from a source directory.
type Installer struct {
	packagesDir string
}

// NewInstaller creates a new Installer.
func NewInstaller(packagesDir string) *Installer {
	return &Installer{packagesDir: packagesDir}
}

// Install installs a package from a source directory.
func (i *Installer) Install(sourcePath string, _ map[string]interface{}) error {
	pkg, err := LoadPackage(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to load package: %w", err)
	}

	if err := pkg.Validate(sourcePath); err != nil {
		return fmt.Errorf("package validation failed: %w", err)
	}

	installedPath := filepath.Join(i.packagesDir, pkg.Name)
	if _, err := os.Stat(installedPath); err == nil {
		return fmt.Errorf("package already installed: %s", pkg.Name)
	}

	if err := os.MkdirAll(installedPath, 0o755); err != nil {
		return fmt.Errorf("failed to create package directory: %w", err)
	}

	if err := copyPackageFiles(sourcePath, installedPath); err != nil {
		os.RemoveAll(installedPath)
		return fmt.Errorf("failed to copy package files: %w", err)
	}

	if err := SavePackage(installedPath, pkg); err != nil {
		os.RemoveAll(installedPath)
		return fmt.Errorf("failed to save package manifest: %w", err)
	}

	return nil
}
