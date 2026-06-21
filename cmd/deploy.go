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

			// Collect enabled providers: use ai_providers.enabled list when available,
			// falling back to the legacy single ai_provider field.
			enabledProviders := cfg.GetEnabledProviders()
			if len(enabledProviders) == 0 && cfg.AIProvider != "" {
				enabledProviders = []string{cfg.AIProvider}
			}
			if len(enabledProviders) == 0 {
				return fmt.Errorf("no AI providers configured. Run 'cockpit setup' first")
			}

			pm := providers.NewProviderManager(providersConfig)
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current working directory: %w", err)
			}

			var deployErrors []string
			for _, providerName := range enabledProviders {
				fmt.Printf("Deploying assets for %s to workspace...\n", providerName)

				if err := pm.Deploy(providerName, cockpitDir, cwd); err != nil {
					deployErrors = append(deployErrors, fmt.Sprintf("  %s: %v", providerName, err))
					fmt.Printf("  ⚠ Failed to deploy for %s: %v\n", providerName, err)
					continue
				}

				fmt.Printf("  ✓ %s deployed\n", providerName)
			}

			if len(deployErrors) > 0 {
				fmt.Printf("\n⚠ Deploy completed with %d error(s):\n", len(deployErrors))
				for _, e := range deployErrors {
					fmt.Println(e)
				}
				return fmt.Errorf("deploy partially failed (%d/%d providers)", len(deployErrors), len(enabledProviders))
			}

			fmt.Printf("\n✓ Configuration successfully compiled and deployed to %s\n", cwd)
			return nil
		},
	}
}
