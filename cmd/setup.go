package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/lleite/aicockpit/internal/config"
	"github.com/lleite/aicockpit/internal/i18n"
	"github.com/lleite/aicockpit/internal/logging"
	"github.com/spf13/cobra"
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
	log.LogInfo("Language selected", map[string]interface{}{"language": language})

	// Step 2: Select AI provider
	fmt.Println()
	fmt.Println(t.T("setup.ai"))
	fmt.Println("1. Claude (Anthropic)")
	fmt.Println("2. GPT (OpenAI)")
	fmt.Println("3. Devin CLI")
	fmt.Println("4. Antigravity")
	fmt.Println("5. Goose")
	fmt.Print("Select (1-5): ")

	aiProviders := []string{"claude", "openai", "devin", "antigravity", "goose"}
	aiProvider := selectOption(aiProviders, "claude")
	cfg.AIProvider = aiProvider
	log.LogInfo("AI provider selected", map[string]interface{}{"provider": aiProvider})

	// Step 3: Create vault
	fmt.Println()
	fmt.Println(t.T("setup.vault"))

	if err := config.Save(cfg); err != nil {
		log.LogError("Failed to save configuration", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Println()
	fmt.Println(t.T("setup.complete"))
	fmt.Printf(t.T("setup.saved")+"\n", config.GetConfigPath())

	// Log the command execution
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
