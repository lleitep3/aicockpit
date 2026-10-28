package personalagent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type routingTestAdapter struct {
	catalog Catalog
}

func (a routingTestAdapter) Catalog(context.Context) (Catalog, error) {
	return a.catalog, nil
}

func (routingTestAdapter) RunTurn(context.Context, RunRequest, func(string), func(ApprovalRequest) (string, error)) (RunResult, error) {
	return RunResult{}, nil
}

func newRoutingTestService(t *testing.T) (*Service, *Store) {
	t.Helper()
	baseDir := t.TempDir()
	conversationDirectory := filepath.Join(baseDir, "conversations")
	projectsDirectory := filepath.Join(baseDir, "projects")
	store, err := NewStoreWithProjectsDir(conversationDirectory, projectsDirectory)
	if err != nil {
		t.Fatalf("NewStoreWithProjectsDir() error = %v", err)
	}
	workspace := t.TempDir()
	discoveredAt := time.Date(2026, time.October, 26, 12, 0, 0, 0, time.UTC)
	adapter := routingTestAdapter{catalog: Catalog{
		ProviderID:   "codex",
		Source:       "test-catalog",
		DiscoveredAt: discoveredAt,
		Models: []Model{
			{ID: "model-fast", Name: "Fast", Default: true, Selectable: true, ReasoningEfforts: []string{"low", "medium"}},
			{ID: "model-deep", Name: "Deep", Selectable: true, ReasoningEfforts: []string{"high", "xhigh"}},
		},
	}}
	providers := &ProviderRegistry{adapters: map[string]Adapter{"codex": adapter}}
	service, err := NewService(store, providers, workspace)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return service, store
}

func TestRoutingDefaultsToAvailableModelAndCanBeCustomized(t *testing.T) {
	service, store := newRoutingTestService(t)
	config, err := service.Routing(context.Background())
	if err != nil {
		t.Fatalf("Routing() error = %v", err)
	}
	if len(config.Rules) != len(knownTaskClasses()) {
		t.Fatalf("default routing rules = %d, want %d", len(config.Rules), len(knownTaskClasses()))
	}
	for _, rule := range config.Rules {
		if rule.ProviderID != "codex" || rule.ModelID != "model-fast" {
			t.Fatalf("default route = %#v, want codex/model-fast", rule)
		}
	}

	config.Rules[0].ModelID = "model-deep"
	config.Rules[0].ReasoningEffort = "high"
	updated, err := service.UpdateRouting(context.Background(), config)
	if err != nil {
		t.Fatalf("UpdateRouting() error = %v", err)
	}
	if updated.UpdatedAt.IsZero() {
		t.Fatal("UpdateRouting() did not timestamp the change")
	}

	loaded, err := store.LoadRouting()
	if err != nil {
		t.Fatalf("LoadRouting() error = %v", err)
	}
	if loaded.Rules[0].ModelID != "model-deep" || loaded.Rules[0].ReasoningEffort != "high" {
		t.Fatalf("stored route = %#v, want customized model and effort", loaded.Rules[0])
	}
	info, err := os.Stat(filepath.Join(store.directory, "routing.config"))
	if err != nil {
		t.Fatalf("stat routing config: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("routing config permissions = %o, want 600", info.Mode().Perm())
	}
}

func TestResolveTurnSelectionUsesConfiguredTaskRoute(t *testing.T) {
	service, _ := newRoutingTestService(t)
	config, err := service.Routing(context.Background())
	if err != nil {
		t.Fatalf("Routing() error = %v", err)
	}
	for index := range config.Rules {
		if config.Rules[index].TaskClass == TaskClassCoding {
			config.Rules[index].ModelID = "model-deep"
			config.Rules[index].ReasoningEffort = "high"
		}
	}
	if _, err := service.UpdateRouting(context.Background(), config); err != nil {
		t.Fatalf("UpdateRouting() error = %v", err)
	}

	selection, classification, err := service.resolveTurnSelection(
		Selection{ProviderID: "codex", ModelID: "model-fast", RoutingMode: RoutingModeAutomatic},
		"Implemente a correção",
		nil,
	)
	if err != nil {
		t.Fatalf("resolveTurnSelection() error = %v", err)
	}
	if classification.TaskClass != TaskClassCoding {
		t.Fatalf("task class = %q, want %q", classification.TaskClass, TaskClassCoding)
	}
	if selection.ProviderID != "codex" || selection.ModelID != "model-deep" || selection.ReasoningEffort != "high" {
		t.Fatalf("resolved selection = %#v, want codex/model-deep/high", selection)
	}
}

func TestUpdateRoutingRejectsMissingClassesAndUnsupportedEffort(t *testing.T) {
	service, _ := newRoutingTestService(t)
	config, err := service.Routing(context.Background())
	if err != nil {
		t.Fatalf("Routing() error = %v", err)
	}
	config.Rules = config.Rules[:len(config.Rules)-1]
	if err := service.validateRouting(context.Background(), config); err == nil || !strings.Contains(err.Error(), "missing route") {
		t.Fatalf("validateRouting() error = %v, want missing route error", err)
	}

	config, err = service.Routing(context.Background())
	if err != nil {
		t.Fatalf("Routing() error = %v", err)
	}
	config.Rules[0].ReasoningEffort = "xhigh"
	if err := service.validateRouting(context.Background(), config); err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("validateRouting() error = %v, want unsupported effort error", err)
	}
}

func TestRoutingHTTPHandlersReadWriteRoutesAndCreateAutomaticConversation(t *testing.T) {
	service, _ := newRoutingTestService(t)
	const host = "127.0.0.1:4321"
	const token = "test-session-token"
	handler := NewHTTPHandler(service, token, host)

	getRequest := httptest.NewRequest(http.MethodGet, "http://"+host+"/api/v1/routing", nil)
	getRequest.Header.Set("Authorization", "Bearer "+token)
	getResponse := httptest.NewRecorder()
	handler.ServeHTTP(getResponse, getRequest)
	if getResponse.Code != http.StatusOK {
		t.Fatalf("GET /routing status = %d, want 200: %s", getResponse.Code, getResponse.Body.String())
	}

	config, err := service.Routing(context.Background())
	if err != nil {
		t.Fatalf("Routing() error = %v", err)
	}
	body, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("marshal route config: %v", err)
	}
	putRequest := httptest.NewRequest(http.MethodPut, "http://"+host+"/api/v1/routing", strings.NewReader(string(body)))
	putRequest.Header.Set("Authorization", "Bearer "+token)
	putRequest.Header.Set("Origin", "http://"+host)
	putResponse := httptest.NewRecorder()
	handler.ServeHTTP(putResponse, putRequest)
	if putResponse.Code != http.StatusOK {
		t.Fatalf("PUT /routing status = %d, want 200: %s", putResponse.Code, putResponse.Body.String())
	}

	createBody := `{"provider_id":"codex","model_id":"model-fast","routing_mode":"automatic"}`
	createRequest := httptest.NewRequest(http.MethodPost, "http://"+host+"/api/v1/conversations", strings.NewReader(createBody))
	createRequest.Header.Set("Authorization", "Bearer "+token)
	createRequest.Header.Set("Origin", "http://"+host)
	createResponse := httptest.NewRecorder()
	handler.ServeHTTP(createResponse, createRequest)
	if createResponse.Code != http.StatusCreated {
		t.Fatalf("POST /conversations status = %d, want 201: %s", createResponse.Code, createResponse.Body.String())
	}
	var conversation Conversation
	if err := json.Unmarshal(createResponse.Body.Bytes(), &conversation); err != nil {
		t.Fatalf("decode created conversation: %v", err)
	}
	if conversation.Selection.RoutingMode != RoutingModeAutomatic {
		t.Fatalf("created routing mode = %q, want automatic", conversation.Selection.RoutingMode)
	}

	summaryRequest := httptest.NewRequest(http.MethodGet, "http://"+host+"/api/v1/conversations?summary=true", nil)
	summaryRequest.Header.Set("Authorization", "Bearer "+token)
	summaryResponse := httptest.NewRecorder()
	handler.ServeHTTP(summaryResponse, summaryRequest)
	if summaryResponse.Code != http.StatusOK {
		t.Fatalf("GET /conversations?summary=true status = %d, want 200: %s", summaryResponse.Code, summaryResponse.Body.String())
	}
	if !strings.Contains(summaryResponse.Body.String(), conversation.ID) || strings.Contains(summaryResponse.Body.String(), `"messages"`) {
		t.Fatalf("conversation summaries response = %s, want metadata without message content", summaryResponse.Body.String())
	}
}

func TestRoutingHTTPRejectsForeignMutationOrigin(t *testing.T) {
	service, _ := newRoutingTestService(t)
	const host = "127.0.0.1:4321"
	handler := NewHTTPHandler(service, "token", host)
	request := httptest.NewRequest(http.MethodPut, "http://"+host+"/api/v1/routing", strings.NewReader(`{"rules":[]}`))
	request.Header.Set("Authorization", "Bearer token")
	request.Header.Set("Origin", "http://localhost:4321")
	request.Header.Set("Sec-Fetch-Site", "cross-site")
	request.Header.Set("Sec-Fetch-Mode", "cors")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("foreign origin status = %d, want 403", response.Code)
	}
}

func TestRoutingHTTPAllowsSameOriginPreviewProxyMutation(t *testing.T) {
	service, _ := newRoutingTestService(t)
	const host = "127.0.0.1:4321"
	handler := NewHTTPHandler(service, "token", host)
	request := httptest.NewRequest(http.MethodPut, "http://"+host+"/api/v1/routing", strings.NewReader("{"))
	request.Header.Set("Authorization", "Bearer token")
	request.Header.Set("Origin", "http://127.0.0.1:32935")
	request.Header.Set("Sec-Fetch-Site", "same-origin")
	request.Header.Set("Sec-Fetch-Mode", "cors")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "invalid_request") {
		t.Fatalf("same-origin proxy request status = %d, want body-validation 400: %s", response.Code, response.Body.String())
	}
}

func TestRoutingHTTPChangesProviderThroughSameOriginPreviewProxy(t *testing.T) {
	service, _ := newRoutingTestService(t)
	service.providers.adapters["antigravity"] = routingTestAdapter{catalog: Catalog{
		ProviderID:   "antigravity",
		Source:       "test-antigravity-catalog",
		DiscoveredAt: time.Now().UTC(),
		Models:       []Model{{ID: "gemini-test", Name: "Gemini Test", Default: true, Selectable: true}},
	}}
	service.providers.quotaMu.Lock()
	service.providers.quotas = map[string]ProviderQuota{
		"antigravity": {Status: "available", CheckedAt: time.Now().UTC()},
	}
	service.providers.quotaMu.Unlock()

	conversation, err := service.CreateConversation(
		context.Background(), "codex", "model-fast", "", "on-request", RoutingModeManual,
	)
	if err != nil {
		t.Fatalf("CreateConversation() error = %v", err)
	}

	const host = "127.0.0.1:4321"
	handler := NewHTTPHandler(service, "token", host)
	body := `{"provider_id":"antigravity","model_id":"gemini-test","approval_policy":"on-request","routing_mode":"manual"}`
	request := httptest.NewRequest(
		http.MethodPatch,
		"http://"+host+"/api/v1/conversations/"+conversation.ID+"/selection",
		strings.NewReader(body),
	)
	request.Header.Set("Authorization", "Bearer token")
	request.Header.Set("Origin", "http://127.0.0.1:32935")
	request.Header.Set("Sec-Fetch-Site", "same-origin")
	request.Header.Set("Sec-Fetch-Mode", "cors")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("PATCH selection status = %d, want 200: %s", response.Code, response.Body.String())
	}

	updated, err := service.Conversation(conversation.ID)
	if err != nil {
		t.Fatalf("Conversation() error = %v", err)
	}
	if updated.Selection.ProviderID != "antigravity" || updated.Selection.ModelID != "gemini-test" {
		t.Fatalf("selection = %+v, want Antigravity / gemini-test", updated.Selection)
	}
}

func TestHTTPServesSandboxedLinkPreview(t *testing.T) {
	service, _ := newRoutingTestService(t)
	const host = "127.0.0.1:4321"
	handler := NewHTTPHandler(service, "token", host)
	for _, tc := range []struct {
		path string
		want string
	}{
		{"/", `sandbox="allow-scripts"`},
		{"/preview.js", "CockpitPreview"},
		{"/quota.js", "CockpitQuota"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "http://"+host+tc.path, nil))
			if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), tc.want) {
				t.Fatalf("GET %s = %d, want preview asset containing %q", tc.path, response.Code, tc.want)
			}
			policy := response.Header().Get("Content-Security-Policy")
			for _, directive := range []string{"frame-src http: https:", "script-src 'self'", "connect-src 'self'", "frame-ancestors 'none'"} {
				if !strings.Contains(policy, directive) {
					t.Errorf("CSP missing %q", directive)
				}
			}
		})
	}
}

func TestHTTPServesMarkdownRendererAndAllowsInlineImagePreviews(t *testing.T) {
	service, _ := newRoutingTestService(t)
	const host = "127.0.0.1:4321"
	handler := NewHTTPHandler(service, "token", host)

	pageRequest := httptest.NewRequest(http.MethodGet, "http://"+host+"/", nil)
	pageResponse := httptest.NewRecorder()
	handler.ServeHTTP(pageResponse, pageRequest)
	if pageResponse.Code != http.StatusOK || !strings.Contains(pageResponse.Body.String(), `src="./markdown.js"`) {
		t.Fatalf("GET / response = %d, want index including Markdown renderer", pageResponse.Code)
	}

	rendererRequest := httptest.NewRequest(http.MethodGet, "http://"+host+"/markdown.js", nil)
	rendererResponse := httptest.NewRecorder()
	handler.ServeHTTP(rendererResponse, rendererRequest)
	if rendererResponse.Code != http.StatusOK || !strings.Contains(rendererResponse.Body.String(), "CockpitMarkdown") {
		t.Fatalf("GET /markdown.js response = %d, want served Markdown renderer", rendererResponse.Code)
	}

	policy := rendererResponse.Header().Get("Content-Security-Policy")
	if !strings.Contains(policy, "img-src 'self' data:") {
		t.Fatalf("Content-Security-Policy = %q, want data images enabled for local previews", policy)
	}

	interactionRequest := httptest.NewRequest(http.MethodGet, "http://"+host+"/interaction.js", nil)
	interactionResponse := httptest.NewRecorder()
	handler.ServeHTTP(interactionResponse, interactionRequest)
	if interactionResponse.Code != http.StatusOK || !strings.Contains(interactionResponse.Body.String(), "CockpitInteractions") {
		t.Fatalf("GET /interaction.js response = %d, want served interaction helpers", interactionResponse.Code)
	}
}
