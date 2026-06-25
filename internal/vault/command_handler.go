package vault

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// CommandHandler provides secure secret injection for command execution
// Applications don't access secrets directly; instead, they request the handler
// to execute commands with secrets injected
type CommandHandler struct {
	vault           *osVault
	allowedCommands map[string]bool
	auditLog        func(command string, keys []string, success bool)
}

// SecretInjection defines how a secret should be injected into a command
type SecretInjection struct {
	SecretKey   string // The key in the vault
	Placeholder string // The placeholder in the command (e.g., {{SECRET}}, $SECRET)
}

// CommandHandlerConfig configures the command handler
type CommandHandlerConfig struct {
	AllowedCommands []string
	EnableAudit     bool
}

// NewCommandHandler creates a new command handler with default configuration
func NewCommandHandler() *CommandHandler {
	return &CommandHandler{
		vault:           newOSVault(),
		allowedCommands: defaultAllowedCommands(),
		auditLog:        defaultAuditLog,
	}
}

// NewCommandHandlerWithConfig creates a command handler with custom configuration
func NewCommandHandlerWithConfig(config CommandHandlerConfig) *CommandHandler {
	handler := &CommandHandler{
		vault:           newOSVault(),
		allowedCommands: make(map[string]bool),
	}

	for _, cmd := range config.AllowedCommands {
		handler.allowedCommands[cmd] = true
	}

	if config.EnableAudit {
		handler.auditLog = defaultAuditLog
	} else {
		handler.auditLog = func(command string, keys []string, success bool) {}
	}

	return handler
}

// ExecuteWithSecret executes a command with secrets securely injected
// This prevents applications from directly accessing secrets while still allowing them to use secrets
func (ch *CommandHandler) ExecuteWithSecret(
	command string,
	args []string,
	injections []SecretInjection,
) (string, error) {

	// Validate command is allowed
	if !ch.isCommandAllowed(command) {
		err := fmt.Errorf("command not allowed: %s", command)
		ch.auditLog(command, ch.extractKeys(injections), false)
		return "", err
	}

	// Resolve secrets and inject into arguments
	resolvedArgs, err := ch.injectSecrets(args, injections)
	if err != nil {
		ch.auditLog(command, ch.extractKeys(injections), false)
		return "", fmt.Errorf("failed to inject secrets: %w", err)
	}

	// Execute the command
	cmd := exec.Command(command, resolvedArgs...)
	output, err := cmd.CombinedOutput()

	// Audit the access
	ch.auditLog(command, ch.extractKeys(injections), err == nil)

	if err != nil {
		return "", fmt.Errorf("command execution failed: %w, output: %s", err, string(output))
	}

	return string(output), nil
}

// ExecuteWithSecretForOutput executes a command and returns only the output (without secrets)
// Useful for commands where you want to capture output but prevent secrets from appearing in logs
func (ch *CommandHandler) ExecuteWithSecretForOutput(
	command string,
	args []string,
	injections []SecretInjection,
) (string, error) {
	output, err := ch.ExecuteWithSecret(command, args, injections)
	if err != nil {
		return "", err
	}

	// Remove any potential secret leaks from output
	// (This is a basic implementation; more sophisticated sanitization may be needed)
	return ch.sanitizeOutput(output, injections), nil
}

// injectSecrets replaces placeholders with actual secret values
func (ch *CommandHandler) injectSecrets(args []string, injections []SecretInjection) ([]string, error) {
	resolvedArgs := make([]string, len(args))

	for i, arg := range args {
		resolvedArg := arg
		var err error

		for _, injection := range injections {
			resolvedArg, err = ch.injectSingleSecret(resolvedArg, injection)
			if err != nil {
				return nil, err
			}
		}

		resolvedArgs[i] = resolvedArg
	}

	return resolvedArgs, nil
}

// injectSingleSecret injects a single secret into a string
func (ch *CommandHandler) injectSingleSecret(input string, injection SecretInjection) (string, error) {
	if injection.Placeholder == "" {
		return "", fmt.Errorf("placeholder cannot be empty")
	}

	secret, err := ch.vault.Get(injection.SecretKey)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve secret %s: %w", injection.SecretKey, err)
	}

	return strings.ReplaceAll(input, injection.Placeholder, secret), nil
}

// isCommandAllowed checks if a command is in the allowed list
func (ch *CommandHandler) isCommandAllowed(command string) bool {
	// Extract just the command name (without path)
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return false
	}

	cmdName := parts[0]
	// Remove path if present
	if idx := strings.LastIndex(cmdName, "/"); idx != -1 {
		cmdName = cmdName[idx+1:]
	}
	if idx := strings.LastIndex(cmdName, "\\"); idx != -1 {
		cmdName = cmdName[idx+1:]
	}

	return ch.allowedCommands[cmdName]
}

// extractKeys extracts secret keys from injections for audit logging
func (ch *CommandHandler) extractKeys(injections []SecretInjection) []string {
	keys := make([]string, len(injections))
	for i, injection := range injections {
		keys[i] = injection.SecretKey
	}
	return keys
}

// sanitizeOutput removes potential secret values from command output
func (ch *CommandHandler) sanitizeOutput(output string, injections []SecretInjection) string {
	sanitized := output

	for _, injection := range injections {
		secret, err := ch.vault.Get(injection.SecretKey)
		if err != nil {
			continue // Skip if we can't retrieve the secret
		}

		// Remove the secret value from output
		sanitized = strings.ReplaceAll(sanitized, secret, "***REDACTED***")
	}

	return sanitized
}

// defaultAllowedCommands returns the default whitelist of allowed commands
func defaultAllowedCommands() map[string]bool {
	return map[string]bool{
		// System utilities (for testing)
		"echo": true,

		// Database clients
		"psql":      true,
		"mysql":     true,
		"mongosh":   true,
		"redis-cli": true,

		// HTTP clients
		"curl": true,
		"wget": true,
		"http": true,

		// Container tools
		"docker":  true,
		"podman":  true,
		"kubectl": true,

		// Cloud tools
		"aws":    true,
		"gcloud": true,
		"az":     true,

		// Development tools
		"git": true,
		"npm": true,
		"pip": true,

		// System tools
		"ssh":   true,
		"scp":   true,
		"rsync": true,
	}
}

// defaultAuditLog provides basic audit logging
func defaultAuditLog(command string, keys []string, success bool) {
	status := "SUCCESS"
	if !success {
		status = "FAILED"
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	fmt.Printf("[AUDIT %s] Command: %s, Secrets: %v, Status: %s\n",
		timestamp, command, keys, status)
}

// SetAllowedCommands updates the list of allowed commands
func (ch *CommandHandler) SetAllowedCommands(commands []string) {
	ch.allowedCommands = make(map[string]bool)
	for _, cmd := range commands {
		ch.allowedCommands[cmd] = true
	}
}

// AddAllowedCommand adds a single command to the allowed list
func (ch *CommandHandler) AddAllowedCommand(command string) {
	ch.allowedCommands[command] = true
}

// RemoveAllowedCommand removes a command from the allowed list
func (ch *CommandHandler) RemoveAllowedCommand(command string) {
	delete(ch.allowedCommands, command)
}

// SetAuditLog sets a custom audit log function
func (ch *CommandHandler) SetAuditLog(auditFunc func(command string, keys []string, success bool)) {
	ch.auditLog = auditFunc
}

// ClearAllSecrets removes all secrets from the vault (factory reset)
func (ch *CommandHandler) ClearAllSecrets() error {
	return ch.vault.ClearAllSecrets()
}
