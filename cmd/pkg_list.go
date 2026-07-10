package cmd

import (
	"fmt"
	"strings"

	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/packages"
	"github.com/spf13/cobra"
)

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
