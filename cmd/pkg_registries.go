package cmd

import (
	"fmt"

	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/packages"
	"github.com/spf13/cobra"
)

// NewPkgRegistriesCommand creates the pkg registries command.
func NewPkgRegistriesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "registries",
		Short: "Manage package registries",
		Long:  "Manage package registries including add, remove, list, enable, and disable operations",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	// Add subcommands
	cmd.AddCommand(NewPkgRegistriesListCommand())
	cmd.AddCommand(NewPkgRegistriesAddCommand())
	cmd.AddCommand(NewPkgRegistriesRemoveCommand())
	cmd.AddCommand(NewPkgRegistriesEnableCommand())
	cmd.AddCommand(NewPkgRegistriesDisableCommand())
	cmd.AddCommand(NewPkgRegistriesInfoCommand())

	return cmd
}

// NewPkgRegistriesListCommand creates the pkg registries list command.
func NewPkgRegistriesListCommand() *cobra.Command {
	var enabled bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all registries",
		Long:  "List all configured package registries",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load config
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Filter registries if --enabled flag is set
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

			// Display results
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

			// Load config
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Check if registry already exists
			for _, reg := range cfg.PackageRegistries {
				if reg.Name == name {
					return fmt.Errorf("registry already exists: %s", name)
				}
			}

			// Set default branch if not provided
			if branch == "" {
				branch = "main"
			}

			// Set default priority if not provided
			if priority == 0 {
				priority = len(cfg.PackageRegistries) + 1
			}

			// Create new registry
			newRegistry := packages.RegistryConfig{
				Name:     name,
				URL:      url,
				Branch:   branch,
				Enabled:  true,
				Priority: priority,
			}

			// Add to config
			cfg.PackageRegistries = append(cfg.PackageRegistries, newRegistry)

			// Save config
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

			// Load config
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Find and remove registry
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

			// Update config
			cfg.PackageRegistries = newRegistries

			// Save config
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

// NewPkgRegistriesEnableCommand creates the pkg registries enable command.
func NewPkgRegistriesEnableCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enable <name>",
		Short: "Enable a registry",
		Long:  "Enable a disabled package registry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			// Load config
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Find and enable registry
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

			// Save config
			if err := cfg.Save(); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			fmt.Printf("✓ Registry enabled: %s\n", name)

			return nil
		},
	}

	return cmd
}

// NewPkgRegistriesDisableCommand creates the pkg registries disable command.
func NewPkgRegistriesDisableCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "disable <name>",
		Short: "Disable a registry",
		Long:  "Disable a package registry without removing it",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			// Load config
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Find and disable registry
			found := false
			for i, reg := range cfg.PackageRegistries {
				if reg.Name == name {
					cfg.PackageRegistries[i].Enabled = false
					found = true
					break
				}
			}

			if !found {
				return fmt.Errorf("registry not found: %s", name)
			}

			// Save config
			if err := cfg.Save(); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			fmt.Printf("✓ Registry disabled: %s\n", name)

			return nil
		},
	}

	return cmd
}

// NewPkgRegistriesInfoCommand creates the pkg registries info command.
func NewPkgRegistriesInfoCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info <name>",
		Short: "Show registry information",
		Long:  "Display detailed information about a specific registry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			// Load config
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Find registry
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

			// Display registry info
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
