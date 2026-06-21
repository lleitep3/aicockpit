package packages

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// PackageManager manages package installation and uninstallation.
type PackageManager struct {
	cockpitDir  string
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

// RunPackageHooks executes a list of hooks from the given package directory.
// Each hook's Script path is relative to packageDir.
// If a hook script does not exist it is skipped with a warning.
func (pm *PackageManager) RunPackageHooks(packageDir string, hooks []Hook) error {
	for _, hook := range hooks {
		scriptPath := filepath.Join(packageDir, hook.Script)

		// Skip missing scripts with a warning instead of hard-failing.
		if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
			fmt.Printf("  ⚠ Hook script not found (skipping): %s\n", hook.Script)
			continue
		}

		desc := hook.Description
		if desc == "" {
			desc = hook.Script
		}
		fmt.Printf("  → Running hook: %s\n", desc)

		// Make the script executable.
		if err := os.Chmod(scriptPath, 0o755); err != nil {
			return fmt.Errorf("failed to chmod hook script %s: %w", hook.Script, err)
		}

		cmd := exec.Command("sh", scriptPath) //nolint:gosec // script path comes from verified package manifest
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Dir = packageDir

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("hook script %s failed: %w", hook.Script, err)
		}
	}
	return nil
}

// SyncPackageAssets copies a package's assets (skills, rules, agents, workflows)
// into the cockpit canonical directories so they are available for provider compilation.
// Each feature entry's directory is copied to <cockpitDir>/<type>/<feature.Name>/.
func (pm *PackageManager) SyncPackageAssets(pkg *Package, installPath string) error {
	type assetGroup struct {
		features []Feature
		dir      string
	}

	groups := []assetGroup{
		{features: pkg.Features.Skills, dir: "skills"},
		{features: pkg.Features.Rules, dir: "rules"},
		{features: pkg.Features.Agents, dir: "agents"},
		{features: pkg.Features.Workflows, dir: "workflows"},
	}

	for _, group := range groups {
		for _, f := range group.features {
			src := filepath.Join(installPath, f.Path)
			dst := filepath.Join(pm.cockpitDir, group.dir, f.Name)

			if _, err := os.Stat(src); os.IsNotExist(err) {
				fmt.Printf("  ⚠ Asset not found, skipping: %s\n", f.Path)
				continue
			}

			if err := os.MkdirAll(dst, 0o755); err != nil {
				return fmt.Errorf("failed to create asset dir %s: %w", dst, err)
			}

			if err := pm.copyDir(src, dst); err != nil {
				return fmt.Errorf("failed to sync asset %s/%s: %w", group.dir, f.Name, err)
			}

			fmt.Printf("  ✓ %s/%s synced to canonical dir\n", group.dir, f.Name)
		}
	}

	// Write gold_rules to ~/.cockpit/rules/<pkg>-gold-rules.md
	// Adapters read all *.md from the rules dir when compiling AGENTS.md / .goosehints,
	// so this injects gold rules into every provider on the next `cockpit deploy`.
	if len(pkg.Features.GoldRules) > 0 {
		rulesDir := filepath.Join(pm.cockpitDir, "rules")
		if err := os.MkdirAll(rulesDir, 0o755); err != nil {
			return fmt.Errorf("failed to create rules dir: %w", err)
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("# Gold Rules — %s\n\n", pkg.Name))
		sb.WriteString("> These rules were injected by the `" + pkg.Name + "` package.\n\n")
		for _, rule := range pkg.Features.GoldRules {
			sb.WriteString("- " + rule + "\n")
		}

		goldRulesPath := filepath.Join(rulesDir, pkg.Name+"-gold-rules.md")
		if err := os.WriteFile(goldRulesPath, []byte(sb.String()), 0o644); err != nil {
			return fmt.Errorf("failed to write gold rules for %s: %w", pkg.Name, err)
		}
		fmt.Printf("  ✓ gold_rules written to rules/%s-gold-rules.md\n", pkg.Name)
	}

	return nil
}

// RemovePackageAssets removes a package's assets from the cockpit canonical directories.
func (pm *PackageManager) RemovePackageAssets(pkg *Package) error {
	type assetGroup struct {
		features []Feature
		dir      string
	}

	groups := []assetGroup{
		{features: pkg.Features.Skills, dir: "skills"},
		{features: pkg.Features.Rules, dir: "rules"},
		{features: pkg.Features.Agents, dir: "agents"},
		{features: pkg.Features.Workflows, dir: "workflows"},
	}

	for _, group := range groups {
		for _, f := range group.features {
			dst := filepath.Join(pm.cockpitDir, group.dir, f.Name)

			if _, err := os.Stat(dst); os.IsNotExist(err) {
				continue // Already gone, no-op
			}

			if err := os.RemoveAll(dst); err != nil {
				return fmt.Errorf("failed to remove asset %s/%s: %w", group.dir, f.Name, err)
			}

			fmt.Printf("  ✓ %s/%s removed from canonical dir\n", group.dir, f.Name)
		}
	}

	// Remove gold_rules file if it exists
	goldRulesPath := filepath.Join(pm.cockpitDir, "rules", pkg.Name+"-gold-rules.md")
	if _, err := os.Stat(goldRulesPath); err == nil {
		if err := os.Remove(goldRulesPath); err != nil {
			return fmt.Errorf("failed to remove gold rules for %s: %w", pkg.Name, err)
		}
		fmt.Printf("  ✓ gold_rules removed from rules/%s-gold-rules.md\n", pkg.Name)
	}

	return nil
}

// TriggerDeploy runs the cockpit deploy command to recompile all canonical assets
// to the active providers. cockpitBin is the path to the cockpit binary; if empty,
// the current process binary is used via os.Executable.
func (pm *PackageManager) TriggerDeploy(cockpitBin string) error {
	if cockpitBin == "" {
		bin, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to resolve cockpit binary: %w", err)
		}
		cockpitBin = bin
	}

	cmd := exec.Command(cockpitBin, "deploy") //nolint:gosec // path from os.Executable
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cockpit deploy failed: %w", err)
	}

	return nil
}

// copyDir recursively copies src directory contents into dst directory.
func (pm *PackageManager) copyDir(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := os.MkdirAll(dstPath, 0o755); err != nil {
				return err
			}
			if err := pm.copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
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
