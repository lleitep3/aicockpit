package router

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// IntentCategory represents the classified destination for a user message.
type IntentCategory string

const (
	IntentHostCheck     IntentCategory = "host_check"
	IntentDatabaseQuery IntentCategory = "database_query"
	IntentDelegation    IntentCategory = "delegation"
	IntentComplexChat   IntentCategory = "complex_chat"
)

// RouteDecision is the structured JSON output returned by the router.
type RouteDecision struct {
	Intent  IntentCategory `json:"intent"`
	Target  string         `json:"target,omitempty"`
	Summary string         `json:"summary,omitempty"`
}

// Router classifies incoming user queries using a local Ollama model.
type Router struct {
	baseURL string
	model   string
	httpCli *http.Client
}

// NewRouter creates an instance of the local router (default: http://localhost:11434, llama3.2:1b).
func NewRouter(baseURL, model string) *Router {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	if model == "" {
		model = "llama3.2:1b"
	}
	return &Router{
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   model,
		httpCli: &http.Client{Timeout: 8 * time.Second},
	}
}

// Route classifies a user message into a RouteDecision.
// If Ollama is unreachable, it falls back to deterministic rule-based heuristics (0 tokens).
func (r *Router) Route(ctx context.Context, userMessage string) RouteDecision {
	clean := strings.TrimSpace(userMessage)
	if clean == "" {
		return RouteDecision{Intent: IntentComplexChat}
	}

	// 1. Try Ollama local model
	decision, err := r.classifyWithOllama(ctx, clean)
	if err == nil && decision.Intent != "" {
		return decision
	}

	// 2. Deterministic Fallback Heuristics
	return r.heuristicFallback(clean)
}

func (r *Router) classifyWithOllama(ctx context.Context, userMessage string) (RouteDecision, error) {
	prompt := fmt.Sprintf(`Você é um classificador de intenções rápido para um assistente de terminal.
Responda APENAS com um objeto JSON válido (sem markdown, sem explicações extras).

Categorias possíveis:
- "database_query": Se o usuário pergunta sobre quantidade de conversas, listar sessões, procurar mensagens antigas ou apagar/limpar sessões.
- "host_check": Se o usuário quer checar o sistema operacional, pipelines de CI do git/github (ex: gh run list, gh pr checks), portas abertas, espaço em disco ou processos.
- "delegation": Se o usuário pede para delegar uma tarefa para outro bot ou especialista (ex: "manda o bot backend rodar x").
- "complex_chat": Qualquer outra pergunta geral, raciocínio complexo, dúvidas conceituais ou pedidos de código.

Mensagem do usuário: %q

JSON:`, userMessage)

	payload := map[string]any{
		"model":  r.model,
		"prompt": prompt,
		"stream": false,
		"format": "json",
		"options": map[string]any{
			"temperature": 0.0,
			"num_predict": 100,
		},
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return RouteDecision{}, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", r.baseURL+"/api/generate", bytes.NewReader(bodyBytes))
	if err != nil {
		return RouteDecision{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.httpCli.Do(req)
	if err != nil {
		return RouteDecision{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return RouteDecision{}, fmt.Errorf("ollama status code %d", resp.StatusCode)
	}

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return RouteDecision{}, err
	}

	var ollamaResp struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal(respBytes, &ollamaResp); err != nil {
		return RouteDecision{}, err
	}

	var decision RouteDecision
	if err := json.Unmarshal([]byte(strings.TrimSpace(ollamaResp.Response)), &decision); err != nil {
		return RouteDecision{}, err
	}

	return decision, nil
}

func (r *Router) heuristicFallback(message string) RouteDecision {
	lower := strings.ToLower(message)

	if strings.Contains(lower, "conversa") || strings.Contains(lower, "sess") || strings.Contains(lower, "limpar") || strings.Contains(lower, "apagar") {
		return RouteDecision{
			Intent:  IntentDatabaseQuery,
			Summary: "Consulta ou ação em sessões do banco de dados",
		}
	}

	if strings.Contains(lower, "pipeline") || strings.Contains(lower, "disco") || strings.Contains(lower, "porta") || strings.Contains(lower, "processo") || strings.Contains(lower, "github") || strings.Contains(lower, "git ") {
		return RouteDecision{
			Intent:  IntentHostCheck,
			Summary: "Verificação de host ou pipelines",
		}
	}

	if strings.Contains(lower, "manda o bot") || strings.Contains(lower, "pede pro bot") || strings.Contains(lower, "delegar") {
		return RouteDecision{
			Intent:  IntentDelegation,
			Summary: "Delegação de tarefa para outro bot",
		}
	}

	return RouteDecision{
		Intent:  IntentComplexChat,
		Summary: "Conversação geral",
	}
}
