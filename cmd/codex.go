package cmd

import (
	"fmt"

	"github.com/lleitep3/aicockpit/internal/codexconfig"
	"github.com/spf13/cobra"
)

// NewCodexCommand creates Codex integration commands.
func NewCodexCommand() *cobra.Command {
	codexCmd := &cobra.Command{
		Use:   "codex",
		Short: "Configure Codex integration",
	}
	codexCmd.AddCommand(NewCodexConfigureSandboxCommand())
	return codexCmd
}

// NewCodexConfigureSandboxCommand creates the idempotent Codex sandbox setup.
func NewCodexConfigureSandboxCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "configure-sandbox",
		Short: "Allow Codex workspace-write access to AICockpit logs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, changed, err := codexconfig.EnsureUserSandboxConfig()
			if err != nil {
				return err
			}
			if changed {
				fmt.Printf("✓ Codex sandbox configuration updated: %s\n", configPath)
			} else {
				fmt.Printf("✓ Codex sandbox configuration already configured: %s\n", configPath)
			}
			return nil
		},
	}
}
