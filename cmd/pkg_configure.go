package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/services"
	"github.com/spf13/cobra"
)

// NewPkgConfigureCommand creates the pkg configure command.
func NewPkgConfigureCommand(svc services.PackageService, cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "configure <package>",
		Short: "Configure an installed package",
		Long:  "Run the configuration script for an installed package.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			packageName := args[0]

			if !svc.PackageExists(packageName) {
				return fmt.Errorf("package not installed: %s", packageName)
			}

			// Path to configure script
			scriptPath := filepath.Join(svc.GetPackageInstallPath(packageName), "bin", "configure")
			if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
				return fmt.Errorf("package %s does not implement a 'configure' script", packageName)
			}

			fmt.Printf("Configuring package: %s\n", packageName)

			// Execute script interactively
			execCmd := exec.Command(scriptPath)
			execCmd.Stdin = os.Stdin
			execCmd.Stdout = os.Stdout
			execCmd.Stderr = os.Stderr

			if err := execCmd.Run(); err != nil {
				return fmt.Errorf("configuration failed: %w", err)
			}

			fmt.Println("Configuration complete.")
			return nil
		},
	}

	// cfg is available for future registry-aware configure logic.
	_ = cfg

	return cmd
}
