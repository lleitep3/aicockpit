package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

// makeRegistryConfig creates a minimal config.yaml in tmpDir/.cockpit/ and
// returns the cockpit directory path.
func makeRegistryConfig(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	cockpitDir := filepath.Join(tmpDir, ".cockpit")
	if err := os.MkdirAll(cockpitDir, 0o755); err != nil {
		t.Fatalf("MkdirAll cockpit: %v", err)
	}
	yaml := `version: "0.1.0"
language: en-us
log_level: info
auto_update_check: false
enabled_providers: []
package_registries:
  - name: default
    url: https://github.com/lleitep3/cockpit-registry
    branch: main
    enabled: true
    priority: 0
`
	if err := os.WriteFile(filepath.Join(cockpitDir, "config.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatalf("WriteFile config: %v", err)
	}
	return cockpitDir
}

func TestNewPkgRegistriesCommand(t *testing.T) {
	cmd := NewPkgRegistriesCommand()

	if cmd.Use != "registries" {
		t.Errorf("Expected command use 'registries', got '%s'", cmd.Use)
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

func TestNewPkgRegistriesListCommand(t *testing.T) {
	cmd := NewPkgRegistriesListCommand()

	if cmd.Use != "list" {
		t.Errorf("Expected command use 'list', got '%s'", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Expected command short description")
	}

	// Check flags
	if cmd.Flag("enabled") == nil {
		t.Error("Expected 'enabled' flag")
	}
}

func TestNewPkgRegistriesAddCommand(t *testing.T) {
	cmd := NewPkgRegistriesAddCommand()

	if cmd.Use != "add <name> <url>" {
		t.Errorf("Expected command use 'add <name> <url>', got '%s'", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Expected command short description")
	}

	// Check flags
	if cmd.Flag("branch") == nil {
		t.Error("Expected 'branch' flag")
	}

	if cmd.Flag("priority") == nil {
		t.Error("Expected 'priority' flag")
	}
}

func TestNewPkgRegistriesRemoveCommand(t *testing.T) {
	cmd := NewPkgRegistriesRemoveCommand()

	if cmd.Use != "remove <name>" {
		t.Errorf("Expected command use 'remove <name>', got '%s'", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Expected command short description")
	}

	// Check flags
	if cmd.Flag("force") == nil {
		t.Error("Expected 'force' flag")
	}
}

func TestNewPkgRegistriesEnableCommand(t *testing.T) {
	cmd := NewPkgRegistriesEnableCommand()

	if cmd.Use != "enable <name>" {
		t.Errorf("Expected command use 'enable <name>', got '%s'", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Expected command short description")
	}
}

func TestNewPkgRegistriesDisableCommand(t *testing.T) {
	cmd := NewPkgRegistriesDisableCommand()

	if cmd.Use != "disable <name>" {
		t.Errorf("Expected command use 'disable <name>', got '%s'", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Expected command short description")
	}
}

func TestNewPkgRegistriesInfoCommand(t *testing.T) {
	cmd := NewPkgRegistriesInfoCommand()

	if cmd.Use != "info <name>" {
		t.Errorf("Expected command use 'info <name>', got '%s'", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Expected command short description")
	}
}

func TestPkgRegistriesCommandHierarchy(t *testing.T) {
	registriesCmd := NewPkgRegistriesCommand()

	// Check that all subcommands are registered
	subcommands := map[string]bool{
		"list":    false,
		"add":     false,
		"remove":  false,
		"enable":  false,
		"disable": false,
		"info":    false,
	}

	for _, cmd := range registriesCmd.Commands() {
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

// ── RunE execution tests ──────────────────────────────────────────────────

func TestPkgRegistriesListCommand_Run(t *testing.T) {
	makeRegistryConfig(t)
	cmd := NewPkgRegistriesListCommand()
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Errorf("list execute error = %v", err)
	}
}

func TestPkgRegistriesListCommand_Run_Enabled(t *testing.T) {
	makeRegistryConfig(t)
	cmd := NewPkgRegistriesListCommand()
	cmd.SetArgs([]string{"--enabled"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("list --enabled execute error = %v", err)
	}
}

func TestPkgRegistriesAddCommand_Run(t *testing.T) {
	makeRegistryConfig(t)
	cmd := NewPkgRegistriesAddCommand()
	cmd.SetArgs([]string{"my-reg", "https://github.com/example/reg"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("add execute error = %v", err)
	}
}

func TestPkgRegistriesRemoveCommand_Run_NotFound(t *testing.T) {
	makeRegistryConfig(t)
	cmd := NewPkgRegistriesRemoveCommand()
	cmd.SetArgs([]string{"nonexistent"})
	if err := cmd.Execute(); err == nil {
		t.Error("remove nonexistent registry should return error")
	}
}

func TestPkgRegistriesRemoveCommand_Run_Found(t *testing.T) {
	makeRegistryConfig(t)
	cmd := NewPkgRegistriesRemoveCommand()
	cmd.SetArgs([]string{"default"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("remove default registry error = %v", err)
	}
}

func TestPkgRegistriesEnableCommand_Run_NotFound(t *testing.T) {
	makeRegistryConfig(t)
	cmd := NewPkgRegistriesEnableCommand()
	cmd.SetArgs([]string{"nonexistent"})
	if err := cmd.Execute(); err == nil {
		t.Error("enable nonexistent registry should return error")
	}
}

func TestPkgRegistriesEnableCommand_Run_Found(t *testing.T) {
	makeRegistryConfig(t)
	cmd := NewPkgRegistriesEnableCommand()
	cmd.SetArgs([]string{"default"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("enable default registry error = %v", err)
	}
}

func TestPkgRegistriesDisableCommand_Run_NotFound(t *testing.T) {
	makeRegistryConfig(t)
	cmd := NewPkgRegistriesDisableCommand()
	cmd.SetArgs([]string{"nonexistent"})
	if err := cmd.Execute(); err == nil {
		t.Error("disable nonexistent registry should return error")
	}
}

func TestPkgRegistriesDisableCommand_Run_Found(t *testing.T) {
	makeRegistryConfig(t)
	cmd := NewPkgRegistriesDisableCommand()
	cmd.SetArgs([]string{"default"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("disable default registry error = %v", err)
	}
}

func TestPkgRegistriesInfoCommand_Run_NotFound(t *testing.T) {
	makeRegistryConfig(t)
	cmd := NewPkgRegistriesInfoCommand()
	cmd.SetArgs([]string{"nonexistent"})
	if err := cmd.Execute(); err == nil {
		t.Error("info nonexistent registry should return error")
	}
}

func TestPkgRegistriesInfoCommand_Run_Found(t *testing.T) {
	makeRegistryConfig(t)
	cmd := NewPkgRegistriesInfoCommand()
	cmd.SetArgs([]string{"default"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("info default registry error = %v", err)
	}
}
