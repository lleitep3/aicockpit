package cmd

import (
	"fmt"

	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/packages"
	"github.com/spf13/cobra"
)

// NewPkgRegistriesRemoveCommand creates the pkg registries remove command.
func NewPkgRegistriesRemoveCommand() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a registry",
		Long:  "Remove a package registry from the configuration",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			found := false
			var newRegistries []packages.RegistryConfig
			for _, reg := range cfg.PackageRegistries {
				if reg.Name == name {
					found = true
					continue
				}
				newRegistries = append(newRegistries, reg)
			}

			if !found {
				return fmt.Errorf("registry not found: %s", name)
			}

			cfg.PackageRegistries = newRegistries

			if err := cfg.Save(); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			fmt.Printf("✓ Registry removed successfully: %s\n", name)

			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Force removal without confirmation")

	return cmd
}
