package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lleite/aicockpit/internal/config"
	"github.com/lleite/aicockpit/internal/i18n"
	"github.com/lleite/aicockpit/internal/logging"
	"github.com/lleite/aicockpit/internal/providers"
	"github.com/spf13/cobra"
)

// NewDeployCommand creates the deploy command.
func NewDeployCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
	return &cobra.Command{
		Use:   "deploy",
		Short: "Compile and deploy rules and skills to AI provider workspaces",
		Long:  "Compile and deploy rules and skills to AI provider workspaces from ~/.cockpit canonical files.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cockpitDir := config.GetCockpitDir()
			providersDstPath := filepath.Join(cockpitDir, "providers.yaml")

			// Load providers configuration
			providersConfig, err := providers.LoadProvidersConfig(providersDstPath)
			if err != nil {
				return fmt.Errorf("failed to load providers configuration: %w", err)
			}

			aiProvider := cfg.AIProvider
			if aiProvider == "" {
				return fmt.Errorf("no AI provider configured. Run 'cockpit setup' first")
			}

			fmt.Printf("Deploying assets for %s to workspace...\n", aiProvider)

			pm := providers.NewProviderManager(providersConfig)
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current working directory: %w", err)
			}

			if err := pm.Deploy(aiProvider, cockpitDir, cwd); err != nil {
				return fmt.Errorf("failed to deploy configuration: %w", err)
			}

			fmt.Printf("✓ Configuration successfully compiled and deployed to %s\n", cwd)
			return nil
		},
	}
}
