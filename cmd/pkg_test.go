package cmd

import (
	"os"
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
