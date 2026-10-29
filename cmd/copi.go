package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/copi"
	"github.com/lleitep3/aicockpit/internal/copi/llm"
	"github.com/lleitep3/aicockpit/internal/copi/storage"
	copitui "github.com/lleitep3/aicockpit/internal/copi/tui"
	"github.com/lleitep3/aicockpit/internal/personalagent"
	"github.com/spf13/cobra"
)

type personalAgentAdapter struct {
	svc       *personalagent.Service
	db        *storage.DB
	llmClient *llm.Client
}

func (a *personalAgentAdapter) List(ctx context.Context, includeArchived bool) ([]personalagent.ConversationSummary, error) {
	if a.db != nil {
		return a.db.ListSummaries(ctx, includeArchived)
	}
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
	if a.db != nil {
		return a.db.GetConversation(ctx, id)
	}
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
	if a.db != nil {
		_ = a.db.DeleteConversation(ctx, convID)
	}

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
	// Extract pure user request line if formatted with system prompt
	rawRequest := prompt
	if idx := strings.LastIndex(prompt, "PEDIDO DO USUÁRIO:\n"); idx != -1 {
		rawRequest = strings.TrimSpace(prompt[idx+len("PEDIDO DO USUÁRIO:\n"):])
	}

	// First check: can we answer deterministically via SQLite without spending tokens?
	lower := strings.ToLower(rawRequest)
	if strings.Contains(lower, "quantas conversas") || strings.Contains(lower, "quais conversas") || strings.Contains(lower, "falam da") || strings.Contains(lower, "falam sobre") || strings.Contains(lower, "conversas") {
		// Extract topic keywords
		words := strings.Fields(rawRequest)
		var keyword string
		for i, w := range words {
			if (strings.EqualFold(w, "da") || strings.EqualFold(w, "sobre") || strings.EqualFold(w, "do") || strings.EqualFold(w, "de")) && i+1 < len(words) {
				keyword = strings.Join(words[i+1:], " ")
				keyword = strings.Trim(keyword, "?.,!")
				break
			}
		}

		if keyword != "" && a.db != nil {
			matches, err := a.db.SearchConversations(ctx, keyword)
			if err == nil {
				if len(matches) == 0 {
					return fmt.Sprintf("🔍 Não encontrei nenhuma conversa ativa falando sobre %q no banco de dados.", keyword), nil
				}
				var b strings.Builder
				b.WriteString(fmt.Sprintf("🔍 Encontrei %d conversa(s) falando sobre %q (0 tokens gastos):\n", len(matches), keyword))
				for _, m := range matches {
					b.WriteString(fmt.Sprintf("  • [%s] \"%s\"\n", m.ID, m.Title))
				}
				return b.String(), nil
			}
		}
	}

	// If LLM direct client is available, execute via direct API (OpenRouter/OpenAI/Anthropic)
	if a.llmClient != nil {
		sysPrompt := "Você é o Copi, o bot mascote e orquestrador autônomo do Cockpit. Seja direto, prático, objetivo e focado em automações."
		resp, err := a.llmClient.Complete(ctx, sysPrompt, rawRequest)
		if err != nil {
			return fmt.Sprintf("⚠️ Erro na chamada da API LLM direta: %v", err), nil
		}
		if resp != "" {
			return resp, nil
		}
	}

	// Discover available providers that have quota or are supported
	candidates := []string{"antigravity", "copilot", "devin", "codex"}
	var selectedProvider string
	var selectedModel string

	for _, candidate := range candidates {
		catalog, err := a.svc.Models(ctx, candidate)
		if err != nil || len(catalog.Models) == 0 {
			continue
		}
		model, err := catalogDefaultOrFirstModel(catalog)
		if err != nil {
			continue
		}
		selectedProvider = candidate
		selectedModel = model.ID
		break
	}

	if selectedProvider == "" {
		return "", fmt.Errorf("nenhum provider de IA funcional encontrado no sistema (antigravity/copilot/devin/codex)")
	}

	// Create conversation with the active provider that has quota/availability
	conversation, err := a.svc.CreateConversation(ctx, selectedProvider, selectedModel, "", "never", personalagent.RoutingModeManual)
	if err != nil {
		return "", fmt.Errorf("criar sessão do Copi no provider %s: %w", selectedProvider, err)
	}

	turnID, err := a.svc.StartTurn(conversation.ID, prompt, nil)
	if err != nil {
		return "", fmt.Errorf("iniciar processamento do Copi (%s): %w", selectedProvider, err)
	}

	// Stream progress and await turn completion
	_, eventsChan, cancel := a.svc.Subscribe(turnID, 0)
	defer cancel()

	timeout := time.After(60 * time.Second)
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-timeout:
			return "Tempo limite esgotado aguardando resposta da IA.", nil
		case ev, ok := <-eventsChan:
			if !ok {
				conv, _ := a.svc.Conversation(conversation.ID)
				if len(conv.Messages) > 0 {
					return conv.Messages[len(conv.Messages)-1].Content, nil
				}
				return "Processamento finalizado.", nil
			}
			if ev.Type == "complete" {
				status, _ := ev.Data["status"].(string)
				if status == "failed" {
					errMsg, _ := ev.Data["error"].(string)
					if errMsg != "" {
						return fmt.Sprintf("Falha ao executar via %s: %s", selectedProvider, errMsg), nil
					}
					return fmt.Sprintf("Falha durante o processamento no provider %s.", selectedProvider), nil
				}
				conv, _ := a.svc.Conversation(conversation.ID)
				if len(conv.Messages) > 0 {
					return conv.Messages[len(conv.Messages)-1].Content, nil
				}
				return "Tarefa concluída com sucesso.", nil
			}
			if ev.Type == "error" {
				errMsg, _ := ev.Data["message"].(string)
				return fmt.Sprintf("Erro (%s): %s", selectedProvider, errMsg), nil
			}
		}
	}
}

func catalogDefaultOrFirstModel(catalog personalagent.Catalog) (personalagent.Model, error) {
	for _, m := range catalog.Models {
		if m.Default {
			return m, nil
		}
	}
	if len(catalog.Models) > 0 {
		return catalog.Models[0], nil
	}
	return personalagent.Model{}, fmt.Errorf("nenhum modelo encontrado")
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

			// Initialize SQLite embedded database
			dbPath := filepath.Join(sandboxDir, "conversations.db")
			sqliteDB, err := storage.Open(dbPath)
			if err != nil {
				return fmt.Errorf("open conversations sqlite db: %w", err)
			}
			defer sqliteDB.Close()

			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			jsonDir := filepath.Join(sandboxDir, "conversations")
			if imported, err := sqliteDB.ImportFromJSONDir(ctx, jsonDir); err == nil && imported > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "📦 Importadas %d conversas para o SQLite (conversations.db)\n", imported)
			}

			store, err := personalagent.NewStore(jsonDir)
			if err != nil {
				return fmt.Errorf("init conversation store: %w", err)
			}

			paSvc, err := personalagent.NewService(store, personalagent.NewProviderRegistry(), sandboxDir)
			if err != nil {
				return fmt.Errorf("init personal agent service: %w", err)
			}

			// Initialize direct LLM client (OpenRouter priority)
			openRouterKey := os.Getenv("OPENROUTER_API_KEY")
			if openRouterKey == "" {
				openRouterKey = llm.ResolveAPIKey("OPENROUTER_API_KEY", "openrouter:api_key")
			}
			if openRouterKey == "" {
				openRouterKey = llm.ResolveAPIKey("OPENROUTER_API_KEY", "openrouter_api_key")
			}

			llmCfg := llm.Config{
				OpenRouterKey:   openRouterKey,
				OpenRouterModel: os.Getenv("OPENROUTER_MODEL"),
				OpenAIKey:       llm.ResolveAPIKey("OPENAI_API_KEY", "openai:api_key"),
				OpenAIModel:     os.Getenv("OPENAI_MODEL"),
				AnthropicKey:    llm.ResolveAPIKey("ANTHROPIC_API_KEY", "anthropic:api_key"),
				AnthropicModel:  os.Getenv("ANTHROPIC_MODEL"),
			}
			llmClient := llm.NewClient(llmCfg)

			adapter := &personalAgentAdapter{svc: paSvc, db: sqliteDB, llmClient: llmClient}
			cfg := copi.DefaultConfig(sandboxDir)

			engine, err := copi.NewEngine(cfg, adapter)
			if err != nil {
				return fmt.Errorf("init copi engine: %w", err)
			}

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
