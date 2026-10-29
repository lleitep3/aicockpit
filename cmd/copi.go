package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/copi"
	copitui "github.com/lleitep3/aicockpit/internal/copi/tui"
	"github.com/lleitep3/aicockpit/internal/personalagent"
	"github.com/spf13/cobra"
)

type personalAgentAdapter struct {
	svc *personalagent.Service
}

func (a *personalAgentAdapter) List(ctx context.Context, includeArchived bool) ([]personalagent.ConversationSummary, error) {
	return a.svc.ConversationSummaries()
}

func (a *personalAgentAdapter) Get(ctx context.Context, id string) (*personalagent.Conversation, error) {
	conv, err := a.svc.Conversation(id)
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

func (a *personalAgentAdapter) SendMessage(ctx context.Context, convID string, message string) error {
	_, err := a.svc.StartTurn(convID, message, nil)
	return err
}

func (a *personalAgentAdapter) DecideAction(ctx context.Context, convID string, actionID string, approved bool) error {
	decision := "approved"
	if !approved {
		decision = "rejected"
	}
	return a.svc.DecideAssistantAction(convID, actionID, decision)
}

// NewCopiCommand creates the 'cockpit copi' CLI command.
func NewCopiCommand() *cobra.Command {
	var sandboxDir string
	var daemonMode bool

	command := &cobra.Command{
		Use:   "copi",
		Short: "Inicia o bot assistente e orquestrador Copi",
		Long: `Copi é o bot mascote e orquestrador proativo do Cockpit.
Ele monitora sessões ativas sem consumir tokens desnecessários,
avisa sobre ações pendentes de aprovação e despacha instruções.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if sandboxDir == "" {
				sandboxDir = config.GetCockpitDir()
			}

			store, err := personalagent.NewStore(filepath.Join(sandboxDir, "conversations"))
			if err != nil {
				return fmt.Errorf("init conversation store: %w", err)
			}

			paSvc, err := personalagent.NewService(store, personalagent.NewProviderRegistry(), sandboxDir)
			if err != nil {
				return fmt.Errorf("init personal agent service: %w", err)
			}

			adapter := &personalAgentAdapter{svc: paSvc}
			cfg := copi.DefaultConfig(sandboxDir)

			engine, err := copi.NewEngine(cfg, adapter)
			if err != nil {
				return fmt.Errorf("init copi engine: %w", err)
			}

			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			if daemonMode {
				fmt.Fprintf(cmd.OutOrStdout(), "🤖 Copi rodando em modo daemon headless (Sandbox: %s)\n", sandboxDir)
				engine.StartAutoScan(ctx)
				<-ctx.Done()
				engine.Stop()
				return nil
			}

			ui := copitui.NewUI(engine, cmd.InOrStdin(), cmd.OutOrStdout())
			return ui.Run(ctx)
		},
	}

	command.Flags().StringVar(&sandboxDir, "sandbox", "", "Diretório da sandbox de atuação do Copi (default: ~/.cockpit)")
	command.Flags().BoolVar(&daemonMode, "daemon", false, "Executa o Copi como daemon contínuo em background (modo servidor)")

	return command
}
