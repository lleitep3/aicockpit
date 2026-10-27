package packages

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTestManifest(t *testing.T, dir string, name string) {
	t.Helper()
	manifestContent := `name: "` + name + `"
version: "1.0.0"
description: "Test package"
author: "Test Author"
license: "MIT"
type: "utility"

requirements:
  cockpit: "0.2.0"

features:
  modules:
    - path: "modules/cmd.go"
      name: "` + name + `"
      description: "Test module"

installation:
  supported_providers:
    - devin
  provider_features:
    devin:
      - modules
  method: "symlink"
`
	manifestPath := filepath.Join(dir, "cockpit-package.yml")
	if err := os.WriteFile(manifestPath, []byte(manifestContent), 0o644); err != nil {
		t.Fatalf("Failed to create test manifest: %v", err)
	}
}

func TestNewPackageLoader(t *testing.T) {
	cockpitDir := t.TempDir()
	loader := NewPackageLoader(cockpitDir)

	expected := filepath.Join(cockpitDir, "packages")
	if loader.packagesDir != expected {
		t.Errorf("Expected packagesDir to be %q, got %q", expected, loader.packagesDir)
	}
}

func TestPackageLoader_GetPackagePath(t *testing.T) {
	cockpitDir := t.TempDir()
	loader := NewPackageLoader(cockpitDir)

	expected := filepath.Join(cockpitDir, "packages", "my-pkg")
	got := loader.GetPackagePath("my-pkg")
	if got != expected {
		t.Errorf("GetPackagePath() = %q, want %q", got, expected)
	}
}

func TestLoadInstalledPackages(t *testing.T) {
	cockpitDir := t.TempDir()
	packagesDir := filepath.Join(cockpitDir, "packages")

	// Create valid package directories
	for _, name := range []string{"pkg-a", "pkg-b"} {
		pkgDir := filepath.Join(packagesDir, name)
		if err := os.MkdirAll(pkgDir, 0o755); err != nil {
			t.Fatalf("setup: %v", err)
		}
		writeTestManifest(t, pkgDir, name)
	}

	// Directory without manifest should be ignored
	emptyDir := filepath.Join(packagesDir, "empty-dir")
	if err := os.MkdirAll(emptyDir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// Non-directory file should be ignored
	if err := os.WriteFile(filepath.Join(packagesDir, "not-a-dir"), []byte(""), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	loader := NewPackageLoader(cockpitDir)
	packages, err := loader.LoadInstalledPackages()
	if err != nil {
		t.Fatalf("LoadInstalledPackages() error = %v", err)
	}

	if len(packages) != 2 {
		t.Errorf("LoadInstalledPackages() returned %d packages, want 2", len(packages))
	}

	seen := make(map[string]bool)
	for _, p := range packages {
		seen[p] = true
	}
	if !seen["pkg-a"] || !seen["pkg-b"] {
		t.Errorf("LoadInstalledPackages() returned unexpected set: %v", packages)
	}
}

func TestLoadInstalledPackages_NoPackagesDir(t *testing.T) {
	cockpitDir := t.TempDir()
	loader := NewPackageLoader(cockpitDir)

	packages, err := loader.LoadInstalledPackages()
	if err != nil {
		t.Fatalf("LoadInstalledPackages() error = %v", err)
	}
	if len(packages) != 0 {
		t.Errorf("LoadInstalledPackages() returned %d packages, want 0", len(packages))
	}
}

func TestLoadInstalledPackages_ReadDirError(t *testing.T) {
	cockpitDir := t.TempDir()
	packagesDir := filepath.Join(cockpitDir, "packages")
	if err := os.WriteFile(packagesDir, []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	loader := NewPackageLoader(cockpitDir)
	_, err := loader.LoadInstalledPackages()
	if err == nil {
		t.Error("LoadInstalledPackages() expected error when packagesDir is not a directory")
	}
}

func TestLoadPackageManifest(t *testing.T) {
	cockpitDir := t.TempDir()
	pkgDir := filepath.Join(cockpitDir, "packages", "test-pkg")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	writeTestManifest(t, pkgDir, "test-pkg")

	loader := NewPackageLoader(cockpitDir)
	pkg, err := loader.LoadPackageManifest("test-pkg")
	if err != nil {
		t.Fatalf("LoadPackageManifest() error = %v", err)
	}
	if pkg.Name != "test-pkg" {
		t.Errorf("LoadPackageManifest() name = %q, want %q", pkg.Name, "test-pkg")
	}
}

func TestLoadPackageManifest_NotFound(t *testing.T) {
	cockpitDir := t.TempDir()
	loader := NewPackageLoader(cockpitDir)

	_, err := loader.LoadPackageManifest("missing")
	if err == nil {
		t.Error("LoadPackageManifest() expected error for missing package")
	}
}

func TestExecutePackageCommand_NoModules(t *testing.T) {
	cockpitDir := t.TempDir()
	pkgDir := filepath.Join(cockpitDir, "packages", "no-modules")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	writeTestManifest(t, pkgDir, "no-modules")

	loader := NewPackageLoader(cockpitDir)
	err := loader.ExecutePackageCommand("no-modules", "test", []string{})
	if err == nil {
		t.Error("ExecutePackageCommand() expected error when package has no modules")
	}
}

func TestExecutePackageCommand_WithModules(t *testing.T) {
	cockpitDir := t.TempDir()
	pkgDir := filepath.Join(cockpitDir, "packages", "with-modules")
	modulesDir := filepath.Join(pkgDir, "modules")
	if err := os.MkdirAll(modulesDir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	writeTestManifest(t, pkgDir, "with-modules")

	loader := NewPackageLoader(cockpitDir)
	err := loader.ExecutePackageCommand("with-modules", "test", []string{})
	if err == nil {
		t.Error("ExecutePackageCommand() expected error because execution is not implemented")
	}
}

func TestGetPackageCommands(t *testing.T) {
	cockpitDir := t.TempDir()
	pkgDir := filepath.Join(cockpitDir, "packages", "cmd-pkg")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	writeTestManifest(t, pkgDir, "cmd-pkg")

	loader := NewPackageLoader(cockpitDir)
	commands, err := loader.GetPackageCommands("cmd-pkg")
	if err != nil {
		t.Fatalf("GetPackageCommands() error = %v", err)
	}
	if len(commands) != 1 || commands[0] != "cmd-pkg" {
		t.Errorf("GetPackageCommands() = %v, want [cmd-pkg]", commands)
	}
}

func TestGetPackageCommands_NoModules(t *testing.T) {
	cockpitDir := t.TempDir()
	pkgDir := filepath.Join(cockpitDir, "packages", "no-cmd")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	manifestContent := `name: "no-cmd"
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

installation:
  supported_providers:
    - devin
  provider_features:
    devin:
      - skills
  method: "symlink"
`
	if err := os.WriteFile(filepath.Join(pkgDir, "cockpit-package.yml"), []byte(manifestContent), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	loader := NewPackageLoader(cockpitDir)
	commands, err := loader.GetPackageCommands("no-cmd")
	if err != nil {
		t.Fatalf("GetPackageCommands() error = %v", err)
	}
	if len(commands) != 0 {
		t.Errorf("GetPackageCommands() returned %d commands, want 0", len(commands))
	}
}

func TestGetPackageCommands_ManifestError(t *testing.T) {
	cockpitDir := t.TempDir()
	loader := NewPackageLoader(cockpitDir)

	_, err := loader.GetPackageCommands("missing")
	if err == nil {
		t.Error("GetPackageCommands() expected error when manifest is missing")
	}
}

func TestCompilePackageModules(t *testing.T) {
	cockpitDir := t.TempDir()
	pkgDir := filepath.Join(cockpitDir, "packages", "compile-pkg")
	modulesDir := filepath.Join(pkgDir, "modules")
	if err := os.MkdirAll(modulesDir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	writeTestManifest(t, pkgDir, "compile-pkg")

	loader := NewPackageLoader(cockpitDir)
	if err := loader.CompilePackageModules("compile-pkg"); err != nil {
		t.Errorf("CompilePackageModules() error = %v", err)
	}
}

func TestCompilePackageModules_NoModules(t *testing.T) {
	cockpitDir := t.TempDir()
	pkgDir := filepath.Join(cockpitDir, "packages", "no-modules")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	writeTestManifest(t, pkgDir, "no-modules")

	loader := NewPackageLoader(cockpitDir)
	err := loader.CompilePackageModules("no-modules")
	if err == nil {
		t.Error("CompilePackageModules() expected error when package has no modules")
	}
}

func TestDiscoverPackageCommands(t *testing.T) {
	cockpitDir := t.TempDir()
	pkgDir := filepath.Join(cockpitDir, "packages", "discover-pkg")
	modulesDir := filepath.Join(pkgDir, "modules")
	if err := os.MkdirAll(modulesDir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(modulesDir, "cmd.go"), []byte("package main"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(modulesDir, "cmd_foo.go"), []byte("package main"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(modulesDir, "other.go"), []byte("package main"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	// Subdirectories should be skipped
	if err := os.MkdirAll(filepath.Join(modulesDir, "nested"), 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	loader := NewPackageLoader(cockpitDir)
	commands, err := loader.DiscoverPackageCommands("discover-pkg")
	if err != nil {
		t.Fatalf("DiscoverPackageCommands() error = %v", err)
	}
	if len(commands) != 2 {
		t.Errorf("DiscoverPackageCommands() returned %d commands, want 2", len(commands))
	}
}

func TestDiscoverPackageCommands_NoModules(t *testing.T) {
	cockpitDir := t.TempDir()
	pkgDir := filepath.Join(cockpitDir, "packages", "no-discover")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	loader := NewPackageLoader(cockpitDir)
	commands, err := loader.DiscoverPackageCommands("no-discover")
	if err != nil {
		t.Fatalf("DiscoverPackageCommands() error = %v", err)
	}
	if len(commands) != 0 {
		t.Errorf("DiscoverPackageCommands() returned %d commands, want 0", len(commands))
	}
}

func TestDiscoverPackageCommands_ReadDirError(t *testing.T) {
	cockpitDir := t.TempDir()
	pkgDir := filepath.Join(cockpitDir, "packages", "bad-modules")
	modulesFile := filepath.Join(pkgDir, "modules")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(modulesFile, []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	loader := NewPackageLoader(cockpitDir)
	_, err := loader.DiscoverPackageCommands("bad-modules")
	if err == nil {
		t.Error("DiscoverPackageCommands() expected error when modules is not a directory")
	}
}

func TestSymlinkPackageModules(t *testing.T) {
	cockpitDir := t.TempDir()
	pkgDir := filepath.Join(cockpitDir, "packages", "symlink-pkg")
	modulesDir := filepath.Join(pkgDir, "modules")
	if err := os.MkdirAll(modulesDir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	writeTestManifest(t, pkgDir, "symlink-pkg")

	loader := NewPackageLoader(cockpitDir)
	if err := loader.SymlinkPackageModules("symlink-pkg"); err != nil {
		t.Errorf("SymlinkPackageModules() error = %v", err)
	}
}

func TestSymlinkPackageModules_NoModules(t *testing.T) {
	cockpitDir := t.TempDir()
	pkgDir := filepath.Join(cockpitDir, "packages", "no-symlink")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	writeTestManifest(t, pkgDir, "no-symlink")

	loader := NewPackageLoader(cockpitDir)
	err := loader.SymlinkPackageModules("no-symlink")
	if err == nil {
		t.Error("SymlinkPackageModules() expected error when package has no modules")
	}
}

func TestRegisterPackageCommands(t *testing.T) {
	cockpitDir := t.TempDir()
	packagesDir := filepath.Join(cockpitDir, "packages")

	for _, name := range []string{"pkg-a", "pkg-b"} {
		pkgDir := filepath.Join(packagesDir, name)
		modulesDir := filepath.Join(pkgDir, "modules")
		if err := os.MkdirAll(modulesDir, 0o755); err != nil {
			t.Fatalf("setup: %v", err)
		}
		if err := os.WriteFile(filepath.Join(modulesDir, "cmd.go"), []byte("package main"), 0o644); err != nil {
			t.Fatalf("setup: %v", err)
		}
		writeTestManifest(t, pkgDir, name)
	}

	loader := NewPackageLoader(cockpitDir)
	commands, err := loader.RegisterPackageCommands()
	if err != nil {
		t.Fatalf("RegisterPackageCommands() error = %v", err)
	}

	if len(commands) != 2 {
		t.Errorf("RegisterPackageCommands() registered %d commands, want 2", len(commands))
	}

	for _, name := range []string{"pkg-a", "pkg-b"} {
		if commands[name] != name {
			t.Errorf("RegisterPackageCommands() mapping for %q = %q, want %q", name, commands[name], name)
		}
	}
}

func TestRegisterPackageCommands_DiscoverError(t *testing.T) {
	cockpitDir := t.TempDir()
	packagesDir := filepath.Join(cockpitDir, "packages")

	// Valid package manifest but modules path is a file, causing DiscoverPackageCommands to fail.
	pkgDir := filepath.Join(packagesDir, "bad-pkg")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	writeTestManifest(t, pkgDir, "bad-pkg")
	if err := os.WriteFile(filepath.Join(pkgDir, "modules"), []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	loader := NewPackageLoader(cockpitDir)
	commands, err := loader.RegisterPackageCommands()
	if err != nil {
		t.Fatalf("RegisterPackageCommands() error = %v", err)
	}
	if len(commands) != 0 {
		t.Errorf("RegisterPackageCommands() returned %d commands, want 0", len(commands))
	}
}

func TestRegisterPackageCommands_LoadError(t *testing.T) {
	cockpitDir := t.TempDir()
	packagesDir := filepath.Join(cockpitDir, "packages")
	if err := os.WriteFile(packagesDir, []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	loader := NewPackageLoader(cockpitDir)
	_, err := loader.RegisterPackageCommands()
	if err == nil {
		t.Error("RegisterPackageCommands() expected error when LoadInstalledPackages fails")
	}
}

func TestRegisterPackageCommands_NoPackagesDir(t *testing.T) {
	cockpitDir := t.TempDir()
	loader := NewPackageLoader(cockpitDir)

	commands, err := loader.RegisterPackageCommands()
	if err != nil {
		t.Fatalf("RegisterPackageCommands() error = %v", err)
	}
	if len(commands) != 0 {
		t.Errorf("RegisterPackageCommands() returned %d commands, want 0", len(commands))
	}
}
