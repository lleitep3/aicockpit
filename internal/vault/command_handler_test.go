package vault

import (
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
