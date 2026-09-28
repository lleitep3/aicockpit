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
	"golang.org/x/mod/semver"
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
				return fmt.Errorf("package lookup failed for %s: %w", packageName, err)
			}

			if version != "" && pkgEntry.Version != version {
				return fmt.Errorf("package version %s not found (available: %s)", version, pkgEntry.Version)
			}

			current := "v" + strings.TrimPrefix(oldPkg.Version, "v")
			available := "v" + strings.TrimPrefix(pkgEntry.Version, "v")
			if !semver.IsValid(current) || !semver.IsValid(available) {
				return fmt.Errorf("cannot safely compare package versions: installed %s, available %s", oldPkg.Version, pkgEntry.Version)
			}
			if semver.Compare(available, current) < 0 && version == "" {
				return fmt.Errorf("refusing automatic downgrade from %s to %s; request %s@%s explicitly", oldPkg.Version, pkgEntry.Version, packageName, pkgEntry.Version)
			}
			if pkgEntry.Version == oldPkg.Version && !force {
				fmt.Printf("Package %s is already up to date (%s)\n", packageName, oldPkg.Version)
				return nil
			}

			fmt.Printf("Upgrading to version: %s\n", pkgEntry.Version)

			packagePath := pkgEntry.Path
			if packagePath == "" {
				packagePath = packageName // Legacy indexes stored packages at the root.
			}
			packageCachePath, err := svc.GetPackageFromCache(registryName, packagePath)
			if err != nil {
				return fmt.Errorf("failed to find package in cache: %w", err)
			}

			candidate, err := packages.LoadPackage(packageCachePath)
			if err != nil {
				return fmt.Errorf("failed to read candidate: %w", err)
			}
			if candidate.Name != packageName || candidate.Version != pkgEntry.Version {
				return fmt.Errorf("candidate identity/version does not match registry index")
			}

			fmt.Printf("\nPerforming upgrade...\n")
			if err := svc.UpgradePackage(packageName, packageCachePath); err != nil {
				return fmt.Errorf("failed to upgrade package: %w", err)
			}

			fmt.Printf("Package %s activated at %s; deploying assets...\n", packageName, pkgEntry.Version)

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
				return fmt.Errorf("package %s activated at %s, but provider deploy failed; retry cockpit deploy: %w", packageName, pkgEntry.Version, err)
			}

			fmt.Printf("✓ Package %s upgraded and deployed successfully to %s\n", packageName, pkgEntry.Version)
			return nil
		},
	}

	cmd.Flags().StringVar(&source, "source", "", "Upgrade from specific registry")
	cmd.Flags().BoolVar(&force, "force", false, "Force upgrade even if versions match")

	return cmd
}
