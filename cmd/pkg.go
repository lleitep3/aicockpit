package cmd

import (
	"github.com/spf13/cobra"
)

// NewPkgCommand creates the pkg command.
func NewPkgCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pkg",
		Short: "Manage AICockpit packages",
		Long:  "Manage packages from registries, including search, install, and uninstall operations",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	// Add subcommands
	cmd.AddCommand(NewPkgSearchCommand())
	cmd.AddCommand(NewPkgInstallCommand())
	cmd.AddCommand(NewPkgUninstallCommand())
	cmd.AddCommand(NewPkgListCommand())
	cmd.AddCommand(NewPkgRegistriesCommand())
	cmd.AddCommand(NewPkgUpgradeCommand())
	cmd.AddCommand(NewPkgConfigureCommand())
	cmd.AddCommand(NewPkgValidateCommand())

	return cmd
}
