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

// NewUninstallCommand creates the uninstall command.
func NewUninstallCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall AICockpit",
		Long:  "Remove AICockpit and all its data",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUninstall(log, cfg, t)
		},
	}
}

func runUninstall(log *logging.Manager, cfg *config.Config, t *i18n.Translator) error {
	startTime := time.Now()
	cockpitDir := config.GetCockpitDir()

	// Confirm uninstall
	fmt.Printf(t.T("uninstall.confirm"), cockpitDir)

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	// Check for affirmative response (y or s for Portuguese)
	if input != "y" && input != "s" && input != "yes" && input != "sim" {
		fmt.Println(t.T("uninstall.cancel"))
		duration := time.Since(startTime)
		log.LogCommand("uninstall", []string{}, "cancelled", 0, duration, "", nil)
		return nil
	}

	// Remove cockpit directory
	if err := os.RemoveAll(cockpitDir); err != nil {
		duration := time.Since(startTime)
		log.LogCommand("uninstall", []string{}, "error", 1, duration, "", err)
		return fmt.Errorf("failed to remove cockpit directory: %w", err)
	}

	fmt.Println(t.T("uninstall.success"))
	duration := time.Since(startTime)
	log.LogCommand("uninstall", []string{}, "success", 0, duration, "", nil)

	return nil
}
