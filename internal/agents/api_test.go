package agents

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const testToken = "sk-test-secret-value"

func TestAPIBackendRunStreams(t *testing.T) {
	tests := []struct {
		name         string
		provider     string
		model        string
		taskModel    string
		body         string
		wantPath     string
		wantText     string
		wantModel    string
		wantSession  string
		wantTools    int
		wantKind     Kind
		anthropicKey bool
	}{
		{
			name:         "anthropic text",
			provider:     ProviderAnthropic,
			model:        "claude-test",
			body:         anthropicSSE("Hello", " world"),
			wantPath:     "/v1/messages",
			wantText:     "Hello world",
			wantModel:    "claude-test",
			wantSession:  "msg_1",
			anthropicKey: true,
		},
		{
			name:        "openai text",
			provider:    ProviderOpenAI,
			model:       "gpt-test",
			body:        openAISSE("Hello", " world"),
			wantPath:    "/v1/chat/completions",
			wantText:    "Hello world",
			wantModel:   "gpt-test",
			wantSession: "chatcmpl-1",
		},
		{
			name:      "ollama uses chat completions",
			provider:  "ollama",
			model:     "llama",
			body:      openAISSE("local", ""),
			wantPath:  "/v1/chat/completions",
			wantText:  "local",
			wantModel: "gpt-test",
		},
		{
			name:      "task model overrides config",
			provider:  ProviderOpenRouter,
			model:     "config-model",
			taskModel: "task-model",
			body:      openAISSE("routed", ""),
			wantPath:  "/v1/chat/completions",
			wantText:  "routed",
			wantModel: "gpt-test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sawTools bool
			backend, _ := newAPIServer(t, tt.provider, tt.model, func(w http.ResponseWriter, r *http.Request) {
				assertTokenHeader(t, r, tt.anthropicKey)
				if r.URL.Path != tt.wantPath {
					t.Errorf("path = %s", r.URL.Path)
				}
				if strings.Contains(r.URL.RequestURI(), testToken) {
					t.Error("token leaked into url")
				}
				var payload map[string]any
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Errorf("decode body: %v", err)
					return
				}
				if _, ok := payload["tools"]; ok {
					sawTools = true
				}
				if payload["stream"] != true {
					t.Errorf("stream = %v", payload["stream"])
				}
				wantModel := tt.model
				if tt.taskModel != "" {
					wantModel = tt.taskModel
				}
				if payload["model"] != wantModel {
					t.Errorf("model = %v", payload["model"])
				}
				w.Header().Set("Content-Type", "text/event-stream")
				_, _ = io.WriteString(w, tt.body)
			})
			var partials int
			result, err := backend.Run(context.Background(), Task{
				Prompt: "say hello",
				Model:  tt.taskModel,
				OnEvent: func(ev Event) {
					if ev.Partial && ev.Kind == KindText {
						partials++
					}
					if strings.Contains(ev.Text, testToken) || strings.Contains(string(ev.Raw), testToken) {
						t.Errorf("secret leaked into event kind %s", ev.Kind)
					}
				},
			})
			if err != nil {
				t.Fatalf("run: %v", err)
			}
			if sawTools {
				t.Fatal("run sent tools")
			}
			if result.Text != tt.wantText || result.Backend != normalizeProvider(tt.provider) {
				t.Fatalf("result text %q backend %q", result.Text, result.Backend)
			}
			if result.Model != tt.wantModel || result.ExitCode != 0 || result.Failed {
				t.Fatalf("result model %q exit %d failed %v", result.Model, result.ExitCode, result.Failed)
			}
			if tt.wantSession != "" && result.SessionID != tt.wantSession {
				t.Fatalf("session = %q", result.SessionID)
			}
			if partials == 0 {
				t.Fatal("expected streamed text")
			}
			if !hasFinal(result.Events) {
				t.Fatal("missing final event")
			}
			requireNoSecret(t, result.Text, result.SessionID, result.Model)
		})
	}
}

func TestAPIBackendToolCallsAreRecorded(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		body     string
		json     bool
	}{
		{name: "anthropic stream", provider: ProviderAnthropic, body: anthropicToolSSE()},
		{name: "openai stream", provider: ProviderOpenAI, body: openAIToolSSE()},
		{name: "anthropic json", provider: ProviderAnthropic, body: anthropicToolJSON(), json: true},
		{name: "openai json", provider: ProviderOpenAI, body: openAIToolJSON(), json: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var hits int
			backend, _ := newAPIServer(t, tt.provider, "model-a", func(w http.ResponseWriter, r *http.Request) {
				hits++
				assertTokenHeader(t, r, tt.provider == ProviderAnthropic)
				if tt.json {
					w.Header().Set("Content-Type", "application/json")
				} else {
					w.Header().Set("Content-Type", "text/event-stream")
				}
				_, _ = io.WriteString(w, tt.body)
			})
			result, err := backend.Run(context.Background(), Task{Prompt: "lookup"})
			if err != nil {
				t.Fatalf("run: %v", err)
			}
			if hits != 1 {
				t.Fatalf("requests = %d", hits)
			}
			if len(result.ToolCalls) != 1 || result.ToolCalls[0].Name != "lookup" {
				t.Fatalf("calls = %+v", result.ToolCalls)
			}
			if string(result.ToolCalls[0].Arguments) != `{"q":"x"}` {
				t.Fatalf("arguments = %s", result.ToolCalls[0].Arguments)
			}
			var sawTool bool
			for _, ev := range result.Events {
				if ev.Kind == KindTool && ev.Tool == "lookup" {
					sawTool = true
				}
			}
			if !sawTool {
				t.Fatal("missing tool event")
			}
			requireNoSecret(t, result.Text)
		})
	}
}

func TestAPIBackendRefusesBeforeRequest(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "file")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	tests := []struct {
		name string
		ctx  context.Context
		task Task
		src  TokenSource
		want error
	}{
		{name: "nil context", task: Task{Prompt: "hi"}, src: tokenValue(testToken), want: ErrNilContext},
		{name: "canceled", ctx: canceled, task: Task{Prompt: "hi"}, src: tokenValue(testToken), want: context.Canceled},
		{name: "empty prompt", ctx: context.Background(), task: Task{Prompt: "  "}, src: tokenValue(testToken), want: ErrEmptyPrompt},
		{name: "bad model", ctx: context.Background(), task: Task{Prompt: "hi", Model: "bad model"}, src: tokenValue(testToken), want: ErrInvalidModel},
		{name: "bad workdir", ctx: context.Background(), task: Task{Prompt: "hi", WorkDir: file}, src: tokenValue(testToken), want: ErrInvalidWorkDir},
		{name: "missing token", ctx: context.Background(), task: Task{Prompt: "hi"}, src: tokenErr(ErrTokenMissing), want: ErrTokenMissing},
		{name: "locked vault", ctx: context.Background(), task: Task{Prompt: "hi"}, src: tokenErr(ErrVaultLocked), want: ErrVaultLocked},
		{name: "unavailable token", ctx: context.Background(), task: Task{Prompt: "hi"}, src: tokenErr(ErrTokenUnavailable), want: ErrTokenUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend, _ := newAPIServer(t, ProviderOpenAI, "gpt-test", func(http.ResponseWriter, *http.Request) {
				t.Error("request was sent")
			})
			backend.tokens = tt.src
			_, err := backend.Run(tt.ctx, tt.task)
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v", err)
			}
			requireNoSecret(t, errString(err))
		})
	}
}

func TestAPIBackendErrors(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		headers map[string]string
		doerErr error
		want    error
		redact  bool
	}{
		{name: "http 401", status: 401, body: `{"error":{"message":"bad key ` + testToken + `"}}`, want: ErrRunFailed, redact: true},
		{name: "http 500", status: 500, body: "upstream failed", want: ErrRunFailed},
		{name: "malformed stream", status: 200, body: "data: {not-json}\n\n", want: ErrBadOutput},
		{name: "empty", status: 200, body: "", want: ErrRunFailed},
		{name: "provider error event", status: 200, body: "data: {\"error\":{\"message\":\"overloaded\"}}\n\n", want: ErrRunFailed},
		{name: "content filter", status: 200, body: "data: {\"choices\":[{\"delta\":{\"content\":\"no\"},\"finish_reason\":\"content_filter\"}]}\n\ndata: [DONE]\n\n", want: ErrRunFailed},
		{name: "transport", doerErr: errors.New("dial " + testToken), want: ErrRunFailed, redact: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var client HTTPDoer
			settings := APISettings{Provider: ProviderOpenAI, Model: "gpt-test", BaseURL: "https://example.com"}
			if tt.doerErr != nil {
				client = doFunc(func(*http.Request) (*http.Response, error) {
					return nil, tt.doerErr
				})
			} else {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assertTokenHeader(t, r, false)
					w.WriteHeader(tt.status)
					_, _ = io.WriteString(w, tt.body)
				}))
				t.Cleanup(server.Close)
				settings.BaseURL = server.URL
				client = server.Client()
			}
			backend, err := NewAPIBackend(settings, tokenValue(testToken), client)
			if err != nil {
				t.Fatal(err)
			}
			result, err := backend.Run(context.Background(), Task{Prompt: "hi"})
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v", err)
			}
			if tt.redact && strings.Contains(errString(err), testToken) {
				t.Fatal("secret leaked into error")
			}
			if tt.redact && !strings.Contains(errString(err), "[redacted]") {
				t.Fatalf("error was not redacted: %v", err)
			}
			requireNoSecret(t, resultText(result))
		})
	}
}

func TestAPIBackendJSONAndReasoning(t *testing.T) {
	const body = "  {\"id\":\"chatcmpl-9\",\"model\":\"json-model\",\"choices\":[{\"index\":0,\"finish_reason\":\"stop\",\"message\":{\"content\":[{\"type\":\"text\",\"text\":\"Hi\"},{\"type\":\"image\",\"text\":\"skip\"}],\"reasoning_content\":\"think\"}}]}"
	backend, _ := newAPIServer(t, ProviderLMStudio, "lm", func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/v1/chat/completions") {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, body)
	})
	result, err := backend.Run(context.Background(), Task{Prompt: "hi"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Text != "Hi" || result.Model != "json-model" || result.SessionID != "chatcmpl-9" {
		t.Fatalf("result = %+v", result)
	}
	var sawReason bool
	for _, ev := range result.Events {
		if ev.Kind == KindReasoning && ev.Text == "think" {
			sawReason = true
		}
	}
	if !sawReason {
		t.Fatal("missing reasoning")
	}
}

func TestAPIBackendAnthropicThinking(t *testing.T) {
	body := "" +
		"event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_9\",\"model\":\"claude-think\"}}\n\n" +
		"event: ping\ndata: {\"type\":\"ping\"}\n\n" +
		": keep-alive\n\n" +
		"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"thinking_delta\",\"thinking\":\"hmm\"}}\n\n" +
		"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"Answer\"}}\n\n" +
		"event: custom\ndata: {\"type\":\"custom\"}\n\n" +
		"event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"}}\n\n" +
		"event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"
	backend, _ := newAPIServer(t, ProviderAnthropic, "claude-test", func(w http.ResponseWriter, r *http.Request) {
		assertTokenHeader(t, r, true)
		if r.Header.Get("anthropic-version") != anthropicVersion {
			t.Error("missing anthropic version")
		}
		_, _ = io.WriteString(w, body)
	})
	result, err := backend.Run(context.Background(), Task{Prompt: "hi"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Text != "Answer" || result.SessionID != "msg_9" {
		t.Fatalf("result = %+v", result)
	}
}

func TestAPIBackendEchoedTokenIsRedacted(t *testing.T) {
	body := strings.ReplaceAll(openAISSE("token "+testToken, ""), "Hello", "token "+testToken)
	backend, _ := newAPIServer(t, ProviderOpenAI, "gpt-test", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, body)
	})
	result, err := backend.Run(context.Background(), Task{Prompt: "hi"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(result.Text, testToken) {
		t.Fatal("secret leaked into result text")
	}
	if !strings.Contains(result.Text, "[redacted]") {
		t.Fatalf("text = %q", result.Text)
	}
}

func TestAPIBackendCancelAndTimeout(t *testing.T) {
	t.Run("cancel", func(t *testing.T) {
		started := make(chan struct{})
		backend, _ := newAPIServer(t, ProviderOpenAI, "gpt-test", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"Hi\"}}]}\n\n")
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			close(started)
			<-r.Context().Done()
		})
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			<-started
			cancel()
		}()
		_, err := backend.Run(ctx, Task{Prompt: "hi"})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("timeout", func(t *testing.T) {
		backend, _ := newAPIServer(t, ProviderOpenAI, "gpt-test", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"Hi\"}}]}\n\n")
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			<-r.Context().Done()
		})
		_, err := backend.Run(context.Background(), Task{Prompt: "hi", Timeout: 50 * time.Millisecond})
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("err = %v", err)
		}
	})
}

func TestAPIBackendDoesNotWriteToken(t *testing.T) {
	dir := t.TempDir()
	backend, _ := newAPIServer(t, ProviderOpenAI, "gpt-test", func(w http.ResponseWriter, r *http.Request) {
		assertTokenHeader(t, r, false)
		_, _ = io.WriteString(w, openAISSE("ok", ""))
	})
	if _, err := backend.Run(context.Background(), Task{Prompt: "hi", WorkDir: dir}); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("workdir gained %d entries", len(entries))
	}
}

func TestAPIBackendBodyLimit(t *testing.T) {
	previous := maxAPIBody
	maxAPIBody = 32
	t.Cleanup(func() { maxAPIBody = previous })
	backend, _ := newAPIServer(t, ProviderOpenAI, "gpt-test", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, strings.Repeat("x", 4096))
	})
	_, err := backend.Run(context.Background(), Task{Prompt: "hi"})
	if !errors.Is(err, ErrRunFailed) {
		t.Fatalf("err = %v", err)
	}
}

func TestAPIBackendRedirectIsNotFollowed(t *testing.T) {
	leak := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("redirect hop received the token")
	}))
	t.Cleanup(leak.Close)
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertTokenHeader(t, r, false)
		http.Redirect(w, r, leak.URL, http.StatusFound)
	}))
	t.Cleanup(origin.Close)
	backend, err := NewAPIBackend(APISettings{
		Provider: ProviderOpenAI,
		Model:    "gpt-test",
		BaseURL:  origin.URL,
	}, tokenValue(testToken), nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = backend.Run(context.Background(), Task{Prompt: "hi"})
	if err == nil || strings.Contains(err.Error(), testToken) {
		t.Fatalf("err = %v", err)
	}
	if !errors.Is(err, ErrRunFailed) {
		t.Fatalf("err = %v", err)
	}
}

func TestResolveAndEndpoint(t *testing.T) {
	secret := testToken
	tests := []struct {
		name    string
		in      APISettings
		want    ResolvedAPI
		wantErr error
	}{
		{
			name: "defaults",
			in:   APISettings{Provider: "Anthropic"},
			want: ResolvedAPI{Provider: ProviderAnthropic, BaseURL: "https://api.anthropic.com", VaultNamespace: DefaultAPIVaultNamespace, VaultKey: "anthropic_api_key", MaxTokens: defaultMaxTokens},
		},
		{
			name: "aliases and overrides",
			in:   APISettings{Provider: "lm-studio", Model: "local", BaseURL: "http://127.0.0.1:9/v1", VaultNamespace: "team", VaultKey: "prod_key", MaxTokens: 32},
			want: ResolvedAPI{Provider: ProviderLMStudio, Model: "local", BaseURL: "http://127.0.0.1:9/v1", VaultNamespace: "team", VaultKey: "prod_key", MaxTokens: 32},
		},
		{name: "unknown", in: APISettings{Provider: secret}, wantErr: ErrUnknownProvider},
		{name: "userinfo", in: APISettings{Provider: "openai", BaseURL: "https://user:" + secret + "@example.com"}, wantErr: ErrInvalidBaseURL},
		{name: "bad scheme", in: APISettings{Provider: "openai", BaseURL: "ftp://example.com"}, wantErr: ErrInvalidBaseURL},
		{name: "bad namespace", in: APISettings{Provider: "openai", VaultNamespace: "Bad"}, wantErr: ErrInvalidVaultRef},
		{name: "bad key", in: APISettings{Provider: "openai", VaultKey: "Bad-" + secret}, wantErr: ErrInvalidVaultRef},
		{name: "bad model", in: APISettings{Provider: "openai", Model: "has space"}, wantErr: ErrInvalidModel},
		{name: "bad max", in: APISettings{Provider: "openai", MaxTokens: -1}, wantErr: ErrInvalidMaxTokens},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.in.Resolve()
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) || strings.Contains(errString(err), secret) {
					t.Fatalf("err = %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("got = %+v", got)
			}
		})
	}

	endpoints := []struct {
		provider string
		base     string
		want     string
	}{
		{ProviderAnthropic, "", ""},
		{ProviderAnthropic, "https://api.anthropic.com", "https://api.anthropic.com/v1/messages"},
		{ProviderAnthropic, "https://example.com/v1", "https://example.com/v1/messages"},
		{ProviderAnthropic, "https://example.com/v1/messages", "https://example.com/v1/messages"},
		{ProviderOpenAI, "https://api.openai.com", "https://api.openai.com/v1/chat/completions"},
		{ProviderOllama, "http://127.0.0.1:11434", "http://127.0.0.1:11434/v1/chat/completions"},
		{ProviderLMStudio, "http://127.0.0.1:1234", "http://127.0.0.1:1234/v1/chat/completions"},
		{ProviderOpenRouter, "https://openrouter.ai/api", "https://openrouter.ai/api/v1/chat/completions"},
		{ProviderOpenAI, "https://example.com/v1/chat/completions", "https://example.com/v1/chat/completions"},
	}
	for _, tt := range endpoints {
		base := tt.base
		if base == "" {
			base = defaultAPIBaseURL(tt.provider)
		}
		got, err := endpointURL(tt.provider, base)
		if err != nil || got != tt.want && tt.want != "" {
			if tt.want == "" {
				continue
			}
			t.Fatalf("endpoint %s %s = %s err %v", tt.provider, base, got, err)
		}
	}
	if _, err := endpointURL(ProviderOpenAI, "http://example.com/path#frag"); !errors.Is(err, ErrInvalidBaseURL) {
		t.Fatalf("fragment err = %v", err)
	}
	if _, err := NewAPIBackend(APISettings{}, tokenValue(testToken), nil); !errors.Is(err, ErrUnknownProvider) {
		t.Fatal(err)
	}
	if _, err := NewAPIBackend(APISettings{Provider: "openai"}, nil, nil); !errors.Is(err, ErrTokenMissing) {
		t.Fatal(err)
	}
	backend, err := NewAPIBackend(APISettings{Provider: "open-router", Model: "m"}, tokenValue(testToken), nil)
	if err != nil || backend.ID() != ProviderOpenRouter {
		t.Fatalf("backend = %+v err %v", backend, err)
	}
	if (*APIBackend)(nil).ID() != "" {
		t.Fatal("nil id")
	}
	if _, err := (*APIBackend)(nil).Run(context.Background(), Task{Prompt: "hi"}); !errors.Is(err, ErrUnknownBackend) {
		t.Fatal(err)
	}
	bare := &APIBackend{tokens: tokenValue(testToken)}
	if _, err := bare.Run(context.Background(), Task{Prompt: "hi"}); !errors.Is(err, ErrUnknownBackend) {
		t.Fatal(err)
	}
	noModel, err := NewAPIBackend(APISettings{Provider: "openai", BaseURL: "https://example.com"}, tokenValue(testToken), doFunc(func(*http.Request) (*http.Response, error) {
		t.Error("request was sent")
		return nil, errors.New("sent")
	}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := noModel.Run(context.Background(), Task{Prompt: "hi"}); !errors.Is(err, ErrModelRequired) {
		t.Fatal(err)
	}
}

func TestProviderRequestTools(t *testing.T) {
	tools := []ToolSpec{{
		Name:        "lookup",
		Description: "find",
		InputSchema: json.RawMessage(`{"type":"object"}`),
	}}
	for _, provider := range []string{ProviderAnthropic, ProviderOpenAI} {
		body, err := encodeBody(provider, "m", "p", 16, tools)
		if err != nil {
			t.Fatal(err)
		}
		if !bytesContains(body, `"lookup"`) || !bytesContains(body, `"tools"`) {
			t.Fatalf("body missing tools: %s", body)
		}
	}
	if _, err := encodeBody(ProviderOpenAI, "m", "p", 16, []ToolSpec{{Name: "bad name"}}); !errors.Is(err, errInvalidTool) {
		t.Fatal(err)
	}
	if _, err := encodeBody(ProviderAnthropic, "m", "p", 16, []ToolSpec{{Name: "ok", InputSchema: json.RawMessage("{")}}); !errors.Is(err, errInvalidTool) {
		t.Fatal(err)
	}
	req, err := newProviderRequest(context.Background(), ResolvedAPI{Provider: ProviderOpenAI, MaxTokens: 8}, "https://example.com/v1/chat/completions", "", "m", "p", nil)
	if !errors.Is(err, ErrTokenMissing) || req != nil {
		t.Fatal(err)
	}
	if err := refuseRedirect(nil, nil); !errors.Is(err, http.ErrUseLastResponse) {
		t.Fatal(err)
	}
	client := defaultHTTPClient()
	if client.CheckRedirect == nil {
		t.Fatal("missing redirect policy")
	}
}

func TestSkipsExtraChoices(t *testing.T) {
	body := "data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"A\"}},{\"index\":1,\"delta\":{\"content\":\"B\"}}]}\n\ndata: {\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n"
	backend, _ := newAPIServer(t, ProviderOpenAI, "gpt-test", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, body)
	})
	result, err := backend.Run(context.Background(), Task{Prompt: "hi"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Text != "A" {
		t.Fatalf("text = %q", result.Text)
	}
}

func TestAPIBackendAnthropicEdges(t *testing.T) {
	t.Run("error event", func(t *testing.T) {
		body := "event: error\ndata: {\"type\":\"error\",\"error\":{\"message\":\"overloaded\"}}\n\n"
		backend, _ := newAPIServer(t, ProviderAnthropic, "m", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, body)
		})
		_, err := backend.Run(context.Background(), Task{Prompt: "hi"})
		if !errors.Is(err, ErrRunFailed) || !strings.Contains(err.Error(), "overloaded") {
			t.Fatal(err)
		}
	})
	t.Run("json text and thinking", func(t *testing.T) {
		body := `{"id":"msg_3","type":"message","model":"claude-json","stop_reason":"end_turn","content":[{"type":"thinking","thinking":"hmm"},{"type":"text","text":"Done"},{"type":"tool_use","name":""}]}`
		backend, _ := newAPIServer(t, ProviderAnthropic, "m", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, body)
		})
		result, err := backend.Run(context.Background(), Task{Prompt: "hi"})
		if err != nil || result.Text != "Done" || result.Model != "claude-json" {
			t.Fatalf("result %+v err %v", result, err)
		}
	})
	t.Run("json error and refusal", func(t *testing.T) {
		backend, _ := newAPIServer(t, ProviderAnthropic, "m", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, `{"type":"error","error":{"message":"nope"}}`)
		})
		if _, err := backend.Run(context.Background(), Task{Prompt: "hi"}); !errors.Is(err, ErrRunFailed) {
			t.Fatal(err)
		}
	})
	t.Run("partial text and bad tool args", func(t *testing.T) {
		body := "event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"Hi\"}}\n\n" +
			"event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":1,\"content_block\":{\"type\":\"tool_use\",\"id\":\"toolu_9\",\"name\":\"lookup\",\"input\":{\"q\":\"y\"}}}\n\n" +
			"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":1,\"delta\":{\"type\":\"input_json_delta\",\"partial_json\":\"tail\"}}\n\n" +
			"event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"refusal\"}}\n\n" +
			"event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"
		backend, _ := newAPIServer(t, ProviderAnthropic, "m", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, body)
		})
		result, err := backend.Run(context.Background(), Task{Prompt: "hi"})
		if !errors.Is(err, ErrRunFailed) {
			t.Fatal(err)
		}
		if len(result.ToolCalls) != 1 || result.ToolCalls[0].Name != "lookup" {
			t.Fatalf("calls = %+v", result.ToolCalls)
		}
	})
	t.Run("empty stop", func(t *testing.T) {
		body := "event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"
		backend, _ := newAPIServer(t, ProviderAnthropic, "m", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, body)
		})
		if _, err := backend.Run(context.Background(), Task{Prompt: "hi"}); !errors.Is(err, ErrRunFailed) {
			t.Fatal(err)
		}
	})
}

func TestAPIBackendNilClientUsesDefault(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertTokenHeader(t, r, false)
		_, _ = io.WriteString(w, openAISSE("via-default", ""))
	}))
	t.Cleanup(server.Close)
	backend, err := NewAPIBackend(APISettings{Provider: ProviderOpenAI, Model: "m", BaseURL: server.URL}, tokenValue("   "), server.Client())
	if err != nil {
		t.Fatal(err)
	}
	backend.tokens = tokenValue("   ")
	if _, err := backend.Run(context.Background(), Task{Prompt: "hi"}); !errors.Is(err, ErrRunFailed) {
		t.Fatal(err)
	}
	backend.tokens = tokenValue(testToken)
	backend.client = nil
	result, err := backend.Run(context.Background(), Task{Prompt: "hi"})
	if err != nil || result.Text != "via-default" {
		t.Fatalf("result %+v err %v", result, err)
	}
}

func TestHelpers(t *testing.T) {
	if providerLabel("nope") != "API provider" {
		t.Fatal(providerLabel("nope"))
	}
	if defaultAPIBaseURL("nope") != "" {
		t.Fatal(defaultAPIBaseURL("nope"))
	}
	if got := normalizeArgs("not-json"); string(got) != `"not-json"` {
		t.Fatalf("args = %s", got)
	}
	if got := normalizeArgs("  "); got != nil {
		t.Fatalf("args = %s", got)
	}
	view := probeRef(context.Background(), nil, tokenRef{name: "OpenAI API", namespace: "agents", key: "openai_api_key"}).Doctor()
	if view.Status != "skipped" || !strings.Contains(view.Message, "unavailable") {
		t.Fatalf("view = %+v", view)
	}
	if redact("prefix sk-test-secret-value suffix", testToken) != "prefix [redacted] suffix" {
		t.Fatal(redact("prefix sk-test-secret-value suffix", testToken))
	}
	spaced := "sk test secret"
	escaped := redact("see "+strings.ReplaceAll(spaced, " ", "+"), spaced)
	if strings.Contains(escaped, spaced) {
		t.Fatal("query escaped token leaked")
	}
	msg := apiErrorMessage([]byte(`{"message":"plain"}`))
	if msg != "plain" {
		t.Fatal(msg)
	}
	if apiErrorMessage(nil) != "" {
		t.Fatal("expected empty")
	}
	if rawMessage("not json") != nil {
		t.Fatal("raw")
	}
	reader := &maxReader{r: strings.NewReader("abc"), limit: 0}
	buf := make([]byte, 8)
	if _, err := reader.Read(buf); err != nil && !errors.Is(err, errBodyLimit) && !errors.Is(err, io.EOF) {
		t.Fatal(err)
	}
	body, err := anthropicBody("m", "p", 4, []ToolSpec{{Name: "lookup"}})
	if err != nil || !bytesContains(body, `"input_schema":{}`) {
		t.Fatalf("body %s err %v", body, err)
	}
	if _, err := openAIBody("m", "p", 4, []ToolSpec{{Name: "lookup", InputSchema: json.RawMessage("{")}}); !errors.Is(err, errInvalidTool) {
		t.Fatal(err)
	}
	if _, err := newProviderRequest(context.Background(), ResolvedAPI{Provider: ProviderOpenAI}, "://bad", testToken, "m", "p", nil); err == nil {
		t.Fatal("expected request error")
	}
	if text, ok := jsonText(json.RawMessage("null")); ok || text != "" {
		t.Fatal("null text")
	}
	if _, ok := jsonText(json.RawMessage("1")); ok {
		t.Fatal("number was text")
	}
}

func TestOpenAIReasoningDelta(t *testing.T) {
	body := "data: {\"choices\":[{\"delta\":{\"reasoning_content\":\"because\"}}]}\n\ndata: {\"choices\":[{\"delta\":{\"content\":\"ok\"},\"finish_reason\":\"length\"}]}\n\n"
	backend, _ := newAPIServer(t, ProviderOllama, "m", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, body)
	})
	result, err := backend.Run(context.Background(), Task{Prompt: "hi"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Text != "ok" {
		t.Fatalf("text = %q", result.Text)
	}
}

func newAPIServer(t *testing.T, provider, model string, handler http.HandlerFunc) (*APIBackend, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	backend, err := NewAPIBackend(APISettings{
		Provider: provider,
		Model:    model,
		BaseURL:  server.URL,
	}, tokenValue(testToken), server.Client())
	if err != nil {
		t.Fatalf("new backend: %v", err)
	}
	return backend, server
}

func assertTokenHeader(t *testing.T, r *http.Request, anthropic bool) {
	t.Helper()
	if anthropic {
		if r.Header.Get("x-api-key") != testToken {
			t.Error("api key header mismatch")
		}
		if r.Header.Get("Authorization") != "" {
			t.Error("anthropic request used a bearer header")
		}
		return
	}
	if r.Header.Get("Authorization") != "Bearer "+testToken {
		t.Error("authorization header mismatch")
	}
	if r.Header.Get("x-api-key") != "" {
		t.Error("openai request used an api key header")
	}
}

type doFunc func(*http.Request) (*http.Response, error)

func (f doFunc) Do(r *http.Request) (*http.Response, error) { return f(r) }

type fixedToken struct {
	value string
	err   error
}

func (f fixedToken) Get(context.Context, string, string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.value, nil
}

func tokenValue(value string) TokenSource { return fixedToken{value: value} }

func tokenErr(err error) TokenSource { return fixedToken{err: err} }

func requireNoSecret(t *testing.T, parts ...string) {
	t.Helper()
	for _, part := range parts {
		if strings.Contains(part, testToken) {
			t.Fatal("secret leaked")
		}
	}
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func resultText(result *Result) string {
	if result == nil {
		return ""
	}
	return result.Text
}

func encodeBody(provider, model, prompt string, maxTokens int, tools []ToolSpec) ([]byte, error) {
	if provider == ProviderAnthropic {
		return anthropicBody(model, prompt, maxTokens, tools)
	}
	return openAIBody(model, prompt, maxTokens, tools)
}

func bytesContains(body []byte, part string) bool {
	return strings.Contains(string(body), part)
}

func anthropicSSE(left, right string) string {
	return "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_1\",\"model\":\"claude-test\"}}\n\n" +
		"event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n" +
		"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"" + left + "\"}}\n\n" +
		"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"" + right + "\"}}\n\n" +
		"event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n\n" +
		"event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"}}\n\n" +
		"event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"
}

func openAISSE(left, right string) string {
	second := ""
	if right != "" {
		second = "data: {\"id\":\"chatcmpl-1\",\"model\":\"gpt-test\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"" + right + "\"}}]}\n\n"
	}
	return "data: {\"id\":\"chatcmpl-1\",\"model\":\"gpt-test\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"" + left + "\"}}]}\n\n" +
		second +
		"data: {\"id\":\"chatcmpl-1\",\"model\":\"gpt-test\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
		"data: [DONE]\n\n"
}

func anthropicToolSSE() string {
	return "event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"tool_use\",\"id\":\"toolu_1\",\"name\":\"lookup\",\"input\":{}}}\n\n" +
		"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"input_json_delta\",\"partial_json\":\"{\\\"q\\\":\\\"x\\\"}\"}}\n\n" +
		"event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n\n" +
		"event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"tool_use\"}}\n\n" +
		"event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"
}

func openAIToolSSE() string {
	return "data: {\"id\":\"chatcmpl-2\",\"model\":\"gpt-test\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call_1\",\"function\":{\"name\":\"lookup\",\"arguments\":\"\"}}]}}]}\n\n" +
		"data: {\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"{\\\"q\\\":\\\"x\\\"}\"}}]}}]}\n\n" +
		"data: {\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"tool_calls\"}]}\n\n" +
		"data: [DONE]\n\n"
}

func anthropicToolJSON() string {
	return `{"id":"msg_2","type":"message","model":"claude-test","stop_reason":"tool_use","content":[{"type":"tool_use","id":"toolu_1","name":"lookup","input":{"q":"x"}}]}`
}

func openAIToolJSON() string {
	return `{"id":"chatcmpl-3","model":"gpt-test","choices":[{"finish_reason":"tool_calls","message":{"content":null,"tool_calls":[{"id":"call_1","type":"function","function":{"name":"lookup","arguments":"{\"q\":\"x\"}"}}]}}]}`
}
