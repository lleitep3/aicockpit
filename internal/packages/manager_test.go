package packages

import (
	"os"
	"path/filepath"
	"strings"
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

func TestRunPackageHooks_Success(t *testing.T) {
	tmpDir := t.TempDir()
	pm := NewPackageManager(tmpDir)

	// Create a simple script that exits 0
	scriptPath := filepath.Join(tmpDir, "scripts", "ok.sh")
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
		t.Fatalf("Failed to create scripts dir: %v", err)
	}
	if err := os.WriteFile(scriptPath, []byte("#!/bin/sh\necho 'hook ran'\n"), 0o755); err != nil {
		t.Fatalf("Failed to create script: %v", err)
	}

	hooks := []Hook{
		{Script: "scripts/ok.sh", Description: "Test hook"},
	}

	err := pm.RunPackageHooks(tmpDir, hooks)
	if err != nil {
		t.Errorf("RunPackageHooks failed unexpectedly: %v", err)
	}
}

func TestRunPackageHooks_MissingScript(t *testing.T) {
	tmpDir := t.TempDir()
	pm := NewPackageManager(tmpDir)

	// Hook pointing to a non-existent script — should be skipped, not error
	hooks := []Hook{
		{Script: "scripts/nonexistent.sh", Description: "Missing hook"},
	}

	err := pm.RunPackageHooks(tmpDir, hooks)
	if err != nil {
		t.Errorf("Expected missing script to be skipped, got error: %v", err)
	}
}

func TestRunPackageHooks_ScriptFails(t *testing.T) {
	tmpDir := t.TempDir()
	pm := NewPackageManager(tmpDir)

	// Create a script that exits non-zero
	scriptPath := filepath.Join(tmpDir, "scripts", "fail.sh")
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
		t.Fatalf("Failed to create scripts dir: %v", err)
	}
	if err := os.WriteFile(scriptPath, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatalf("Failed to create script: %v", err)
	}

	hooks := []Hook{
		{Script: "scripts/fail.sh", Description: "Failing hook"},
	}

	err := pm.RunPackageHooks(tmpDir, hooks)
	if err == nil {
		t.Error("Expected error when hook script fails, got nil")
	}
}

func TestRunPackageHooks_EmptyHooks(t *testing.T) {
	tmpDir := t.TempDir()
	pm := NewPackageManager(tmpDir)

	// Empty hook list — should be a no-op
	err := pm.RunPackageHooks(tmpDir, []Hook{})
	if err != nil {
		t.Errorf("RunPackageHooks with empty list should not fail: %v", err)
	}
}

func TestRunPackageHooks_NoDescription(t *testing.T) {
	tmpDir := t.TempDir()
	pm := NewPackageManager(tmpDir)

	scriptPath := filepath.Join(tmpDir, "scripts", "nodesc.sh")
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
		t.Fatalf("Failed to create scripts dir: %v", err)
	}
	if err := os.WriteFile(scriptPath, []byte("#!/bin/sh\necho 'ok'\n"), 0o755); err != nil {
		t.Fatalf("Failed to create script: %v", err)
	}

	// Hook with no description — uses script path as fallback
	hooks := []Hook{
		{Script: "scripts/nodesc.sh"},
	}

	err := pm.RunPackageHooks(tmpDir, hooks)
	if err != nil {
		t.Errorf("RunPackageHooks failed: %v", err)
	}
}

// ── SyncPackageAssets ────────────────────────────────────────────────────────

func TestSyncPackageAssets_CopiesAllAssetTypes(t *testing.T) {
	tmpDir := t.TempDir()
	cockpitDir := filepath.Join(tmpDir, "cockpit")
	pm := NewPackageManager(cockpitDir)

	installPath := filepath.Join(tmpDir, "pkg-install")
	assetDirs := []string{
		"skills/my-skill",
		"rules/my-rule",
		"agents/my-agent",
		"workflows/my-flow",
	}
	for _, d := range assetDirs {
		dir := filepath.Join(installPath, d)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("setup: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("# content"), 0o644); err != nil {
			t.Fatalf("setup: %v", err)
		}
	}

	pkg := &Package{
		Name: "test-pkg",
		Features: Features{
			Skills:    []Feature{{Path: "skills/my-skill", Name: "my-skill"}},
			Rules:     []Feature{{Path: "rules/my-rule", Name: "my-rule"}},
			Agents:    []Feature{{Path: "agents/my-agent", Name: "my-agent"}},
			Workflows: []Feature{{Path: "workflows/my-flow", Name: "my-flow"}},
		},
	}

	if err := pm.SyncPackageAssets(pkg, installPath); err != nil {
		t.Fatalf("SyncPackageAssets failed: %v", err)
	}

	expected := []string{
		filepath.Join(cockpitDir, "skills", "my-skill", "SKILL.md"),
		filepath.Join(cockpitDir, "rules", "my-rule", "SKILL.md"),
		filepath.Join(cockpitDir, "agents", "my-agent", "SKILL.md"),
		filepath.Join(cockpitDir, "workflows", "my-flow", "SKILL.md"),
	}
	for _, p := range expected {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected file missing: %s", p)
		}
	}
}

func TestSyncPackageAssets_SkipsMissingSource(t *testing.T) {
	tmpDir := t.TempDir()
	pm := NewPackageManager(tmpDir)

	pkg := &Package{
		Name:     "test-pkg",
		Features: Features{Skills: []Feature{{Path: "skills/ghost", Name: "ghost"}}},
	}
	// Missing source — should warn and skip, not error
	if err := pm.SyncPackageAssets(pkg, tmpDir); err != nil {
		t.Errorf("expected no error for missing source, got: %v", err)
	}
}

func TestSyncPackageAssets_NoFeatures(t *testing.T) {
	tmpDir := t.TempDir()
	pm := NewPackageManager(tmpDir)
	pkg := &Package{Name: "empty-pkg", Features: Features{}}
	if err := pm.SyncPackageAssets(pkg, tmpDir); err != nil {
		t.Errorf("expected no error for empty features, got: %v", err)
	}
}

// ── RemovePackageAssets ──────────────────────────────────────────────────────

func TestRemovePackageAssets_RemovesExistingAssets(t *testing.T) {
	tmpDir := t.TempDir()
	cockpitDir := filepath.Join(tmpDir, "cockpit")
	pm := NewPackageManager(cockpitDir)

	dirs := []string{
		filepath.Join(cockpitDir, "skills", "my-skill"),
		filepath.Join(cockpitDir, "rules", "my-rule"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("setup: %v", err)
		}
	}

	pkg := &Package{
		Name: "test-pkg",
		Features: Features{
			Skills: []Feature{{Name: "my-skill"}},
			Rules:  []Feature{{Name: "my-rule"}},
		},
	}

	if err := pm.RemovePackageAssets(pkg); err != nil {
		t.Fatalf("RemovePackageAssets failed: %v", err)
	}

	for _, d := range dirs {
		if _, err := os.Stat(d); !os.IsNotExist(err) {
			t.Errorf("expected dir to be removed: %s", d)
		}
	}
}

func TestRemovePackageAssets_NoOpWhenNotPresent(t *testing.T) {
	tmpDir := t.TempDir()
	pm := NewPackageManager(tmpDir)
	pkg := &Package{
		Name:     "test-pkg",
		Features: Features{Skills: []Feature{{Name: "ghost-skill"}}},
	}
	if err := pm.RemovePackageAssets(pkg); err != nil {
		t.Errorf("expected no error for already-missing assets, got: %v", err)
	}
}

func TestRemovePackageAssets_NoFeatures(t *testing.T) {
	tmpDir := t.TempDir()
	pm := NewPackageManager(tmpDir)
	pkg := &Package{Name: "empty-pkg", Features: Features{}}
	if err := pm.RemovePackageAssets(pkg); err != nil {
		t.Errorf("expected no error for empty features, got: %v", err)
	}
}

// ── copyDir ──────────────────────────────────────────────────────────────────

func TestCopyDir_CopiesNestedStructure(t *testing.T) {
	tmpDir := t.TempDir()
	pm := NewPackageManager(tmpDir)

	src := filepath.Join(tmpDir, "src")
	if err := os.MkdirAll(filepath.Join(src, "sub"), 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(src, "root.txt"), []byte("root"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(src, "sub", "child.txt"), []byte("child"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	dst := filepath.Join(tmpDir, "dst")
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	if err := pm.copyDir(src, dst); err != nil {
		t.Fatalf("copyDir failed: %v", err)
	}

	for _, rel := range []string{"root.txt", filepath.Join("sub", "child.txt")} {
		p := filepath.Join(dst, rel)
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected copied file missing: %s", p)
		}
	}
}

func TestCopyDir_InvalidSrc(t *testing.T) {
	tmpDir := t.TempDir()
	pm := NewPackageManager(tmpDir)
	err := pm.copyDir("/nonexistent/path", tmpDir)
	if err == nil {
		t.Error("expected error for invalid source dir")
	}
}

// ── TriggerDeploy ────────────────────────────────────────────────────────────

func TestTriggerDeploy_InvalidBinary(t *testing.T) {
	tmpDir := t.TempDir()
	pm := NewPackageManager(tmpDir)
	err := pm.TriggerDeploy("/nonexistent/cockpit-binary")
	if err == nil {
		t.Error("expected error for invalid cockpit binary")
	}
}

func TestTriggerDeploy_FailingCommand(t *testing.T) {
	tmpDir := t.TempDir()
	pm := NewPackageManager(tmpDir)

	script := filepath.Join(tmpDir, "fake-cockpit")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	err := pm.TriggerDeploy(script)
	if err == nil {
		t.Error("expected error when deploy command fails")
	}
}

// ── GoldRules ────────────────────────────────────────────────────────────────

func TestSyncPackageAssets_WritesGoldRules(t *testing.T) {
	tmpDir := t.TempDir()
	cockpitDir := filepath.Join(tmpDir, "cockpit")
	pm := NewPackageManager(cockpitDir)

	pkg := &Package{
		Name: "rtk",
		Features: Features{
			GoldRules: []string{
				"Always prefix terminal commands with rtk",
				"Never run git without rtk prefix",
			},
		},
	}

	if err := pm.SyncPackageAssets(pkg, tmpDir); err != nil {
		t.Fatalf("SyncPackageAssets failed: %v", err)
	}

	goldPath := filepath.Join(cockpitDir, "rules", "rtk-gold-rules.md")
	data, err := os.ReadFile(goldPath)
	if err != nil {
		t.Fatalf("gold rules file not created: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "Always prefix terminal commands with rtk") {
		t.Errorf("expected gold rule in file, got:\n%s", content)
	}
	if !strings.Contains(content, "Never run git without rtk prefix") {
		t.Errorf("expected second gold rule in file, got:\n%s", content)
	}
	if !strings.Contains(content, "# Gold Rules") {
		t.Errorf("expected section header in gold rules file, got:\n%s", content)
	}
}

func TestSyncPackageAssets_NoGoldRulesSkipped(t *testing.T) {
	tmpDir := t.TempDir()
	pm := NewPackageManager(tmpDir)

	pkg := &Package{
		Name:     "no-rules-pkg",
		Features: Features{GoldRules: []string{}},
	}

	if err := pm.SyncPackageAssets(pkg, tmpDir); err != nil {
		t.Errorf("expected no error with empty gold rules, got: %v", err)
	}

	goldPath := filepath.Join(tmpDir, "rules", "no-rules-pkg-gold-rules.md")
	if _, err := os.Stat(goldPath); err == nil {
		t.Error("expected no gold rules file to be created when none defined")
	}
}

func TestRemovePackageAssets_RemovesGoldRules(t *testing.T) {
	tmpDir := t.TempDir()
	cockpitDir := filepath.Join(tmpDir, "cockpit")
	pm := NewPackageManager(cockpitDir)

	// Pre-create the gold rules file
	rulesDir := filepath.Join(cockpitDir, "rules")
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	goldPath := filepath.Join(rulesDir, "rtk-gold-rules.md")
	if err := os.WriteFile(goldPath, []byte("# Gold Rules\n"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	pkg := &Package{
		Name:     "rtk",
		Features: Features{GoldRules: []string{"some rule"}},
	}

	if err := pm.RemovePackageAssets(pkg); err != nil {
		t.Fatalf("RemovePackageAssets failed: %v", err)
	}

	if _, err := os.Stat(goldPath); !os.IsNotExist(err) {
		t.Error("expected gold rules file to be removed after uninstall")
	}
}
