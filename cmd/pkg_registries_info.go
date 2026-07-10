package cmd

import (
	"fmt"

	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/packages"
	"github.com/spf13/cobra"
)

// NewPkgRegistriesInfoCommand creates the pkg registries info command.
func NewPkgRegistriesInfoCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info <name>",
		Short: "Show registry information",
		Long:  "Display detailed information about a specific registry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			var registry *packages.RegistryConfig
			for i, reg := range cfg.PackageRegistries {
				if reg.Name == name {
					registry = &cfg.PackageRegistries[i]
					break
				}
			}

			if registry == nil {
				return fmt.Errorf("registry not found: %s", name)
			}

			status := "enabled"
			if !registry.Enabled {
				status = "disabled"
			}

			fmt.Printf("Registry: %s\n", registry.Name)
			fmt.Printf("URL: %s\n", registry.URL)
			fmt.Printf("Branch: %s\n", registry.Branch)
			fmt.Printf("Status: %s\n", status)
			fmt.Printf("Priority: %d\n", registry.Priority)

			return nil
		},
	}

	return cmd
}
