package cmd

import (
	"fmt"
	"strings"

	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/packages"
	"github.com/lleitep3/aicockpit/internal/services"
	"github.com/spf13/cobra"
)

// NewPkgInstallCommand creates the pkg install command.
func NewPkgInstallCommand(svc services.PackageService, cfg *config.Config) *cobra.Command {
	var (
		source           string
		withDependencies bool
		interactive      bool
		force            bool
	)

	cmd := &cobra.Command{
		Use:   "install <package>[@version]",
		Short: "Install a package",
		Long:  "Install a package from a registry",
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

			// Get registries to search
			var registriesToSearch []packages.RegistryConfig
			if source != "" {
				// Install from specific registry
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
				// Search in all enabled registries
				registriesToSearch = cfg.PackageRegistries
			}

			// Find package
			fmt.Printf("Searching for package: %s\n", packageName)
			pkgEntry, registryName, err := svc.GetPackage(packageName, registriesToSearch)
			if err != nil {
				return fmt.Errorf("package not found: %s", packageName)
			}

			// Check version if specified
			if version != "" && pkgEntry.Version != version {
				return fmt.Errorf("package version %s not found (available: %s)", version, pkgEntry.Version)
			}

			// Check if already installed
			if svc.PackageExists(packageName) && !force {
				return fmt.Errorf("package already installed: %s (use --force to reinstall)", packageName)
			}

			// Display package info
			fmt.Printf("\nPackage: %s\n", pkgEntry.Name)
			fmt.Printf("Version: %s\n", pkgEntry.Version)
			fmt.Printf("Author: %s\n", pkgEntry.Author)
			fmt.Printf("Description: %s\n", pkgEntry.Description)
			fmt.Printf("Registry: %s\n", registryName)
			fmt.Printf("License: %s\n", pkgEntry.License)

			// Copy package from registry cache
			fmt.Printf("\nCopying package from cache...\n")

			// Get package path from cache
			packageCachePath, err := svc.GetPackageFromCache(registryName, packageName)
			if err != nil {
				return fmt.Errorf("failed to find package in cache: %w", err)
			}

			// Load manifest from cache to get hooks BEFORE copying
			cachedPkg, err := packages.LoadPackage(packageCachePath)
			if err != nil {
				return fmt.Errorf("failed to load package manifest from cache: %w", err)
			}

			// Run pre_install hooks from cache directory
			if len(cachedPkg.Installation.PreInstall) > 0 {
				fmt.Printf("\nRunning pre-install hooks...\n")
				if err := svc.RunPackageHooks(packageCachePath, cachedPkg.Installation.PreInstall); err != nil {
					return fmt.Errorf("pre-install hook failed: %w", err)
				}
			}

			// Copy package to installation directory
			installPath := svc.GetPackageInstallPath(packageName)
			if err := copyDirectory(packageCachePath, installPath); err != nil {
				return fmt.Errorf("failed to copy package: %w", err)
			}

			// Load the downloaded package manifest
			downloadedPkg, err := packages.LoadPackage(installPath)
			if err != nil {
				return fmt.Errorf("failed to load downloaded package: %w", err)
			}

			// Validate the downloaded package
			if err := downloadedPkg.Validate(installPath); err != nil {
				return fmt.Errorf("downloaded package validation failed: %w", err)
			}

			// Save package manifest to installation directory
			if err := packages.SavePackage(installPath, downloadedPkg); err != nil {
				return fmt.Errorf("failed to save package manifest: %w", err)
			}

			fmt.Printf("✓ Package installed successfully\n")
			fmt.Printf("  Location: %s\n", installPath)

			// Run post_install hooks from the installation directory
			if len(downloadedPkg.Installation.PostInstall) > 0 {
				fmt.Printf("\nRunning post-install hooks...\n")
				if err := svc.RunPackageHooks(installPath, downloadedPkg.Installation.PostInstall); err != nil {
					return fmt.Errorf("post-install hook failed: %w", err)
				}
			}

			// Sync package assets (skills/rules/agents/workflows) to canonical dirs
			hasAssets := len(downloadedPkg.Features.Skills) > 0 ||
				len(downloadedPkg.Features.Rules) > 0 ||
				len(downloadedPkg.Features.Agents) > 0 ||
				len(downloadedPkg.Features.Workflows) > 0 ||
				len(downloadedPkg.Features.KB) > 0
			if hasAssets {
				fmt.Printf("\nSyncing assets to canonical dirs...\n")
				if err := svc.SyncPackageAssets(downloadedPkg, installPath); err != nil {
					fmt.Printf("  ⚠ Asset sync warning: %v\n", err)
				}

				fmt.Printf("\nDeploying to active providers...\n")
				if err := svc.TriggerDeploy(""); err != nil {
					fmt.Printf("  ⚠ Deploy warning: %v\n", err)
				}
			}

			// Install dependencies if requested
			if withDependencies && len(downloadedPkg.Dependencies) > 0 {
				fmt.Printf("\nInstalling dependencies...\n")
				for _, dep := range downloadedPkg.Dependencies {
					fmt.Printf("  Installing dependency: %s (%s)\n", dep.Name, dep.Version)

					// Recursively install dependency
					depCmd := NewPkgInstallCommand(svc, cfg)
					depArgs := []string{dep.Name}
					if withDependencies {
						depArgs = append(depArgs, "--with-dependencies")
					}

					if err := depCmd.RunE(depCmd, depArgs); err != nil {
						if !dep.Optional {
							return fmt.Errorf("failed to install required dependency %s: %w", dep.Name, err)
						}
						fmt.Printf("  Warning: failed to install optional dependency %s: %v\n", dep.Name, err)
					} else {
						fmt.Printf("  ✓ Dependency %s installed\n", dep.Name)
					}
				}
			}

			if interactive {
				fmt.Printf("  Note: Interactive configuration not yet implemented\n")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&source, "source", "", "Install from specific registry")
	cmd.Flags().BoolVar(&withDependencies, "with-dependencies", false, "Install with dependencies")
	cmd.Flags().BoolVar(&interactive, "interactive", false, "Interactive configuration")
	cmd.Flags().BoolVar(&force, "force", false, "Force installation")

	return cmd
}
