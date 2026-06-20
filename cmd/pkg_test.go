package cmd

import (
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

	// Test with package name
	err := cmd.RunE(cmd, []string{"hello-world"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
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

	// Test with package name
	err := cmd.RunE(cmd, []string{"hello-world"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
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
