package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestNewPkgCommand(t *testing.T) {
	cmd := NewPkgCommand()

	if cmd.Use != "pkg" {
		t.Errorf("Expected command use 'pkg', got '%s'", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Expected command short description")
	}

	if cmd.Long == "" {
		t.Error("Expected command long description")
	}

	// Check that subcommands are registered
	if len(cmd.Commands()) == 0 {
		t.Error("Expected subcommands to be registered")
	}
}

func TestNewPkgSearchCommand(t *testing.T) {
	cmd := NewPkgSearchCommand()

	if cmd.Use != "search [query]" {
		t.Errorf("Expected command use 'search [query]', got '%s'", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Expected command short description")
	}

	if cmd.Long == "" {
		t.Error("Expected command long description")
	}

	// Check flags
	if cmd.Flag("source") == nil {
		t.Error("Expected 'source' flag")
	}

	if cmd.Flag("category") == nil {
		t.Error("Expected 'category' flag")
	}

	if cmd.Flag("tag") == nil {
		t.Error("Expected 'tag' flag")
	}

	if cmd.Flag("detailed") == nil {
		t.Error("Expected 'detailed' flag")
	}
}

func TestNewPkgSearchCommandExecution(t *testing.T) {
	cmd := NewPkgSearchCommand()

	// Test with no arguments should fail
	err := cmd.RunE(cmd, []string{})
	if err == nil {
		t.Error("Expected error when no query provided")
	}

	// Test with query argument
	err = cmd.RunE(cmd, []string{"hello"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestNewPkgInstallCommand(t *testing.T) {
	cmd := NewPkgInstallCommand()

	if cmd.Use != "install <package>[@version]" {
		t.Errorf("Expected command use 'install <package>[@version]', got '%s'", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Expected command short description")
	}

	if cmd.Long == "" {
		t.Error("Expected command long description")
	}

	// Check flags
	if cmd.Flag("source") == nil {
		t.Error("Expected 'source' flag")
	}

	if cmd.Flag("with-dependencies") == nil {
		t.Error("Expected 'with-dependencies' flag")
	}

	if cmd.Flag("interactive") == nil {
		t.Error("Expected 'interactive' flag")
	}

	if cmd.Flag("force") == nil {
		t.Error("Expected 'force' flag")
	}
}

func TestNewPkgInstallCommandExecution(t *testing.T) {
	cmd := NewPkgInstallCommand()

	// Test with non-existent package - will fail because package doesn't exist
	// This is expected behavior in test environment
	err := cmd.RunE(cmd, []string{"nonexistent-package-xyz"})
	if err == nil {
		t.Error("Expected error when package not found")
	}
}

func TestNewPkgUninstallCommand(t *testing.T) {
	cmd := NewPkgUninstallCommand()

	if cmd.Use != "uninstall <package>" {
		t.Errorf("Expected command use 'uninstall <package>', got '%s'", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Expected command short description")
	}

	if cmd.Long == "" {
		t.Error("Expected command long description")
	}

	// Check flags
	if cmd.Flag("force") == nil {
		t.Error("Expected 'force' flag")
	}
}

func TestNewPkgUninstallCommandExecution(t *testing.T) {
	cmd := NewPkgUninstallCommand()

	// Test with non-existent package - will fail because package is not installed
	// This is expected behavior in test environment
	err := cmd.RunE(cmd, []string{"nonexistent-package-xyz"})
	if err == nil {
		t.Error("Expected error when package not found")
	}
}

func TestNewPkgListCommand(t *testing.T) {
	cmd := NewPkgListCommand()

	if cmd.Use != "list" {
		t.Errorf("Expected command use 'list', got '%s'", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Expected command short description")
	}

	if cmd.Long == "" {
		t.Error("Expected command long description")
	}

	// Check flags
	if cmd.Flag("source") == nil {
		t.Error("Expected 'source' flag")
	}

	if cmd.Flag("detailed") == nil {
		t.Error("Expected 'detailed' flag")
	}
}

func TestNewPkgListCommandExecution(t *testing.T) {
	cmd := NewPkgListCommand()

	// Test execution
	err := cmd.RunE(cmd, []string{})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestPkgCommandHierarchy(t *testing.T) {
	pkgCmd := NewPkgCommand()

	// Check that all subcommands are registered
	subcommands := map[string]bool{
		"search":    false,
		"install":   false,
		"uninstall": false,
		"list":      false,
	}

	for _, cmd := range pkgCmd.Commands() {
		if _, exists := subcommands[cmd.Name()]; exists {
			subcommands[cmd.Name()] = true
		}
	}

	for cmd, found := range subcommands {
		if !found {
			t.Errorf("Expected subcommand '%s' not found", cmd)
		}
	}
}

// ── copyFile ──────────────────────────────────────────────────────────────

func TestCopyFile_Success(t *testing.T) {
	src := filepath.Join(t.TempDir(), "src.txt")
	dst := filepath.Join(t.TempDir(), "dst.txt")

	if err := os.WriteFile(src, []byte("hello"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copyFile() error = %v", err)
	}
	data, _ := os.ReadFile(dst)
	if string(data) != "hello" {
		t.Errorf("dst content = %q, want %q", string(data), "hello")
	}
}

func TestCopyFile_MissingSrc(t *testing.T) {
	err := copyFile("/nonexistent/src.txt", t.TempDir()+"/dst.txt")
	if err == nil {
		t.Error("copyFile() expected error for missing source, got nil")
	}
}

// ── copyDirectory ─────────────────────────────────────────────────────────

func TestCopyDirectory_Success(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := filepath.Join(t.TempDir(), "dst")

	// Create nested structure.
	subDir := filepath.Join(srcDir, "sub")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "a.txt"), []byte("a"), 0o644); err != nil {
		t.Fatalf("WriteFile a.txt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "b.txt"), []byte("b"), 0o644); err != nil {
		t.Fatalf("WriteFile b.txt: %v", err)
	}

	if err := copyDirectory(srcDir, dstDir); err != nil {
		t.Fatalf("copyDirectory() error = %v", err)
	}

	// Verify both files exist in destination.
	for _, rel := range []string{"a.txt", "sub/b.txt"} {
		if _, err := os.Stat(filepath.Join(dstDir, rel)); err != nil {
			t.Errorf("expected %s to exist in dst: %v", rel, err)
		}
	}
}

func TestCopyDirectory_MissingSrc(t *testing.T) {
	err := copyDirectory("/nonexistent/src", t.TempDir()+"/dst")
	if err == nil {
		t.Error("copyDirectory() expected error for missing source, got nil")
	}
}

func TestCopyDirectory_NestedDirs(t *testing.T) {
	src := t.TempDir()
	dst := filepath.Join(t.TempDir(), "dest")

	// Create nested structure: src/sub/file.txt
	subDir := filepath.Join(src, "sub")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "file.txt"), []byte("nested"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "root.txt"), []byte("root"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := copyDirectory(src, dst); err != nil {
		t.Fatalf("copyDirectory nested error = %v", err)
	}

	// Verify nested file
	data, err := os.ReadFile(filepath.Join(dst, "sub", "file.txt"))
	if err != nil || string(data) != "nested" {
		t.Error("nested file not copied correctly")
	}

	// Verify root file
	data, err = os.ReadFile(filepath.Join(dst, "root.txt"))
	if err != nil || string(data) != "root" {
		t.Error("root file not copied correctly")
	}

	// Verify permissions preserved
	info, err := os.Stat(filepath.Join(dst, "root.txt"))
	if err != nil || info.Mode()&0o755 != 0o755 {
		t.Error("file permissions not preserved")
	}
}

func TestCopyFile_PreservesPermissions(t *testing.T) {
	src := filepath.Join(t.TempDir(), "exec.sh")
	if err := os.WriteFile(src, []byte("#!/bin/sh"), 0o755); err != nil {
		t.Fatal(err)
	}

	dst := filepath.Join(t.TempDir(), "exec_copy.sh")
	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copyFile error = %v", err)
	}

	info, err := os.Stat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&0o755 != 0o755 {
		t.Errorf("expected mode 0755, got %o", info.Mode())
	}
}

func TestCopyFile_WriteError(t *testing.T) {
	src := filepath.Join(t.TempDir(), "source.txt")
	os.WriteFile(src, []byte("data"), 0o644)

	// Destination is a directory — can't write a file there
	dstDir := t.TempDir()
	dst := filepath.Join(dstDir, "subdir", "nested", "file.txt")
	// Make the parent read-only so write fails
	os.MkdirAll(filepath.Join(dstDir, "subdir", "nested"), 0o555)
	t.Cleanup(func() { os.Chmod(filepath.Join(dstDir, "subdir", "nested"), 0o755) })

	err := copyFile(src, dst)
	if err == nil {
		t.Error("expected error writing to read-only directory")
	}
}

func TestCopyDirectory_MkdirAllError(t *testing.T) {
	src := t.TempDir()
	os.WriteFile(filepath.Join(src, "f.txt"), []byte("x"), 0o644)

	// Try to create dst where parent is read-only
	parent := t.TempDir()
	os.Chmod(parent, 0o555)
	t.Cleanup(func() { os.Chmod(parent, 0o755) })

	dst := filepath.Join(parent, "newdir")
	err := copyDirectory(src, dst)
	if err == nil {
		t.Error("expected error from MkdirAll on read-only parent")
	}
}

func TestCopyDirectory_UnreadableFile(t *testing.T) {
	src := t.TempDir()
	// Create a file that can't be read
	f := filepath.Join(src, "secret.txt")
	os.WriteFile(f, []byte("secret"), 0o000)
	t.Cleanup(func() { os.Chmod(f, 0o644) })

	dst := filepath.Join(t.TempDir(), "dst")
	err := copyDirectory(src, dst)
	if err == nil {
		t.Error("expected error copying unreadable file")
	}
}

func TestCopyDirectory_RecursiveSubdirError(t *testing.T) {
	src := t.TempDir()
	subdir := filepath.Join(src, "sub")
	os.MkdirAll(subdir, 0o755)
	// Make subdir unreadable so ReadDir fails inside recursive call
	os.Chmod(subdir, 0o000)
	t.Cleanup(func() { os.Chmod(subdir, 0o755) })

	dst := filepath.Join(t.TempDir(), "dst")
	err := copyDirectory(src, dst)
	if err == nil {
		t.Error("expected error from recursive subdir copy")
	}
}

// ── pkg list / install / search RunE ─────────────────────────────────────

func makePkgConfig(t *testing.T) {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	cockpitDir := filepath.Join(tmpDir, ".cockpit")
	if err := os.MkdirAll(cockpitDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	yaml := `version: "0.1.0"
language: en-us
log_level: info
auto_update_check: false
enabled_providers: []
package_registries: []
`
	if err := os.WriteFile(filepath.Join(cockpitDir, "config.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatalf("WriteFile config: %v", err)
	}
}

func TestNewPkgListCommand_NoPackages(t *testing.T) {
	makePkgConfig(t)
	cmd := NewPkgListCommand()
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Errorf("list (no packages) error = %v", err)
	}
}

func TestNewPkgListCommand_UnknownRegistry(t *testing.T) {
	makePkgConfig(t)
	cmd := NewPkgListCommand()
	cmd.SetArgs([]string{"--source", "nonexistent"})
	if err := cmd.Execute(); err == nil {
		t.Error("list --source nonexistent should error")
	}
}

func TestNewPkgSearchCommand_NoResults(t *testing.T) {
	makePkgConfig(t)
	cmd := NewPkgSearchCommand()
	cmd.SetArgs([]string{"no-such-pkg"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("search (no results) error = %v", err)
	}
}

func TestNewPkgInstallCommand_RegistryNotFound(t *testing.T) {
	makePkgConfig(t)
	cmd := NewPkgInstallCommand()
	cmd.SetArgs([]string{"--source", "ghost", "some-pkg"})
	if err := cmd.Execute(); err == nil {
		t.Error("install --source ghost should error")
	}
}

func TestNewPkgInstallCommand_PackageNotFound(t *testing.T) {
	makePkgConfig(t)
	cmd := NewPkgInstallCommand()
	cmd.SetArgs([]string{"totally-unknown-pkg"})
	if err := cmd.Execute(); err == nil {
		t.Error("install unknown package should error")
	}
}

func TestNewPkgUninstallCommand_NotInstalled(t *testing.T) {
	makePkgConfig(t)
	cmd := NewPkgUninstallCommand()
	cmd.SetArgs([]string{"ghost-package"})
	if err := cmd.Execute(); err == nil {
		t.Error("uninstall not-installed package should error")
	}
}

func TestNewPkgUpgradeCommand_NotInstalled(t *testing.T) {
	makePkgConfig(t)
	cmd := NewPkgUpgradeCommand()
	cmd.SetArgs([]string{"ghost-package"})
	if err := cmd.Execute(); err == nil {
		t.Error("upgrade not-installed package should error")
	}
}

func TestNewPkgConfigureCommand_NotInstalled(t *testing.T) {
	makePkgConfig(t)
	cmd := NewPkgConfigureCommand()
	cmd.SetArgs([]string{"ghost-package"})
	if err := cmd.Execute(); err == nil {
		t.Error("configure not-installed package should error")
	}
}

func TestNewPkgValidateCommand_NotInstalled(t *testing.T) {
	makePkgConfig(t)
	cmd := NewPkgValidateCommand()
	cmd.SetArgs([]string{"ghost-package"})
	if err := cmd.Execute(); err == nil {
		t.Error("validate not-installed package should error")
	}
}

// ── Deep RunE tests for pkg install ───────────────────────────────────────

func TestNewPkgInstallCommand_WithVersion(t *testing.T) {
	makePkgConfig(t)
	cmd := NewPkgInstallCommand()
	cmd.SetArgs([]string{"some-pkg@1.0.0"})
	// Should fail because config has no registries
	if err := cmd.Execute(); err == nil {
		t.Error("install with version on no registries should error")
	}
}

func TestNewPkgInstallCommand_WithForce(t *testing.T) {
	makePkgConfig(t)
	cmd := NewPkgInstallCommand()
	cmd.SetArgs([]string{"--force", "some-pkg"})
	if err := cmd.Execute(); err == nil {
		t.Error("install --force with no registries should error")
	}
}

func TestNewPkgInstallCommand_WithDependencies(t *testing.T) {
	makePkgConfig(t)
	cmd := NewPkgInstallCommand()
	cmd.SetArgs([]string{"--with-dependencies", "some-pkg"})
	if err := cmd.Execute(); err == nil {
		t.Error("install --with-dependencies on no registries should error")
	}
}

func TestNewPkgInstallCommand_Interactive(t *testing.T) {
	makePkgConfig(t)
	cmd := NewPkgInstallCommand()
	cmd.SetArgs([]string{"--interactive", "some-pkg"})
	if err := cmd.Execute(); err == nil {
		t.Error("install --interactive on no registries should error")
	}
}

// ── Deep RunE tests for pkg upgrade ───────────────────────────────────────

func TestNewPkgUpgradeCommand_WithVersion(t *testing.T) {
	makePkgConfig(t)
	cmd := NewPkgUpgradeCommand()
	cmd.SetArgs([]string{"some-pkg@2.0.0"})
	if err := cmd.Execute(); err == nil {
		t.Error("upgrade with version on non-installed package should error")
	}
}

func TestNewPkgUpgradeCommand_WithForce(t *testing.T) {
	makePkgConfig(t)
	cmd := NewPkgUpgradeCommand()
	cmd.SetArgs([]string{"--force", "some-pkg"})
	if err := cmd.Execute(); err == nil {
		t.Error("upgrade --force on non-installed package should error")
	}
}

func TestNewPkgUpgradeCommand_WithSource(t *testing.T) {
	makePkgConfig(t)
	cmd := NewPkgUpgradeCommand()
	cmd.SetArgs([]string{"--source", "nonexistent-reg", "some-pkg"})
	if err := cmd.Execute(); err == nil {
		t.Error("upgrade --source nonexistent on non-installed should error")
	}
}

// ── Deep RunE tests for pkg uninstall ─────────────────────────────────────

func TestNewPkgUninstallCommand_WithForce(t *testing.T) {
	makePkgConfig(t)
	cmd := NewPkgUninstallCommand()
	cmd.SetArgs([]string{"--force", "ghost-package"})
	if err := cmd.Execute(); err == nil {
		t.Error("uninstall --force on non-installed package should error")
	}
}

// ── Pkg configure with installed package but no configure script ──────────

func TestNewPkgConfigureCommand_NoConfigure(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	cockpitDir := filepath.Join(tmpDir, ".cockpit")
	pkgDir := filepath.Join(cockpitDir, "packages", "test-pkg")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	// Write minimal manifest
	manifest := `name: test-pkg
version: "1.0.0"
description: Test
author: Test
license: MIT
requirements:
  cockpit: ">=0.1.0"
`
	if err := os.WriteFile(filepath.Join(pkgDir, "cockpit-package.yml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	// Write config
	cfgYaml := `version: "0.1.0"
language: en-us
log_level: info
auto_update_check: false
enabled_providers: []
package_registries: []
`
	if err := os.WriteFile(filepath.Join(cockpitDir, "config.yaml"), []byte(cfgYaml), 0o644); err != nil {
		t.Fatalf("WriteFile config: %v", err)
	}

	cmd := NewPkgConfigureCommand()
	cmd.SetArgs([]string{"test-pkg"})
	// Should fail because no configure script exists
	if err := cmd.Execute(); err == nil {
		t.Error("configure with no configure script should error")
	}
}

func TestNewPkgConfigureCommand_WithScript(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	cockpitDir := filepath.Join(tmpDir, ".cockpit")
	pkgDir := filepath.Join(cockpitDir, "packages", "test-pkg")
	binDir := filepath.Join(pkgDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	manifest := `name: test-pkg
version: "1.0.0"
description: Test
author: Test
license: MIT
requirements:
  cockpit: ">=0.1.0"
`
	if err := os.WriteFile(filepath.Join(pkgDir, "cockpit-package.yml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile manifest: %v", err)
	}
	// Write configure script
	if err := os.WriteFile(filepath.Join(binDir, "configure"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("WriteFile configure: %v", err)
	}
	cfgYaml := `version: "0.1.0"
language: en-us
log_level: info
auto_update_check: false
enabled_providers: []
package_registries: []
`
	if err := os.WriteFile(filepath.Join(cockpitDir, "config.yaml"), []byte(cfgYaml), 0o644); err != nil {
		t.Fatalf("WriteFile config: %v", err)
	}

	cmd := NewPkgConfigureCommand()
	cmd.SetArgs([]string{"test-pkg"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("configure with valid script error = %v", err)
	}
}

func TestNewPkgConfigureCommand_ScriptFails(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	cockpitDir := filepath.Join(tmpDir, ".cockpit")
	pkgDir := filepath.Join(cockpitDir, "packages", "test-pkg")
	binDir := filepath.Join(pkgDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	manifest := `name: test-pkg
version: "1.0.0"
description: Test
author: Test
license: MIT
requirements:
  cockpit: ">=0.1.0"
`
	if err := os.WriteFile(filepath.Join(pkgDir, "cockpit-package.yml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile manifest: %v", err)
	}
	// Write a failing configure script
	if err := os.WriteFile(filepath.Join(binDir, "configure"), []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatalf("WriteFile configure: %v", err)
	}
	cfgYaml := `version: "0.1.0"
language: en-us
log_level: info
auto_update_check: false
enabled_providers: []
package_registries: []
`
	if err := os.WriteFile(filepath.Join(cockpitDir, "config.yaml"), []byte(cfgYaml), 0o644); err != nil {
		t.Fatalf("WriteFile config: %v", err)
	}

	cmd := NewPkgConfigureCommand()
	cmd.SetArgs([]string{"test-pkg"})
	if err := cmd.Execute(); err == nil {
		t.Error("configure with failing script should error")
	}
}

// ── Pkg validate with installed package ───────────────────────────────────

func TestNewPkgValidateCommand_NoValidateScript(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	cockpitDir := filepath.Join(tmpDir, ".cockpit")
	pkgDir := filepath.Join(cockpitDir, "packages", "test-pkg")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	manifest := `name: test-pkg
version: "1.0.0"
description: Test
author: Test
license: MIT
requirements:
  cockpit: ">=0.1.0"
`
	if err := os.WriteFile(filepath.Join(pkgDir, "cockpit-package.yml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	cfgYaml := `version: "0.1.0"
language: en-us
log_level: info
auto_update_check: false
enabled_providers: []
package_registries: []
`
	if err := os.WriteFile(filepath.Join(cockpitDir, "config.yaml"), []byte(cfgYaml), 0o644); err != nil {
		t.Fatalf("WriteFile config: %v", err)
	}

	cmd := NewPkgValidateCommand()
	cmd.SetArgs([]string{"test-pkg"})
	if err := cmd.Execute(); err == nil {
		t.Error("validate with no validate script should error")
	}
}

func TestNewPkgValidateCommand_WithScript(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	cockpitDir := filepath.Join(tmpDir, ".cockpit")
	pkgDir := filepath.Join(cockpitDir, "packages", "test-pkg")
	binDir := filepath.Join(pkgDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	manifest := `name: test-pkg
version: "1.0.0"
description: Test
author: Test
license: MIT
requirements:
  cockpit: ">=0.1.0"
`
	if err := os.WriteFile(filepath.Join(pkgDir, "cockpit-package.yml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "validate"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("WriteFile validate: %v", err)
	}
	cfgYaml := `version: "0.1.0"
language: en-us
log_level: info
auto_update_check: false
enabled_providers: []
package_registries: []
`
	if err := os.WriteFile(filepath.Join(cockpitDir, "config.yaml"), []byte(cfgYaml), 0o644); err != nil {
		t.Fatalf("WriteFile config: %v", err)
	}

	cmd := NewPkgValidateCommand()
	cmd.SetArgs([]string{"test-pkg"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("validate with valid script error = %v", err)
	}
}

func TestNewPkgValidateCommand_ScriptFails(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	cockpitDir := filepath.Join(tmpDir, ".cockpit")
	pkgDir := filepath.Join(cockpitDir, "packages", "test-pkg")
	binDir := filepath.Join(pkgDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	manifest := `name: test-pkg
version: "1.0.0"
description: Test
author: Test
license: MIT
requirements:
  cockpit: ">=0.1.0"
`
	if err := os.WriteFile(filepath.Join(pkgDir, "cockpit-package.yml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "validate"), []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatalf("WriteFile validate: %v", err)
	}
	cfgYaml := `version: "0.1.0"
language: en-us
log_level: info
auto_update_check: false
enabled_providers: []
package_registries: []
`
	if err := os.WriteFile(filepath.Join(cockpitDir, "config.yaml"), []byte(cfgYaml), 0o644); err != nil {
		t.Fatalf("WriteFile config: %v", err)
	}

	cmd := NewPkgValidateCommand()
	cmd.SetArgs([]string{"test-pkg"})
	if err := cmd.Execute(); err == nil {
		t.Error("validate with failing script should error")
	}
}

// ── Pkg install with installed package (already exists) ───────────────────

func TestNewPkgInstallCommand_AlreadyInstalled(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	cockpitDir := filepath.Join(tmpDir, ".cockpit")
	pkgDir := filepath.Join(cockpitDir, "packages", "test-pkg")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	manifest := `name: test-pkg
version: "1.0.0"
description: Test
author: Test
license: MIT
requirements:
  cockpit: ">=0.1.0"
`
	if err := os.WriteFile(filepath.Join(pkgDir, "cockpit-package.yml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	cfgYaml := `version: "0.1.0"
language: en-us
log_level: info
auto_update_check: false
enabled_providers: []
package_registries:
  - name: test-registry
    url: /tmp/fake-registry
    enabled: true
`
	if err := os.WriteFile(filepath.Join(cockpitDir, "config.yaml"), []byte(cfgYaml), 0o644); err != nil {
		t.Fatalf("WriteFile config: %v", err)
	}

	cmd := NewPkgInstallCommand()
	cmd.SetArgs([]string{"test-pkg"})
	// Should fail because package already exists
	if err := cmd.Execute(); err == nil {
		t.Error("install already installed package (without --force) should error")
	}
}

// ── Pkg uninstall on installed package ────────────────────────────────────

func TestNewPkgUninstallCommand_Installed(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	cockpitDir := filepath.Join(tmpDir, ".cockpit")
	pkgDir := filepath.Join(cockpitDir, "packages", "test-pkg")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	manifest := `name: test-pkg
version: "1.0.0"
description: Test
author: Test
license: MIT
requirements:
  cockpit: ">=0.1.0"
`
	if err := os.WriteFile(filepath.Join(pkgDir, "cockpit-package.yml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	cfgYaml := `version: "0.1.0"
language: en-us
log_level: info
auto_update_check: false
enabled_providers: []
package_registries: []
`
	if err := os.WriteFile(filepath.Join(cockpitDir, "config.yaml"), []byte(cfgYaml), 0o644); err != nil {
		t.Fatalf("WriteFile config: %v", err)
	}

	cmd := NewPkgUninstallCommand()
	cmd.SetArgs([]string{"test-pkg"})
	// Should succeed — uninstall a real package
	if err := cmd.Execute(); err != nil {
		t.Errorf("uninstall installed package error = %v", err)
	}
}

// ── Pkg upgrade on installed package with no registry ─────────────────────

func TestNewPkgUpgradeCommand_InstalledNoRegistry(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	cockpitDir := filepath.Join(tmpDir, ".cockpit")
	pkgDir := filepath.Join(cockpitDir, "packages", "test-pkg")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	manifest := `name: test-pkg
version: "1.0.0"
description: Test
author: Test
license: MIT
requirements:
  cockpit: ">=0.1.0"
`
	if err := os.WriteFile(filepath.Join(pkgDir, "cockpit-package.yml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	cfgYaml := `version: "0.1.0"
language: en-us
log_level: info
auto_update_check: false
enabled_providers: []
package_registries: []
`
	if err := os.WriteFile(filepath.Join(cockpitDir, "config.yaml"), []byte(cfgYaml), 0o644); err != nil {
		t.Fatalf("WriteFile config: %v", err)
	}

	cmd := NewPkgUpgradeCommand()
	cmd.SetArgs([]string{"test-pkg"})
	// Should fail because no registries have the package
	if err := cmd.Execute(); err == nil {
		t.Error("upgrade with no registry should error")
	}
}

// ── Full pkg install/upgrade/search integration tests ─────────────────────

// setupLocalGitRegistry creates a local bare git repo that serves as a registry,
// then configures the cockpit config to use it. Returns the HOME dir.
func setupLocalGitRegistry(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	// Prevent TriggerDeploy from re-invoking the test binary (which hangs)
	t.Setenv("COCKPIT_SKIP_DEPLOY", "1")

	cockpitDir := filepath.Join(tmpDir, ".cockpit")
	packagesDir := filepath.Join(cockpitDir, "packages")
	os.MkdirAll(packagesDir, 0o755)

	// Create a working directory with registry content
	workDir := filepath.Join(tmpDir, "registry-work")
	os.MkdirAll(filepath.Join(workDir, "hello-pkg", "skills"), 0o755)

	// Write package-index.yaml
	os.WriteFile(filepath.Join(workDir, "package-index.yaml"), []byte(`packages:
  - name: hello-pkg
    version: "2.0.0"
    author: TestAuthor
    description: A test package
    license: MIT
`), 0o644)

	// Write manifest with a skill so validation passes, but TriggerDeploy will
	// call os.Executable (the test binary). We accept the deploy warning.
	os.WriteFile(filepath.Join(workDir, "hello-pkg", "cockpit-package.yml"), []byte(`name: hello-pkg
version: "2.0.0"
description: A test package
author: TestAuthor
license: MIT
requirements:
  cockpit: ">=0.1.0"
features:
  skills:
    - name: hello-skill
      path: skills/hello.md
installation:
  supported_providers:
    - all
  provider_features:
    all:
      - skills
`), 0o644)

	// Write feature file
	os.MkdirAll(filepath.Join(workDir, "hello-pkg", "skills"), 0o755)
	os.WriteFile(filepath.Join(workDir, "hello-pkg", "skills", "hello.md"), []byte("# Hello Skill\n"), 0o644)

	// Init git repo, commit, and use it as a bare repo reference
	for _, args := range [][]string{
		{"git", "-C", workDir, "init", "-b", "main"},
		{"git", "-C", workDir, "config", "user.email", "test@test.com"},
		{"git", "-C", workDir, "config", "user.name", "Test"},
		{"git", "-C", workDir, "add", "."},
		{"git", "-C", workDir, "commit", "-m", "init"},
	} {
		c := exec.Command(args[0], args[1:]...)
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git setup %v failed: %v\n%s", args, err, out)
		}
	}

	// Create a bare clone to use as the "remote" registry URL
	bareDir := filepath.Join(tmpDir, "registry.git")
	c := exec.Command("git", "clone", "--bare", workDir, bareDir)
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("git clone --bare failed: %v\n%s", err, out)
	}

	// Write config pointing to the local bare repo
	cfgYaml := `version: "0.1.0"
language: en-us
log_level: info
auto_update_check: false
enabled_providers: []
package_registries:
  - name: test-registry
    url: ` + bareDir + `
    branch: main
    enabled: true
    priority: 1
`
	os.WriteFile(filepath.Join(cockpitDir, "config.yaml"), []byte(cfgYaml), 0o644)

	return tmpDir
}

func TestNewPkgInstallCommand_FullInstall(t *testing.T) {
	setupLocalGitRegistry(t)

	cmd := NewPkgInstallCommand()
	cmd.SetArgs([]string{"hello-pkg"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("pkg install error = %v", err)
	}
}

func TestNewPkgInstallCommand_IntegrationWithVersion(t *testing.T) {
	setupLocalGitRegistry(t)

	cmd := NewPkgInstallCommand()
	cmd.SetArgs([]string{"hello-pkg@2.0.0"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("pkg install @version error = %v", err)
	}
}

func TestNewPkgInstallCommand_IntegrationWrongVersion(t *testing.T) {
	setupLocalGitRegistry(t)

	cmd := NewPkgInstallCommand()
	cmd.SetArgs([]string{"hello-pkg@9.9.9"})
	if err := cmd.Execute(); err == nil {
		t.Error("expected error for wrong version")
	}
}

func TestNewPkgInstallCommand_IntegrationSource(t *testing.T) {
	setupLocalGitRegistry(t)

	cmd := NewPkgInstallCommand()
	cmd.SetArgs([]string{"--source", "test-registry", "hello-pkg"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("pkg install --source error = %v", err)
	}
}

func TestNewPkgInstallCommand_IntegrationSourceNotFound(t *testing.T) {
	setupLocalGitRegistry(t)

	cmd := NewPkgInstallCommand()
	cmd.SetArgs([]string{"--source", "nonexistent", "hello-pkg"})
	if err := cmd.Execute(); err == nil {
		t.Error("expected error for nonexistent source")
	}
}

func TestNewPkgInstallCommand_IntegrationAlreadyInstalled(t *testing.T) {
	tmpDir := setupLocalGitRegistry(t)

	// Install first
	cmd := NewPkgInstallCommand()
	cmd.SetArgs([]string{"hello-pkg"})
	_ = cmd.Execute()

	// Try again without --force
	cmd2 := NewPkgInstallCommand()
	cmd2.SetArgs([]string{"hello-pkg"})
	if err := cmd2.Execute(); err == nil {
		t.Error("expected error: already installed")
	}

	_ = tmpDir
}

func TestNewPkgInstallCommand_IntegrationForce(t *testing.T) {
	setupLocalGitRegistry(t)

	// Install first
	cmd := NewPkgInstallCommand()
	cmd.SetArgs([]string{"hello-pkg"})
	_ = cmd.Execute()

	// Force reinstall
	cmd2 := NewPkgInstallCommand()
	cmd2.SetArgs([]string{"--force", "hello-pkg"})
	if err := cmd2.Execute(); err != nil {
		t.Errorf("pkg install --force error = %v", err)
	}
}

func TestNewPkgInstallCommand_IntegrationInteractive(t *testing.T) {
	setupLocalGitRegistry(t)

	cmd := NewPkgInstallCommand()
	cmd.SetArgs([]string{"--interactive", "hello-pkg"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("pkg install --interactive error = %v", err)
	}
}

func TestNewPkgUpgradeCommand_FullUpgrade(t *testing.T) {
	tmpDir := setupLocalGitRegistry(t)

	// Install old version first (fake)
	cockpitDir := filepath.Join(tmpDir, ".cockpit")
	pkgDir := filepath.Join(cockpitDir, "packages", "hello-pkg")
	os.MkdirAll(pkgDir, 0o755)
	os.WriteFile(filepath.Join(pkgDir, "cockpit-package.yml"), []byte(`name: hello-pkg
version: "1.0.0"
description: A test package
author: TestAuthor
license: MIT
requirements:
  cockpit: ">=0.1.0"
`), 0o644)

	cmd := NewPkgUpgradeCommand()
	cmd.SetArgs([]string{"hello-pkg"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("pkg upgrade error = %v", err)
	}
}

func TestNewPkgUpgradeCommand_AlreadyUpToDate(t *testing.T) {
	tmpDir := setupLocalGitRegistry(t)

	cockpitDir := filepath.Join(tmpDir, ".cockpit")
	pkgDir := filepath.Join(cockpitDir, "packages", "hello-pkg")
	os.MkdirAll(pkgDir, 0o755)
	os.WriteFile(filepath.Join(pkgDir, "cockpit-package.yml"), []byte(`name: hello-pkg
version: "2.0.0"
description: A test package
author: TestAuthor
license: MIT
requirements:
  cockpit: ">=0.1.0"
`), 0o644)

	cmd := NewPkgUpgradeCommand()
	cmd.SetArgs([]string{"hello-pkg"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("pkg upgrade (up to date) error = %v", err)
	}
}

func TestNewPkgUpgradeCommand_IntegrationNotInstalled(t *testing.T) {
	setupLocalGitRegistry(t)

	cmd := NewPkgUpgradeCommand()
	cmd.SetArgs([]string{"nonexistent-pkg"})
	if err := cmd.Execute(); err == nil {
		t.Error("expected error: package not installed")
	}
}

func TestNewPkgUpgradeCommand_IntegrationWithSource(t *testing.T) {
	tmpDir := setupLocalGitRegistry(t)

	cockpitDir := filepath.Join(tmpDir, ".cockpit")
	pkgDir := filepath.Join(cockpitDir, "packages", "hello-pkg")
	os.MkdirAll(pkgDir, 0o755)
	os.WriteFile(filepath.Join(pkgDir, "cockpit-package.yml"), []byte(`name: hello-pkg
version: "1.0.0"
description: A test package
author: TestAuthor
license: MIT
requirements:
  cockpit: ">=0.1.0"
`), 0o644)

	cmd := NewPkgUpgradeCommand()
	cmd.SetArgs([]string{"--source", "test-registry", "hello-pkg"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("pkg upgrade --source error = %v", err)
	}
}

func TestNewPkgUpgradeCommand_SourceNotFound(t *testing.T) {
	tmpDir := setupLocalGitRegistry(t)

	cockpitDir := filepath.Join(tmpDir, ".cockpit")
	pkgDir := filepath.Join(cockpitDir, "packages", "hello-pkg")
	os.MkdirAll(pkgDir, 0o755)
	os.WriteFile(filepath.Join(pkgDir, "cockpit-package.yml"), []byte(`name: hello-pkg
version: "1.0.0"
description: A test package
author: TestAuthor
license: MIT
requirements:
  cockpit: ">=0.1.0"
`), 0o644)

	cmd := NewPkgUpgradeCommand()
	cmd.SetArgs([]string{"--source", "nonexistent", "hello-pkg"})
	if err := cmd.Execute(); err == nil {
		t.Error("expected error: registry not found")
	}
}

func TestNewPkgUpgradeCommand_Force(t *testing.T) {
	tmpDir := setupLocalGitRegistry(t)

	cockpitDir := filepath.Join(tmpDir, ".cockpit")
	pkgDir := filepath.Join(cockpitDir, "packages", "hello-pkg")
	os.MkdirAll(pkgDir, 0o755)
	os.WriteFile(filepath.Join(pkgDir, "cockpit-package.yml"), []byte(`name: hello-pkg
version: "2.0.0"
description: A test package
author: TestAuthor
license: MIT
requirements:
  cockpit: ">=0.1.0"
`), 0o644)

	// Force upgrade even when already up to date
	cmd := NewPkgUpgradeCommand()
	cmd.SetArgs([]string{"--force", "hello-pkg"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("pkg upgrade --force error = %v", err)
	}
}

func TestNewPkgUpgradeCommand_WrongVersion(t *testing.T) {
	tmpDir := setupLocalGitRegistry(t)

	cockpitDir := filepath.Join(tmpDir, ".cockpit")
	pkgDir := filepath.Join(cockpitDir, "packages", "hello-pkg")
	os.MkdirAll(pkgDir, 0o755)
	os.WriteFile(filepath.Join(pkgDir, "cockpit-package.yml"), []byte(`name: hello-pkg
version: "1.0.0"
description: A test package
author: TestAuthor
license: MIT
requirements:
  cockpit: ">=0.1.0"
`), 0o644)

	cmd := NewPkgUpgradeCommand()
	cmd.SetArgs([]string{"hello-pkg@9.9.9"})
	if err := cmd.Execute(); err == nil {
		t.Error("expected error: version not found")
	}
}

func setupLocalGitRegistryWithHooks(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("COCKPIT_SKIP_DEPLOY", "1")

	cockpitDir := filepath.Join(tmpDir, ".cockpit")
	packagesDir := filepath.Join(cockpitDir, "packages")
	os.MkdirAll(packagesDir, 0o755)

	// Create a working directory with registry content
	workDir := filepath.Join(tmpDir, "registry-work")
	os.MkdirAll(filepath.Join(workDir, "hook-pkg", "skills"), 0o755)
	os.MkdirAll(filepath.Join(workDir, "hook-pkg", "bin"), 0o755)

	// Write package-index.yaml
	os.WriteFile(filepath.Join(workDir, "package-index.yaml"), []byte(`packages:
  - name: hook-pkg
    version: "1.0.0"
    author: TestAuthor
    description: A package with hooks
    license: MIT
`), 0o644)

	// Write hook scripts
	os.WriteFile(filepath.Join(workDir, "hook-pkg", "bin", "pre-install.sh"), []byte("#!/bin/sh\necho pre-install\n"), 0o755)
	os.WriteFile(filepath.Join(workDir, "hook-pkg", "bin", "post-install.sh"), []byte("#!/bin/sh\necho post-install\n"), 0o755)

	// Write manifest with hooks
	os.WriteFile(filepath.Join(workDir, "hook-pkg", "cockpit-package.yml"), []byte(`name: hook-pkg
version: "1.0.0"
description: A package with hooks
author: TestAuthor
license: MIT
requirements:
  cockpit: ">=0.1.0"
features:
  skills:
    - name: hook-skill
      path: skills/hook.md
installation:
  supported_providers:
    - all
  provider_features:
    all:
      - skills
  pre_install:
    - script: "bin/pre-install.sh"
      description: "Run pre-install"
  post_install:
    - script: "bin/post-install.sh"
      description: "Run post-install"
`), 0o644)

	// Write feature file
	os.WriteFile(filepath.Join(workDir, "hook-pkg", "skills", "hook.md"), []byte("# Hook Skill\n"), 0o644)

	// Init git repo
	for _, args := range [][]string{
		{"git", "-C", workDir, "init", "-b", "main"},
		{"git", "-C", workDir, "config", "user.email", "test@test.com"},
		{"git", "-C", workDir, "config", "user.name", "Test"},
		{"git", "-C", workDir, "add", "."},
		{"git", "-C", workDir, "commit", "-m", "init"},
	} {
		c := exec.Command(args[0], args[1:]...)
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git setup %v failed: %v\n%s", args, err, out)
		}
	}

	bareDir := filepath.Join(tmpDir, "registry.git")
	c := exec.Command("git", "clone", "--bare", workDir, bareDir)
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("git clone --bare failed: %v\n%s", err, out)
	}

	cfgYaml := `version: "0.1.0"
language: en-us
log_level: info
auto_update_check: false
enabled_providers: []
package_registries:
  - name: test-registry
    url: ` + bareDir + `
    branch: main
    enabled: true
    priority: 0
`
	os.WriteFile(filepath.Join(cockpitDir, "config.yaml"), []byte(cfgYaml), 0o644)
	return tmpDir
}

func TestNewPkgInstallCommand_WithHooks(t *testing.T) {
	setupLocalGitRegistryWithHooks(t)

	cmd := NewPkgInstallCommand()
	cmd.SetArgs([]string{"hook-pkg"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("pkg install with hooks error = %v", err)
	}
}

func TestNewPkgInstallCommand_WithInteractive(t *testing.T) {
	setupLocalGitRegistry(t)

	cmd := NewPkgInstallCommand()
	cmd.SetArgs([]string{"--interactive", "hello-pkg"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("pkg install --interactive error = %v", err)
	}
}

func TestNewPkgInstallCommand_IntegrationWithDependencies(t *testing.T) {
	// The hello-pkg has no dependencies, so --with-dependencies is a no-op path but exercises the flag
	setupLocalGitRegistry(t)

	cmd := NewPkgInstallCommand()
	cmd.SetArgs([]string{"--force", "--with-dependencies", "hello-pkg"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("pkg install --with-dependencies error = %v", err)
	}
}

func TestNewPkgSearchCommand_FullSearch(t *testing.T) {
	setupLocalGitRegistry(t)

	cmd := NewPkgSearchCommand()
	cmd.SetArgs([]string{"hello"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("pkg search error = %v", err)
	}
}

func TestNewPkgSearchCommand_NotFound(t *testing.T) {
	setupLocalGitRegistry(t)

	cmd := NewPkgSearchCommand()
	cmd.SetArgs([]string{"nonexistent-xyz"})
	// Search with no results should still succeed (prints "no packages found")
	if err := cmd.Execute(); err != nil {
		t.Errorf("pkg search (no results) error = %v", err)
	}
}

func TestNewPkgSearchCommand_Detailed(t *testing.T) {
	setupLocalGitRegistry(t)

	cmd := NewPkgSearchCommand()
	cmd.SetArgs([]string{"--detailed", "hello"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("pkg search --detailed error = %v", err)
	}
}

func TestNewPkgSearchCommand_NoQuery(t *testing.T) {
	setupLocalGitRegistry(t)

	cmd := NewPkgSearchCommand()
	cmd.SetArgs([]string{})
	// No query, no category, no tag → error
	if err := cmd.Execute(); err == nil {
		t.Error("expected error: no search criteria")
	}
}

func TestNewPkgSearchCommand_Source(t *testing.T) {
	setupLocalGitRegistry(t)

	cmd := NewPkgSearchCommand()
	cmd.SetArgs([]string{"--source", "test-registry", "hello"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("pkg search --source error = %v", err)
	}
}

func TestNewPkgSearchCommand_SourceNotFound(t *testing.T) {
	setupLocalGitRegistry(t)

	cmd := NewPkgSearchCommand()
	cmd.SetArgs([]string{"--source", "nonexistent", "hello"})
	if err := cmd.Execute(); err == nil {
		t.Error("expected error: registry not found")
	}
}

func TestNewPkgSearchCommand_Category(t *testing.T) {
	setupLocalGitRegistry(t)

	cmd := NewPkgSearchCommand()
	cmd.SetArgs([]string{"--category", "test"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("pkg search --category error = %v", err)
	}
}

func TestNewPkgSearchCommand_Tag(t *testing.T) {
	setupLocalGitRegistry(t)

	cmd := NewPkgSearchCommand()
	cmd.SetArgs([]string{"--tag", "test"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("pkg search --tag error = %v", err)
	}
}

func TestNewPkgListCommand_WithPackages(t *testing.T) {
	setupLocalGitRegistry(t)

	// List — should find hello-pkg in registry
	listCmd := NewPkgListCommand()
	if err := listCmd.Execute(); err != nil {
		t.Errorf("pkg list (with packages) error = %v", err)
	}
}

func TestNewPkgListCommand_Detailed(t *testing.T) {
	setupLocalGitRegistry(t)

	listCmd := NewPkgListCommand()
	listCmd.SetArgs([]string{"--detailed"})
	if err := listCmd.Execute(); err != nil {
		t.Errorf("pkg list --detailed error = %v", err)
	}
}

func TestNewPkgListCommand_Source(t *testing.T) {
	setupLocalGitRegistry(t)

	listCmd := NewPkgListCommand()
	listCmd.SetArgs([]string{"--source", "test-registry"})
	if err := listCmd.Execute(); err != nil {
		t.Errorf("pkg list --source error = %v", err)
	}
}

func TestNewPkgListCommand_SourceNotFound(t *testing.T) {
	setupLocalGitRegistry(t)

	listCmd := NewPkgListCommand()
	listCmd.SetArgs([]string{"--source", "nonexistent"})
	if err := listCmd.Execute(); err == nil {
		t.Error("expected error: registry not found")
	}
}

// ── Full pkg uninstall integration ────────────────────────────────────────

func TestNewPkgUninstallCommand_FullUninstall(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cockpitDir := filepath.Join(tmpDir, ".cockpit")
	pkgDir := filepath.Join(cockpitDir, "packages", "hello-pkg")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	manifest := `name: hello-pkg
version: "1.0.0"
description: A test package
author: TestAuthor
license: MIT
requirements:
  cockpit: ">=0.1.0"
installation:
  supported_providers:
    - all
  provider_features:
    all:
      - skills
  post_uninstall:
    - script: "echo goodbye"
      description: "Say goodbye"
`
	if err := os.WriteFile(filepath.Join(pkgDir, "cockpit-package.yml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	cfgYaml := `version: "0.1.0"
language: en-us
log_level: info
auto_update_check: false
enabled_providers: []
package_registries: []
`
	if err := os.WriteFile(filepath.Join(cockpitDir, "config.yaml"), []byte(cfgYaml), 0o644); err != nil {
		t.Fatalf("WriteFile config: %v", err)
	}

	cmd := NewPkgUninstallCommand()
	cmd.SetArgs([]string{"hello-pkg"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("pkg uninstall error = %v", err)
	}
}

func TestNewPkgUninstallCommand_WithAssets(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("COCKPIT_SKIP_DEPLOY", "1")

	cockpitDir := filepath.Join(tmpDir, ".cockpit")
	pkgDir := filepath.Join(cockpitDir, "packages", "asset-pkg")
	skillsDir := filepath.Join(pkgDir, "skills")
	os.MkdirAll(skillsDir, 0o755)
	os.WriteFile(filepath.Join(skillsDir, "my-skill.md"), []byte("# Skill\n"), 0o644)

	manifest := `name: asset-pkg
version: "1.0.0"
description: Asset test
author: TestAuthor
license: MIT
requirements:
  cockpit: ">=0.1.0"
features:
  skills:
    - name: my-skill
      path: skills/my-skill.md
installation:
  supported_providers:
    - all
  provider_features:
    all:
      - skills
`
	os.WriteFile(filepath.Join(pkgDir, "cockpit-package.yml"), []byte(manifest), 0o644)
	cfgYaml := `version: "0.1.0"
language: en-us
log_level: info
auto_update_check: false
enabled_providers: []
package_registries: []
`
	os.WriteFile(filepath.Join(cockpitDir, "config.yaml"), []byte(cfgYaml), 0o644)

	cmd := NewPkgUninstallCommand()
	cmd.SetArgs([]string{"asset-pkg"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("pkg uninstall with assets error = %v", err)
	}
}

func TestNewPkgUninstallCommand_WithPreUninstallHook(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cockpitDir := filepath.Join(tmpDir, ".cockpit")
	pkgDir := filepath.Join(cockpitDir, "packages", "hook-pkg")
	binDir := filepath.Join(pkgDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	// Write hook script
	if err := os.WriteFile(filepath.Join(binDir, "pre-uninstall.sh"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("WriteFile hook: %v", err)
	}
	manifest := `name: hook-pkg
version: "1.0.0"
description: Hook test
author: TestAuthor
license: MIT
requirements:
  cockpit: ">=0.1.0"
installation:
  supported_providers:
    - all
  provider_features:
    all:
      - skills
  pre_uninstall:
    - script: "bin/pre-uninstall.sh"
      description: "Pre-uninstall cleanup"
`
	if err := os.WriteFile(filepath.Join(pkgDir, "cockpit-package.yml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile manifest: %v", err)
	}
	cfgYaml := `version: "0.1.0"
language: en-us
log_level: info
auto_update_check: false
enabled_providers: []
package_registries: []
`
	if err := os.WriteFile(filepath.Join(cockpitDir, "config.yaml"), []byte(cfgYaml), 0o644); err != nil {
		t.Fatalf("WriteFile config: %v", err)
	}

	cmd := NewPkgUninstallCommand()
	cmd.SetArgs([]string{"hook-pkg"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("pkg uninstall with pre-uninstall hook error = %v", err)
	}
}

func TestNewPkgUninstallCommand_WithFailingHook_Force(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cockpitDir := filepath.Join(tmpDir, ".cockpit")
	pkgDir := filepath.Join(cockpitDir, "packages", "fail-pkg")
	binDir := filepath.Join(pkgDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "pre-uninstall.sh"), []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatalf("WriteFile hook: %v", err)
	}
	manifest := `name: fail-pkg
version: "1.0.0"
description: Fail hook test
author: TestAuthor
license: MIT
requirements:
  cockpit: ">=0.1.0"
installation:
  supported_providers:
    - all
  provider_features:
    all:
      - skills
  pre_uninstall:
    - script: "bin/pre-uninstall.sh"
      description: "Pre-uninstall failing hook"
`
	if err := os.WriteFile(filepath.Join(pkgDir, "cockpit-package.yml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile manifest: %v", err)
	}
	cfgYaml := `version: "0.1.0"
language: en-us
log_level: info
auto_update_check: false
enabled_providers: []
package_registries: []
`
	if err := os.WriteFile(filepath.Join(cockpitDir, "config.yaml"), []byte(cfgYaml), 0o644); err != nil {
		t.Fatalf("WriteFile config: %v", err)
	}

	// Without --force, should error
	cmd := NewPkgUninstallCommand()
	cmd.SetArgs([]string{"fail-pkg"})
	if err := cmd.Execute(); err == nil {
		t.Error("uninstall with failing hook (no --force) should error")
	}

	// With --force, should succeed
	cmd2 := NewPkgUninstallCommand()
	cmd2.SetArgs([]string{"--force", "fail-pkg"})
	if err := cmd2.Execute(); err != nil {
		t.Errorf("uninstall --force with failing hook error = %v", err)
	}
}
