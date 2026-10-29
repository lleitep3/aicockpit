package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/lleitep3/aicockpit/internal/vault"
)

// Provider defines the interface for direct LLM API execution.
type Provider interface {
	Name() string
	Complete(ctx context.Context, systemPrompt, userMessage string) (string, error)
}

// Client manages direct LLM API providers with fallback.
type Client struct {
	providers []Provider
	httpCli   *http.Client
}

// Config holds configuration for LLM clients.
type Config struct {
	OpenRouterKey   string
	OpenRouterModel string
	OpenAIKey       string
	OpenAIModel     string
	AnthropicKey    string
	AnthropicModel  string
	BaseURL         string
}

// NewClient creates a new multi-provider LLM client.
func NewClient(cfg Config) *Client {
	var providers []Provider
	httpClient := &http.Client{Timeout: 90 * time.Second}

	// 1. OpenRouter (Primary default for flexible model routing)
	if cfg.OpenRouterKey != "" {
		model := cfg.OpenRouterModel
		if model == "" {
			model = "google/gemini-2.5-flash" // ultra-fast & cost efficient default
		}
		providers = append(providers, NewOpenRouter(cfg.OpenRouterKey, model, httpClient))
	}

	// 2. OpenAI / Compatible
	if cfg.OpenAIKey != "" {
		model := cfg.OpenAIModel
		if model == "" {
			model = "gpt-4o-mini"
		}
		providers = append(providers, NewOpenAI(cfg.OpenAIKey, model, cfg.BaseURL, httpClient))
	}

	// 3. Anthropic Direct
	if cfg.AnthropicKey != "" {
		model := cfg.AnthropicModel
		if model == "" {
			model = "claude-3-5-haiku-20241022"
		}
		providers = append(providers, NewAnthropic(cfg.AnthropicKey, model, httpClient))
	}

	return &Client{
		providers: providers,
		httpCli:   httpClient,
	}
}

// ResolveAPIKey attempts to load a key from environment or Cockpit vault.
func ResolveAPIKey(envVar, vaultKey string) string {
	if val := os.Getenv(envVar); val != "" {
		return val
	}
	v := vault.NewOSVault()
	if val, err := v.Get(vaultKey); err == nil && val != "" {
		return val
	}
	return ""
}

// Complete executes prompt against the first functional provider with fallback.
func (c *Client) Complete(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	if len(c.providers) == 0 {
		return "", fmt.Errorf("nenhuma chave de API configurada (defina OPENROUTER_API_KEY ou use 'cockpit vault set openrouter_api_key <key>')")
	}

	var lastErr error
	for _, p := range c.providers {
		resp, err := p.Complete(ctx, systemPrompt, userMessage)
		if err == nil && strings.TrimSpace(resp) != "" {
			return resp, nil
		}
		lastErr = fmt.Errorf("%s failed: %w", p.Name(), err)
	}

	return "", lastErr
}

// --- OpenRouter Provider ---

type OpenRouter struct {
	apiKey  string
	model   string
	httpCli *http.Client
}

func NewOpenRouter(apiKey, model string, cli *http.Client) *OpenRouter {
	return &OpenRouter{apiKey: apiKey, model: model, httpCli: cli}
}

func (o *OpenRouter) Name() string {
	return "OpenRouter (" + o.model + ")"
}

type openAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatRequest struct {
	Model       string              `json:"model"`
	Messages    []openAIChatMessage `json:"messages"`
	Temperature float64             `json:"temperature"`
	MaxTokens   int                 `json:"max_tokens,omitempty"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (o *OpenRouter) Complete(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	reqBody := openAIChatRequest{
		Model: o.model,
		Messages: []openAIChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userMessage},
		},
		Temperature: 0.2,
		MaxTokens:   2048,
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+o.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "https://github.com/lleitep3/aicockpit")
	req.Header.Set("X-Title", "Cockpit Copi Bot")

	resp, err := o.httpCli.Do(req)
	if err != nil {
		return "", fmt.Errorf("openrouter request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	var parsed openAIChatResponse
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return "", fmt.Errorf("decode json: %w (body: %s)", err, string(respBytes))
	}

	if parsed.Error != nil && parsed.Error.Message != "" {
		return "", fmt.Errorf("api error: %s", parsed.Error.Message)
	}

	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("no completion returned from openrouter")
	}

	return strings.TrimSpace(parsed.Choices[0].Message.Content), nil
}

// --- OpenAI Provider ---

type OpenAI struct {
	apiKey  string
	model   string
	baseURL string
	httpCli *http.Client
}

func NewOpenAI(apiKey, model, baseURL string, cli *http.Client) *OpenAI {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	return &OpenAI{apiKey: apiKey, model: model, baseURL: baseURL, httpCli: cli}
}

func (oa *OpenAI) Name() string {
	return "OpenAI (" + oa.model + ")"
}

func (oa *OpenAI) Complete(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	reqBody := openAIChatRequest{
		Model: oa.model,
		Messages: []openAIChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userMessage},
		},
		Temperature: 0.2,
	}

	bodyBytes, _ := json.Marshal(reqBody)
	url := strings.TrimRight(oa.baseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+oa.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := oa.httpCli.Do(req)
	if err != nil {
		return "", fmt.Errorf("openai request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	var parsed openAIChatResponse
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return "", fmt.Errorf("decode json: %w", err)
	}

	if parsed.Error != nil && parsed.Error.Message != "" {
		return "", fmt.Errorf("api error: %s", parsed.Error.Message)
	}

	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("no completion returned")
	}

	return strings.TrimSpace(parsed.Choices[0].Message.Content), nil
}

// --- Anthropic Provider ---

type Anthropic struct {
	apiKey  string
	model   string
	httpCli *http.Client
}

func NewAnthropic(apiKey, model string, cli *http.Client) *Anthropic {
	return &Anthropic{apiKey: apiKey, model: model, httpCli: cli}
}

func (a *Anthropic) Name() string {
	return "Anthropic (" + a.model + ")"
}

func (a *Anthropic) Complete(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	reqMap := map[string]any{
		"model":      a.model,
		"max_tokens": 2048,
		"system":     systemPrompt,
		"messages": []map[string]string{
			{"role": "user", "content": userMessage},
		},
	}

	bodyBytes, _ := json.Marshal(reqMap)
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}

	req.Header.Set("x-api-key", a.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpCli.Do(req)
	if err != nil {
		return "", fmt.Errorf("anthropic request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	var parsed struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}

	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return "", fmt.Errorf("decode json: %w", err)
	}

	if parsed.Error != nil && parsed.Error.Message != "" {
		return "", fmt.Errorf("api error: %s", parsed.Error.Message)
	}

	if len(parsed.Content) == 0 {
		return "", fmt.Errorf("no content returned from anthropic")
	}

	return strings.TrimSpace(parsed.Content[0].Text), nil
}
