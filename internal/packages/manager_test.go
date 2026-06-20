package packages

import (
	"os"
	"path/filepath"
	"testing"
)

func createTestPackage(t *testing.T, dir string) string {
	packageDir := filepath.Join(dir, "test-package")
	if err := os.MkdirAll(packageDir, 0o755); err != nil {
		t.Fatalf("Failed to create package directory: %v", err)
	}

	// Create manifest
	manifestPath := filepath.Join(packageDir, "cockpit-package.yml")
	manifestContent := `
name: "test-package"
version: "1.0.0"
description: "Test package"
author: "Test Author"
license: "MIT"
type: "utility"

requirements:
  cockpit: "0.2.0"

features:
  skills:
    - path: "skills/test.go"
      name: "test-skill"
      description: "Test skill"

installation:
  supported_providers:
    - devin
  provider_features:
    devin:
      - skills
  method: "symlink"
`

	if err := os.WriteFile(manifestPath, []byte(manifestContent), 0o644); err != nil {
		t.Fatalf("Failed to create manifest: %v", err)
	}

	// Create skills directory
	skillsDir := filepath.Join(packageDir, "skills")
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		t.Fatalf("Failed to create skills directory: %v", err)
	}

	// Create test skill file
	skillPath := filepath.Join(skillsDir, "test.go")
	if err := os.WriteFile(skillPath, []byte("package skills\n"), 0o644); err != nil {
		t.Fatalf("Failed to create skill file: %v", err)
	}

	return packageDir
}

func TestNewPackageManager(t *testing.T) {
	tmpDir := t.TempDir()

	pm := NewPackageManager(tmpDir)

	if pm.GetCockpitDir() != tmpDir {
		t.Errorf("Expected cockpit dir %s, got %s", tmpDir, pm.GetCockpitDir())
	}

	expectedPackagesDir := filepath.Join(tmpDir, "packages")
	if pm.GetPackagesDir() != expectedPackagesDir {
		t.Errorf("Expected packages dir %s, got %s", expectedPackagesDir, pm.GetPackagesDir())
	}
}

func TestInstallPackage(t *testing.T) {
	tmpDir := t.TempDir()
	pm := NewPackageManager(tmpDir)

	// Create test package
	packageDir := createTestPackage(t, tmpDir)

	// Install package
	err := pm.InstallPackage(packageDir, nil)
	if err != nil {
		t.Fatalf("InstallPackage failed: %v", err)
	}

	// Verify package was installed
	installedPath := filepath.Join(pm.GetPackagesDir(), "test-package")
	if _, err := os.Stat(installedPath); os.IsNotExist(err) {
		t.Error("Package was not installed")
	}

	// Verify manifest exists
	manifestPath := filepath.Join(installedPath, "cockpit-package.yml")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		t.Error("Manifest was not copied")
	}

	// Verify skill file was copied
	skillPath := filepath.Join(installedPath, "skills", "test.go")
	if _, err := os.Stat(skillPath); os.IsNotExist(err) {
		t.Error("Skill file was not copied")
	}
}

func TestInstallPackageDuplicate(t *testing.T) {
	tmpDir := t.TempDir()
	pm := NewPackageManager(tmpDir)

	// Create test package
	packageDir := createTestPackage(t, tmpDir)

	// Install package first time
	if err := pm.InstallPackage(packageDir, nil); err != nil {
		t.Fatalf("First install failed: %v", err)
	}

	// Try to install again
	err := pm.InstallPackage(packageDir, nil)
	if err == nil {
		t.Error("Expected error when installing duplicate package")
	}
}

func TestUninstallPackage(t *testing.T) {
	tmpDir := t.TempDir()
	pm := NewPackageManager(tmpDir)

	// Create and install test package
	packageDir := createTestPackage(t, tmpDir)
	if err := pm.InstallPackage(packageDir, nil); err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	// Uninstall package
	err := pm.UninstallPackage("test-package")
	if err != nil {
		t.Fatalf("UninstallPackage failed: %v", err)
	}

	// Verify package was removed
	installedPath := filepath.Join(pm.GetPackagesDir(), "test-package")
	if _, err := os.Stat(installedPath); err == nil {
		t.Error("Package was not removed")
	}
}

func TestUninstallPackageNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	pm := NewPackageManager(tmpDir)

	err := pm.UninstallPackage("nonexistent-package")
	if err == nil {
		t.Error("Expected error when uninstalling nonexistent package")
	}
}

func TestGetInstalledPackage(t *testing.T) {
	tmpDir := t.TempDir()
	pm := NewPackageManager(tmpDir)

	// Create and install test package
	packageDir := createTestPackage(t, tmpDir)
	if err := pm.InstallPackage(packageDir, nil); err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	// Get installed package
	pkg, err := pm.GetInstalledPackage("test-package")
	if err != nil {
		t.Fatalf("GetInstalledPackage failed: %v", err)
	}

	if pkg.Name != "test-package" {
		t.Errorf("Expected package name 'test-package', got '%s'", pkg.Name)
	}

	if pkg.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", pkg.Version)
	}
}

func TestGetInstalledPackageNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	pm := NewPackageManager(tmpDir)

	_, err := pm.GetInstalledPackage("nonexistent-package")
	if err == nil {
		t.Error("Expected error when getting nonexistent package")
	}
}

func TestListInstalledPackages(t *testing.T) {
	tmpDir := t.TempDir()
	pm := NewPackageManager(tmpDir)

	// Create and install test package
	packageDir := createTestPackage(t, tmpDir)
	if err := pm.InstallPackage(packageDir, nil); err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	// List packages
	packages, err := pm.ListInstalledPackages()
	if err != nil {
		t.Fatalf("ListInstalledPackages failed: %v", err)
	}

	if len(packages) != 1 {
		t.Errorf("Expected 1 package, got %d", len(packages))
	}

	if packages[0].Name != "test-package" {
		t.Errorf("Expected package name 'test-package', got '%s'", packages[0].Name)
	}
}

func TestPackageExists(t *testing.T) {
	tmpDir := t.TempDir()
	pm := NewPackageManager(tmpDir)

	// Create and install test package
	packageDir := createTestPackage(t, tmpDir)
	if err := pm.InstallPackage(packageDir, nil); err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	// Check if package exists
	if !pm.PackageExists("test-package") {
		t.Error("Expected package to exist")
	}

	// Check if nonexistent package exists
	if pm.PackageExists("nonexistent-package") {
		t.Error("Expected nonexistent package to not exist")
	}
}

func TestValidatePackageAtPath(t *testing.T) {
	tmpDir := t.TempDir()
	pm := NewPackageManager(tmpDir)

	// Create test package
	packageDir := createTestPackage(t, tmpDir)

	// Validate package
	err := pm.ValidatePackage(packageDir)
	if err != nil {
		t.Fatalf("ValidatePackage failed: %v", err)
	}
}

func TestGetPackageInstallPath(t *testing.T) {
	tmpDir := t.TempDir()
	pm := NewPackageManager(tmpDir)

	expectedPath := filepath.Join(tmpDir, "packages", "test-package")
	actualPath := pm.GetPackageInstallPath("test-package")

	if actualPath != expectedPath {
		t.Errorf("Expected path %s, got %s", expectedPath, actualPath)
	}
}
