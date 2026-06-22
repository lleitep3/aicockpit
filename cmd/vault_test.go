package cmd

import (
	"bytes"
	"testing"

	"github.com/lleite/aicockpit/internal/config"
	"github.com/lleite/aicockpit/internal/i18n"
	"github.com/lleite/aicockpit/internal/logging"
	"github.com/zalando/go-keyring"
)

func TestVaultCommands(t *testing.T) {
	// Enable mock keyring for testing
	keyring.MockInit()
	log, _ := logging.NewManager("")
	cfg := &config.Config{Version: "0.1.0", Language: "en-us"}
	translator := i18n.New("en-us")

	t.Run("Test NewVaultCommand", func(t *testing.T) {
		cmd := NewVaultCommand(log, cfg, translator)
		if cmd == nil {
			t.Fatal("Expected command, got nil")
		}
		if cmd.Use != "vault" {
			t.Errorf("Expected Use to be 'vault', got '%s'", cmd.Use)
		}
	})

	t.Run("Test Vault Set and Get", func(t *testing.T) {
		key := "test_key_cli"
		value := "secret_cli_value"

		// Set via CLI args with --value flag
		setCmd := NewVaultSetCommand(log, cfg, translator)
		setCmd.SetArgs([]string{key, "--value", value})

		var out bytes.Buffer
		setCmd.SetOut(&out)
		setCmd.SetErr(&out)

		err := setCmd.Execute()
		if err != nil {
			t.Fatalf("Failed to execute set command: %v", err)
		}

		// Get via CLI
		getCmd := NewVaultGetCommand(log, cfg, translator)
		getCmd.SetArgs([]string{key})

		out.Reset()
		getCmd.SetOut(&out)
		getCmd.SetErr(&out)

		err = getCmd.Execute()
		if err != nil {
			t.Fatalf("Failed to execute get command: %v", err)
		}

		if out.String() != value {
			t.Errorf("Expected output %q, got %q", value, out.String())
		}
	})

	t.Run("Test Vault Remove", func(t *testing.T) {
		key := "test_key_remove"
		value := "val"

		// Set first
		setCmd := NewVaultSetCommand(log, cfg, translator)
		setCmd.SetArgs([]string{key, "--value", value})
		_ = setCmd.Execute()

		// Remove via CLI
		removeCmd := NewVaultRemoveCommand(log, cfg, translator)
		removeCmd.SetArgs([]string{key})

		var out bytes.Buffer
		removeCmd.SetOut(&out)
		removeCmd.SetErr(&out)

		err := removeCmd.Execute()
		if err != nil {
			t.Fatalf("Failed to execute remove command: %v", err)
		}

		// Verify it's gone
		getCmd := NewVaultGetCommand(log, cfg, translator)
		getCmd.SetArgs([]string{key})
		err = getCmd.Execute()
		if err == nil {
			t.Errorf("Expected error when getting removed key, got nil")
		}
	})
}
