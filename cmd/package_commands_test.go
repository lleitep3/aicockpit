package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lleitep3/aicockpit/internal/packages"
	"github.com/spf13/cobra"
)

// ── getPackageCommandName ──────────────────────────────────────────────────

func TestGetPackageCommandName_UsesModuleName(t *testing.T) {
	pkg := &packages.Package{
		Features: packages.Features{
			Modules: []packages.Feature{{Name: "my-cmd", Path: "bin/my-cmd"}},
		},
	}
	got := getPackageCommandName(pkg, "fallback")
	if got != "my-cmd" {
		t.Errorf("getPackageCommandName() = %q, want %q", got, "my-cmd")
	}
}

func TestGetPackageCommandName_FallsBackToPackageName(t *testing.T) {
	pkg := &packages.Package{} // no modules
	got := getPackageCommandName(pkg, "my-package")
	if got != "my-package" {
		t.Errorf("getPackageCommandName() = %q, want %q", got, "my-package")
	}
}

func TestGetPackageCommandName_EmptyModuleName(t *testing.T) {
	pkg := &packages.Package{
		Features: packages.Features{
			Modules: []packages.Feature{{Path: "bin/cmd"}}, // Name is ""
		},
	}
	got := getPackageCommandName(pkg, "fallback")
	if got != "fallback" {
		t.Errorf("getPackageCommandName() = %q, want %q", got, "fallback")
	}
}

// ── executePackageCommand ─────────────────────────────────────────────────

func TestExecutePackageCommand_NoExecutable(t *testing.T) {
	tmpDir := t.TempDir()
	// No bin/ dir at all — must return an error.
	err := executePackageCommand("mypkg", tmpDir, nil)
	if err == nil {
		t.Error("executePackageCommand() expected error when no binary exists, got nil")
	}
}

func TestExecutePackageCommand_ModulesOnly(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a modules dir (no bin/) — exercises the "Go modules not implemented" branch.
	if err := os.MkdirAll(filepath.Join(tmpDir, "modules"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	err := executePackageCommand("mypkg", tmpDir, nil)
	if err == nil {
		t.Error("executePackageCommand() expected error for Go modules, got nil")
	}
}

func TestExecutePackageCommand_BinScript(t *testing.T) {
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	// Write a tiny shell script that exits 0 and make it executable.
	script := filepath.Join(binDir, "mypkg")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := executePackageCommand("mypkg", tmpDir, nil); err != nil {
		t.Errorf("executePackageCommand() error = %v", err)
	}
}

func TestExecutePackageCommand_FallbackBinScript(t *testing.T) {
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	// Script named differently from the package — exercises the fallback-to-first-entry branch.
	script := filepath.Join(binDir, "other-name")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := executePackageCommand("mypkg", tmpDir, nil); err != nil {
		t.Errorf("executePackageCommand() fallback error = %v", err)
	}
}

// ── hasCommand ────────────────────────────────────────────────────────────

func TestHasCommand_WithModulesDir(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, "modules"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if !hasCommand(tmpDir, "any-cmd") {
		t.Error("hasCommand() = false, want true when modules/ exists")
	}
}

func TestHasCommand_NoModulesDir(t *testing.T) {
	tmpDir := t.TempDir() // empty — no modules/
	if hasCommand(tmpDir, "any-cmd") {
		t.Error("hasCommand() = true, want false when modules/ is missing")
	}
}

// ── CreateDynamicCommand ──────────────────────────────────────────────────

func TestCreateDynamicCommand(t *testing.T) {
	cmd := CreateDynamicCommand("my-dynamic")
	if cmd == nil {
		t.Fatal("CreateDynamicCommand() returned nil")
	}
	if cmd.Use != "my-dynamic" {
		t.Errorf("Use = %q, want %q", cmd.Use, "my-dynamic")
	}
}

// ── LoadPackageCommands ───────────────────────────────────────────────────

func TestLoadPackageCommands_NoPackagesDir(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	root := &cobra.Command{Use: "root"}
	if err := LoadPackageCommands(root); err != nil {
		t.Errorf("LoadPackageCommands() with no packages dir error = %v", err)
	}
}

func TestLoadPackageCommands_WithValidPackage(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cockpitDir := filepath.Join(tmpDir, ".cockpit")
	pkgDir := filepath.Join(cockpitDir, "packages", "hello")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	// Write a minimal valid manifest.
	manifest := `name: hello
version: "1.0.0"
description: Hello package
author: Test
license: MIT
requirements:
  cockpit: ">=0.1.0"
features:
  modules:
    - path: bin/hello
      name: hello
`
	if err := os.WriteFile(filepath.Join(pkgDir, "cockpit-package.yml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile manifest: %v", err)
	}

	root := &cobra.Command{Use: "root"}
	if err := LoadPackageCommands(root); err != nil {
		t.Errorf("LoadPackageCommands() error = %v", err)
	}
}

func TestLoadPackageCommands_PackageWithoutManifest(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cockpitDir := filepath.Join(tmpDir, ".cockpit")
	// Create a dir in packages/ that has no cockpit-package.yml.
	noManifestDir := filepath.Join(cockpitDir, "packages", "no-manifest")
	if err := os.MkdirAll(noManifestDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	root := &cobra.Command{Use: "root"}
	// Should silently skip the package and return nil.
	if err := LoadPackageCommands(root); err != nil {
		t.Errorf("LoadPackageCommands() error = %v", err)
	}
}
