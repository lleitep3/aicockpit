package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lleite/aicockpit/internal/config"
	"github.com/lleite/aicockpit/internal/i18n"
	"github.com/lleite/aicockpit/internal/logger"
	"github.com/spf13/cobra"
)

// NewInfoCommand creates the info command.
func NewInfoCommand(log *logger.Logger, cfg *config.Config, t *i18n.Translator) *cobra.Command {
	return &cobra.Command{
		Use:   "info",
		Short: t.T("info.title"),
		Long:  t.T("info.title"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInfo(log, cfg, t)
		},
	}
}

func runInfo(log *logger.Logger, cfg *config.Config, t *i18n.Translator) error {
	fmt.Println(t.T("info.title"))
	fmt.Println("=" + strings.Repeat("=", 49))
	fmt.Println()

	cockpitDir := config.GetCockpitDir()
	configPath := config.GetConfigPath()

	fmt.Printf("%s: %s\n", t.T("version"), cfg.Version)
	fmt.Printf("%s: %s\n", t.T("language"), cfg.Language)
	fmt.Printf("%s: %s\n", t.T("log_level"), cfg.LogLevel)
	fmt.Printf("%s: %s\n", t.T("ai_provider"), cfg.AIProvider)
	fmt.Println()
	fmt.Printf("%s: %s\n", t.T("info.dir"), cockpitDir)
	fmt.Printf("%s: %s\n", t.T("info.config"), configPath)

	// Show log file path
	logDir := filepath.Join(cockpitDir, "logs")
	if entries, err := os.ReadDir(logDir); err == nil && len(entries) > 0 {
		latestLog := entries[len(entries)-1]
		fmt.Printf("%s: %s\n", t.T("info.log"), filepath.Join(logDir, latestLog.Name()))
	}

	fmt.Println()

	// List installed packages
	fmt.Println(t.T("info.packages") + ":")
	packagesDir := filepath.Join(cockpitDir, "packages")
	entries, err := os.ReadDir(packagesDir)
	if err != nil || len(entries) == 0 {
		fmt.Println("  " + t.T("info.no_packages"))
	} else {
		for _, entry := range entries {
			if entry.IsDir() {
				fmt.Printf("  - %s\n", entry.Name())
			}
		}
	}

	log.Info("Cockpit info displayed")
	return nil
}
