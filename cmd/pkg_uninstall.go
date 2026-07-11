package cmd

import (
	"fmt"
	"time"

	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/events"
	"github.com/lleitep3/aicockpit/internal/services"
	"github.com/spf13/cobra"
)

// NewPkgUninstallCommand creates the pkg uninstall command.
func NewPkgUninstallCommand(svc services.PackageService, cfg *config.Config) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "uninstall <package>",
		Short: "Uninstall a package",
		Long:  "Uninstall a package from AICockpit",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			packageName := args[0]

			// Check if package exists
			if !svc.PackageExists(packageName) {
				return fmt.Errorf("package not found: %s", packageName)
			}

			// Get package info
			pkg, err := svc.GetInstalledPackage(packageName)
			if err != nil {
				return fmt.Errorf("failed to get package info: %w", err)
			}

			// Display package info
			fmt.Printf("Package: %s\n", pkg.Name)
			fmt.Printf("Version: %s\n", pkg.Version)
			fmt.Printf("Author: %s\n", pkg.Author)
			fmt.Printf("Description: %s\n", pkg.Description)

			// Run pre_uninstall hooks from the install dir (before files are removed)
			installPath := svc.GetPackageInstallPath(packageName)
			if len(pkg.Installation.PreUninstall) > 0 {
				fmt.Printf("\nRunning pre-uninstall hooks...\n")
				if err := svc.RunPackageHooks(installPath, pkg.Installation.PreUninstall); err != nil {
					if !force {
						return fmt.Errorf("pre-uninstall hook failed: %w", err)
					}
					fmt.Printf("  Warning: pre-uninstall hook failed (--force): %v\n", err)
				}
			}

			// Remove package assets from canonical dirs before deleting package files
			hasAssets := len(pkg.Features.Skills) > 0 ||
				len(pkg.Features.Rules) > 0 ||
				len(pkg.Features.Agents) > 0 ||
				len(pkg.Features.Workflows) > 0 ||
				len(pkg.Features.KB) > 0
			if hasAssets {
				fmt.Printf("\nRemoving assets from canonical dirs...\n")
				if err := svc.RemovePackageAssets(pkg); err != nil {
					fmt.Printf("  ⚠ Asset removal warning: %v\n", err)
				}
			}

			// Uninstall package
			fmt.Printf("\nUninstalling package: %s\n", packageName)
			err = svc.UninstallPackage(packageName)
			if err != nil {
				return fmt.Errorf("failed to uninstall package: %w", err)
			}

			fmt.Printf("✓ Package uninstalled successfully\n")

			// Emit package uninstalled event
			svc.EmitEvent(events.Event{
				Topic: events.TopicPackageUninstalled,
				Payload: events.PackageUninstalledPayload{
					PackageName: packageName,
					Version:     pkg.Version,
					InstallPath: installPath,
					Timestamp:   time.Now(),
				},
			})

			// Redeploy to providers after removing assets
			if hasAssets {
				fmt.Printf("\nRedeploying to active providers...\n")
				if err := svc.TriggerDeploy(""); err != nil {
					fmt.Printf("  ⚠ Deploy warning: %v\n", err)
				}
			}

			// Run post_uninstall hooks — note: package files are gone, so scripts
			// must be self-contained or rely only on system-level paths.
			if len(pkg.Installation.PostUninstall) > 0 {
				fmt.Printf("\nRunning post-uninstall hooks...\n")
				// PostUninstall scripts were already removed with the package files.
				// We warn the user rather than fail silently.
				fmt.Printf("  ⚠ post_uninstall hooks defined but package files were already removed.\n")
				fmt.Printf("  Tip: use pre_uninstall for cleanup that needs the package files.\n")
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Force uninstallation")

	// Ensure cfg is used (suppress unused import warning); cfg is available for
	// future registry-aware uninstall logic.
	_ = cfg

	return cmd
}
