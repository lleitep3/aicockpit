package cmd

import (
	"fmt"

	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/packages"
	"github.com/spf13/cobra"
)

// NewPkgRegistriesListCommand creates the pkg registries list command.
func NewPkgRegistriesListCommand() *cobra.Command {
	var enabled bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all registries",
		Long:  "List all configured package registries",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			registries := cfg.PackageRegistries
			if enabled {
				var enabledRegistries []packages.RegistryConfig
				for _, reg := range registries {
					if reg.Enabled {
						enabledRegistries = append(enabledRegistries, reg)
					}
				}
				registries = enabledRegistries
			}

			if len(registries) == 0 {
				fmt.Println("No registries configured")
				return nil
			}

			fmt.Printf("Configured Registries (%d):\n\n", len(registries))

			for i, reg := range registries {
				status := "enabled"
				if !reg.Enabled {
					status = "disabled"
				}

				fmt.Printf("%d. %s (priority: %d) - %s\n", i+1, reg.Name, reg.Priority, status)
				fmt.Printf("   URL: %s\n", reg.URL)
				fmt.Printf("   Branch: %s\n", reg.Branch)
				fmt.Println()
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&enabled, "enabled", false, "Show only enabled registries")

	return cmd
}
