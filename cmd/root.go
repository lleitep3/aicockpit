package cmd

import (
	"github.com/lleite/aicockpit/internal/config"
	"github.com/lleite/aicockpit/internal/i18n"
	"github.com/lleite/aicockpit/internal/logging"
	"github.com/spf13/cobra"
)

// NewRootCommand creates the root command for the CLI.
func NewRootCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:     "cockpit",
		Short:   t.T("welcome"),
		Version: cfg.Version,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			t.SetLanguage(cfg.Language)
		},
	}

	// Add subcommands
	rootCmd.AddCommand(NewSetupCommand(log, cfg, t))
	rootCmd.AddCommand(NewInfoCommand(log, cfg, t))
	rootCmd.AddCommand(NewDoctorCommand(log, cfg, t))
	rootCmd.AddCommand(NewUninstallCommand(log, cfg, t))
	rootCmd.AddCommand(NewMetricsCommand(log, cfg, t))
	rootCmd.AddCommand(NewKBCommand(log, cfg, t))
	rootCmd.AddCommand(NewPkgCommand())

	// Add flags
	rootCmd.PersistentFlags().StringVar(&cfg.Language, "language", cfg.Language, "Set language (en-us, pt-br)")
	rootCmd.PersistentFlags().StringVar(&cfg.LogLevel, "log-level", cfg.LogLevel, "Set log level (debug, info, warn, error)")

	return rootCmd
}
