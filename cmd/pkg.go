package cmd

import (
	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/events"
	"github.com/lleitep3/aicockpit/internal/services"
	"github.com/spf13/cobra"
)

// newPkgServiceFunc can be replaced in tests.
var newPkgServiceFunc = func(cockpitDir string) services.PackageService {
	bus := events.New()
	// Placeholder log subscriber — future issues can replace TriggerDeploy entirely.
	for _, topic := range []events.Topic{
		events.TopicPackageInstalled,
		events.TopicPackageUninstalled,
		events.TopicPackageUpgraded,
	} {
		bus.Subscribe(topic, func(_ events.Event) error { return nil })
	}
	return services.NewPackageService(cockpitDir, bus)
}

// NewPkgCommand creates the pkg command.
func NewPkgCommand(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pkg",
		Short: "Manage AICockpit packages",
		Long:  "Manage packages from registries, including search, install, and uninstall operations",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	svc := newPkgServiceFunc(config.GetCockpitDir())

	// Add subcommands
	cmd.AddCommand(NewPkgSearchCommand(svc, cfg))
	cmd.AddCommand(NewPkgInstallCommand(svc, cfg))
	cmd.AddCommand(NewPkgUninstallCommand(svc, cfg))
	cmd.AddCommand(NewPkgListCommand(svc, cfg))
	cmd.AddCommand(NewPkgRegistriesCommand())
	cmd.AddCommand(NewPkgUpgradeCommand(svc, cfg))
	cmd.AddCommand(NewPkgConfigureCommand(svc, cfg))
	cmd.AddCommand(NewPkgValidateCommand(svc, cfg))

	return cmd
}
