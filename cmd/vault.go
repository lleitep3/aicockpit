package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/i18n"
	"github.com/lleitep3/aicockpit/internal/logging"
	"github.com/lleitep3/aicockpit/internal/vault"
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
	vaultCmd.AddCommand(NewVaultLockCommand(log, cfg, t))
	vaultCmd.AddCommand(NewVaultUnlockCommand(log, cfg, t))
	vaultCmd.AddCommand(NewVaultStatusCommand(log, cfg, t))
	vaultCmd.AddCommand(NewVaultSetMasterPasswordCommand(log, cfg, t))
	vaultCmd.AddCommand(NewVaultChangeMasterPasswordCommand(log, cfg, t))
	vaultCmd.AddCommand(NewVaultDisableMasterPasswordCommand(log, cfg, t))
	vaultCmd.AddCommand(NewVaultFactoryResetCommand(log, cfg, t))

	return vaultCmd
}

// checkVaultAccess checks if vault access is allowed based on lock state
func checkVaultAccess(operation string) error {
	lm := vault.NewLockManager("")

	// Check if current process (package) has access
	currentPackage := getCurrentProcessName()

	if !lm.CanPackageAccess(currentPackage) {
		status := lm.GetStatus()

		fmt.Printf("🔒 Vault is locked. Access denied for '%s'.\n", currentPackage)
		fmt.Println()
		fmt.Println("To unlock:")
		if status.GlobalUnlock {
			fmt.Println("  Vault is already unlocked globally")
		} else {
			fmt.Println("  cockpit vault unlock              # Unlock globally")
			fmt.Printf("  cockpit vault unlock %s    # Unlock for this package\n", currentPackage)
		}
		fmt.Println()
		fmt.Println("For status:")
		fmt.Println("  cockpit vault status")

		return fmt.Errorf("vault is locked")
	}

	return nil
}

// getCurrentProcessName returns the current process/package name
func getCurrentProcessName() string {
	exePath, err := os.Executable()
	if err != nil {
		return "unknown"
	}

	// Extract package name from executable path
	if filepath.Base(exePath) == "cockpit" {
		// CLI itself has access
		return "cockpit-cli"
	}

	// Try to extract from .cockpit/packages/ path
	if strings.Contains(exePath, "/.cockpit/packages/") {
		parts := strings.Split(exePath, "/")
		for i, part := range parts {
			if part == ".cockpit" && i+2 < len(parts) && parts[i+1] == "packages" {
				return parts[i+2]
			}
		}
	}

	// Fallback to executable name
	exeName := filepath.Base(exePath)
	exeName = strings.TrimSuffix(exeName, filepath.Ext(exeName))
	return exeName
}

// NewVaultSetCommand creates the vault set subcommand.
func NewVaultSetCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
	var valueFlag string
	var namespaceFlag string

	setCmd := &cobra.Command{
		Use:   "set <key>",
		Short: "Store a secret securely",
		Long:  "Store a secret securely. If --value is not provided, you will be prompted to enter it securely.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Only check vault access if NOT using namespace (namespace provides isolation)
			if namespaceFlag == "" {
				if err := checkVaultAccess("set"); err != nil {
					return err
				}
			}

			key := args[0]
			value := valueFlag
			namespace := namespaceFlag

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

			// Use NamespacedVault if namespace is specified, otherwise use OSVault for backward compatibility
			var v vault.Manager
			if namespace != "" {
				v = vault.NewNamespacedVault(namespace)
			} else {
				// DEPRECATED: Direct OSVault access for backward compatibility
				v = vault.NewOSVault()
			}

			if err := v.Set(key, value); err != nil {
				return fmt.Errorf("failed to store secret: %w", err)
			}

			namespaceMsg := ""
			if namespace != "" {
				namespaceMsg = fmt.Sprintf(" in namespace '%s'", namespace)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Secret for '%s'%s securely stored in the OS vault.\n", key, namespaceMsg)
			return nil
		},
	}

	setCmd.Flags().StringVar(&valueFlag, "value", "", "The secret value (discouraged, leaves traces in shell history)")
	setCmd.Flags().StringVar(&namespaceFlag, "namespace", "", "Vault namespace for isolation (recommended for security)")

	return setCmd
}

// NewVaultGetCommand creates the vault get subcommand.
func NewVaultGetCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
	var namespaceFlag string

	getCmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Retrieve a secret",
		Long:  "Retrieve a secret from the vault and print it to standard output.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Only check vault access if NOT using namespace (namespace provides isolation)
			if namespaceFlag == "" {
				if err := checkVaultAccess("get"); err != nil {
					return err
				}
			}

			key := args[0]
			namespace := namespaceFlag

			// Use NamespacedVault if namespace is specified, otherwise use OSVault for backward compatibility
			var v vault.Manager
			if namespace != "" {
				v = vault.NewNamespacedVault(namespace)
			} else {
				// DEPRECATED: Direct OSVault access for backward compatibility
				v = vault.NewOSVault()
			}

			value, err := v.Get(key)
			if err != nil {
				return fmt.Errorf("failed to retrieve secret: %w", err)
			}

			// Print exactly the value so it can be piped or captured
			fmt.Fprint(cmd.OutOrStdout(), value)
			return nil
		},
	}

	getCmd.Flags().StringVar(&namespaceFlag, "namespace", "", "Vault namespace for isolation (recommended for security)")

	return getCmd
}

// NewVaultRemoveCommand creates the vault remove subcommand.
func NewVaultRemoveCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
	var namespaceFlag string

	removeCmd := &cobra.Command{
		Use:   "remove <key>",
		Short: "Remove a secret",
		Long:  "Delete a securely stored secret from the vault.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Only check vault access if NOT using namespace (namespace provides isolation)
			if namespaceFlag == "" {
				if err := checkVaultAccess("remove"); err != nil {
					return err
				}
			}

			key := args[0]
			namespace := namespaceFlag

			// Use NamespacedVault if namespace is specified, otherwise use OSVault for backward compatibility
			var v vault.Manager
			if namespace != "" {
				v = vault.NewNamespacedVault(namespace)
			} else {
				// DEPRECATED: Direct OSVault access for backward compatibility
				v = vault.NewOSVault()
			}

			if err := v.Delete(key); err != nil {
				return fmt.Errorf("failed to remove secret: %w", err)
			}

			namespaceMsg := ""
			if namespace != "" {
				namespaceMsg = fmt.Sprintf(" from namespace '%s'", namespace)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Secret for '%s'%s successfully removed from the vault.\n", key, namespaceMsg)
			return nil
		},
	}

	removeCmd.Flags().StringVar(&namespaceFlag, "namespace", "", "Vault namespace for isolation (recommended for security)")

	return removeCmd
}
