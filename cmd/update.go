package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/i18n"
	"github.com/lleitep3/aicockpit/internal/logging"
	"github.com/lleitep3/aicockpit/internal/update"
	"github.com/lleitep3/aicockpit/internal/version"
	"github.com/spf13/cobra"
)

// NewUpdateCommand creates the update command.
func NewUpdateCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Update AICockpit to the latest version",
		Long:  "Check for updates and upgrade AICockpit to the latest version. Will automatically run setup after update.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(log, cfg, t)
		},
	}
}

func runUpdate(log *logging.Manager, cfg *config.Config, t *i18n.Translator) error {
	fmt.Println(t.T("update.checking"))

	updateService := update.NewUpdateService()
	latestVersion, releaseURL, err := updateService.CheckForUpdates()
	if err != nil {
		return fmt.Errorf(t.T("update.check_failed"), err)
	}

	currentVersion := version.GetVersion()

	if latestVersion == "" {
		fmt.Printf("✓ AICockpit is already up to date (version %s)\n", currentVersion)
		return nil
	}

	fmt.Printf(t.T("update.available")+"\n", latestVersion, currentVersion)
	fmt.Printf(t.T("update.changelog")+"\n", releaseURL)
	fmt.Print(t.T("update.prompt"))

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	if input != "y" && input != "yes" && input != "s" && input != "sim" {
		fmt.Println(t.T("update.cancel"))
		return nil
	}

	fmt.Printf(t.T("update.updating")+"\n", latestVersion)

	// Perform the update
	if err := performUpdate(latestVersion); err != nil {
		return fmt.Errorf(t.T("update.failed"), err)
	}

	fmt.Printf(t.T("update.success")+"\n", latestVersion)

	// Update the last check timestamp
	now := time.Now().Format(time.RFC3339)
	if err := cfg.SetLastUpdateCheck(now); err != nil {
		log.LogWarn(fmt.Sprintf("failed to update last check timestamp: %v", err), nil)
	}

	// Ask if user wants to run setup
	fmt.Print("Would you like to run setup now? (y/n): ")
	reader = bufio.NewReader(os.Stdin)
	input, _ = reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	if input == "y" || input == "yes" || input == "s" || input == "sim" {
		fmt.Println("Running setup...")
		return runSetup(log, cfg, t)
	}

	return nil
}

// performUpdate performs the actual update by pulling the latest version from git
func performUpdate(targetVersion string) error {
	// Check if we're in a git repository
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		return fmt.Errorf("not in a git repository, automatic update not available")
	}

	// Fetch latest changes
	fmt.Println("Fetching latest changes...")
	if err := runCommand("git", "fetch", "origin"); err != nil {
		return fmt.Errorf("failed to fetch: %w", err)
	}

	// Checkout the latest version tag
	fmt.Printf("Checking out version %s...\n", targetVersion)
	if err := runCommand("git", "checkout", "v"+targetVersion); err != nil {
		return fmt.Errorf("failed to checkout version: %w", err)
	}

	// Pull latest changes
	fmt.Println("Pulling latest changes...")
	if err := runCommand("git", "pull", "origin", "v"+targetVersion); err != nil {
		return fmt.Errorf("failed to pull: %w", err)
	}

	// Rebuild the application
	fmt.Println("Rebuilding AICockpit...")
	if err := runCommand("make", "build"); err != nil {
		return fmt.Errorf("failed to build: %w", err)
	}

	// Install locally
	fmt.Println("Installing AICockpit...")
	if err := runCommand("make", "install-local"); err != nil {
		return fmt.Errorf("failed to install: %w", err)
	}

	return nil
}

// runCommand executes a shell command
func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
