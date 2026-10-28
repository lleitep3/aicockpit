package personalagent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

type captureTurnAdapter struct {
	catalog    Catalog
	lastPrompt string
	lastWork   string
	aiResponse string
}

func (a *captureTurnAdapter) Catalog(context.Context) (Catalog, error) {
	return a.catalog, nil
}

func (a *captureTurnAdapter) RunTurn(_ context.Context, req RunRequest, onDelta func(string), _ func(ApprovalRequest) (string, error)) (RunResult, error) {
	a.lastPrompt = req.Prompt
	a.lastWork = req.Workspace
	if a.aiResponse != "" {
		onDelta(a.aiResponse)
	} else {
		onDelta("ok")
	}
	return RunResult{NativeSessionID: "sess_1", NativeTurnID: "nturn_1"}, nil
}

func TestProjectLifecycleAndConversationBinding(t *testing.T) {
	service, _ := newRoutingTestService(t)
	projectWorkspace := t.TempDir()

	project, err := service.CreateProject(context.Background(), ProjectInput{
		Name:            "Cockpit Core",
		Description:     "CLI e runtime local",
		Workspace:       projectWorkspace,
		Repositories:    []string{"/home/lleite/projects/aicockpit", "https://github.com/lleitep3/aicockpit"},
		Instructions:    "Sempre rode go test antes de finalizar.",
		ProviderID:      "codex",
		ModelID:         "model-deep",
		ReasoningEffort: "high",
		ApprovalPolicy:  "never",
		RoutingMode:     RoutingModeManual,
	})
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	if project.Name != "Cockpit Core" || len(project.Repositories) != 2 {
		t.Fatalf("created project = %#v", project)
	}

	conv, err := service.CreateConversationInProject(context.Background(), project.ID, "", "", "", "", "")
	if err != nil {
		t.Fatalf("CreateConversationInProject() error = %v", err)
	}
	if conv.ProjectID != project.ID {
		t.Fatalf("conv.ProjectID = %q, want %q", conv.ProjectID, project.ID)
	}
	if conv.Workspace != project.Workspace {
		t.Fatalf("conv.Workspace = %q, want %q", conv.Workspace, project.Workspace)
	}
	if conv.Selection.ModelID != "model-deep" || conv.Selection.ReasoningEffort != "high" || conv.Selection.ApprovalPolicy != "never" {
		t.Fatalf("conv.Selection = %#v, want project defaults", conv.Selection)
	}

	updated, err := service.UpdateProject(context.Background(), project.ID, ProjectInput{
		Name:         "Cockpit Core v2",
		Description:  "Atualizado",
		Instructions: "Use TDD",
	})
	if err != nil {
		t.Fatalf("UpdateProject() error = %v", err)
	}
	if updated.Name != "Cockpit Core v2" {
		t.Fatalf("updated.Name = %q, want Cockpit Core v2", updated.Name)
	}

	if err := service.DeleteProject(project.ID); err != nil {
		t.Fatalf("DeleteProject() error = %v", err)
	}
	reloadedConv, err := service.Conversation(conv.ID)
	if err != nil {
		t.Fatalf("Conversation() error = %v", err)
	}
	if reloadedConv.ProjectID != "" {
		t.Fatalf("reloadedConv.ProjectID = %q, want empty after project deletion", reloadedConv.ProjectID)
	}
}

func TestConversationSummariesIncludeMessageCounts(t *testing.T) {
	service, store := newRoutingTestService(t)
	empty, err := service.CreateConversation(context.Background(), "codex", "model-fast", "", "on-request", RoutingModeManual)
	if err != nil {
		t.Fatalf("CreateConversation(empty) error = %v", err)
	}
	active, err := service.CreateConversation(context.Background(), "codex", "model-fast", "", "on-request", RoutingModeManual)
	if err != nil {
		t.Fatalf("CreateConversation(active) error = %v", err)
	}
	active.Messages = []Message{{ID: "msg_1", Role: "user", Content: "hello"}}
	if err := store.Save(active); err != nil {
		t.Fatalf("Save(active) error = %v", err)
	}

	summaries, err := service.ConversationSummaries()
	if err != nil {
		t.Fatalf("ConversationSummaries() error = %v", err)
	}
	counts := make(map[string]int, len(summaries))
	for _, summary := range summaries {
		counts[summary.ID] = summary.MessageCount
	}
	if len(summaries) != 2 || counts[empty.ID] != 0 || counts[active.ID] != 1 {
		t.Fatalf("conversation summary message counts = %#v, want empty=0 and active=1", counts)
	}
}

func TestStartTurnInjectsProjectContextAndAIDraft(t *testing.T) {
	baseDir := t.TempDir()
	store, err := NewStoreWithProjectsDir(filepath.Join(baseDir, "conversations"), filepath.Join(baseDir, "projects"))
	if err != nil {
		t.Fatalf("NewStoreWithProjectsDir() error = %v", err)
	}
	defaultWorkspace := t.TempDir()
	projectWorkspace := t.TempDir()
	adapter := &captureTurnAdapter{
		catalog: Catalog{
			ProviderID: "codex",
			Source:     "test",
			Models: []Model{
				{ID: "model-fast", Name: "Fast", Default: true, Selectable: true},
			},
		},
		aiResponse: `{"name":"API Pagamentos","description":"Serviço de cobrança","workspace":"","repositories":["https://github.com/org/pay"],"instructions":"Siga Clean Architecture e valide idempotência."}`,
	}
	providers := &ProviderRegistry{adapters: map[string]Adapter{"codex": adapter}}
	service, err := NewService(store, providers, defaultWorkspace)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	draft, err := service.DraftProjectWithAI(context.Background(), "Criar serviço de pagamentos com idempotência", "codex", "model-fast", "")
	if err != nil {
		t.Fatalf("DraftProjectWithAI() error = %v", err)
	}
	if draft.Name != "API Pagamentos" || len(draft.Repositories) != 1 || !strings.Contains(draft.Instructions, "idempotência") {
		t.Fatalf("DraftProjectWithAI() = %#v", draft)
	}

	draft.Workspace = projectWorkspace
	project, err := service.CreateProject(context.Background(), draft)
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	conv, err := service.CreateConversationInProject(context.Background(), project.ID, "codex", "model-fast", "", "on-request", "manual")
	if err != nil {
		t.Fatalf("CreateConversationInProject() error = %v", err)
	}

	adapter.aiResponse = "resposta final"
	if _, err := service.StartTurn(conv.ID, "Crie o endpoint POST /charges", nil); err != nil {
		t.Fatalf("StartTurn() error = %v", err)
	}
	if err := service.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}

	if adapter.lastWork != project.Workspace {
		t.Fatalf("turn workspace = %q, want %q", adapter.lastWork, project.Workspace)
	}
	if !strings.Contains(adapter.lastPrompt, "[Contexto do Projeto Cockpit]") ||
		!strings.Contains(adapter.lastPrompt, "API Pagamentos") ||
		!strings.Contains(adapter.lastPrompt, "https://github.com/org/pay") ||
		!strings.Contains(adapter.lastPrompt, "Siga Clean Architecture") {
		t.Fatalf("turn prompt missing project context: %s", adapter.lastPrompt)
	}
}

func TestProjectHTTPHandlers(t *testing.T) {
	service, _ := newRoutingTestService(t)
	const host = "127.0.0.1:4321"
	const token = "test-session-token"
	handler := NewHTTPHandler(service, token, host)

	createReq := httptest.NewRequest(http.MethodPost, "http://"+host+"/api/v1/projects", strings.NewReader(`{"name":"Novo projeto"}`))
	createReq.Header.Set("Authorization", "Bearer "+token)
	createReq.Header.Set("Origin", "http://"+host)
	createRec := httptest.NewRecorder()
	handler.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("POST /projects status = %d, want 201: %s", createRec.Code, createRec.Body.String())
	}
	var created Project
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode project: %v", err)
	}

	patchReq := httptest.NewRequest(
		http.MethodPatch,
		"http://"+host+"/api/v1/projects/"+created.ID,
		strings.NewReader(`{"name":"Projeto Renomeado","repositories":["repo-a"],"instructions":"Use Go 1.26"}`),
	)
	patchReq.Header.Set("Authorization", "Bearer "+token)
	patchReq.Header.Set("Origin", "http://"+host)
	patchRec := httptest.NewRecorder()
	handler.ServeHTTP(patchRec, patchReq)
	if patchRec.Code != http.StatusOK {
		t.Fatalf("PATCH /projects/:id status = %d, want 200: %s", patchRec.Code, patchRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "http://"+host+"/api/v1/projects", nil)
	listReq.Header.Set("Authorization", "Bearer "+token)
	listRec := httptest.NewRecorder()
	handler.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK || !strings.Contains(listRec.Body.String(), "Projeto Renomeado") {
		t.Fatalf("GET /projects = %d %s", listRec.Code, listRec.Body.String())
	}
}

func TestConversationRootGroupingAndParentAssignment(t *testing.T) {
	service, _ := newRoutingTestService(t)
	const host = "127.0.0.1:4321"
	const token = "test-session-token"
	handler := NewHTTPHandler(service, token, host)

	project, err := service.CreateProject(context.Background(), ProjectInput{
		Name: "Cockpit Evolution",
	})
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	root, err := service.CreateConversationWithParent(
		context.Background(), project.ID, "", "Conversa Raiz", "codex", "model-fast", "", "on-request", RoutingModeManual,
	)
	if err != nil {
		t.Fatalf("CreateConversationWithParent(root) error = %v", err)
	}
	if root.Title != "Conversa Raiz" || root.RootConversationID != "" {
		t.Fatalf("unexpected root conversation: %+v", root)
	}

	spawnBody := `{"parent_conversation_id":"` + root.ID + `","title":"Sub-sessão 1","provider_id":"codex","model_id":"model-fast"}`
	spawnReq := httptest.NewRequest(http.MethodPost, "http://"+host+"/api/v1/conversations", strings.NewReader(spawnBody))
	spawnReq.Header.Set("Authorization", "Bearer "+token)
	spawnReq.Header.Set("Origin", "http://"+host)
	spawnRec := httptest.NewRecorder()
	handler.ServeHTTP(spawnRec, spawnReq)
	if spawnRec.Code != http.StatusCreated {
		t.Fatalf("POST /conversations (spawn) status = %d: %s", spawnRec.Code, spawnRec.Body.String())
	}
	var child1 Conversation
	if err := json.Unmarshal(spawnRec.Body.Bytes(), &child1); err != nil {
		t.Fatalf("decode child1: %v", err)
	}
	if child1.ParentConversationID != root.ID || child1.RootConversationID != root.ID || child1.ProjectID != project.ID {
		t.Fatalf("child1 linkage = parent:%q root:%q project:%q", child1.ParentConversationID, child1.RootConversationID, child1.ProjectID)
	}

	grandChild, err := service.CreateConversationWithParent(
		context.Background(), "", child1.ID, "Sub-sessão 2", "codex", "model-fast", "", "on-request", RoutingModeManual,
	)
	if err != nil {
		t.Fatalf("CreateConversationWithParent(grandChild) error = %v", err)
	}
	if grandChild.ParentConversationID != child1.ID || grandChild.RootConversationID != root.ID {
		t.Fatalf("grandChild linkage = parent:%q root:%q, want parent %q root %q", grandChild.ParentConversationID, grandChild.RootConversationID, child1.ID, root.ID)
	}

	if _, err := service.AssignConversationParent(root.ID, root.ID); err == nil {
		t.Fatal("expected error when assigning conversation as its own parent")
	}
	if _, err := service.AssignConversationParent(root.ID, child1.ID); err == nil {
		t.Fatal("expected error when creating circular parent hierarchy")
	}

	standalone, err := service.CreateConversation(context.Background(), "codex", "model-fast", "", "on-request", RoutingModeManual)
	if err != nil {
		t.Fatalf("CreateConversation(standalone) error = %v", err)
	}
	patchReq := httptest.NewRequest(
		http.MethodPatch,
		"http://"+host+"/api/v1/conversations/"+standalone.ID+"/parent",
		strings.NewReader(`{"parent_conversation_id":"`+root.ID+`"}`),
	)
	patchReq.Header.Set("Authorization", "Bearer "+token)
	patchReq.Header.Set("Origin", "http://"+host)
	patchRec := httptest.NewRecorder()
	handler.ServeHTTP(patchRec, patchReq)
	if patchRec.Code != http.StatusOK {
		t.Fatalf("PATCH /conversations/:id/parent status = %d: %s", patchRec.Code, patchRec.Body.String())
	}

	summaries, err := service.ConversationSummaries()
	if err != nil {
		t.Fatalf("ConversationSummaries() error = %v", err)
	}
	linkedCount := 0
	for _, summary := range summaries {
		if summary.RootConversationID == root.ID {
			linkedCount++
		}
	}
	if linkedCount != 3 {
		t.Fatalf("summaries with RootConversationID %q = %d, want 3", root.ID, linkedCount)
	}
}

func TestArchiveConversationAndHTTP(t *testing.T) {
	service, _ := newRoutingTestService(t)
	const host = "127.0.0.1:4321"
	const token = "test-session-token"
	handler := NewHTTPHandler(service, token, host)

	conv, err := service.CreateConversation(context.Background(), "codex", "model-fast", "", "on-request", RoutingModeManual)
	if err != nil {
		t.Fatalf("CreateConversation() error = %v", err)
	}
	originalUpdatedAt := conv.UpdatedAt

	archiveReq := httptest.NewRequest(
		http.MethodPatch,
		"http://"+host+"/api/v1/conversations/"+conv.ID+"/archive",
		strings.NewReader(`{"archived":true}`),
	)
	archiveReq.Header.Set("Authorization", "Bearer "+token)
	archiveReq.Header.Set("Origin", "http://"+host)
	archiveRec := httptest.NewRecorder()
	handler.ServeHTTP(archiveRec, archiveReq)
	if archiveRec.Code != http.StatusOK {
		t.Fatalf("PATCH /conversations/:id/archive status = %d: %s", archiveRec.Code, archiveRec.Body.String())
	}

	var archivedConv Conversation
	if err := json.Unmarshal(archiveRec.Body.Bytes(), &archivedConv); err != nil {
		t.Fatalf("decode archived conversation: %v", err)
	}
	if !archivedConv.Archived {
		t.Fatalf("expected conversation to be archived, got %+v", archivedConv)
	}
	if !archivedConv.UpdatedAt.Equal(originalUpdatedAt) {
		t.Fatalf("expected UpdatedAt to remain %v, got %v", originalUpdatedAt, archivedConv.UpdatedAt)
	}

	summaries, err := service.ConversationSummaries()
	if err != nil {
		t.Fatalf("ConversationSummaries() error = %v", err)
	}
	if len(summaries) != 1 || !summaries[0].Archived {
		t.Fatalf("expected summary Archived=true, got %+v", summaries)
	}

	unarchiveConv, err := service.ArchiveConversation(conv.ID, false)
	if err != nil {
		t.Fatalf("ArchiveConversation(false) error = %v", err)
	}
	if unarchiveConv.Archived {
		t.Fatalf("expected conversation to be unarchived, got %+v", unarchiveConv)
	}
}
