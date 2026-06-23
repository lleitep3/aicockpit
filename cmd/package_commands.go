package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/packages"
	"github.com/spf13/cobra"
)

// LoadPackageCommands loads and registers commands from installed packages
func LoadPackageCommands(rootCmd *cobra.Command) error {
	cockpitDir := config.GetCockpitDir()
	packagesDir := filepath.Join(cockpitDir, "packages")

	// Check if packages directory exists
	if _, err := os.Stat(packagesDir); err != nil {
		if os.IsNotExist(err) {
			return nil // No packages installed yet
		}
		return fmt.Errorf("failed to check packages directory: %w", err)
	}

	// Read installed packages
	entries, err := os.ReadDir(packagesDir)
	if err != nil {
		return fmt.Errorf("failed to read packages directory: %w", err)
	}

	// Load each package
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		packageName := entry.Name()
		packagePath := filepath.Join(packagesDir, packageName)

		// Check if it's a valid package
		manifestPath := filepath.Join(packagePath, "cockpit-package.yml")
		if _, err := os.Stat(manifestPath); err != nil {
			continue
		}

		// Load package manifest
		pkg, err := packages.LoadPackage(packagePath)
		if err != nil {
			fmt.Printf("Warning: failed to load package %s: %v\n", packageName, err)
			continue
		}

		// Create a wrapper command for this package
		cmd := createPackageCommand(pkg, packageName, packagePath)
		if cmd != nil {
			rootCmd.AddCommand(cmd)
		}
	}

	return nil
}

// createPackageCommand creates a command wrapper for a package
func createPackageCommand(pkg *packages.Package, packageName, packagePath string) *cobra.Command {
	// Try to get the command name from the package features
	commandName := getPackageCommandName(pkg, packageName)

	return &cobra.Command{
		Use:   commandName,
		Short: pkg.Description,
		Long:  pkg.Description,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Silence usage so cobra doesn't print usage on package script failure
			cmd.SilenceUsage = true

			// Try to execute package command
			err := executePackageCommand(packageName, packagePath, args)
			if err != nil {
				var exitErr *exec.ExitError
				if errors.As(err, &exitErr) {
					// Propagate the original script's exit code transparently
					os.Exit(exitErr.ExitCode())
				}
				return err
			}
			return nil
		},
	}
}

// getPackageCommandName extracts the command name from package features
func getPackageCommandName(pkg *packages.Package, packageName string) string {
	// Check if package has modules with a defined command name
	if len(pkg.Features.Modules) > 0 {
		// Get the first module's name if available
		if pkg.Features.Modules[0].Name != "" {
			return pkg.Features.Modules[0].Name
		}
	}

	// Fall back to package name
	return packageName
}

// executePackageCommand executes a command from a package
func executePackageCommand(packageName, packagePath string, args []string) error {
	// Check if there's a script to execute
	scriptPath := filepath.Join(packagePath, "bin", packageName)
	if _, err := os.Stat(scriptPath); err == nil {
		// Execute the script
		return executeScript(scriptPath, args)
	}

	// Also try the package name directly (for packages like hello-world with hello command)
	// This is a fallback for when the command name differs from the package name
	entries, err := os.ReadDir(filepath.Join(packagePath, "bin"))
	if err == nil && len(entries) > 0 {
		// Execute the first script found
		scriptPath := filepath.Join(packagePath, "bin", entries[0].Name())
		if !entries[0].IsDir() {
			return executeScript(scriptPath, args)
		}
	}

	// Check if there's a Go module we can execute
	modulesPath := filepath.Join(packagePath, "modules")
	if _, err := os.Stat(modulesPath); err == nil {
		// For now, return a message indicating this needs to be implemented
		return fmt.Errorf("package %s has Go modules but execution is not yet implemented", packageName)
	}

	return fmt.Errorf("package %s has no executable", packageName)
}

// executeScript executes a script file
func executeScript(scriptPath string, args []string) error {
	// Import os/exec
	cmd := exec.Command(scriptPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

// CreateDynamicCommand creates a dynamic command that can load subcommands from packages
func CreateDynamicCommand(commandName string) *cobra.Command {
	return &cobra.Command{
		Use:   commandName,
		Short: fmt.Sprintf("Execute %s command from installed package", commandName),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Silence usage so cobra doesn't print usage on package script failure
			cmd.SilenceUsage = true

			// Find which package provides this command
			cockpitDir := config.GetCockpitDir()
			packagesDir := filepath.Join(cockpitDir, "packages")

			entries, err := os.ReadDir(packagesDir)
			if err != nil {
				return fmt.Errorf("failed to read packages directory: %w", err)
			}

			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}

				packageName := entry.Name()
				packagePath := filepath.Join(packagesDir, packageName)

				// Check if this package has the command
				if hasCommand(packagePath, commandName) {
					err := executePackageCommand(packageName, packagePath, args)
					if err != nil {
						var exitErr *exec.ExitError
						if errors.As(err, &exitErr) {
							// Propagate the original script's exit code transparently
							os.Exit(exitErr.ExitCode())
						}
						return err
					}
					return nil
				}
			}

			return fmt.Errorf("command %s not found in any installed package", commandName)
		},
	}
}

// hasCommand checks if a package provides a specific command
func hasCommand(packagePath, commandName string) bool {
	// Check if the package has a modules directory with the command
	modulesPath := filepath.Join(packagePath, "modules")
	if _, err := os.Stat(modulesPath); err != nil {
		return false
	}

	// For now, assume the package name matches the command name
	// In the future, we could parse the package manifest to get the actual command names
	return true
}
