package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/i18n"
	"github.com/lleitep3/aicockpit/internal/logging"
	"github.com/lleitep3/aicockpit/internal/providers"
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

	// Step 3: Select AI providers (multiple allowed) from providers.yaml
	fmt.Println()
	fmt.Println(t.T("setup.ai"))

	// Load providers configuration
	providersConfig, err := providers.LoadProvidersConfig(providersDstPath)
	if err != nil {
		fmt.Printf("✗ Failed to load providers configuration: %v\n", err)
		return err
	}

	// Get all provider options (enabled in config)
	providerOptions := providersConfig.GetProviderOptions()
	if len(providerOptions) == 0 {
		return fmt.Errorf("no providers available in configuration")
	}

	// Sort for deterministic display order
	sort.Slice(providerOptions, func(i, j int) bool {
		return providerOptions[i].Name < providerOptions[j].Name
	})

	// Display provider options
	fmt.Println("Available AI providers (you can select multiple):")
	for i, opt := range providerOptions {
		fmt.Printf("  %d. %s\n", i+1, opt.DisplayName)
	}
	fmt.Printf("Enter numbers separated by commas (e.g. 1,3) or press Enter for [1]: ")

	selectedNames := selectMultiple(providerOptions)
	if len(selectedNames) == 0 {
		return fmt.Errorf("at least one provider must be selected")
	}

	cfg.AIProvider = selectedNames[0] // first selection is the primary provider
	fmt.Printf("✓ AI providers selected: %s\n", strings.Join(selectedNames, ", "))

	// Update config with selected providers and language
	if err := updateConfigWithProviders(configDstPath, selectedNames, language); err != nil {
		fmt.Printf("✗ Failed to update config: %v\n", err)
		return err
	}

	// Step 4: Run deploy for ALL selected providers
	pm := providers.NewProviderManager(providersConfig)
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	fmt.Println()
	deployErrors := 0
	for _, providerName := range selectedNames {
		fmt.Printf("Deploying assets for %s...\n", providerName)
		if err := pm.Deploy(providerName, cockpitDir, cwd); err != nil {
			fmt.Printf("  ✗ Failed to deploy %s: %v\n", providerName, err)
			deployErrors++
		} else {
			fmt.Printf("  ✓ %s deployed successfully\n", providerName)
		}
	}

	if deployErrors > 0 {
		return fmt.Errorf("%d provider(s) failed to deploy", deployErrors)
	}

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

// selectMultiple reads a comma-separated list of option numbers from stdin and
// returns the selected provider names. Falls back to the first option on invalid input.
func selectMultiple(options []providers.ProviderOption) []string {
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		if len(options) > 0 {
			return []string{options[0].Name}
		}
		return nil
	}

	parts := strings.Split(input, ",")
	seen := make(map[string]bool)
	var selected []string

	for _, part := range parts {
		part = strings.TrimSpace(part)
		idx, err := strconv.Atoi(part)
		if err != nil || idx < 1 || idx > len(options) {
			continue
		}
		name := options[idx-1].Name
		if !seen[name] {
			seen[name] = true
			selected = append(selected, name)
		}
	}

	if len(selected) == 0 && len(options) > 0 {
		return []string{options[0].Name}
	}
	return selected
}

// updateConfigWithProviders updates the config file with the selected providers and language.
func updateConfigWithProviders(configPath string, providerNames []string, language string) error {
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

	// Primary provider (first selected) for backward compatibility
	configMap["ai_provider"] = providerNames[0]

	// Language
	configMap["language"] = language

	// ai_providers section
	if aiProviders, ok := configMap["ai_providers"].(map[string]interface{}); ok {
		aiProviders["enabled"] = providerNames
	} else {
		configMap["ai_providers"] = map[string]interface{}{
			"enabled": providerNames,
		}
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

	fmt.Println("✓ config.yaml updated with selected providers")
	return nil
}
