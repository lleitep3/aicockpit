package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/events"
	"github.com/lleitep3/aicockpit/internal/packages"
	"github.com/lleitep3/aicockpit/internal/services"
	"github.com/spf13/cobra"
)

// NewPkgUpgradeCommand creates the pkg upgrade command.
func NewPkgUpgradeCommand(svc services.PackageService, cfg *config.Config) *cobra.Command {
	var (
		source string
		force  bool
	)

	cmd := &cobra.Command{
		Use:   "upgrade <package>[@version]",
		Short: "Upgrade a package",
		Long:  "Upgrade a package to a specific version or the latest available version",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			packageSpec := args[0]

			// Parse package name and version
			parts := strings.Split(packageSpec, "@")
			packageName := parts[0]
			version := ""
			if len(parts) > 1 {
				version = parts[1]
			}

			if !svc.PackageExists(packageName) {
				return fmt.Errorf("package not installed: %s", packageName)
			}

			oldPkg, err := svc.GetInstalledPackage(packageName)
			if err != nil {
				return fmt.Errorf("failed to load installed package: %w", err)
			}

			fmt.Printf("Current version: %s\n", oldPkg.Version)

			// Get registries to search
			var registriesToSearch []packages.RegistryConfig
			if source != "" {
				found := false
				for _, reg := range cfg.PackageRegistries {
					if reg.Name == source {
						registriesToSearch = append(registriesToSearch, reg)
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("registry not found: %s", source)
				}
			} else {
				registriesToSearch = cfg.PackageRegistries
			}

			fmt.Printf("Searching for package: %s\n", packageName)
			pkgEntry, registryName, err := svc.GetPackage(packageName, registriesToSearch)
			if err != nil {
				return fmt.Errorf("package not found in registry: %s", packageName)
			}

			if version != "" && pkgEntry.Version != version {
				return fmt.Errorf("package version %s not found (available: %s)", version, pkgEntry.Version)
			}

			if pkgEntry.Version == oldPkg.Version && !force {
				fmt.Printf("Package %s is already up to date (%s)\n", packageName, oldPkg.Version)
				return nil
			}

			fmt.Printf("Upgrading to version: %s\n", pkgEntry.Version)

			packageCachePath, err := svc.GetPackageFromCache(registryName, packageName)
			if err != nil {
				return fmt.Errorf("failed to find package in cache: %w", err)
			}

			fmt.Printf("\nPerforming upgrade...\n")
			if err := svc.UpgradePackage(packageName, packageCachePath); err != nil {
				return fmt.Errorf("failed to upgrade package: %w", err)
			}

			fmt.Printf("✓ Package %s upgraded successfully to %s\n", packageName, pkgEntry.Version)

			// Emit package upgraded event
			svc.EmitEvent(events.Event{
				Topic: events.TopicPackageUpgraded,
				Payload: events.PackageUpgradedPayload{
					PackageName: packageName,
					OldVersion:  oldPkg.Version,
					NewVersion:  pkgEntry.Version,
					InstallPath: svc.GetPackageInstallPath(packageName),
					Timestamp:   time.Now(),
				},
			})

			// Redeploy to active providers
			fmt.Printf("\nRedeploying to active providers...\n")
			if err := svc.TriggerDeploy(""); err != nil {
				fmt.Printf("  ⚠ Deploy warning: %v\n", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&source, "source", "", "Upgrade from specific registry")
	cmd.Flags().BoolVar(&force, "force", false, "Force upgrade even if versions match")

	return cmd
}
