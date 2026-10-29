package router_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lleitep3/aicockpit/internal/copi/router"
	"github.com/stretchr/testify/assert"
)

func TestNewRouterDefaults(t *testing.T) {
	r := router.NewRouter("", "")
	assert.NotNil(t, r)
}

func TestRouterWithOllamaSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/generate", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"response": "{\"intent\":\"host_check\",\"summary\":\"Checar pipelines do GitHub\"}"
		}`))
	}))
	defer server.Close()

	r := router.NewRouter(server.URL, "llama3.2:1b")
	decision := r.Route(context.Background(), "como estao os pipelines do repo?")

	assert.Equal(t, router.IntentHostCheck, decision.Intent)
	assert.Equal(t, "Checar pipelines do GitHub", decision.Summary)
}

func TestRouterWithOllamaNon200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`error`))
	}))
	defer server.Close()

	r := router.NewRouter(server.URL, "llama3.2:1b")
	decision := r.Route(context.Background(), "quantas conversas temos?")
	// Falls back to heuristic
	assert.Equal(t, router.IntentDatabaseQuery, decision.Intent)
}

func TestRouterWithOllamaInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"response": "not-valid-json"}`))
	}))
	defer server.Close()

	r := router.NewRouter(server.URL, "llama3.2:1b")
	decision := r.Route(context.Background(), "quantas conversas temos?")
	// Falls back to heuristic
	assert.Equal(t, router.IntentDatabaseQuery, decision.Intent)
}

func TestRouterFallbackHeuristics(t *testing.T) {
	// Pointing to a closed server port so it triggers fallback
	r := router.NewRouter("http://127.0.0.1:54321", "llama3.2:1b")

	ctx := context.Background()

	// Empty
	assert.Equal(t, router.IntentComplexChat, r.Route(ctx, "").Intent)

	// Database query
	assert.Equal(t, router.IntentDatabaseQuery, r.Route(ctx, "quantas conversas temos?").Intent)
	assert.Equal(t, router.IntentDatabaseQuery, r.Route(ctx, "limpar conversas vazias").Intent)
	assert.Equal(t, router.IntentDatabaseQuery, r.Route(ctx, "procurar sessão anterior").Intent)

	// Host check
	assert.Equal(t, router.IntentHostCheck, r.Route(ctx, "verificar pipelines no github").Intent)
	assert.Equal(t, router.IntentHostCheck, r.Route(ctx, "quanto espaço em disco tenho?").Intent)
	assert.Equal(t, router.IntentHostCheck, r.Route(ctx, "qual o status do git status?").Intent)

	// Delegation
	assert.Equal(t, router.IntentDelegation, r.Route(ctx, "manda o bot backend rodar os testes").Intent)

	// Complex chat default
	assert.Equal(t, router.IntentComplexChat, r.Route(ctx, "me explique o teorema de Pitágoras").Intent)
}
