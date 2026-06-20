package packages

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// PackageManager manages package installation and uninstallation.
type PackageManager struct {
	cockpitDir string
	packagesDir string
}

// NewPackageManager creates a new package manager.
func NewPackageManager(cockpitDir string) *PackageManager {
	return &PackageManager{
		cockpitDir:  cockpitDir,
		packagesDir: filepath.Join(cockpitDir, "packages"),
	}
}

// InstallPackage installs a package from a source directory.
func (pm *PackageManager) InstallPackage(sourcePath string, config map[string]interface{}) error {
	// Load package manifest
	pkg, err := LoadPackage(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to load package: %w", err)
	}

	// Validate package
	if err := pkg.Validate(); err != nil {
		return fmt.Errorf("package validation failed: %w", err)
	}

	// Check if package already installed
	installedPath := filepath.Join(pm.packagesDir, pkg.Name)
	if _, err := os.Stat(installedPath); err == nil {
		return fmt.Errorf("package already installed: %s", pkg.Name)
	}

	// Create package directory
	if err := os.MkdirAll(installedPath, 0o755); err != nil {
		return fmt.Errorf("failed to create package directory: %w", err)
	}

	// Copy package files
	if err := pm.copyPackageFiles(sourcePath, installedPath); err != nil {
		// Cleanup on error
		os.RemoveAll(installedPath)
		return fmt.Errorf("failed to copy package files: %w", err)
	}

	// Save package manifest
	if err := SavePackage(installedPath, pkg); err != nil {
		os.RemoveAll(installedPath)
		return fmt.Errorf("failed to save package manifest: %w", err)
	}

	return nil
}

// UninstallPackage uninstalls a package.
func (pm *PackageManager) UninstallPackage(packageName string) error {
	installedPath := filepath.Join(pm.packagesDir, packageName)

	// Check if package exists
	if _, err := os.Stat(installedPath); os.IsNotExist(err) {
		return fmt.Errorf("package not found: %s", packageName)
	}

	// Load package manifest
	pkg, err := LoadPackage(installedPath)
	if err != nil {
		return fmt.Errorf("failed to load package manifest: %w", err)
	}

	// Create backup
	backupPath := filepath.Join(pm.cockpitDir, "backups", fmt.Sprintf("%s_%s_%s", 
		pkg.Name, pkg.Version, time.Now().Format("2006-01-02T15:04:05Z")))
	
	if err := pm.backupPackage(installedPath, backupPath); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Remove package directory
	if err := os.RemoveAll(installedPath); err != nil {
		return fmt.Errorf("failed to remove package directory: %w", err)
	}

	return nil
}

// GetInstalledPackage returns an installed package.
func (pm *PackageManager) GetInstalledPackage(packageName string) (*Package, error) {
	installedPath := filepath.Join(pm.packagesDir, packageName)

	if _, err := os.Stat(installedPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("package not found: %s", packageName)
	}

	return LoadPackage(installedPath)
}

// ListInstalledPackages returns all installed packages.
func (pm *PackageManager) ListInstalledPackages() ([]*Package, error) {
	// Create packages directory if it doesn't exist
	if err := os.MkdirAll(pm.packagesDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create packages directory: %w", err)
	}

	entries, err := os.ReadDir(pm.packagesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read packages directory: %w", err)
	}

	var packages []*Package
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pkg, err := LoadPackage(filepath.Join(pm.packagesDir, entry.Name()))
		if err != nil {
			// Skip packages with invalid manifests
			continue
		}

		packages = append(packages, pkg)
	}

	return packages, nil
}

// PackageExists checks if a package is installed.
func (pm *PackageManager) PackageExists(packageName string) bool {
	installedPath := filepath.Join(pm.packagesDir, packageName)
	_, err := os.Stat(installedPath)
	return err == nil
}

// ValidatePackage validates a package at a given path.
func (pm *PackageManager) ValidatePackage(packagePath string) error {
	pkg, err := LoadPackage(packagePath)
	if err != nil {
		return err
	}

	return pkg.Validate()
}

// copyPackageFiles copies package files from source to destination.
func (pm *PackageManager) copyPackageFiles(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if entry.Name() == "cockpit-package.yml" {
				continue // Skip manifest, we'll save it separately
			}

			if err := os.MkdirAll(dstPath, 0o755); err != nil {
				return err
			}

			if err := pm.copyPackageFiles(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if entry.Name() == "cockpit-package.yml" {
				continue // Skip manifest
			}

			data, err := os.ReadFile(srcPath)
			if err != nil {
				return err
			}

			if err := os.WriteFile(dstPath, data, 0o644); err != nil {
				return err
			}
		}
	}

	return nil
}

// backupPackage creates a backup of a package.
func (pm *PackageManager) backupPackage(src, dst string) error {
	// Create backup directory
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}

	// Copy package files
	return pm.copyPackageFiles(src, dst)
}

// GetPackageInstallPath returns the installation path for a package.
func (pm *PackageManager) GetPackageInstallPath(packageName string) string {
	return filepath.Join(pm.packagesDir, packageName)
}

// GetPackagesDir returns the packages directory.
func (pm *PackageManager) GetPackagesDir() string {
	return pm.packagesDir
}

// GetCockpitDir returns the cockpit directory.
func (pm *PackageManager) GetCockpitDir() string {
	return pm.cockpitDir
}
