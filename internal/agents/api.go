package agents

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

const (
	// ProviderAnthropic calls the Anthropic Messages API.
	ProviderAnthropic = "anthropic"
	// ProviderOpenAI calls an OpenAI Chat Completions endpoint.
	ProviderOpenAI = "openai"
	// ProviderOllama calls an OpenAI-compatible endpoint, defaulting to local Ollama.
	ProviderOllama = "ollama"
	// ProviderLMStudio calls an OpenAI-compatible endpoint, defaulting to LM Studio.
	ProviderLMStudio = "lmstudio"
	// ProviderOpenRouter calls an OpenAI-compatible endpoint, defaulting to OpenRouter.
	ProviderOpenRouter = "openrouter"

	// DefaultAPIVaultNamespace is the vault namespace used when none is configured.
	DefaultAPIVaultNamespace = "agents"

	defaultMaxTokens = 4096
	maxMaxTokens     = 1_000_000
	anthropicVersion = "2023-06-01"
)

var (
	// ErrUnknownProvider is returned when the API provider id is not supported.
	ErrUnknownProvider = errors.New("unknown api provider")
	// ErrTokenMissing is returned when the vault has no usable provider token.
	ErrTokenMissing = errors.New("provider token is not configured")
	// ErrTokenUnavailable is returned when the token cannot be read.
	ErrTokenUnavailable = errors.New("provider token is unavailable")
	// ErrVaultLocked is returned when the vault is locked or access is denied.
	ErrVaultLocked = errors.New("vault is locked")
	// ErrInvalidBaseURL is returned when the API base URL cannot be used.
	ErrInvalidBaseURL = errors.New("invalid api base url")
	// ErrInvalidVaultRef is returned when the vault namespace or key is not usable.
	ErrInvalidVaultRef = errors.New("invalid vault reference")
	// ErrModelRequired is returned when neither config nor the task sets a model.
	ErrModelRequired = errors.New("model is required")
	// ErrInvalidMaxTokens is returned when max_tokens is outside the accepted range.
	ErrInvalidMaxTokens = errors.New("invalid max tokens")
)

var (
	apiNamespacePattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)
	apiVaultKeyPattern  = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)
	toolNamePattern     = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]{0,63}$`)
	errInvalidTool      = errors.New("invalid tool")
	errStreamEnd        = errors.New("stream end")
	errBodyLimit        = errors.New("response exceeded size limit")
)

// APISettings is the non-secret HTTP provider block stored in the cockpit config.
// The token itself is read from the vault and is not a field of this struct.
type APISettings struct {
	// Provider is anthropic, openai, ollama, lmstudio, or openrouter.
	Provider string `yaml:"provider,omitempty"`
	// Model is the default model. A task model overrides it.
	Model string `yaml:"model,omitempty"`
	// BaseURL is the API root. Empty selects the provider default.
	// It must not include userinfo. A full endpoint path is also accepted.
	BaseURL string `yaml:"base_url,omitempty"`
	// VaultNamespace is the vault namespace. Empty uses DefaultAPIVaultNamespace.
	VaultNamespace string `yaml:"vault_namespace,omitempty"`
	// VaultKey is the vault key inside VaultNamespace. Empty uses <provider>_api_key.
	VaultKey string `yaml:"vault_key,omitempty"`
	// MaxTokens limits the completion. Empty uses 4096. Anthropic requires this field.
	MaxTokens int `yaml:"max_tokens,omitempty"`
}

// ResolvedAPI is APISettings with defaults filled in.
type ResolvedAPI struct {
	// Provider is the canonical provider id.
	Provider string
	// Model is the configured model and may be empty when the task supplies one.
	Model string
	// BaseURL is the API root without a trailing slash.
	BaseURL string
	// VaultNamespace is the namespace that holds the provider token.
	VaultNamespace string
	// VaultKey is the key inside VaultNamespace.
	VaultKey string
	// MaxTokens is the completion limit sent to the provider.
	MaxTokens int
}

// HTTPDoer sends one provider request.
// Tests inject an httptest client. The default client does not follow redirects,
// so the provider token is not forwarded to another host.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// TokenSource reads one provider token.
// Implementations must not log or persist the value.
type TokenSource interface {
	Get(ctx context.Context, namespace, key string) (string, error)
}

// APIBackend calls a provider HTTP API as a Backend.
// Run performs one plain completion. ToolSpec values can be serialized for a
// later phase; Run does not send tools and does not execute tool calls.
type APIBackend struct {
	resolved ResolvedAPI
	tokens   TokenSource
	client   HTTPDoer
}

var _ Backend = (*APIBackend)(nil)

// NewAPIBackend validates settings and returns an API backend.
// A nil client uses a client that does not follow redirects.
// A nil token source is rejected so a call cannot start without a vault reader.
func NewAPIBackend(settings APISettings, tokens TokenSource, client HTTPDoer) (*APIBackend, error) {
	resolved, err := settings.Resolve()
	if err != nil {
		return nil, err
	}
	if tokens == nil {
		return nil, fmt.Errorf("agents api: %w", ErrTokenMissing)
	}
	if client == nil {
		client = defaultHTTPClient()
	}
	return &APIBackend{resolved: resolved, tokens: tokens, client: client}, nil
}

// ID returns the canonical provider id.
func (b *APIBackend) ID() string {
	if b == nil {
		return ""
	}
	return b.resolved.Provider
}

// Resolve checks settings and fills defaults.
// An empty model is kept so a task can supply one. The token is not read here.
func (s APISettings) Resolve() (ResolvedAPI, error) {
	provider := normalizeProvider(s.Provider)
	if provider == "" {
		return ResolvedAPI{}, fmt.Errorf("agents api: %w", ErrUnknownProvider)
	}
	model := strings.TrimSpace(s.Model)
	if model != "" && !modelPattern.MatchString(model) {
		return ResolvedAPI{}, fmt.Errorf("agents api: %w", ErrInvalidModel)
	}
	base := strings.TrimSpace(s.BaseURL)
	if base == "" {
		base = defaultAPIBaseURL(provider)
	}
	if _, err := endpointURL(provider, base); err != nil {
		return ResolvedAPI{}, err
	}
	namespace := strings.TrimSpace(s.VaultNamespace)
	if namespace == "" {
		namespace = DefaultAPIVaultNamespace
	}
	if !apiNamespacePattern.MatchString(namespace) {
		return ResolvedAPI{}, fmt.Errorf("agents api: %w", ErrInvalidVaultRef)
	}
	key := strings.TrimSpace(s.VaultKey)
	if key == "" {
		key = defaultAPIVaultKey(provider)
	}
	if !apiVaultKeyPattern.MatchString(key) {
		return ResolvedAPI{}, fmt.Errorf("agents api: %w", ErrInvalidVaultRef)
	}
	maxTokens := s.MaxTokens
	if maxTokens == 0 {
		maxTokens = defaultMaxTokens
	}
	if maxTokens < 1 || maxTokens > maxMaxTokens {
		return ResolvedAPI{}, fmt.Errorf("agents api: %w", ErrInvalidMaxTokens)
	}
	return ResolvedAPI{
		Provider:       provider,
		Model:          model,
		BaseURL:        strings.TrimRight(base, "/"),
		VaultNamespace: namespace,
		VaultKey:       key,
		MaxTokens:      maxTokens,
	}, nil
}

// Run executes one plain completion and streams events through task.OnEvent.
// The provider token is read here and is not written to disk. A locked vault
// or a missing token returns before any HTTP request. Tool calls in the
// response are recorded on Result and are not executed.
func (b *APIBackend) Run(ctx context.Context, task Task) (*Result, error) {
	if b == nil || b.resolved.Provider == "" {
		return nil, fmt.Errorf("agents run: %w", ErrUnknownBackend)
	}
	runCtx, cancel, err := bindContext(ctx, task.Timeout)
	if err != nil {
		return nil, err
	}
	defer cancel()

	task, err = normalizeTask(task)
	if err != nil {
		return nil, err
	}
	model := b.resolved.Model
	if task.Model != "" {
		model = task.Model
	}
	if model == "" {
		return nil, fmt.Errorf("agents run: %w", ErrModelRequired)
	}
	endpoint, err := endpointURL(b.resolved.Provider, b.resolved.BaseURL)
	if err != nil {
		return nil, err
	}
	if b.tokens == nil {
		return nil, fmt.Errorf("agents run: %w", ErrTokenMissing)
	}
	token, err := b.tokens.Get(runCtx, b.resolved.VaultNamespace, b.resolved.VaultKey)
	if err != nil {
		return nil, err
	}
	defer func() { token = "" }()

	req, err := newProviderRequest(runCtx, b.resolved, endpoint, token, model, task.Prompt, nil)
	if err != nil {
		return nil, fmt.Errorf("agents run %s: %s: %w", b.ID(), clip(redact(err.Error(), token)), ErrRunFailed)
	}
	result := &Result{Backend: b.ID(), ExitCode: -1, Model: model}
	client := b.client
	if client == nil {
		client = defaultHTTPClient()
	}
	resp, err := client.Do(req)
	if err != nil {
		if runCtx.Err() != nil {
			return result, fmt.Errorf("agents run %s: %w", b.ID(), runCtx.Err())
		}
		return result, fmt.Errorf("agents run %s: %s: %w", b.ID(), clip(redact(err.Error(), token)), ErrRunFailed)
	}
	defer resp.Body.Close()
	result.ExitCode = resp.StatusCode
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		detail := fmt.Sprintf("http %d: %s", resp.StatusCode, readErrorMessage(resp.Body, token))
		return result, fmt.Errorf("agents run %s: %s: %w", b.ID(), clip(detail), ErrRunFailed)
	}

	summary, err := parseProvider(b.resolved.Provider, &maxReader{r: resp.Body, limit: maxAPIBody}, model, func(ev Event) {
		ev = scrubEvent(ev, token)
		if task.OnEvent != nil {
			task.OnEvent(ev.clone())
		}
		applyEvent(result, ev)
	})
	result.ToolCalls = scrubCalls(summary.calls, token)
	if summary.model != "" {
		result.Model = redact(summary.model, token)
	}
	if summary.session != "" && result.SessionID == "" {
		result.SessionID = redact(summary.session, token)
	}
	if runCtx.Err() != nil {
		return result, fmt.Errorf("agents run %s: %w", b.ID(), runCtx.Err())
	}
	if err != nil {
		return result, fmt.Errorf("agents run %s: %s: %w", b.ID(), clip(redact(err.Error(), token)), publicErr(err))
	}
	if summary.failed || result.Failed {
		detail := clip(result.Text)
		if detail == "" {
			detail = "provider reported a failure"
		}
		return result, fmt.Errorf("agents run %s: %s: %w", b.ID(), redact(detail, token), ErrRunFailed)
	}
	if !hasFinal(result.Events) {
		return result, fmt.Errorf("agents run %s: missing final result: %w", b.ID(), ErrRunFailed)
	}
	result.ExitCode = 0
	return result, nil
}

func publicErr(err error) error {
	if errors.Is(err, ErrBadOutput) {
		return ErrBadOutput
	}
	return ErrRunFailed
}

// KnownAPIProviders returns the provider ids this package can call, in doctor order.
func KnownAPIProviders() []string {
	return []string{
		ProviderAnthropic,
		ProviderOpenAI,
		ProviderOllama,
		ProviderLMStudio,
		ProviderOpenRouter,
	}
}

func normalizeProvider(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case ProviderAnthropic:
		return ProviderAnthropic
	case ProviderOpenAI:
		return ProviderOpenAI
	case ProviderOllama:
		return ProviderOllama
	case ProviderLMStudio, "lm-studio", "lm_studio":
		return ProviderLMStudio
	case ProviderOpenRouter, "open-router":
		return ProviderOpenRouter
	default:
		return ""
	}
}

func providerLabel(id string) string {
	switch id {
	case ProviderAnthropic:
		return "Anthropic API"
	case ProviderOpenAI:
		return "OpenAI API"
	case ProviderOllama:
		return "Ollama API"
	case ProviderLMStudio:
		return "LM Studio API"
	case ProviderOpenRouter:
		return "OpenRouter API"
	default:
		return "API provider"
	}
}

func defaultAPIVaultKey(provider string) string {
	return provider + "_api_key"
}

func defaultAPIBaseURL(provider string) string {
	switch provider {
	case ProviderAnthropic:
		return "https://api.anthropic.com"
	case ProviderOpenAI:
		return "https://api.openai.com"
	case ProviderOllama:
		return "http://127.0.0.1:11434"
	case ProviderLMStudio:
		return "http://127.0.0.1:1234"
	case ProviderOpenRouter:
		return "https://openrouter.ai/api"
	default:
		return ""
	}
}

func defaultHTTPClient() *http.Client {
	return &http.Client{
		CheckRedirect: refuseRedirect,
	}
}

func refuseRedirect(*http.Request, []*http.Request) error {
	return http.ErrUseLastResponse
}
