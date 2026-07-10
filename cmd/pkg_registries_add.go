package cmd

import (
	"fmt"

	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/packages"
	"github.com/spf13/cobra"
)

// NewPkgRegistriesAddCommand creates the pkg registries add command.
func NewPkgRegistriesAddCommand() *cobra.Command {
	var (
		branch   string
		priority int
	)

	cmd := &cobra.Command{
		Use:   "add <name> <url>",
		Short: "Add a new registry",
		Long:  "Add a new package registry to the configuration",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			url := args[1]

			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			for _, reg := range cfg.PackageRegistries {
				if reg.Name == name {
					return fmt.Errorf("registry already exists: %s", name)
				}
			}

			if branch == "" {
				branch = "main"
			}

			if priority == 0 {
				priority = len(cfg.PackageRegistries) + 1
			}

			newRegistry := packages.RegistryConfig{
				Name:     name,
				URL:      url,
				Branch:   branch,
				Enabled:  true,
				Priority: priority,
			}

			cfg.PackageRegistries = append(cfg.PackageRegistries, newRegistry)

			if err := cfg.Save(); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			fmt.Printf("✓ Registry added successfully\n")
			fmt.Printf("  Name: %s\n", name)
			fmt.Printf("  URL: %s\n", url)
			fmt.Printf("  Branch: %s\n", branch)
			fmt.Printf("  Priority: %d\n", priority)

			return nil
		},
	}

	cmd.Flags().StringVar(&branch, "branch", "main", "Git branch to use")
	cmd.Flags().IntVar(&priority, "priority", 0, "Registry priority (lower = first)")

	return cmd
}
