package vault

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/zalando/go-keyring"
)

func TestCommandHandler(t *testing.T) {
	// Enable mock keyring for testing
	keyring.MockInit()

	handler := NewCommandHandler()

	// Setup test secrets
	testSecrets := map[string]string{
		"test_api_key": "sk-test-1234567890",
		"test_db_pass": "database_password_123",
	}

	for key, value := range testSecrets {
		err := handler.vault.Set(key, value)
		if err != nil {
			t.Fatalf("Failed to setup test secret %s: %v", key, err)
		}
	}

	// Cleanup
	defer func() {
		for key := range testSecrets {
			handler.vault.Delete(key)
		}
	}()

	t.Run("Test Execute With Secret", func(t *testing.T) {
		// Test with echo command (should be available on most systems)
		output, err := handler.ExecuteWithSecret(
			"echo",
			[]string{"API_KEY: {{API_KEY}}"},
			[]SecretInjection{
				{SecretKey: "test_api_key", Placeholder: "{{API_KEY}}"},
			},
		)

		if err != nil {
			t.Fatalf("Failed to execute command: %v", err)
		}

		if !strings.Contains(output, "sk-test-1234567890") {
			t.Errorf("Expected output to contain secret, got: %s", output)
		}
	})

	t.Run("Test Command Not Allowed", func(t *testing.T) {
		_, err := handler.ExecuteWithSecret(
			"disallowed_command",
			[]string{"test"},
			[]SecretInjection{
				{SecretKey: "test_api_key", Placeholder: "{{API_KEY}}"},
			},
		)

		if err == nil {
			t.Error("Expected error for disallowed command, got nil")
		}

		if !strings.Contains(err.Error(), "command not allowed") {
			t.Errorf("Expected 'command not allowed' error, got: %v", err)
		}
	})

	t.Run("Test Multiple Secret Injection", func(t *testing.T) {
		output, err := handler.ExecuteWithSecret(
			"echo",
			[]string{"API: {{API_KEY}}, DB: {{DB_PASS}}"},
			[]SecretInjection{
				{SecretKey: "test_api_key", Placeholder: "{{API_KEY}}"},
				{SecretKey: "test_db_pass", Placeholder: "{{DB_PASS}}"},
			},
		)

		if err != nil {
			t.Fatalf("Failed to execute command: %v", err)
		}

		if !strings.Contains(output, "sk-test-1234567890") {
			t.Errorf("Expected output to contain API key, got: %s", output)
		}

		if !strings.Contains(output, "database_password_123") {
			t.Errorf("Expected output to contain DB password, got: %s", output)
		}
	})

	t.Run("Test Secret Not Found", func(t *testing.T) {
		_, err := handler.ExecuteWithSecret(
			"echo",
			[]string{"SECRET: {{NONEXISTENT}}"},
			[]SecretInjection{
				{SecretKey: "nonexistent_key", Placeholder: "{{NONEXISTENT}}"},
			},
		)

		if err == nil {
			t.Error("Expected error for nonexistent secret, got nil")
		}

		if !strings.Contains(err.Error(), "failed to retrieve secret") {
			t.Errorf("Expected secret retrieval error, got: %v", err)
		}
	})

	t.Run("Test Custom Allowed Commands", func(t *testing.T) {
		customHandler := NewCommandHandlerWithConfig(CommandHandlerConfig{
			AllowedCommands: []string{"custom_test_cmd"},
			EnableAudit:     true,
		})

		// Should fail with default command
		_, err := customHandler.ExecuteWithSecret(
			"echo",
			[]string{"test"},
			[]SecretInjection{},
		)

		if err == nil {
			t.Error("Expected error for non-whitelisted command, got nil")
		}
	})

	t.Run("Test Output Sanitization", func(t *testing.T) {
		output, err := handler.ExecuteWithSecretForOutput(
			"echo",
			[]string{"The key is sk-test-1234567890"},
			[]SecretInjection{
				{SecretKey: "test_api_key", Placeholder: "sk-test-1234567890"},
			},
		)

		if err != nil {
			t.Fatalf("Failed to execute command: %v", err)
		}

		// The secret should be redacted from output
		if strings.Contains(output, "sk-test-1234567890") {
			t.Error("Expected secret to be redacted from output")
		}

		if !strings.Contains(output, "***REDACTED***") {
			t.Error("Expected redaction placeholder in output")
		}
	})
}

func TestCommandHandlerConfig(t *testing.T) {
	t.Run("Test Custom Configuration", func(t *testing.T) {
		config := CommandHandlerConfig{
			AllowedCommands: []string{"my_custom_app"},
			EnableAudit:     false,
		}

		handler := NewCommandHandlerWithConfig(config)

		// Verify custom command is allowed
		if !handler.isCommandAllowed("my_custom_app") {
			t.Error("Expected custom command to be allowed")
		}

		// Verify default commands are not allowed
		if handler.isCommandAllowed("curl") {
			t.Error("Expected default command to be disallowed with custom config")
		}
	})

	t.Run("Test Add/Remove Commands", func(t *testing.T) {
		handler := NewCommandHandler()

		// Add custom command
		handler.AddAllowedCommand("new_command")
		if !handler.isCommandAllowed("new_command") {
			t.Error("Expected newly added command to be allowed")
		}

		// Remove command
		handler.RemoveAllowedCommand("curl")
		if handler.isCommandAllowed("curl") {
			t.Error("Expected removed command to be disallowed")
		}
	})
}

func newTestCommandHandler(t *testing.T) *CommandHandler {
	t.Helper()
	keyring.MockInit()
	handler := NewCommandHandler()
	handler.vault.indexPath = filepath.Join(t.TempDir(), ".vault_index.json")
	return handler
}

func TestCommandHandler_SetAllowedCommands(t *testing.T) {
	handler := newTestCommandHandler(t)

	handler.SetAllowedCommands([]string{"whoami", "pwd"})

	if handler.isCommandAllowed("echo") {
		t.Error("echo should not be allowed after SetAllowedCommands")
	}
	if !handler.isCommandAllowed("whoami") {
		t.Error("whoami should be allowed")
	}
	if !handler.isCommandAllowed("pwd") {
		t.Error("pwd should be allowed")
	}
}

func TestCommandHandler_SetAuditLog(t *testing.T) {
	handler := newTestCommandHandler(t)

	var audited bool
	handler.SetAuditLog(func(command string, keys []string, success bool) {
		audited = true
		if command != "disallowed_cmd" {
			t.Errorf("expected command disallowed_cmd, got %q", command)
		}
		if success {
			t.Error("expected audit to record failure for disallowed command")
		}
	})

	_, err := handler.ExecuteWithSecret("disallowed_cmd", []string{"test"}, nil)
	if err == nil {
		t.Fatal("expected error for disallowed command")
	}
	if !audited {
		t.Error("custom audit log function was not called")
	}
}

func TestCommandHandler_ClearAllSecrets(t *testing.T) {
	handler := newTestCommandHandler(t)

	if err := handler.vault.Set("clear-key", "clear-value"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	if err := handler.ClearAllSecrets(); err != nil {
		t.Fatalf("ClearAllSecrets: %v", err)
	}

	if _, err := handler.vault.Get("clear-key"); err == nil {
		t.Error("secret should be removed after ClearAllSecrets")
	}
}

func TestCommandHandler_ExecuteWithSecretForOutput_Error(t *testing.T) {
	handler := newTestCommandHandler(t)

	_, err := handler.ExecuteWithSecretForOutput(
		"disallowed_cmd",
		[]string{"test"},
		nil,
	)
	if err == nil {
		t.Fatal("expected error when command is not allowed")
	}
}

func TestCommandHandler_IsCommandAllowed(t *testing.T) {
	handler := newTestCommandHandler(t)

	if !handler.isCommandAllowed("echo") {
		t.Error("echo should be allowed")
	}
	if !handler.isCommandAllowed("/usr/bin/echo") {
		t.Error("absolute unix path should be allowed")
	}
	if !handler.isCommandAllowed("C:\\Windows\\System32\\echo") {
		t.Error("windows-style path should be allowed")
	}
	if handler.isCommandAllowed("") {
		t.Error("empty command should not be allowed")
	}
	if handler.isCommandAllowed("notallowed") {
		t.Error("notallowed should not be allowed")
	}
}

func TestCommandHandler_SanitizeOutput_SecretMissing(t *testing.T) {
	handler := newTestCommandHandler(t)

	output := handler.sanitizeOutput("sensitive output", []SecretInjection{
		{SecretKey: "missing-secret", Placeholder: "{{SECRET}}"},
	})

	if output != "sensitive output" {
		t.Errorf("expected output unchanged, got %q", output)
	}
}

func TestCommandHandler_ExecuteWithSecret_CommandFails(t *testing.T) {
	handler := newTestCommandHandler(t)

	_, err := handler.ExecuteWithSecret("git", []string{"--not-a-valid-git-option-xyz"}, nil)
	if err == nil {
		t.Fatal("expected error when allowed command exits non-zero")
	}
	if !strings.Contains(err.Error(), "command execution failed") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSecretInjection(t *testing.T) {
	t.Run("Test Placeholder Replacement", func(t *testing.T) {
		keyring.MockInit()
		handler := NewCommandHandler()

		testKey := "test_injection"
		testValue := "secret123"
		handler.vault.Set(testKey, testValue)
		defer handler.vault.Delete(testKey)

		injection := SecretInjection{
			SecretKey:   testKey,
			Placeholder: "{{SECRET}}",
		}

		input := "The secret is {{SECRET}}"
		output, err := handler.injectSingleSecret(input, injection)

		if err != nil {
			t.Fatalf("Failed to inject secret: %v", err)
		}

		expected := "The secret is secret123"
		if output != expected {
			t.Errorf("Expected %q, got %q", expected, output)
		}
	})

	t.Run("Test Empty Placeholder", func(t *testing.T) {
		keyring.MockInit()
		handler := NewCommandHandler()

		injection := SecretInjection{
			SecretKey:   "test_key",
			Placeholder: "",
		}

		_, err := handler.injectSingleSecret("test", injection)
		if err == nil {
			t.Error("Expected error for empty placeholder, got nil")
		}
	})
}
