package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lleite/aicockpit/internal/config"
	"github.com/lleite/aicockpit/internal/i18n"
	"github.com/lleite/aicockpit/internal/logging"
	"github.com/lleite/aicockpit/internal/providers"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// NewSetupCommand creates the setup command.
func NewSetupCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: t.T("setup.welcome"),
		Long:  t.T("setup.welcome"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSetup(log, cfg, t)
		},
	}
}

func runSetup(log *logging.Manager, cfg *config.Config, t *i18n.Translator) error {
	startTime := time.Now()
	fmt.Println(t.T("setup.welcome"))
	fmt.Println()

	// Step 1: Select language
	fmt.Println(t.T("setup.language"))
	fmt.Println("1. English (en-us)")
	fmt.Println("2. Português Brasileiro (pt-br)")
	fmt.Print("Select (1-2): ")

	language := selectOption([]string{"en-us", "pt-br"}, "en-us")
	t.SetLanguage(language)
	cfg.Language = language
	fmt.Printf("✓ Language selected: %s\n", language)

	// Step 2: Copy the entire .cockpit directory to ~/.cockpit
	fmt.Println()
	fmt.Println("Synchronizing ~/.cockpit folder...")

	cockpitDir := config.GetCockpitDir()

	// Determine repository .cockpit source path
	cockpitRepoPath := filepath.Join(filepath.Dir(os.Args[0]), "..", ".cockpit")
	if _, err := os.Stat(cockpitRepoPath); err != nil {
		cockpitRepoPath = filepath.Join(os.Getenv("HOME"), "projects", "aicockpit", ".cockpit")
	}

	if err := copyDirectory(cockpitRepoPath, cockpitDir); err != nil {
		fmt.Printf("✗ Failed to setup ~/.cockpit directory: %v\n", err)
		return err
	}
	fmt.Println("✓ ~/.cockpit folder synchronized")

	providersDstPath := filepath.Join(cockpitDir, "providers.yaml")
	configDstPath := filepath.Join(cockpitDir, "config.yaml")

	// Step 3: Select AI provider from providers.yaml
	fmt.Println()
	fmt.Println(t.T("setup.ai"))

	// Load providers configuration
	providersConfig, err := providers.LoadProvidersConfig(providersDstPath)
	if err != nil {
		fmt.Printf("✗ Failed to load providers configuration: %v\n", err)
		return err
	}

	// Get enabled provider options
	providerOptions := providersConfig.GetProviderOptions()
	if len(providerOptions) == 0 {
		return fmt.Errorf("no providers available in configuration")
	}

	// Display provider options
	for i, opt := range providerOptions {
		fmt.Printf("%d. %s\n", i+1, opt.DisplayName)
	}
	fmt.Print("Select (1-" + fmt.Sprintf("%d", len(providerOptions)) + "): ")

	// Get provider names for selection
	providerNames := make([]string, len(providerOptions))
	for i, opt := range providerOptions {
		providerNames[i] = opt.Name
	}

	aiProvider := selectOption(providerNames, "antigravity")
	cfg.AIProvider = aiProvider
	fmt.Printf("✓ AI provider selected: %s\n", aiProvider)

	// Update config with selected provider and language
	if err := updateConfigWithProvider(configDstPath, aiProvider, language); err != nil {
		fmt.Printf("✗ Failed to update config: %v\n", err)
		return err
	}

	// Step 4: Run deploy for the selected provider using ProviderManager
	fmt.Println()
	fmt.Printf("Deploying assets for %s to workspace...\n", aiProvider)

	pm := providers.NewProviderManager(providersConfig)
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	if err := pm.Deploy(aiProvider, cockpitDir, cwd); err != nil {
		fmt.Printf("✗ Failed to deploy configuration: %v\n", err)
		return err
	}
	fmt.Printf("✓ Configuration successfully deployed to %s\n", cwd)

	fmt.Println()
	fmt.Println(t.T("setup.complete"))
	fmt.Printf(t.T("setup.saved")+"\n", config.GetConfigPath())

	duration := time.Since(startTime)
	log.LogCommand("setup", []string{}, "success", 0, duration, "", nil)

	return nil
}

// selectOption reads user input and returns the selected option.
func selectOption(options []string, defaultOption string) string {
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	// Parse input as number
	var index int
	_, err := fmt.Sscanf(input, "%d", &index)
	if err != nil || index < 1 || index > len(options) {
		return defaultOption
	}

	return options[index-1]
}
func copyConfigFile(src, dst string) error {
	return copyFile(src, dst)
}

// updateConfigWithProvider updates the config file with the selected provider and language
func updateConfigWithProvider(configPath, provider, language string) error {
	// Read the config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse as map to update fields
	var configMap map[string]interface{}
	if err := yaml.Unmarshal(data, &configMap); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// Update ai_provider
	configMap["ai_provider"] = provider

	// Update language
	configMap["language"] = language

	// Update ai_providers.enabled
	if aiProviders, ok := configMap["ai_providers"].(map[string]interface{}); ok {
		aiProviders["enabled"] = []string{provider}
	}

	// Marshal back to YAML
	updatedData, err := yaml.Marshal(configMap)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write back to file
	if err := os.WriteFile(configPath, updatedData, 0o644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Println("✓ config.yaml updated with selected provider")
	return nil
}
