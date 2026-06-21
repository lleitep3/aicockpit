package cmd

import (
	"testing"
)

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
