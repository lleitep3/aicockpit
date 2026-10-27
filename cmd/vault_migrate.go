package cmd

import (
	"errors"
	"fmt"

	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/i18n"
	"github.com/lleitep3/aicockpit/internal/logging"
	"github.com/lleitep3/aicockpit/internal/vault"
	"github.com/spf13/cobra"
)

// NewVaultMigrateStateCommand provides the only path from forgeable v1 state
// to v2. Legacy grants are discarded and credentials are untouched.
func NewVaultMigrateStateCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
	var confirmed bool
	command := &cobra.Command{
		Use:   "migrate-state",
		Short: "Migrate legacy vault lock state to v2",
		Long:  "Create a fresh v2 lock state. Legacy package grants and global unlocks are discarded; vault credentials are preserved.",
		RunE: func(cmd *cobra.Command, args []string) error {
			backup, err := vault.MigrateState(vault.DefaultLockStatePath(), nil, confirmed)
			if errors.Is(err, vault.ErrMigrationConfirmation) && !confirmed {
				fmt.Fprintln(cmd.OutOrStdout(), "Legacy state will be backed up and discarded; credentials are preserved, but the vault will remain locked.")
				fmt.Fprintln(cmd.OutOrStdout(), "Type MIGRATE to continue, or rerun with --confirm for automation:")
				var answer string
				if _, scanErr := fmt.Fscan(cmd.InOrStdin(), &answer); scanErr != nil || answer != "MIGRATE" {
					return fmt.Errorf("migration cancelled; rerun with --confirm")
				}
				backup, err = vault.MigrateState(vault.DefaultLockStatePath(), nil, true)
			}
			if err != nil {
				return fmt.Errorf("failed to migrate vault lock state: %w", err)
			}
			if backup != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Legacy state backed up to %s; vault is locked.\n", backup)
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "Vault lock state is already v2 and valid; no migration was needed.")
			}
			return nil
		},
	}
	command.Flags().BoolVar(&confirmed, "confirm", false, "confirm discarding legacy lock grants")
	return command
}
