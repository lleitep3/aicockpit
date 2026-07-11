package cmd

import (
	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/services"
	"github.com/spf13/cobra"
)

// newPkgServiceFunc can be replaced in tests.
var newPkgServiceFunc = func(cockpitDir string) services.PackageService {
	return services.NewPackageService(cockpitDir)
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
