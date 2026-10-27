package packages

import (
	"fmt"
	"os"
	"path/filepath"
)

// packageExists reports whether a package is installed.
func packageExists(packagesDir, packageName string) bool {
	installedPath := filepath.Join(packagesDir, packageName)
	_, err := os.Stat(installedPath)
	return err == nil
}

// PackageManager coordinates package lifecycle operations by delegating to
// focused collaborators (Installer, Uninstaller, Upgrader, AssetSyncer, etc.).
type PackageManager struct {
	cockpitDir  string
	packagesDir string
	installer   *Installer
	uninstaller *Uninstaller
	upgrader    *Upgrader
	syncer      *AssetSyncer
	hookRunner  *HookRunner
	backup      *BackupManager
	deployer    *Deployer
}

// NewPackageManager creates a new package manager.
func NewPackageManager(cockpitDir string) *PackageManager {
	packagesDir := filepath.Join(cockpitDir, "packages")
	installer := NewInstaller(packagesDir)
	hookRunner := NewHookRunner()
	syncer := NewAssetSyncer(cockpitDir)
	backup := NewBackupManager(cockpitDir)

	return &PackageManager{
		cockpitDir:  cockpitDir,
		packagesDir: packagesDir,
		installer:   installer,
		uninstaller: NewUninstaller(cockpitDir, packagesDir, backup),
		upgrader:    NewUpgrader(cockpitDir, packagesDir, installer, hookRunner, syncer, backup),
		syncer:      syncer,
		hookRunner:  hookRunner,
		backup:      backup,
		deployer:    NewDeployer(),
	}
}

// InstallPackage installs a package from a source directory.
func (pm *PackageManager) InstallPackage(sourcePath string, config map[string]interface{}) error {
	return pm.installer.Install(sourcePath, config)
}

// UninstallPackage uninstalls a package.
func (pm *PackageManager) UninstallPackage(packageName string) error {
	return pm.uninstaller.Uninstall(packageName)
}

// GetInstalledPackage returns an installed package.
func (pm *PackageManager) GetInstalledPackage(packageName string) (*Package, error) {
	installedPath := filepath.Join(pm.packagesDir, packageName)

	if _, err := os.Stat(installedPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("package not found: %s", packageName)
	}

	return LoadPackage(installedPath)
}

// UpgradePackage upgrades an installed package from a new source directory.
func (pm *PackageManager) UpgradePackage(packageName string, sourcePath string) error {
	return pm.upgrader.Upgrade(packageName, sourcePath)
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
	return packageExists(pm.packagesDir, packageName)
}

// ValidatePackage validates a package at a given path.
func (pm *PackageManager) ValidatePackage(packagePath string) error {
	pkg, err := LoadPackage(packagePath)
	if err != nil {
		return err
	}

	return pkg.Validate(packagePath)
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

// RunPackageHooks executes a list of hooks from the given package directory.
func (pm *PackageManager) RunPackageHooks(packageDir string, hooks []Hook) error {
	return pm.hookRunner.Run(packageDir, hooks)
}

// SyncPackageAssets copies a package's assets into the cockpit canonical dirs.
func (pm *PackageManager) SyncPackageAssets(pkg *Package, installPath string) error {
	return pm.syncer.Sync(pkg, installPath)
}

// RemovePackageAssets removes a package's assets from the cockpit canonical dirs.
func (pm *PackageManager) RemovePackageAssets(pkg *Package) error {
	return pm.syncer.Remove(pkg)
}

// TriggerDeploy runs the cockpit deploy command to recompile canonical assets.
func (pm *PackageManager) TriggerDeploy(cockpitBin string) error {
	return pm.deployer.Trigger(cockpitBin)
}

// copyDir delegates to the package-level copyDir helper.
func (pm *PackageManager) copyDir(src, dst string) error {
	return copyDir(src, dst)
}

// copyPackageFiles delegates to the package-level copyPackageFiles helper.
func (pm *PackageManager) copyPackageFiles(src, dst string) error {
	return copyPackageFiles(src, dst)
}

// backupPackage delegates to the package-level backupPackage helper.
func (pm *PackageManager) backupPackage(src, dst string) error {
	return backupPackage(src, dst)
}
