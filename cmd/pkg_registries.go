package cmd

import (
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

	cmd.AddCommand(NewPkgRegistriesListCommand())
	cmd.AddCommand(NewPkgRegistriesAddCommand())
	cmd.AddCommand(NewPkgRegistriesRemoveCommand())
	cmd.AddCommand(NewPkgRegistriesEnableCommand())
	cmd.AddCommand(NewPkgRegistriesDisableCommand())
	cmd.AddCommand(NewPkgRegistriesInfoCommand())

	return cmd
}
