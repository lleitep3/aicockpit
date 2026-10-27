package cmd

import (
	"fmt"

	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/spf13/cobra"
)

// NewPkgRegistriesEnableCommand creates the pkg registries enable command.
func NewPkgRegistriesEnableCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enable <name>",
		Short: "Enable a registry",
		Long:  "Enable a disabled package registry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			found := false
			for i, reg := range cfg.PackageRegistries {
				if reg.Name == name {
					cfg.PackageRegistries[i].Enabled = true
					found = true
					break
				}
			}

			if !found {
				return fmt.Errorf("registry not found: %s", name)
			}

			if err := cfg.Save(); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			fmt.Printf("✓ Registry enabled: %s\n", name)

			return nil
		},
	}

	return cmd
}
