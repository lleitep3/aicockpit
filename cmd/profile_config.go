package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/profileconfig"
	"github.com/lleitep3/aicockpit/internal/vault"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func packageConfigStore() profileconfig.Store {
	return profileconfig.Store{Dir: filepath.Join(config.GetCockpitDir(), "package-config")}
}

// Package context is supplied by the dispatcher, not a security identity against
// hostile code running as the same OS user. See docs/PACKAGE_CONFIG.md.
func packageActor() string {
	if actor := os.Getenv("COCKPIT_PACKAGE_CONTEXT"); actor != "" {
		return actor
	}
	return profileconfig.Operator
}

var profileVaultAccess = func(actor string) error {
	lm := vault.NewLockManager("")
	if lm.InitializationError() != nil || !lm.CanPackageAccess(actor) {
		return errors.New("vault access denied; unlock the vault for the calling package")
	}
	return nil
}

func checkNamespaceAccess(namespace string, write bool) error {
	actor := packageActor()
	if namespace == "" {
		if actor != profileconfig.Operator {
			return errors.New("packages must use their own --namespace")
		}
		return checkVaultAccess("legacy")
	}
	if err := packageConfigStore().Check(actor, namespace, write); err != nil {
		return err
	}
	return profileVaultAccess(actor)
}

func NewConfigCommand() *cobra.Command {
	var namespace, profile string
	root := &cobra.Command{Use: "config", Short: "Manage package profiles, public settings and vault references"}
	root.PersistentFlags().StringVar(&namespace, "namespace", "", "Owner package (defaults to the calling package)")
	root.PersistentFlags().StringVar(&profile, "profile", "", "Explicit account/environment profile")
	resolve := func(needProfile bool) (string, error) {
		ns := namespace
		if ns == "" && packageActor() != profileconfig.Operator {
			ns = packageActor()
		}
		if !profileconfig.ValidName(ns) {
			return "", errors.New("choose a package with --namespace")
		}
		if needProfile && !profileconfig.ValidName(profile) {
			return "", errors.New("choose a profile with --profile")
		}
		return ns, nil
	}
	set := &cobra.Command{Use: "set <field> <public-value>", Short: "Store a non-secret setting; use secret for credentials", Args: cobra.ExactArgs(2), RunE: func(c *cobra.Command, args []string) error {
		ns, err := resolve(true)
		if err != nil {
			return err
		}
		return packageConfigStore().Set(packageActor(), ns, profile, args[0], profileconfig.Field{Value: &args[1]})
	}}
	var stdin bool
	secret := &cobra.Command{Use: "secret <field>", Short: "Store a secret in the vault using hidden input or --stdin", Args: cobra.ExactArgs(1), RunE: func(c *cobra.Command, args []string) error {
		ns, err := resolve(true)
		if err != nil {
			return err
		}
		if !profileconfig.ValidName(args[0]) {
			return errors.New("invalid field name")
		}
		if err := checkNamespaceAccess(ns, true); err != nil {
			return err
		}
		value, err := readProfileSecret(c, stdin)
		if err != nil {
			return err
		}
		key := profileconfig.SecretKey(profile, args[0])
		// Persist the reference first: a failed keyring write leaves an explicit
		// unconfigured reference, never a plaintext fallback or an orphan secret.
		if err := packageConfigStore().Set(packageActor(), ns, profile, args[0], profileconfig.Field{SecretKey: key}); err != nil {
			return err
		}
		if err := profileVaultAccess(packageActor()); err != nil {
			return err
		}
		if err := vault.NewNamespacedVault(ns).Set(key, value); err != nil {
			return errors.New("vault write failed; reference retained, retry secret input after resolving vault access")
		}
		return nil
	}}
	secret.Flags().BoolVar(&stdin, "stdin", false, "Read the secret from stdin (never pass credentials in arguments)")
	show := &cobra.Command{Use: "show", Short: "Show public values and secret references as JSON; never resolves secrets", Args: cobra.NoArgs, RunE: func(c *cobra.Command, args []string) error {
		ns, err := resolve(true)
		if err != nil {
			return err
		}
		fields, err := packageConfigStore().Profile(packageActor(), ns, profile)
		if err != nil {
			return err
		}
		return json.NewEncoder(c.OutOrStdout()).Encode(fields)
	}}
	list := &cobra.Command{Use: "list", Short: "List profiles and namespace readers as JSON", Args: cobra.NoArgs, RunE: func(c *cobra.Command, args []string) error {
		ns, err := resolve(false)
		if err != nil {
			return err
		}
		doc, err := packageConfigStore().Read(packageActor(), ns)
		if err != nil {
			return err
		}
		names := make([]string, 0, len(doc.Profiles))
		for name := range doc.Profiles {
			names = append(names, name)
		}
		sort.Strings(names)
		return json.NewEncoder(c.OutOrStdout()).Encode(map[string]any{"profiles": names, "readers": doc.Readers})
	}}
	root.AddCommand(set, secret, show, list)
	for _, action := range []string{"grant", "revoke"} {
		root.AddCommand(&cobra.Command{Use: action + " <reader-package>", Short: action + " read access to all profiles and secrets in the namespace (operator only)", Args: cobra.ExactArgs(1), RunE: func(c *cobra.Command, args []string) error {
			ns, err := resolve(false)
			if err != nil {
				return err
			}
			if packageActor() != profileconfig.Operator {
				return errors.New("only the operator may change grants")
			}
			if err := profileVaultAccess(profileconfig.Operator); err != nil {
				return err
			}
			return packageConfigStore().Grant(packageActor(), ns, args[0], action == "grant")
		}})
	}
	var bindings []string
	run := &cobra.Command{Use: "exec --env ENV=field -- <command> [args...]", Short: "Inject selected fields into a trusted child process only", Args: cobra.MinimumNArgs(1), RunE: func(c *cobra.Command, args []string) error {
		ns, err := resolve(true)
		if err != nil {
			return err
		}
		if c.ArgsLenAtDash() != 0 || len(bindings) == 0 {
			return errors.New("provide --env bindings and separate the command with --")
		}
		fields, err := packageConfigStore().Profile(packageActor(), ns, profile)
		if err != nil {
			return err
		}
		env, err := profileconfig.Environment(fields, bindings, os.Environ(), func(key string) (string, error) {
			if err := checkNamespaceAccess(ns, false); err != nil {
				return "", err
			}
			return vault.NewNamespacedVault(ns).Get(key)
		})
		if err != nil {
			return err
		}
		child := exec.CommandContext(c.Context(), args[0], args[1:]...)
		child.Env = env
		child.Stdin = c.InOrStdin()
		child.Stdout = c.OutOrStdout()
		child.Stderr = c.ErrOrStderr()
		if err := child.Run(); err != nil {
			return errors.New("child command failed; inspect its exit/output without exposing credentials")
		}
		return nil
	}}
	run.Flags().StringArrayVar(&bindings, "env", nil, "Explicit ENV_NAME=profile_field binding (repeatable)")
	root.AddCommand(run)
	return root
}

func readProfileSecret(c *cobra.Command, stdin bool) (string, error) {
	var data []byte
	var err error
	if stdin {
		data, err = io.ReadAll(io.LimitReader(c.InOrStdin(), 65537))
	} else {
		fmt.Fprint(c.ErrOrStderr(), "Secret (hidden): ")
		data, err = term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(c.ErrOrStderr())
	}
	if err != nil {
		return "", errors.New("cannot read secret; use an interactive terminal or --stdin")
	}
	value := strings.TrimSuffix(strings.TrimSuffix(string(data), "\n"), "\r")
	if value == "" || len(data) > 65536 || strings.ContainsRune(value, 0) {
		return "", errors.New("secret must be nonempty, at most 64 KiB and contain no NUL")
	}
	return value, nil
}
