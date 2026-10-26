package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/hindsight"
	"github.com/lleitep3/aicockpit/internal/i18n"
	"github.com/lleitep3/aicockpit/internal/logging"
	"github.com/lleitep3/aicockpit/internal/vault"
	"github.com/spf13/cobra"
)

// memoryService is the lifecycle used by the memory command and doctor.
type memoryService interface {
	Up(ctx context.Context) (hindsight.Report, error)
	Down(ctx context.Context) (hindsight.Report, error)
	Status(ctx context.Context) (hindsight.Report, error)
}

var newMemoryManager = func(cfg *config.Config) memoryService {
	return hindsight.NewManager(cfg.Memory.Hindsight, hindsight.NewDockerCLI(), hindsight.HTTPProber{}, vaultHindsightSecrets{})
}

type vaultGetter interface {
	Get(key string) (string, error)
}

var authorizeHindsightVault = func(namespace string) error {
	return checkNamespaceAccess(namespace, false)
}

var openHindsightVault = func(namespace string) vaultGetter {
	return vault.NewNamespacedVault(namespace)
}

// vaultHindsightSecrets reads Hindsight secrets from the cockpit vault.
// Values are returned to the caller and are never logged.
type vaultHindsightSecrets struct{}

func (vaultHindsightSecrets) Get(ctx context.Context, namespace, key string) (string, error) {
	if ctx != nil && ctx.Err() != nil {
		return "", ctx.Err()
	}
	if err := authorizeHindsightVault(namespace); err != nil {
		return "", err
	}
	value, err := openHindsightVault(namespace).Get(key)
	if err != nil {
		return "", fmt.Errorf("read vault key %s/%s: %w", namespace, key, err)
	}
	if strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("vault key %s/%s is empty", namespace, key)
	}
	if strings.ContainsAny(value, "\r\n\x00") {
		return "", fmt.Errorf("vault key %s/%s contains a newline", namespace, key)
	}
	return value, nil
}

// NewMemoryCommand creates cockpit memory up, down, and status.
func NewMemoryCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "memory",
		Short: "Manage the optional local Hindsight memory server",
		Long: `Manage an optional local Hindsight server (Docker image ghcr.io/vectorize-io/hindsight).

The API listens on port 8888 and the control plane on port 9999. Data is stored
with embedded pg0 in a named volume. LLM API keys are read from the cockpit vault
and passed only as container environment variables.

Enable it in ~/.cockpit/config.yaml:

  memory:
    hindsight:
      enabled: true
      llm_provider: openai

When it is enabled, kb search also recalls one project bank and one user bank,
and kb index retains documents in the project bank. Names default to
project-<name> and user-<name>, or set memory.hindsight.banks. If the server
is unreachable, search and index keep the local knowledge base.

Store the key without putting it in the shell history:

  cockpit vault set llm_api_key --namespace hindsight --stdin`,
	}
	cmd.PersistentFlags().BoolVar(&jsonOut, "json", false, "Print machine-readable JSON")
	cmd.AddCommand(
		newMemoryAction(log, "up", "Start the local Hindsight server", func(ctx context.Context) (hindsight.Report, error) {
			return newMemoryManager(cfg).Up(ctx)
		}, &jsonOut),
		newMemoryAction(log, "down", "Stop the local Hindsight server and keep its data volume", func(ctx context.Context) (hindsight.Report, error) {
			return newMemoryManager(cfg).Down(ctx)
		}, &jsonOut),
		newMemoryAction(log, "status", "Show whether the local Hindsight server is running and healthy", func(ctx context.Context) (hindsight.Report, error) {
			return newMemoryManager(cfg).Status(ctx)
		}, &jsonOut),
	)
	return cmd
}

func newMemoryAction(log *logging.Manager, use, short string, run func(context.Context) (hindsight.Report, error), jsonOut *bool) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			report, err := run(ctx)
			if err != nil {
				return err
			}
			if log != nil {
				_ = log.LogInfo("hindsight "+use+": "+report.State+" "+report.Health, nil)
			}
			return printMemoryReport(cmd, report, jsonOut != nil && *jsonOut)
		},
	}
}

func printMemoryReport(cmd *cobra.Command, report hindsight.Report, asJSON bool) error {
	if asJSON {
		encoded, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return fmt.Errorf("encode memory status: %w", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(encoded))
		return nil
	}
	fmt.Fprint(cmd.OutOrStdout(), report.FormatText())
	return nil
}
