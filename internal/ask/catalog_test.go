package ask

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/lleitep3/aicockpit/internal/agents"
)

type installedRunner struct {
	bins map[string]bool
}

func (r installedRunner) Start(context.Context, agents.Spec) (agents.Proc, error) {
	return nil, errors.New("not used")
}

func (r installedRunner) Installed(binary string) bool {
	return r.bins[binary]
}

type missingRunner struct{}

func (missingRunner) Start(context.Context, agents.Spec) (agents.Proc, error) {
	return nil, agents.ErrNotInstalled
}

type tokenDouble struct {
	ok        bool
	err       error
	namespace string
	key       string
	value     string
}

func (t *tokenDouble) Get(context.Context, string, string) (string, error) {
	if t.value == "" {
		return "token", nil
	}
	return t.value, nil
}

func (t *tokenDouble) Configured(_ context.Context, namespace, key string) (bool, error) {
	t.namespace = namespace
	t.key = key
	return t.ok, t.err
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) Do(req *http.Request) (*http.Response, error) { return f(req) }

func TestLiveCatalogAutoAndExplicit(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name      string
		opts      LiveOptions
		flag      string
		config    string
		wantRef   string
		wantErr   string
		wantAvail bool
	}{
		{
			name:    "picks claude",
			opts:    LiveOptions{Runner: installedRunner{bins: map[string]bool{"claude": true}}},
			wantRef: "claude",
		},
		{
			name:    "skips to codex",
			opts:    LiveOptions{Runner: installedRunner{bins: map[string]bool{"codex": true}}},
			wantRef: "codex",
		},
		{
			name: "picks configured api",
			opts: LiveOptions{
				Runner: installedRunner{},
				API:    agents.APISettings{Provider: "openai", Model: "gpt-test"},
				Tokens: &tokenDouble{ok: true},
			},
			wantRef: "api:openai",
		},
		{
			name:    "none installed",
			opts:    LiveOptions{Runner: installedRunner{}},
			wantErr: "no agent backend available",
		},
		{
			name: "api token missing is reported",
			opts: LiveOptions{
				Runner: installedRunner{},
				API:    agents.APISettings{Provider: "anthropic", Model: "claude-sonnet-4-5"},
				Tokens: &tokenDouble{},
			},
			wantErr: "token is not configured",
		},
		{
			name:    "explicit claude missing",
			opts:    LiveOptions{Runner: installedRunner{}},
			flag:    "claude",
			wantErr: "claude is not installed",
		},
		{
			name:    "config names codex",
			opts:    LiveOptions{Runner: installedRunner{bins: map[string]bool{"codex": true, "claude": true}}},
			config:  "codex",
			wantRef: "codex",
		},
		{
			name: "explicit api provider",
			opts: LiveOptions{
				API:    agents.APISettings{Provider: "anthropic", Model: "claude-sonnet-4-5", VaultNamespace: "agents"},
				Tokens: &tokenDouble{ok: true},
			},
			flag:    "api:openai",
			wantRef: "api:openai",
		},
		{
			name: "bare api uses config",
			opts: LiveOptions{
				API:    agents.APISettings{Provider: "ollama", Model: "llama"},
				Tokens: &tokenDouble{ok: true},
			},
			flag:    "api",
			wantRef: "api:ollama",
		},
		{
			name:    "bare api without provider",
			opts:    LiveOptions{Tokens: &tokenDouble{ok: true}},
			flag:    "api",
			wantErr: "agents.api.provider is not set",
		},
		{
			name: "vault locked",
			opts: LiveOptions{
				API:    agents.APISettings{Provider: "openai", Model: "gpt-test"},
				Tokens: &tokenDouble{err: agents.ErrVaultLocked},
			},
			flag:    "api:openai",
			wantErr: "vault is locked",
		},
		{
			name: "probe unavailable",
			opts: LiveOptions{
				API:    agents.APISettings{Provider: "openai", Model: "gpt-test"},
				Probe:  &tokenDouble{err: errors.New("index down")},
				Tokens: &tokenDouble{ok: true},
			},
			flag:    "api",
			wantErr: "token status unavailable",
		},
		{
			name: "invalid api settings",
			opts: LiveOptions{
				API:    agents.APISettings{Provider: "openai", MaxTokens: -1},
				Tokens: &tokenDouble{ok: true},
			},
			flag:    "api",
			wantErr: "configuration is invalid",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			catalog := NewLiveCatalog(tt.opts)
			ref, backend, err := resolve(ctx, tt.flag, tt.config, catalog)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if displayRef(ref) != tt.wantRef {
				t.Fatalf("ref = %+v", ref)
			}
			if backend == nil || backend.ID() == "" {
				t.Fatalf("backend = %#v", backend)
			}
		})
	}
}

func TestLiveCatalogAPIModelSwitch(t *testing.T) {
	var sawModel string
	var calls int
	doer := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		sawModel = string(body)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"model":"claude-sonnet-4-5","content":[{"type":"text","text":"ok"}]}`)),
		}, nil
	})
	catalog := NewLiveCatalog(LiveOptions{
		API: agents.APISettings{
			Provider: "anthropic",
			Model:    "claude-sonnet-4-5",
			BaseURL:  "https://api.anthropic.com",
			VaultKey: "anthropic_api_key",
		},
		Tokens: &tokenDouble{ok: true, value: "secret-token"},
		HTTP:   doer,
	})
	switched, err := catalog.Open(context.Background(), Ref{Kind: KindAPI, ID: "openai", Raw: "api:openai"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := switched.Run(context.Background(), agents.Task{Prompt: "hi"}); !errors.Is(err, agents.ErrModelRequired) {
		t.Fatalf("switched model = %v", err)
	}
	if calls != 0 {
		t.Fatal("switched provider called HTTP")
	}
	same, err := catalog.Open(context.Background(), Ref{Kind: KindAPI, ID: "anthropic", Raw: "api:anthropic"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := same.Run(context.Background(), agents.Task{Prompt: "hi"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Text != "ok" || !strings.Contains(sawModel, "claude-sonnet-4-5") {
		t.Fatalf("result = %+v body = %s", result, sawModel)
	}
	if strings.Contains(sawModel, "secret-token") {
		t.Fatal("model assertion captured the token")
	}
}

func TestLiveCatalogAliasesAndDetect(t *testing.T) {
	ctx := context.Background()
	probe := &tokenDouble{ok: true}
	catalog := NewLiveCatalog(LiveOptions{
		Runner: missingRunner{},
		API:    agents.APISettings{Provider: "open-router", Model: "gpt-test", VaultKey: "router_key"},
		Tokens: probe,
	})
	candidates, err := catalog.Auto(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 3 || candidates[2].Ref.Raw != "api:openrouter" || !candidates[2].Available {
		t.Fatalf("candidates = %+v", candidates)
	}
	if probe.namespace != agents.DefaultAPIVaultNamespace || probe.key != "router_key" {
		t.Fatalf("probe = %s/%s", probe.namespace, probe.key)
	}
	noProbe := NewLiveCatalog(LiveOptions{API: agents.APISettings{Provider: "openai", Model: "m"}})
	got, err := noProbe.Check(ctx, Ref{Kind: KindAPI, Raw: "api"})
	if err != nil || got.Available || got.Reason != "token status unavailable" {
		t.Fatalf("candidate = %+v %v", got, err)
	}
	if _, err := noProbe.Open(ctx, Ref{Kind: KindAPI, ID: "openai", Raw: "api:openai"}); !errors.Is(err, agents.ErrTokenMissing) {
		t.Fatalf("open = %v", err)
	}
	if _, err := catalog.Open(ctx, Ref{Kind: "other", Raw: "other"}); !errors.Is(err, agents.ErrUnknownBackend) {
		t.Fatalf("unknown open = %v", err)
	}
	unknown, err := catalog.Check(ctx, Ref{Kind: KindCLI, ID: "gemini", Raw: "gemini"})
	if err != nil || unknown.Reason != "unknown backend" {
		t.Fatalf("unknown cli = %+v %v", unknown, err)
	}
	other, err := catalog.Check(ctx, Ref{Kind: "other", Raw: "other"})
	if err != nil || other.Reason != "unknown backend" {
		t.Fatalf("other = %+v %v", other, err)
	}
	var nilCatalog *LiveCatalog
	if _, err := nilCatalog.Auto(nil); err != nil { //nolint:staticcheck // SA1012: nil context is the case under test.
		t.Fatal(err)
	}
	missing, err := nilCatalog.Check(nil, Ref{Raw: "claude"}) //nolint:staticcheck // SA1012: nil context is the case under test.
	if err != nil || missing.Reason != "unavailable" {
		t.Fatalf("nil check = %+v %v", missing, err)
	}
	if _, err := nilCatalog.Open(ctx, Ref{Kind: KindCLI, ID: "claude"}); !errors.Is(err, ErrNoBackend) {
		t.Fatalf("nil open = %v", err)
	}
	if canonicalProvider("") != "" || canonicalProvider("nope") != "" || canonicalProvider("lm-studio") != "lmstudio" {
		t.Fatal("canonical provider")
	}
	installed := NewLiveCatalog(LiveOptions{Runner: nil})
	status, err := installed.Check(ctx, Ref{Kind: KindCLI, ID: "claude", Raw: "claude"})
	if err != nil {
		t.Fatal(err)
	}
	if status.Available && status.Ref.Raw != "claude" {
		t.Fatalf("host claude = %+v", status)
	}
}
