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

// NewPkgValidateCommand creates the pkg validate command.
func NewPkgValidateCommand(svc services.PackageService, cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate <package>",
		Short: "Validate an installed package configuration",
		Long:  "Run the validation script for an installed package to ensure it is correctly configured.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			packageName := args[0]

			if !svc.PackageExists(packageName) {
				return fmt.Errorf("package not installed: %s", packageName)
			}

			// Path to validate script
			scriptPath := filepath.Join(svc.GetPackageInstallPath(packageName), "bin", "validate")
			if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
				return fmt.Errorf("package %s does not implement a 'validate' script", packageName)
			}

			fmt.Printf("Validating package: %s\n", packageName)

			// Execute script
			execCmd := exec.Command(scriptPath)
			execCmd.Stdout = os.Stdout
			execCmd.Stderr = os.Stderr

			if err := execCmd.Run(); err != nil {
				return fmt.Errorf("validation failed: %w", err)
			}

			fmt.Println("Validation successful.")
			return nil
		},
	}

	// cfg is available for future registry-aware validate logic.
	_ = cfg

	return cmd
}
