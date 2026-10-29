package llm_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/lleitep3/aicockpit/internal/copi/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenRouterSuccessAndErrors(t *testing.T) {
	// 1. Success
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"choices": [{"message": {"content": "Olá via OpenRouter!"}}]
		}`))
	}))
	defer server.Close()

	oa := llm.NewOpenAI("test-key", "gemini-flash", server.URL, server.Client())
	resp, err := oa.Complete(context.Background(), "sys", "user")
	require.NoError(t, err)
	assert.Equal(t, "Olá via OpenRouter!", resp)

	// 2. API Error JSON
	errServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{
			"error": {"message": "Invalid model or quota exceeded"}
		}`))
	}))
	defer errServer.Close()

	oaErr := llm.NewOpenAI("test-key", "model-x", errServer.URL, errServer.Client())
	_, err = oaErr.Complete(context.Background(), "sys", "user")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "quota exceeded")

	// 3. Empty choices
	emptyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices": []}`))
	}))
	defer emptyServer.Close()

	oaEmpty := llm.NewOpenAI("test-key", "model-x", emptyServer.URL, emptyServer.Client())
	_, err = oaEmpty.Complete(context.Background(), "sys", "user")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no completion")
}

func TestAnthropicComplete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "test-key", r.Header.Get("x-api-key"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"content": [{"text": "Resposta Anthropic"}]
		}`))
	}))
	defer server.Close()

	anthropic := llm.NewAnthropic("test-key", "claude-3-haiku", server.Client())
	assert.Equal(t, "Anthropic (claude-3-haiku)", anthropic.Name())
}

func TestResolveAPIKey(t *testing.T) {
	_ = os.Setenv("TEST_OPENROUTER_KEY", "env_secret_123")
	defer os.Unsetenv("TEST_OPENROUTER_KEY")

	val := llm.ResolveAPIKey("TEST_OPENROUTER_KEY", "non_existent_vault_key")
	assert.Equal(t, "env_secret_123", val)

	empty := llm.ResolveAPIKey("NON_EXISTENT_ENV_KEY", "non_existent_vault_key")
	assert.Empty(t, empty)
}

func TestClientConfigAndFallback(t *testing.T) {
	// All providers initialized in client
	cfg := llm.Config{
		OpenRouterKey: "or-key",
		OpenAIKey:     "oa-key",
		AnthropicKey:  "ant-key",
	}
	cli := llm.NewClient(cfg)
	assert.NotNil(t, cli)

	// Empty client
	emptyClient := llm.NewClient(llm.Config{})
	_, err := emptyClient.Complete(context.Background(), "sys", "user")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nenhuma chave de API")
}
