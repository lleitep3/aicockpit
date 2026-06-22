package cmd

import (
	"fmt"
	"syscall"

	"github.com/lleite/aicockpit/internal/config"
	"github.com/lleite/aicockpit/internal/i18n"
	"github.com/lleite/aicockpit/internal/logging"
	"github.com/lleite/aicockpit/internal/vault"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// NewVaultCommand creates the root vault command.
func NewVaultCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
	vaultCmd := &cobra.Command{
		Use:   "vault",
		Short: "Manage secure secrets and credentials",
		Long:  "Store and retrieve secure credentials (API keys, tokens) using the OS native keyring.",
	}

	vaultCmd.AddCommand(NewVaultSetCommand(log, cfg, t))
	vaultCmd.AddCommand(NewVaultGetCommand(log, cfg, t))
	vaultCmd.AddCommand(NewVaultRemoveCommand(log, cfg, t))

	return vaultCmd
}

// NewVaultSetCommand creates the vault set subcommand.
func NewVaultSetCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
	var valueFlag string

	setCmd := &cobra.Command{
		Use:   "set <key>",
		Short: "Store a secret securely",
		Long:  "Store a secret securely. If --value is not provided, you will be prompted to enter it securely.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			value := valueFlag

			// Prompt if value not provided
			if value == "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Enter secret for '%s': ", key)
				bytePassword, err := term.ReadPassword(int(syscall.Stdin))
				fmt.Fprintln(cmd.OutOrStdout()) // Print newline after hidden input
				if err != nil {
					return fmt.Errorf("failed to read secret securely: %w", err)
				}
				value = string(bytePassword)
			}

			if value == "" {
				return fmt.Errorf("secret value cannot be empty")
			}

			v := vault.NewOSVault()
			if err := v.Set(key, value); err != nil {
				return fmt.Errorf("failed to store secret: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Secret for '%s' securely stored in the OS vault.\n", key)
			return nil
		},
	}

	setCmd.Flags().StringVar(&valueFlag, "value", "", "The secret value (discouraged, leaves traces in shell history)")

	return setCmd
}

// NewVaultGetCommand creates the vault get subcommand.
func NewVaultGetCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
	getCmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Retrieve a secret",
		Long:  "Retrieve a secret from the vault and print it to standard output.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]

			v := vault.NewOSVault()
			value, err := v.Get(key)
			if err != nil {
				return fmt.Errorf("failed to retrieve secret: %w", err)
			}

			// Print exactly the value so it can be piped or captured
			fmt.Fprint(cmd.OutOrStdout(), value)
			return nil
		},
	}

	return getCmd
}

// NewVaultRemoveCommand creates the vault remove subcommand.
func NewVaultRemoveCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
	removeCmd := &cobra.Command{
		Use:   "remove <key>",
		Short: "Remove a secret",
		Long:  "Delete a securely stored secret from the vault.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]

			v := vault.NewOSVault()
			if err := v.Delete(key); err != nil {
				return fmt.Errorf("failed to remove secret: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Secret for '%s' successfully removed from the vault.\n", key)
			return nil
		},
	}

	return removeCmd
}
