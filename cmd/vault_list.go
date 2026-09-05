package cmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/i18n"
	"github.com/lleitep3/aicockpit/internal/logging"
	"github.com/lleitep3/aicockpit/internal/vault"
	"github.com/spf13/cobra"
)

// NewVaultListCommand creates the vault list subcommand.
func NewVaultListCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
	var namespaceFlag string

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List stored secrets",
		Long:  "List all secrets in the vault with their creation and last update timestamps.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Namespace mode bypasses lock checks
			if namespaceFlag == "" {
				if err := checkVaultAccess("list"); err != nil {
					return err
				}
			}

			var v vault.Manager
			if namespaceFlag != "" {
				v = vault.NewNamespacedVault(namespaceFlag)
			} else {
				v = vault.NewOSVault()
			}

			secrets, err := v.List()
			if err != nil {
				return fmt.Errorf("failed to list secrets: %w", err)
			}

			if len(secrets) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No secrets found.")
				return nil
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "KEY\tCREATED\tUPDATED")
			for _, s := range secrets {
				fmt.Fprintf(w, "%s\t%s\t%s\n", s.Key, s.Created, s.Updated)
			}
			if err := w.Flush(); err != nil {
				return fmt.Errorf("failed to write list output: %w", err)
			}
			return nil
		},
	}

	listCmd.Flags().StringVar(&namespaceFlag, "namespace", "", "Vault namespace for isolation")

	return listCmd
}
