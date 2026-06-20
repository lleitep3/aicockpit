package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lleite/aicockpit/internal/config"
	"github.com/lleite/aicockpit/internal/packages"
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

			if withDependencies {
				fmt.Printf("  Note: Dependency installation not yet implemented\n")
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

			// Uninstall package
			fmt.Printf("\nUninstalling package: %s\n", packageName)
			err = pm.UninstallPackage(packageName)
			if err != nil {
				return fmt.Errorf("failed to uninstall package: %w", err)
			}

			fmt.Printf("✓ Package uninstalled successfully\n")
			fmt.Printf("  Backup created at: %s.backup\n", pm.GetPackageInstallPath(packageName))

			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Force uninstallation")

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

// parseGitHubURL parses a repository URL and extracts owner, repo, and branch.
// Expected format: https://github.com/{owner}/{repo} or http://localhost:8888
func parseGitHubURL(url string) (owner, repo, branch string, err error) {
	originalURL := url

	// Remove https:// or http:// prefix
	if strings.HasPrefix(url, "https://") {
		url = strings.TrimPrefix(url, "https://")
	} else if strings.HasPrefix(url, "http://") {
		url = strings.TrimPrefix(url, "http://")
	}

	// Remove github.com/ prefix if present
	if strings.HasPrefix(url, "github.com/") {
		url = strings.TrimPrefix(url, "github.com/")
	}

	// Split by /
	parts := strings.Split(url, "/")

	// For GitHub URLs, we need at least owner/repo
	// For other URLs (like localhost:8888), we use the original URL as owner
	if len(parts) >= 2 && !strings.Contains(parts[0], ":") {
		// GitHub-style URL (owner/repo)
		owner = parts[0]
		repo = strings.TrimSuffix(parts[1], ".git")
		branch = "main" // Default branch
		return owner, repo, branch, nil
	}

	// For non-GitHub URLs (like localhost:8888), return the original URL as-is
	// This will be used directly in constructRawGitHubURL
	return originalURL, "", "main", nil
}

// extractBranchFromURL extracts the branch name from a GitHub URL.
// Expected format: https://github.com/{owner}/{repo}/tree/{branch}/{path}
func extractBranchFromURL(url string) string {
	// Look for /tree/ in the URL
	if !strings.Contains(url, "/tree/") {
		return ""
	}

	// Split by /tree/
	parts := strings.Split(url, "/tree/")
	if len(parts) < 2 {
		return ""
	}

	// Get the part after /tree/
	remainder := parts[1]

	// Split by / to get the branch
	branchParts := strings.Split(remainder, "/")
	if len(branchParts) > 0 {
		return branchParts[0]
	}

	return ""
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
