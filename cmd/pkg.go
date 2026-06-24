package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/packages"
	"github.com/spf13/cobra"
)

// NewPkgCommand creates the pkg command.
func NewPkgCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pkg",
		Short: "Manage AICockpit packages",
		Long:  "Manage packages from registries, including search, install, and uninstall operations",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	// Add subcommands
	cmd.AddCommand(NewPkgSearchCommand())
	cmd.AddCommand(NewPkgInstallCommand())
	cmd.AddCommand(NewPkgUninstallCommand())
	cmd.AddCommand(NewPkgListCommand())
	cmd.AddCommand(NewPkgRegistriesCommand())
	cmd.AddCommand(NewPkgUpgradeCommand())
	cmd.AddCommand(NewPkgConfigureCommand())
	cmd.AddCommand(NewPkgValidateCommand())

	return cmd
}

// NewPkgSearchCommand creates the pkg search command.
func NewPkgSearchCommand() *cobra.Command {
	var (
		source   string
		category string
		tag      string
		detailed bool
	)

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search for packages in registries",
		Long:  "Search for packages in registries by name, description, category, or tags",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := ""
			if len(args) > 0 {
				query = args[0]
			}

			// Validate input
			if query == "" && category == "" && tag == "" {
				return fmt.Errorf("please provide a search query, category, or tag")
			}

			// Load config
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Create registry manager
			cockpitDir := config.GetCockpitDir()
			rm := packages.NewRegistryManager(cockpitDir)

			// Get registries to search
			var registriesToSearch []packages.RegistryConfig
			if source != "" {
				// Search in specific registry
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

			// Perform search
			var results []packages.PackageIndexEntry
			if category != "" {
				results, err = rm.SearchByCategory(category, registriesToSearch)
			} else if tag != "" {
				results, err = rm.SearchByTag(tag, registriesToSearch)
			} else {
				results, err = rm.SearchPackages(query, registriesToSearch)
			}

			if err != nil {
				return fmt.Errorf("search failed: %w", err)
			}

			// Display results
			if len(results) == 0 {
				fmt.Println("No packages found")
				return nil
			}

			fmt.Printf("Found %d package(s):\n\n", len(results))

			for i, pkg := range results {
				fmt.Printf("%d. %s (%s)\n", i+1, pkg.Name, pkg.Version)
				fmt.Printf("   Author: %s\n", pkg.Author)
				fmt.Printf("   Description: %s\n", pkg.Description)
				fmt.Printf("   Category: %s\n", pkg.Category)
				fmt.Printf("   Status: %s\n", pkg.Status)
				fmt.Printf("   Providers: %s\n", strings.Join(pkg.SupportedProviders, ", "))

				if detailed {
					fmt.Printf("   License: %s\n", pkg.License)
					fmt.Printf("   Tags: %s\n", strings.Join(pkg.Tags, ", "))
					fmt.Printf("   Features: %s\n", strings.Join(pkg.Features, ", "))
					fmt.Printf("   Released: %s\n", pkg.ReleasedAt)
				}
				fmt.Println()
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&source, "source", "", "Search in specific registry")
	cmd.Flags().StringVar(&category, "category", "", "Search by category")
	cmd.Flags().StringVar(&tag, "tag", "", "Search by tag")
	cmd.Flags().BoolVar(&detailed, "detailed", false, "Show detailed information")

	return cmd
}

// NewPkgInstallCommand creates the pkg install command.
func NewPkgInstallCommand() *cobra.Command {
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

			// Load config
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Create registry manager
			cockpitDir := config.GetCockpitDir()
			rm := packages.NewRegistryManager(cockpitDir)

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
			pkgEntry, registryName, err := rm.GetPackage(packageName, registriesToSearch)
			if err != nil {
				return fmt.Errorf("package not found: %s", packageName)
			}

			// Check version if specified
			if version != "" && pkgEntry.Version != version {
				return fmt.Errorf("package version %s not found (available: %s)", version, pkgEntry.Version)
			}

			// Create package manager
			pm := packages.NewPackageManager(cockpitDir)

			// Check if already installed
			if pm.PackageExists(packageName) && !force {
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
			cache := packages.NewRegistryCache(config.GetCockpitDir())
			packageCachePath, err := cache.GetPackageFromCache(registryName, packageName)
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
				if err := pm.RunPackageHooks(packageCachePath, cachedPkg.Installation.PreInstall); err != nil {
					return fmt.Errorf("pre-install hook failed: %w", err)
				}
			}

			// Copy package to installation directory
			installPath := pm.GetPackageInstallPath(packageName)
			if err := copyDirectory(packageCachePath, installPath); err != nil {
				return fmt.Errorf("failed to copy package: %w", err)
			}

			// Load the downloaded package manifest
			downloadedPkg, err := packages.LoadPackage(installPath)
			if err != nil {
				return fmt.Errorf("failed to load downloaded package: %w", err)
			}

			// Validate the downloaded package
			if err := downloadedPkg.Validate(); err != nil {
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
				if err := pm.RunPackageHooks(installPath, downloadedPkg.Installation.PostInstall); err != nil {
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
				if err := pm.SyncPackageAssets(downloadedPkg, installPath); err != nil {
					fmt.Printf("  ⚠ Asset sync warning: %v\n", err)
				}

				fmt.Printf("\nDeploying to active providers...\n")
				if err := pm.TriggerDeploy(""); err != nil {
					fmt.Printf("  ⚠ Deploy warning: %v\n", err)
				}
			}

			// Install dependencies if requested
			if withDependencies && len(downloadedPkg.Dependencies) > 0 {
				fmt.Printf("\nInstalling dependencies...\n")
				for _, dep := range downloadedPkg.Dependencies {
					fmt.Printf("  Installing dependency: %s (%s)\n", dep.Name, dep.Version)

					// Recursively install dependency
					depCmd := NewPkgInstallCommand()
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

// NewPkgUninstallCommand creates the pkg uninstall command.
func NewPkgUninstallCommand() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "uninstall <package>",
		Short: "Uninstall a package",
		Long:  "Uninstall a package from AICockpit",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			packageName := args[0]

			// Load config
			cockpitDir := config.GetCockpitDir()

			// Create package manager
			pm := packages.NewPackageManager(cockpitDir)

			// Check if package exists
			if !pm.PackageExists(packageName) {
				return fmt.Errorf("package not found: %s", packageName)
			}

			// Get package info
			pkg, err := pm.GetInstalledPackage(packageName)
			if err != nil {
				return fmt.Errorf("failed to get package info: %w", err)
			}

			// Display package info
			fmt.Printf("Package: %s\n", pkg.Name)
			fmt.Printf("Version: %s\n", pkg.Version)
			fmt.Printf("Author: %s\n", pkg.Author)
			fmt.Printf("Description: %s\n", pkg.Description)

			// Run pre_uninstall hooks from the install dir (before files are removed)
			installPath := pm.GetPackageInstallPath(packageName)
			if len(pkg.Installation.PreUninstall) > 0 {
				fmt.Printf("\nRunning pre-uninstall hooks...\n")
				if err := pm.RunPackageHooks(installPath, pkg.Installation.PreUninstall); err != nil {
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
				if err := pm.RemovePackageAssets(pkg); err != nil {
					fmt.Printf("  ⚠ Asset removal warning: %v\n", err)
				}
			}

			// Uninstall package
			fmt.Printf("\nUninstalling package: %s\n", packageName)
			err = pm.UninstallPackage(packageName)
			if err != nil {
				return fmt.Errorf("failed to uninstall package: %w", err)
			}

			fmt.Printf("✓ Package uninstalled successfully\n")

			// Redeploy to providers after removing assets
			if hasAssets {
				fmt.Printf("\nRedeploying to active providers...\n")
				if err := pm.TriggerDeploy(""); err != nil {
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

	return cmd
}

// NewPkgUpgradeCommand creates the pkg upgrade command.
func NewPkgUpgradeCommand() *cobra.Command {
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

			// Load config
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			cockpitDir := config.GetCockpitDir()
			pm := packages.NewPackageManager(cockpitDir)

			if !pm.PackageExists(packageName) {
				return fmt.Errorf("package not installed: %s", packageName)
			}

			oldPkg, err := pm.GetInstalledPackage(packageName)
			if err != nil {
				return fmt.Errorf("failed to load installed package: %w", err)
			}

			fmt.Printf("Current version: %s\n", oldPkg.Version)

			// Create registry manager
			rm := packages.NewRegistryManager(cockpitDir)

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
			pkgEntry, registryName, err := rm.GetPackage(packageName, registriesToSearch)
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

			cache := packages.NewRegistryCache(config.GetCockpitDir())
			packageCachePath, err := cache.GetPackageFromCache(registryName, packageName)
			if err != nil {
				return fmt.Errorf("failed to find package in cache: %w", err)
			}

			fmt.Printf("\nPerforming upgrade...\n")
			if err := pm.UpgradePackage(packageName, packageCachePath); err != nil {
				return fmt.Errorf("failed to upgrade package: %w", err)
			}

			fmt.Printf("✓ Package %s upgraded successfully to %s\n", packageName, pkgEntry.Version)

			// Redeploy to active providers
			fmt.Printf("\nRedeploying to active providers...\n")
			if err := pm.TriggerDeploy(""); err != nil {
				fmt.Printf("  ⚠ Deploy warning: %v\n", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&source, "source", "", "Upgrade from specific registry")
	cmd.Flags().BoolVar(&force, "force", false, "Force upgrade even if versions match")

	return cmd
}

// NewPkgListCommand creates the pkg list command.
func NewPkgListCommand() *cobra.Command {
	var (
		source   string
		detailed bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available packages",
		Long:  "List all available packages from registries",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load config
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Create registry manager
			cockpitDir := config.GetCockpitDir()
			rm := packages.NewRegistryManager(cockpitDir)

			// Get registries to list
			var registriesToList []packages.RegistryConfig
			if source != "" {
				// List from specific registry
				found := false
				for _, reg := range cfg.PackageRegistries {
					if reg.Name == source {
						registriesToList = append(registriesToList, reg)
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("registry not found: %s", source)
				}
			} else {
				// List from all enabled registries
				registriesToList = cfg.PackageRegistries
			}

			// Get packages
			pkgs, err := rm.ListPackages(registriesToList)
			if err != nil {
				return fmt.Errorf("failed to list packages: %w", err)
			}

			// Display results
			if len(pkgs) == 0 {
				fmt.Println("No packages found")
				return nil
			}

			fmt.Printf("Available Packages (%d):\n\n", len(pkgs))

			for i, pkg := range pkgs {
				fmt.Printf("%d. %s (%s)\n", i+1, pkg.Name, pkg.Version)
				fmt.Printf("   Author: %s\n", pkg.Author)
				fmt.Printf("   Description: %s\n", pkg.Description)
				fmt.Printf("   Category: %s\n", pkg.Category)
				fmt.Printf("   Status: %s\n", pkg.Status)

				if detailed {
					fmt.Printf("   License: %s\n", pkg.License)
					fmt.Printf("   Providers: %s\n", strings.Join(pkg.SupportedProviders, ", "))
					fmt.Printf("   Features: %s\n", strings.Join(pkg.Features, ", "))
					fmt.Printf("   Released: %s\n", pkg.ReleasedAt)
				}
				fmt.Println()
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&source, "source", "", "List from specific registry")
	cmd.Flags().BoolVar(&detailed, "detailed", false, "Show detailed information")

	return cmd
}

// copyDirectory copies a directory recursively
func copyDirectory(src, dst string) error {
	// Create destination directory
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Read source directory
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("failed to read source directory: %w", err)
	}

	// Copy each entry
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			// Recursively copy subdirectory
			if err := copyDirectory(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			// Copy file
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// copyFile copies a single file
func copyFile(src, dst string) error {
	// Read source file
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	// Get source file info to preserve permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat source file: %w", err)
	}

	// Write to destination file with same permissions
	if err := os.WriteFile(dst, data, srcInfo.Mode()); err != nil {
		return fmt.Errorf("failed to write destination file: %w", err)
	}

	return nil
}

// NewPkgConfigureCommand creates the pkg configure command.
func NewPkgConfigureCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "configure <package>",
		Short: "Configure an installed package",
		Long:  "Run the configuration script for an installed package.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			packageName := args[0]
			cockpitDir := config.GetCockpitDir()
			pm := packages.NewPackageManager(cockpitDir)

			if !pm.PackageExists(packageName) {
				return fmt.Errorf("package not installed: %s", packageName)
			}

			// Path to configure script
			scriptPath := filepath.Join(pm.GetPackageInstallPath(packageName), "bin", "configure")
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

	return cmd
}

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
