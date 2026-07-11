package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/i18n"
	"github.com/lleitep3/aicockpit/internal/logging"
	"github.com/lleitep3/aicockpit/internal/update"
	"github.com/lleitep3/aicockpit/internal/version"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// NewRootCommand creates the root command for the CLI.
func NewRootCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:     "cockpit",
		Short:   t.T("welcome"),
		Version: cfg.Version,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			t.SetLanguage(cfg.Language)

			// Skip update check for certain commands
			if cmd.Name() == "update" || cmd.Name() == "setup" {
				return
			}

			// Check for updates
			checkForUpdates(log, cfg, t)
		},
	}

	// Add subcommands
	rootCmd.AddCommand(NewSetupCommand(log, cfg, t))
	rootCmd.AddCommand(NewDeployCommand(log, cfg, t))
	rootCmd.AddCommand(NewInfoCommand(log, cfg, t))
	rootCmd.AddCommand(NewDoctorCommand(log, cfg, t))
	rootCmd.AddCommand(NewUninstallCommand(log, cfg, t))
	rootCmd.AddCommand(NewVaultCommand(log, cfg, t))
	rootCmd.AddCommand(NewMetricsCommand(log, cfg, t))
	rootCmd.AddCommand(NewKBCommand(log, cfg, t))
	rootCmd.AddCommand(NewPkgCommand())
	rootCmd.AddCommand(NewUpdateCommand(log, cfg, t))

	// Load commands from installed packages
	if err := LoadPackageCommands(rootCmd); err != nil {
		// Log warning but don't fail
		log.LogWarn(fmt.Sprintf("failed to load package commands: %v", err), nil)
	}

	// Add flags
	rootCmd.PersistentFlags().StringVar(&cfg.Language, "language", cfg.Language, "Set language (en-us, pt-br)")
	rootCmd.PersistentFlags().StringVar(&cfg.LogLevel, "log-level", cfg.LogLevel, "Set log level (debug, info, warn, error)")

	return rootCmd
}

// checkForUpdates checks if a new version is available and prompts the user.
// In non-interactive environments (no TTY) the update notification is logged
// but stdin is never read, so the CLI never hangs in scripts or CI pipelines.
func checkForUpdates(log *logging.Manager, cfg *config.Config, t *i18n.Translator) {
	interactive := term.IsTerminal(int(os.Stdin.Fd()))
	checkForUpdatesWithReader(log, cfg, t, os.Stdin, interactive)
}

// updateChecker is an interface for checking updates, allowing test injection.
type updateChecker interface {
	CheckForUpdates() (string, string, error)
}

// checkForUpdatesWithReader is the testable core of checkForUpdates.
// It receives an explicit reader and interactivity flag so tests can inject
// a fake stdin without requiring a real TTY.
func checkForUpdatesWithReader(log *logging.Manager, cfg *config.Config, t *i18n.Translator, stdin io.Reader, interactive bool) {
	checkForUpdatesWithService(log, cfg, t, stdin, interactive, update.NewUpdateService())
}

// checkForUpdatesWithService is the fully injectable core that accepts an updateChecker.
func checkForUpdatesWithService(log *logging.Manager, cfg *config.Config, t *i18n.Translator, stdin io.Reader, interactive bool, svc updateChecker) {
	if !cfg.ShouldCheckUpdate() {
		return
	}

	latestVersion, releaseURL, err := svc.CheckForUpdates()
	if err != nil {
		// Don't fail the command if update check fails, just log it
		log.LogWarn(fmt.Sprintf(t.T("update.check_failed"), err), nil)
		return
	}

	// Update the last check timestamp
	now := time.Now().Format(time.RFC3339)
	if err := cfg.SetLastUpdateCheck(now); err != nil {
		log.LogWarn(fmt.Sprintf("failed to update last check timestamp: %v", err), nil)
	}

	// If no new version is available, return silently
	if latestVersion == "" {
		return
	}

	currentVersion := version.GetVersion()

	// Non-interactive: log the available update and return without blocking stdin
	if !interactive {
		log.LogInfo(fmt.Sprintf("update available: %s (current: %s) — run 'cockpit update' to upgrade", latestVersion, currentVersion), nil)
		return
	}

	fmt.Println()
	fmt.Printf(t.T("update.available")+"\n", latestVersion, currentVersion)
	fmt.Printf(t.T("update.changelog")+"\n", releaseURL)
	fmt.Print(t.T("update.prompt"))

	reader := bufio.NewReader(stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	if input == "y" || input == "yes" || input == "s" || input == "sim" {
		fmt.Printf(t.T("update.updating")+"\n", latestVersion)

		// Direct user to use the update command for full update
		fmt.Println("Please run 'cockpit update' to perform the update.")
	} else {
		fmt.Println(t.T("update.cancel"))
	}
	fmt.Println()
}
