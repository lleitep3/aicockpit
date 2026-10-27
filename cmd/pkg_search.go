package cmd

import (
	"fmt"
	"strings"

	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/packages"
	"github.com/lleitep3/aicockpit/internal/services"
	"github.com/spf13/cobra"
)

// NewPkgSearchCommand creates the pkg search command.
func NewPkgSearchCommand(svc services.PackageService, cfg *config.Config) *cobra.Command {
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
			var (
				results []packages.PackageIndexEntry
				err     error
			)
			if category != "" {
				results, err = svc.SearchByCategory(category, registriesToSearch)
			} else if tag != "" {
				results, err = svc.SearchByTag(tag, registriesToSearch)
			} else {
				results, err = svc.SearchPackages(query, registriesToSearch)
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
