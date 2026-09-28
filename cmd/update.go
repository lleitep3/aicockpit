package cmd

import (
	"bufio"
	"context"
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
	options := unifiedOptions{}
	command := &cobra.Command{
		Use:     "update",
		Short:   "Update AICockpit to the latest version",
		Long:    "Update the binary, installed packages, or both. Without flags, use the interactive binary update. Scoped updates require --yes; --check only inspects versions and refreshes registry caches. Binary updates require Git and Go. Reports distinguish partial failures.",
		Example: "  cockpit update --all --check\n  cockpit update --all --yes --report update-result.json\n  cockpit update --packages-only --yes",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if options.all || options.binaryOnly || options.packagesOnly || options.yes || options.check || options.report != "" {
				return runUnifiedUpdate(cmd, cfg, options)
			}
			return runUpdate(log, cfg, t)
		},
	}
	command.Flags().BoolVar(&options.all, "all", false, "Update the binary and installed packages")
	command.Flags().BoolVar(&options.binaryOnly, "binary-only", false, "Update only the binary")
	command.Flags().BoolVar(&options.packagesOnly, "packages-only", false, "Update only installed packages")
	command.Flags().BoolVarP(&options.yes, "yes", "y", false, "Apply without prompts; never runs interactive setup")
	command.Flags().BoolVar(&options.check, "check", false, "Check versions without activation or deploy (refreshes registry caches)")
	command.Flags().StringVar(&options.report, "report", "", "Write a versioned JSON outcome report to this file")
	command.MarkFlagsMutuallyExclusive("all", "binary-only", "packages-only")
	return command
}

func runUpdate(log *logging.Manager, cfg *config.Config, t *i18n.Translator) error {
	return runUpdateWithDeps(log, cfg, t, update.NewUpdateService(), os.Stdin)
}

// performUpdateFunc is the function used to perform the actual update. Tests can override this.
var performUpdateFunc = performUpdate

// runUpdateWithDeps is the testable core of runUpdate.
func runUpdateWithDeps(log *logging.Manager, cfg *config.Config, t *i18n.Translator, svc updateChecker, stdin *os.File) error {
	fmt.Println(t.T("update.checking"))

	latestVersion, releaseURL, err := svc.CheckForUpdates()
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

	reader := bufio.NewReader(stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	if input != "y" && input != "yes" && input != "s" && input != "sim" {
		fmt.Println(t.T("update.cancel"))
		return nil
	}

	fmt.Printf(t.T("update.updating")+"\n", latestVersion)

	// Perform the update
	if err := performUpdateFunc(latestVersion); err != nil {
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
	input, _ = reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	if input == "y" || input == "yes" || input == "s" || input == "sim" {
		fmt.Println("Running setup...")
		executable, err := os.Executable()
		if err != nil {
			return err
		}
		return runCommand(executable, "setup")
	}

	return nil
}

// performUpdate builds the official release outside the caller's repository.
func performUpdate(targetVersion string) error {
	fmt.Printf("Preparing official binary %s in isolated staging...\n", targetVersion)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	backup, err := update.NewBinaryUpdater().Update(ctx, targetVersion)
	if err != nil {
		return err
	}
	fmt.Printf("Previous executable preserved: %s\n", backup)
	return nil
}

// runCommandFunc is the function used to run commands. Tests can override this.
var runCommandFunc = runCommandDefault

// runCommand executes a shell command
func runCommand(name string, args ...string) error {
	return runCommandFunc(name, args...)
}

func runCommandDefault(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
