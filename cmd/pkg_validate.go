package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/packages"
	"github.com/spf13/cobra"
)

// NewPkgValidateCommand creates the pkg validate command.
func NewPkgValidateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate <package>",
		Short: "Validate an installed package configuration",
		Long:  "Run the validation script for an installed package to ensure it is correctly configured.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			packageName := args[0]
			cockpitDir := config.GetCockpitDir()
			pm := packages.NewPackageManager(cockpitDir)

			if !pm.PackageExists(packageName) {
				return fmt.Errorf("package not installed: %s", packageName)
			}

			// Path to validate script
			scriptPath := filepath.Join(pm.GetPackageInstallPath(packageName), "bin", "validate")
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

	return cmd
}
