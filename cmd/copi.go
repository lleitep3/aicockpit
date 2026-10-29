package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

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
	summaries, err := a.svc.ConversationSummaries()
	if err != nil {
		return nil, err
	}
	if includeArchived {
		return summaries, nil
	}
	filtered := make([]personalagent.ConversationSummary, 0, len(summaries))
	for _, s := range summaries {
		if !s.Archived {
			filtered = append(filtered, s)
		}
	}
	return filtered, nil
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

func (a *personalAgentAdapter) ArchiveSession(ctx context.Context, convID string) error {
	// First mark as archived in service
	_, _ = a.svc.ArchiveConversation(convID, true)

	// If empty/idle, remove the file completely from disk so it doesn't pollute
	convFile := filepath.Join(config.GetCockpitDir(), "conversations", convID+".json")
	if err := os.Remove(convFile); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (a *personalAgentAdapter) Ask(ctx context.Context, prompt string) (string, error) {
	// Look up available providers for the assistant
	providers := a.svc.AssistantProviders()
	if len(providers) == 0 {
		return "", fmt.Errorf("nenhum provider de IA disponível no momento")
	}

	// Create or reuse an assistant session
	assistant, err := a.svc.CreateAssistantSession(ctx, "", "")
	if err != nil {
		return "", fmt.Errorf("criar sessão do copiloto: %w", err)
	}

	turnID, err := a.svc.StartTurn(assistant.ID, prompt, nil)
	if err != nil {
		return "", fmt.Errorf("iniciar processamento do copiloto: %w", err)
	}

	// Await turn completion
	_, eventsChan, cancel := a.svc.Subscribe(turnID, 0)
	defer cancel()

	timeout := time.After(45 * time.Second)
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-timeout:
			return "Tempo limite esgotado aguardando resposta da IA.", nil
		case ev, ok := <-eventsChan:
			if !ok {
				conv, _ := a.svc.Conversation(assistant.ID)
				if len(conv.Messages) > 0 {
					return conv.Messages[len(conv.Messages)-1].Content, nil
				}
				return "Processamento finalizado.", nil
			}
			if ev.Type == "turn_completed" {
				conv, _ := a.svc.Conversation(assistant.ID)
				if len(conv.Messages) > 0 {
					return conv.Messages[len(conv.Messages)-1].Content, nil
				}
				return "Tarefa concluída com sucesso.", nil
			}
			if ev.Type == "turn_failed" {
				return "Ocorreu uma falha durante o processamento da tarefa.", nil
			}
		}
	}
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
