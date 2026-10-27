package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/lleitep3/aicockpit/internal/agents"
	"github.com/lleitep3/aicockpit/internal/ask"
	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/i18n"
	"github.com/lleitep3/aicockpit/internal/logging"
	"github.com/lleitep3/aicockpit/internal/memory"
	"github.com/lleitep3/aicockpit/internal/memoryhindsight"
	"github.com/spf13/cobra"
)

// openAskMemory opens the local knowledge-base memory and, when Hindsight is
// enabled, the optional Hindsight memory. A Hindsight error leaves the local
// memory usable. Tests replace it.
var openAskMemory = defaultOpenAskMemory

// newAskCatalog builds the worker catalog. Tests replace it.
var newAskCatalog = defaultAskCatalog

// NewAskCommand creates the cockpit ask command.
func NewAskCommand(log *logging.Manager, cfg *config.Config, _ *i18n.Translator) *cobra.Command {
	var (
		backend  string
		model    string
		workdir  string
		timeout  time.Duration
		noMemory bool
		jsonOut  bool
	)
	cmd := &cobra.Command{
		Use:   "ask <task>",
		Short: "Run one assistant turn",
		Long: `Run one assistant turn on a local CLI or a provider API.

The turn recalls relevant memory, sends the task with that context, streams the
answer, and retains a short record of the task and the result. It does not loop
on tools.

--backend selects the worker: claude, codex, api, or api:<provider>
(anthropic, openai, ollama, lmstudio, openrouter). Without --backend, cockpit
uses agents.backend from config.yaml. When that is also empty, it picks the
first available worker: claude, then codex, then the configured API provider
when its vault token is present.

--no-memory skips recall and retain. --json prints one result object instead
of streaming text.

Example configuration:

  agents:
    backend: claude
    api:
      provider: anthropic
      model: claude-sonnet-4-5`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAskCommand(cmd, log, cfg, ask.Request{
				Task:     args[0],
				Backend:  backend,
				Model:    model,
				WorkDir:  workdir,
				Timeout:  timeout,
				NoMemory: noMemory,
				JSON:     jsonOut,
			})
		},
	}
	cmd.Flags().StringVar(&backend, "backend", "", "Worker: claude, codex, api, or api:<provider>")
	cmd.Flags().StringVar(&model, "model", "", "Model override")
	cmd.Flags().StringVar(&workdir, "workdir", "", "Working directory for the worker")
	cmd.Flags().DurationVar(&timeout, "timeout", 0, "Worker time limit (for example 2m)")
	cmd.Flags().BoolVar(&noMemory, "no-memory", false, "Skip memory recall and retain")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Print the result as JSON")
	return cmd
}

func runAskCommand(cmd *cobra.Command, log *logging.Manager, cfg *config.Config, req ask.Request) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	var local, extra memory.Memory
	if !req.NoMemory {
		var err error
		local, extra, err = openAskMemory(cfg)
		if local == nil {
			if err == nil {
				err = errors.New("ask: memory is not configured")
			}
			return err
		}
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "warning: hindsight memory skipped: %v\n", err)
		}
	}
	configBackend := ""
	if cfg != nil {
		configBackend = cfg.Agents.Backend
	}
	outcome, err := ask.Run(ctx, req, ask.Deps{
		Memory:        local,
		Extra:         extra,
		Catalog:       newAskCatalog(cfg),
		Out:           cmd.OutOrStdout(),
		ErrOut:        cmd.ErrOrStderr(),
		ConfigBackend: configBackend,
	})
	if log != nil && outcome != nil && outcome.Backend != "" {
		_ = log.LogInfo("ask "+outcome.Backend, nil)
	}
	return err
}

func defaultAskCatalog(cfg *config.Config) ask.Catalog {
	var settings agents.APISettings
	if cfg != nil {
		settings = cfg.Agents.API
	}
	source := agents.NewVaultTokenSource()
	return ask.NewLiveCatalog(ask.LiveOptions{
		API:    settings,
		Tokens: source,
		Probe:  source,
	})
}

func defaultOpenAskMemory(cfg *config.Config) (memory.Memory, memory.Memory, error) {
	if cfg == nil {
		return nil, nil, errors.New("ask: nil config")
	}
	roots := cfg.KB.Roots
	if len(roots) == 0 {
		roots = []string{filepath.Join(config.GetCockpitDir(), "kb")}
	}
	if err := os.MkdirAll(roots[0], 0o755); err != nil {
		return nil, nil, fmt.Errorf("ask: memory root: %w", err)
	}
	local, err := memory.NewKB(roots, filepath.Join(config.GetCockpitDir(), ".kb-index.json"))
	if err != nil {
		return nil, nil, fmt.Errorf("ask: memory: %w", err)
	}
	extra, err := openHindsightAskMemory(cfg)
	if err != nil {
		return local, nil, err
	}
	return local, extra, nil
}

func openHindsightAskMemory(cfg *config.Config) (memory.Memory, error) {
	if cfg == nil || !cfg.Memory.Hindsight.Enabled {
		return nil, nil
	}
	token, err := hindsightAPIToken(cfg.Memory.Hindsight)
	if err != nil {
		return nil, err
	}
	contributor, err := newHindsightIntegration(cfg.Memory.Hindsight, memoryhindsight.CurrentIdentity(), token)
	if err != nil {
		return nil, err
	}
	integration, ok := contributor.(*memoryhindsight.Integration)
	if !ok || integration == nil || !integration.Enabled() {
		return nil, nil
	}
	return integration.Memory(), nil
}
