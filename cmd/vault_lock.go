package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/i18n"
	"github.com/lleitep3/aicockpit/internal/logging"
	"github.com/lleitep3/aicockpit/internal/vault"
	"github.com/spf13/cobra"
)

func NewVaultLockCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
	var reasonFlag string

	lockCmd := &cobra.Command{
		Use:   "lock [package]",
		Short: "Lock the vault (globally or for specific package)",
		Long:  "Lock the vault to prevent secret access. Without arguments, locks globally. With package name, locks only for that package.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mp := vault.NewMasterPassword()

			// Check for dev mode flag to skip master password
			devMode := os.Getenv("COCKPIT_DEV_MODE") == "true"

			// If master password not set and not in dev mode, require setting it first
			if !mp.IsEnabled() && !devMode {
				fmt.Println("⚠️  Master password is not set. For security, you must set it first.")
				fmt.Println("  Run: cockpit vault set-master-password")
				fmt.Println("  Or run: COCKPIT_DEV_MODE=true cockpit vault lock (not recommended)")
				return fmt.Errorf("master password not set")
			}

			// Require master password if enabled and not in dev mode
			if mp.IsEnabled() && !devMode {
				if err := mp.PromptAndValidate(); err != nil {
					return err
				}
			}

			lm := vault.NewLockManager("")

			if reasonFlag == "" {
				if len(args) > 0 {
					reasonFlag = fmt.Sprintf("Manual lock for package: %s", args[0])
				} else {
					reasonFlag = "Manual lock by user"
				}
			}

			if len(args) > 0 {
				// Lock specific package
				packageName := args[0]
				err := lm.LockPackage(packageName)
				if err != nil {
					return fmt.Errorf("failed to lock package: %w", err)
				}

				fmt.Printf("✓ Package '%s' locked successfully\n", packageName)
				fmt.Println("  This package can no longer access vault secrets")
			} else {
				// Lock globally
				err := lm.Lock(reasonFlag)
				if err != nil {
					return fmt.Errorf("failed to lock vault: %w", err)
				}

				fmt.Println("✓ Vault locked successfully")
				fmt.Println("  Use 'cockpit vault unlock' to access secrets")
			}

			return nil
		},
	}

	lockCmd.Flags().StringVar(&reasonFlag, "reason", "", "Reason for locking")

	return lockCmd
}

func NewVaultUnlockCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
	var reasonFlag string
	var timeoutFlag string

	unlockCmd := &cobra.Command{
		Use:   "unlock [package]",
		Short: "Unlock the vault (globally or for specific package)",
		Long:  "Unlock the vault to allow secret access. Without arguments, unlocks globally. With package name, unlocks only for that package.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mp := vault.NewMasterPassword()

			// Check for dev mode flag to skip master password
			devMode := os.Getenv("COCKPIT_DEV_MODE") == "true"

			// If master password not set and not in dev mode, require setting it first
			if !mp.IsEnabled() && !devMode {
				fmt.Println("⚠️  Master password is not set. For security, you must set it first.")
				fmt.Println("  Run: cockpit vault set-master-password")
				fmt.Println("  Or run: COCKPIT_DEV_MODE=true cockpit vault lock (not recommended)")
				return fmt.Errorf("master password not set")
			}

			// Require master password if enabled and not in dev mode
			if mp.IsEnabled() && !devMode {
				if err := mp.PromptAndValidate(); err != nil {
					return err
				}
			}

			lm := vault.NewLockManager("")

			if reasonFlag == "" {
				if len(args) > 0 {
					reasonFlag = fmt.Sprintf("Manual unlock for package: %s", args[0])
				} else {
					reasonFlag = "Manual unlock by user"
				}
			}

			if len(args) > 0 {
				// Unlock for specific package
				packageName := args[0]
				err := lm.UnlockPackage(packageName, reasonFlag)
				if err != nil {
					return fmt.Errorf("failed to unlock package: %w", err)
				}

				fmt.Printf("✓ Package '%s' unlocked successfully\n", packageName)
				fmt.Println("  Only this package can access vault secrets")
			} else {
				// Unlock globally
				err := lm.Unlock(reasonFlag)
				if err != nil {
					return fmt.Errorf("failed to unlock vault: %w", err)
				}

				fmt.Println("✓ Vault unlocked successfully")

				// Handle auto-lock timeout
				if timeoutFlag != "" {
					duration, err := time.ParseDuration(timeoutFlag)
					if err != nil {
						return fmt.Errorf("invalid timeout format: %w", err)
					}

					fmt.Printf("  Auto-lock in: %v\n", duration)
					lm.SetAutoLockTimeout(duration)
				}
			}

			return nil
		},
	}

	unlockCmd.Flags().StringVar(&reasonFlag, "reason", "", "Reason for unlocking")
	unlockCmd.Flags().StringVar(&timeoutFlag, "timeout", "", "Auto-lock after duration (e.g., '1h', '30m')")

	return unlockCmd
}

func NewVaultStatusCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show vault lock status",
		Long:  "Show current vault lock status, including which packages have access and lock history.",
		RunE: func(cmd *cobra.Command, args []string) error {
			lm := vault.NewLockManager("")
			status := lm.GetStatus()

			fmt.Println("=== Vault Lock Status ===")
			fmt.Println()

			// Overall status
			if status.IsLocked {
				fmt.Printf("Status: 🔒 LOCKED\n")
				if !status.LockedAt.IsZero() {
					fmt.Printf("Locked at: %s\n", status.LockedAt.Format("2006-01-02 15:04:05"))
				}
				if status.LockedBy != "" {
					fmt.Printf("Locked by: %s\n", status.LockedBy)
				}
				if status.UnlockReason != "" {
					fmt.Printf("Reason: %s\n", status.UnlockReason)
				}
			} else {
				fmt.Println("Status: 🔓 UNLOCKED")
				if status.UnlockReason != "" {
					fmt.Printf("Reason: %s\n", status.UnlockReason)
				}
			}

			fmt.Println()

			// Global unlock status
			if status.GlobalUnlock {
				fmt.Println("Global Access: 🔓 All packages can access vault")
			} else {
				fmt.Println("Global Access: 🔒 Vault is locked")
			}

			fmt.Println()

			// Package-specific locks
			if len(status.PackageLocks) > 0 {
				fmt.Println("Package Access:")
				fmt.Println("================")

				for pkg, isUnlocked := range status.PackageLocks {
					if isUnlocked {
						fmt.Printf("  ✓ %s: 🔓 Unlocked\n", pkg)
					} else {
						fmt.Printf("  ✗ %s: 🔒 Locked\n", pkg)
					}
				}
			} else {
				fmt.Println("Package Access: No package-specific locks")
			}

			fmt.Println()

			// Summary
			if status.IsLocked {
				fmt.Println("Summary: Vault is locked. Use 'cockpit vault unlock' to access secrets.")
				if len(status.UnlockedPackages) > 0 {
					fmt.Printf("        Only %d packages have access: %v\n",
						len(status.UnlockedPackages), status.UnlockedPackages)
				}
			} else {
				fmt.Println("Summary: Vault is unlocked. All packages can access secrets.")
			}

			return nil
		},
	}

	return statusCmd
}

func NewVaultSetMasterPasswordCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
	setMasterCmd := &cobra.Command{
		Use:   "set-master-password",
		Short: "Set master password for vault operations",
		Long:  "Set a master password required for lock/unlock operations. This provides additional security.",
		RunE: func(cmd *cobra.Command, args []string) error {
			mp := vault.NewMasterPassword()

			// Prompt for new password
			fmt.Print("Enter new master password: ")
			password1, err := vault.PromptPassword()
			if err != nil {
				return fmt.Errorf("failed to read password: %w", err)
			}

			if len(password1) < 8 {
				return fmt.Errorf("password must be at least 8 characters")
			}

			// Confirm password
			fmt.Print("Confirm master password: ")
			password2, err := vault.PromptPassword()
			if err != nil {
				return fmt.Errorf("failed to read password: %w", err)
			}

			if password1 != password2 {
				return fmt.Errorf("passwords do not match")
			}

			// Set password
			err = mp.SetPassword(password1)
			if err != nil {
				return fmt.Errorf("failed to set master password: %w", err)
			}

			fmt.Println("✓ Master password set successfully")
			fmt.Println("  You will be prompted for this password for lock/unlock operations")

			return nil
		},
	}

	return setMasterCmd
}

func NewVaultDisableMasterPasswordCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
	disableMasterCmd := &cobra.Command{
		Use:   "disable-master-password",
		Short: "Disable master password protection",
		Long:  "Disable master password protection. This reduces security (not recommended in production).",
		RunE: func(cmd *cobra.Command, args []string) error {
			mp := vault.NewMasterPassword()

			if !mp.IsEnabled() {
				fmt.Println("Master password is already disabled")
				return nil
			}

			fmt.Print("WARNING: Disabling master password reduces security. Continue? (type 'DISABLE'): ")
			var confirmation string
			fmt.Scanln(&confirmation)
			if confirmation != "DISABLE" {
				return fmt.Errorf("operation cancelled")
			}

			err := mp.Disable()
			if err != nil {
				return fmt.Errorf("failed to disable master password: %w", err)
			}

			fmt.Println("✓ Master password disabled")
			fmt.Println("  Vault operations no longer require password")

			return nil
		},
	}

	return disableMasterCmd
}

func NewVaultChangeMasterPasswordCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
	changeMasterCmd := &cobra.Command{
		Use:   "change-master-password",
		Short: "Change the master password",
		Long:  "Change the master password. Requires entering the current password first.",
		RunE: func(cmd *cobra.Command, args []string) error {
			mp := vault.NewMasterPassword()

			if !mp.IsEnabled() {
				return fmt.Errorf("master password is not set. Use 'cockpit vault set-master-password' first")
			}

			// Prompt for old password
			fmt.Print("Enter current master password: ")
			oldPassword, err := vault.PromptPassword()
			if err != nil {
				return fmt.Errorf("failed to read password: %w", err)
			}

			// Validate old password
			if !mp.Validate(oldPassword) {
				return fmt.Errorf("invalid current password")
			}

			// Prompt for new password
			fmt.Print("Enter new master password: ")
			newPassword1, err := vault.PromptPassword()
			if err != nil {
				return fmt.Errorf("failed to read password: %w", err)
			}

			if len(newPassword1) < 8 {
				return fmt.Errorf("password must be at least 8 characters")
			}

			// Confirm new password
			fmt.Print("Confirm new master password: ")
			newPassword2, err := vault.PromptPassword()
			if err != nil {
				return fmt.Errorf("failed to read password: %w", err)
			}

			if newPassword1 != newPassword2 {
				return fmt.Errorf("new passwords do not match")
			}

			// Change password
			err = mp.ChangePassword(oldPassword, newPassword1)
			if err != nil {
				return fmt.Errorf("failed to change password: %w", err)
			}

			fmt.Println("✓ Master password changed successfully")

			return nil
		},
	}

	return changeMasterCmd
}

func NewVaultFactoryResetCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
	resetCmd := &cobra.Command{
		Use:   "factory-reset",
		Short: "Factory reset - delete all secrets and reset vault",
		Long:  "Delete all secrets from the vault and reset all configurations. Use this if you forgot your master password and want to start fresh.",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("⚠️  WARNING: This will delete ALL secrets and reset vault configuration!")
			fmt.Println("  This action cannot be undone.")
			fmt.Print("Type 'FACTORY-RESET' to confirm: ")
			var confirmation string
			fmt.Scanln(&confirmation)

			if confirmation != "FACTORY-RESET" {
				return fmt.Errorf("factory reset cancelled")
			}

			// Clear all secrets
			v := vault.NewOSVault()
			err := v.ClearAllSecrets()
			if err != nil {
				return fmt.Errorf("failed to clear secrets: %w", err)
			}

			// Reset lock manager state
			lm := vault.NewLockManager("")
			lm.Lock("Factory reset")

			// Disable master password
			mp := vault.NewMasterPassword()
			mp.Disable()

			// Remove lock state file
			os.Remove("/home/lleite/.cockpit/vault/lock_state.json")
			os.Remove("/home/lleite/.cockpit/vault/master_password.dat")
			os.Remove("/home/lleite/.cockpit/vault/permissions.json")

			fmt.Println("✓ Factory reset complete")
			fmt.Println("  All secrets deleted")
			fmt.Println("  Vault configuration reset")
			fmt.Println("  You can now set a new master password with: cockpit vault set-master-password")

			return nil
		},
	}

	return resetCmd
}
