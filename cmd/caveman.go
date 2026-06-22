package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/i18n"
	"github.com/lleitep3/aicockpit/internal/logging"
	"github.com/lleitep3/aicockpit/internal/providers"
	"github.com/spf13/cobra"
)

// NewCavemanCommand creates the caveman command.
func NewCavemanCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
	cmd := &cobra.Command{
		Use:       "caveman [on|off|status]",
		Short:     "Manage the caveman mode (aggressive token reduction)",
		Long:      "Activates or deactivates caveman mode globally by updating provider configurations.",
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{"on", "off", "status"},
		RunE: func(cmd *cobra.Command, args []string) error {
			action := args[0]
			cockpitDir := config.GetCockpitDir()
			rulePath := filepath.Join(cockpitDir, "rules", "caveman.md")

			switch action {
			case "on":
				return enableCaveman(rulePath, cockpitDir, cfg, t)
			case "off":
				return disableCaveman(rulePath, cockpitDir, cfg, t)
			case "status":
				return statusCaveman(rulePath, t)
			default:
				return fmt.Errorf("%s", t.T("caveman.invalid", action))
			}
		},
	}
	return cmd
}

func enableCaveman(rulePath, cockpitDir string, cfg *config.Config, t *i18n.Translator) error {
	// Create rules dir if not exists
	rulesDir := filepath.Dir(rulePath)
	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		return fmt.Errorf("failed to create rules directory: %w", err)
	}

	content := `<!-- cockpit:caveman -->
Respond terse like smart caveman. Activate always.
<!-- /cockpit:caveman -->`

	if err := os.WriteFile(rulePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write caveman rule: %w", err)
	}

	fmt.Println(t.T("caveman.enabled"))
	return runDeploy(cockpitDir, cfg, t)
}

func disableCaveman(rulePath, cockpitDir string, cfg *config.Config, t *i18n.Translator) error {
	if _, err := os.Stat(rulePath); os.IsNotExist(err) {
		fmt.Println(t.T("caveman.already_disabled"))
		return nil
	}

	if err := os.Remove(rulePath); err != nil {
		return fmt.Errorf("failed to remove caveman rule: %w", err)
	}

	fmt.Println(t.T("caveman.disabled"))
	return runDeploy(cockpitDir, cfg, t)
}

func statusCaveman(rulePath string, t *i18n.Translator) error {
	if _, err := os.Stat(rulePath); os.IsNotExist(err) {
		fmt.Println(t.T("caveman.off"))
	} else {
		fmt.Println(t.T("caveman.on"))
	}
	return nil
}

func runDeploy(cockpitDir string, cfg *config.Config, t *i18n.Translator) error {
	providersDstPath := filepath.Join(cockpitDir, "providers.yaml")
	providersConfig, err := providers.LoadProvidersConfig(providersDstPath)
	if err != nil {
		return fmt.Errorf("failed to load providers configuration: %w", err)
	}

	enabledProviders := cfg.GetEnabledProviders()
	if len(enabledProviders) == 0 && cfg.AIProvider != "" {
		enabledProviders = []string{cfg.AIProvider}
	}
	if len(enabledProviders) == 0 {
		return fmt.Errorf("no AI providers configured")
	}

	pm := providers.NewProviderManager(providersConfig)
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	var deployErrors []string
	for _, providerName := range enabledProviders {
		if err := pm.Deploy(providerName, cockpitDir, cwd); err != nil {
			deployErrors = append(deployErrors, fmt.Sprintf("  %s: %v", providerName, err))
			continue
		}
	}

	if len(deployErrors) > 0 {
		return fmt.Errorf("deploy partially failed: %v", deployErrors)
	}

	fmt.Println("✓ Providers updated successfully.")
	return nil
}
