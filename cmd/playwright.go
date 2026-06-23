package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/i18n"
	"github.com/lleitep3/aicockpit/internal/logging"
	"github.com/lleitep3/aicockpit/internal/playwright"
	"github.com/spf13/cobra"
)

func NewPlaywrightCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
	playwrightCmd := &cobra.Command{
		Use:   "playwright",
		Short: "Browser automation using Playwright",
	}

	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start Playwright server with user profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			url, _ := cmd.Flags().GetString("url")
			profile, _ := cmd.Flags().GetString("profile")

			if profile == "" {
				home, _ := os.UserHomeDir()
				profile = filepath.Join(home, ".cockpit", "browser_profile")
			}

			driver, err := playwright.NewDriver(profile)
			if err != nil {
				return fmt.Errorf("failed to initialize driver: %w", err)
			}
			defer driver.Close()

			if url != "" {
				if err := driver.Goto(url); err != nil {
					log.LogError("Failed to navigate", map[string]interface{}{"error": err})
				}
			}

			server := playwright.NewServer(driver)
			log.LogInfo("Playwright server started on 127.0.0.1:9091", nil)
			fmt.Println("Playwright server started on 127.0.0.1:9091")
			return server.Start()
		},
	}
	startCmd.Flags().String("url", "", "URL to open on start")
	startCmd.Flags().String("profile", "", "Path to User Data Dir")

	clickCmd := &cobra.Command{
		Use:   "click [selector]",
		Short: "Click an element",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return sendAction("click", args[0], "", "", "")
		},
	}

	typeCmd := &cobra.Command{
		Use:   "type [selector] [text]",
		Short: "Type text into an element",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return sendAction("type", args[0], args[1], "", "")
		},
	}

	evalCmd := &cobra.Command{
		Use:   "eval [js]",
		Short: "Evaluate JavaScript on the page",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return sendAction("eval", "", "", "", args[0])
		},
	}

	playwrightCmd.AddCommand(startCmd, clickCmd, typeCmd, evalCmd)
	return playwrightCmd
}

func sendAction(action, selector, text, url, js string) error {
	reqBody := playwright.ActionRequest{
		Action:   action,
		Selector: selector,
		Text:     text,
		URL:      url,
		JS:       js,
	}
	b, _ := json.Marshal(reqBody)

	resp, err := http.Post("http://127.0.0.1:9091/action", "application/json", bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("server error (is it running?): %w", err)
	}
	defer resp.Body.Close()

	var result playwright.ActionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("action failed: %s", result.Error)
	}

	if result.Result != nil {
		fmt.Printf("Result: %v\n", result.Result)
	} else {
		fmt.Println("Success")
	}
	return nil
}
