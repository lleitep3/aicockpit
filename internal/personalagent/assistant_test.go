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

type assistantToolAdapter struct {
	catalog   Catalog
	tool      string
	args      json.RawMessage
	toolReply chan string
	requests  chan RunRequest
	approvals chan string
}

func (a *assistantToolAdapter) Catalog(context.Context) (Catalog, error) {
	return a.catalog, nil
}

func (a *assistantToolAdapter) RunTurn(
	ctx context.Context,
	request RunRequest,
	onDelta func(string),
	onApproval func(ApprovalRequest) (string, error),
) (RunResult, error) {
	if request.OnToolCall != nil {
		a.requests <- request
		decision, err := onApproval(ApprovalRequest{Method: "item/commandExecution/requestApproval"})
		if err != nil {
			return RunResult{}, err
		}
		a.approvals <- decision
		result, err := request.OnToolCall(ctx, a.tool, a.args)
		if err != nil {
			return RunResult{}, err
		}
		a.toolReply <- result
	}
	onDelta("done")
	return RunResult{NativeSessionID: "test-session", NativeTurnID: "test-turn"}, nil
}

func newAssistantTestService(t *testing.T, tool string, arguments string) (*Service, *assistantToolAdapter) {
	t.Helper()
	base := t.TempDir()
	store, err := NewStoreWithProjectsDir(filepath.Join(base, "conversations"), filepath.Join(base, "projects"))
	if err != nil {
		t.Fatalf("NewStoreWithProjectsDir() error = %v", err)
	}
	adapter := &assistantToolAdapter{
		catalog: Catalog{
			ProviderID: "codex",
			Source:     "assistant-test",
			Models: []Model{
				{ID: "model-fast", Name: "Fast", Default: true, Selectable: true},
			},
		},
		tool:      tool,
		args:      json.RawMessage(arguments),
		toolReply: make(chan string, 1),
		requests:  make(chan RunRequest, 1),
		approvals: make(chan string, 1),
	}
	providers := &ProviderRegistry{adapters: map[string]Adapter{"codex": adapter}}
	service, err := NewService(store, providers, t.TempDir())
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return service, adapter
}

func waitForAssistantAction(t *testing.T, service *Service, assistantID string) AssistantAction {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		conversation, err := service.Conversation(assistantID)
		if err == nil && len(conversation.AssistantActions) > 0 {
			return conversation.AssistantActions[0]
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("assistant action was not persisted")
	return AssistantAction{}
}

func TestAssistantActionRequiresDecisionAndIsSingleUse(t *testing.T) {
	tests := []struct {
		name            string
		decision        string
		wantStatus      string
		wantDestination int
	}{
		{name: "approval sends exactly one turn", decision: "approve", wantStatus: "queued", wantDestination: 1},
		{name: "rejection sends no turn", decision: "reject", wantStatus: "rejected", wantDestination: 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service, adapter := newAssistantTestService(
				t, "propose_send_message",
				`{"conversation_id":"destination","content":"Approved text"}`,
			)
			assistant, err := service.CreateAssistantSession(context.Background(), "model-fast", "")
			if err != nil {
				t.Fatalf("CreateAssistantSession() error = %v", err)
			}
			destination, err := service.CreateConversation(
				context.Background(), "codex", "model-fast", "", "on-request", RoutingModeManual,
			)
			if err != nil {
				t.Fatalf("CreateConversation() error = %v", err)
			}
			adapter.args, err = json.Marshal(map[string]string{
				"conversation_id": destination.ID,
				"content":         "Approved text",
			})
			if err != nil {
				t.Fatalf("encode fixture tool arguments: %v", err)
			}
			if _, err := service.StartTurn(assistant.ID, "Send a message", nil); err != nil {
				t.Fatalf("StartTurn() error = %v", err)
			}
			request := <-adapter.requests
			if request.Sandbox != "read-only" || request.Workspace == service.Workspace() {
				t.Fatalf("assistant execution boundary = sandbox %q workspace %q", request.Sandbox, request.Workspace)
			}
			workspaceInfo, err := os.Stat(request.Workspace)
			if err != nil || workspaceInfo.Mode().Perm() != 0o700 {
				t.Fatalf("assistant workspace mode = %v, error = %v; want 0700", workspaceInfo, err)
			}
			if decision := <-adapter.approvals; decision != "decline" {
				t.Fatalf("assistant command approval = %q, want decline", decision)
			}
			action := waitForAssistantAction(t, service, assistant.ID)
			before, err := service.Conversation(destination.ID)
			if err != nil {
				t.Fatalf("load destination before decision: %v", err)
			}
			if len(before.Turns) != 0 {
				t.Fatalf("destination turns before decision = %d, want 0", len(before.Turns))
			}
			if err := service.DecideAssistantAction(assistant.ID, action.ID, test.decision); err != nil {
				t.Fatalf("DecideAssistantAction() error = %v", err)
			}
			if err := service.DecideAssistantAction(assistant.ID, action.ID, test.decision); err == nil {
				t.Fatal("repeated decision unexpectedly succeeded")
			}
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if err := service.Shutdown(ctx); err != nil {
				t.Fatalf("Shutdown() error = %v", err)
			}
			after, err := service.Conversation(destination.ID)
			if err != nil {
				t.Fatalf("load destination after decision: %v", err)
			}
			if got := len(after.Turns); got != test.wantDestination {
				t.Fatalf("destination turns = %d, want %d", got, test.wantDestination)
			}
			if test.wantDestination == 1 && after.Messages[0].Content != "Approved text" {
				t.Fatalf("sent message = %q, want exact approved text", after.Messages[0].Content)
			}
			afterAssistant, err := service.Conversation(assistant.ID)
			if err != nil {
				t.Fatalf("reload assistant session: %v", err)
			}
			if got := afterAssistant.AssistantActions[0].Status; got != test.wantStatus {
				t.Fatalf("action status = %q, want %q", got, test.wantStatus)
			}
			if hash := afterAssistant.AssistantActions[0].PayloadHash; len(hash) != 64 {
				t.Fatalf("action payload hash length = %d, want SHA-256 hex", len(hash))
			}
			select {
			case <-adapter.toolReply:
			default:
				t.Fatal("provider did not receive tool result")
			}
		})
	}
}

func TestEffectiveSelectionIgnoresCatalogRefreshTimestamp(t *testing.T) {
	first := Selection{
		ProviderID: "codex", ModelID: "model-fast", ReasoningEffort: "low",
		ApprovalPolicy: "on-request", RoutingMode: RoutingModeAutomatic,
		CatalogSource: "codex-app-server", CatalogDiscoveredAt: time.Now().UTC(),
	}
	refreshed := first
	refreshed.CatalogDiscoveredAt = first.CatalogDiscoveredAt.Add(time.Second)
	if !sameEffectiveSelection(first, refreshed) {
		t.Fatal("catalog refresh timestamp invalidated an unchanged effective selection")
	}
	refreshed.ModelID = "model-deep"
	if sameEffectiveSelection(first, refreshed) {
		t.Fatal("changed model was treated as the same effective selection")
	}
}

func TestAssistantToolsRejectUnknownFieldsAndAssistantTranscriptReads(t *testing.T) {
	if err := decodeToolArguments(json.RawMessage(`{"conversation_id":"x","unexpected":true}`), &struct {
		ConversationID string `json:"conversation_id"`
	}{}); err == nil {
		t.Fatal("decodeToolArguments() accepted an unknown field")
	}
	service, _ := newAssistantTestService(t, "read_conversation", `{"conversation_id":"missing"}`)
	assistant, err := service.CreateAssistantSession(context.Background(), "model-fast", "")
	if err != nil {
		t.Fatalf("CreateAssistantSession() error = %v", err)
	}
	if _, err := service.runAssistantTool(context.Background(), assistant.ID, "turn_test", "read_conversation", json.RawMessage(`{"conversation_id":"`+assistant.ID+`"}`)); err == nil {
		t.Fatal("read_conversation() exposed an assistant transcript")
	}
	assistant, err = service.Conversation(assistant.ID)
	if err != nil {
		t.Fatalf("reload assistant session: %v", err)
	}
	if len(assistant.AssistantAudit) != 1 || assistant.AssistantAudit[0].Status != "failed" ||
		assistant.AssistantAudit[0].Error == "" {
		t.Fatalf("invalid tool audit = %+v, want one failed event", assistant.AssistantAudit)
	}
}

func TestAssistantApprovalInvalidatesWhenDestinationSelectionChanges(t *testing.T) {
	service, adapter := newAssistantTestService(
		t, "propose_send_message",
		`{"content":"Approved text"}`,
	)
	assistant, err := service.CreateAssistantSession(context.Background(), "model-fast", "")
	if err != nil {
		t.Fatalf("CreateAssistantSession() error = %v", err)
	}
	destination, err := service.CreateConversation(context.Background(), "codex", "model-fast", "", "", "")
	if err != nil {
		t.Fatalf("CreateConversation() error = %v", err)
	}
	arguments, err := json.Marshal(map[string]string{
		"conversation_id": destination.ID,
		"content":         "Approved text",
	})
	if err != nil {
		t.Fatalf("encode fixture arguments: %v", err)
	}
	// The provider adapter reads its request payload from the fixture instance.
	adapter.args = arguments
	if _, err := service.StartTurn(assistant.ID, "Send a message", nil); err != nil {
		t.Fatalf("StartTurn() error = %v", err)
	}
	action := waitForAssistantAction(t, service, assistant.ID)
	if _, err := service.UpdateSelection(
		context.Background(), destination.ID, "codex", "model-fast", "", "never", RoutingModeManual,
	); err != nil {
		t.Fatalf("UpdateSelection() error = %v", err)
	}
	if err := service.DecideAssistantAction(assistant.ID, action.ID, "approve"); err == nil {
		t.Fatal("approval unexpectedly succeeded after destination settings changed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := service.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	after, err := service.Conversation(destination.ID)
	if err != nil {
		t.Fatalf("reload destination: %v", err)
	}
	if len(after.Turns) != 0 {
		t.Fatalf("destination turns = %d, want 0", len(after.Turns))
	}
	afterAssistant, err := service.Conversation(assistant.ID)
	if err != nil {
		t.Fatalf("reload assistant session: %v", err)
	}
	if got := afterAssistant.AssistantActions[0].Status; got != "invalidated" {
		t.Fatalf("action status = %q, want invalidated", got)
	}
}

func TestAssistantActionExpiryCannotSendMessage(t *testing.T) {
	service, adapter := newAssistantTestService(
		t, "propose_send_message",
		`{}`,
	)
	assistant, err := service.CreateAssistantSession(context.Background(), "model-fast", "")
	if err != nil {
		t.Fatalf("CreateAssistantSession() error = %v", err)
	}
	destination, err := service.CreateConversation(context.Background(), "codex", "model-fast", "", "", "")
	if err != nil {
		t.Fatalf("CreateConversation() error = %v", err)
	}
	adapter.args, err = json.Marshal(map[string]string{
		"conversation_id": destination.ID,
		"content":         "Expired text",
	})
	if err != nil {
		t.Fatalf("encode fixture arguments: %v", err)
	}
	if _, err := service.StartTurn(assistant.ID, "Send a message", nil); err != nil {
		t.Fatalf("StartTurn() error = %v", err)
	}
	action := waitForAssistantAction(t, service, assistant.ID)
	if err := service.updateAssistantAction(assistant.ID, action.ID, func(item *AssistantAction) {
		item.ExpiresAt = time.Now().UTC().Add(-time.Second)
	}); err != nil {
		t.Fatalf("expire fixture action: %v", err)
	}
	if err := service.DecideAssistantAction(assistant.ID, action.ID, "approve"); err == nil {
		t.Fatal("expired action approval unexpectedly succeeded")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := service.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	after, err := service.Conversation(destination.ID)
	if err != nil {
		t.Fatalf("reload destination: %v", err)
	}
	if len(after.Turns) != 0 {
		t.Fatalf("expired action created %d destination turn(s)", len(after.Turns))
	}
	afterAssistant, err := service.Conversation(assistant.ID)
	if err != nil {
		t.Fatalf("reload assistant session: %v", err)
	}
	if afterAssistant.AssistantActions[0].Status != "expired" {
		t.Fatalf("expired action status = %q", afterAssistant.AssistantActions[0].Status)
	}
}

func TestAssistantReadAuditsMessageIDsWithoutCopyingTranscript(t *testing.T) {
	service, _ := newAssistantTestService(t, "read_conversation", `{}`)
	assistant, err := service.CreateAssistantSession(context.Background(), "model-fast", "")
	if err != nil {
		t.Fatalf("CreateAssistantSession() error = %v", err)
	}
	destination, err := service.CreateConversation(context.Background(), "codex", "model-fast", "", "", "")
	if err != nil {
		t.Fatalf("CreateConversation() error = %v", err)
	}
	destination.Messages = []Message{{ID: "message-1", Role: "user", Content: "Private transcript text"}}
	if err := service.store.Save(destination); err != nil {
		t.Fatalf("save transcript fixture: %v", err)
	}
	result, err := service.runAssistantTool(
		context.Background(), assistant.ID, "turn_test", "read_conversation",
		json.RawMessage(`{"conversation_id":"`+destination.ID+`"}`),
	)
	if err != nil {
		t.Fatalf("runAssistantTool() error = %v", err)
	}
	var transcript struct {
		Messages []struct {
			ID      string `json:"id"`
			Content string `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal([]byte(result), &transcript); err != nil {
		t.Fatalf("decode transcript tool result: %v", err)
	}
	if len(transcript.Messages) != 1 || transcript.Messages[0].ID != "message-1" {
		t.Fatalf("transcript tool result = %+v", transcript)
	}
	assistant, err = service.Conversation(assistant.ID)
	if err != nil {
		t.Fatalf("reload assistant session: %v", err)
	}
	audit := assistant.AssistantAudit[len(assistant.AssistantAudit)-1]
	if len(audit.MessageIDs) != 1 || audit.MessageIDs[0] != "message-1" {
		t.Fatalf("read audit = %+v, want message ID only", audit)
	}
	if strings.Contains(audit.Tool, "Private transcript text") || strings.Contains(string(mustJSON(audit)), "Private transcript text") {
		t.Fatal("audit copied transcript content")
	}
}

func TestAssistantHTTPCreateAndAuthorization(t *testing.T) {
	service, adapter := newAssistantTestService(t, "propose_send_message", `{}`)
	const host = "127.0.0.1:4321"
	const token = "assistant-test-token"
	handler := NewHTTPHandler(service, token, host)
	destination, err := service.CreateConversation(context.Background(), "codex", "model-fast", "", "", "")
	if err != nil {
		t.Fatalf("CreateConversation() error = %v", err)
	}
	adapter.args, err = json.Marshal(map[string]string{
		"conversation_id": destination.ID,
		"content":         "HTTP-approved text",
	})
	if err != nil {
		t.Fatalf("encode fixture arguments: %v", err)
	}

	providersRequest := httptest.NewRequest(http.MethodGet, "http://"+host+"/api/v1/assistant/providers", nil)
	providersRequest.Header.Set("Authorization", "Bearer "+token)
	providersResponse := httptest.NewRecorder()
	handler.ServeHTTP(providersResponse, providersRequest)
	if providersResponse.Code != http.StatusOK || !strings.Contains(providersResponse.Body.String(), `"tools"`) {
		t.Fatalf("GET assistant providers = %d %s", providersResponse.Code, providersResponse.Body.String())
	}

	createRequest := httptest.NewRequest(
		http.MethodPost,
		"http://"+host+"/api/v1/assistant/sessions",
		strings.NewReader(`{"model_id":"model-fast"}`),
	)
	createRequest.Header.Set("Authorization", "Bearer "+token)
	createRequest.Header.Set("Origin", "http://"+host)
	createResponse := httptest.NewRecorder()
	handler.ServeHTTP(createResponse, createRequest)
	if createResponse.Code != http.StatusCreated || !strings.Contains(createResponse.Body.String(), `"assistant_mode":true`) {
		t.Fatalf("POST assistant sessions = %d %s", createResponse.Code, createResponse.Body.String())
	}
	var created Conversation
	if err := json.Unmarshal(createResponse.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode assistant session: %v", err)
	}

	messageRequest := httptest.NewRequest(
		http.MethodPost,
		"http://"+host+"/api/v1/assistant/sessions/"+created.ID+"/messages",
		strings.NewReader(`{"content":"Analyze this image","attachments":[{"name":"capture.png","media_type":"image/png","content":"iVBORw0KGgo="}]}`),
	)
	messageRequest.Header.Set("Authorization", "Bearer "+token)
	messageRequest.Header.Set("Origin", "http://"+host)
	messageResponse := httptest.NewRecorder()
	handler.ServeHTTP(messageResponse, messageRequest)
	if messageResponse.Code != http.StatusAccepted {
		t.Fatalf("POST assistant message = %d %s", messageResponse.Code, messageResponse.Body.String())
	}
	runRequest := <-adapter.requests
	if len(runRequest.Attachments) != 1 || runRequest.Attachments[0].MediaType != "image/png" {
		t.Fatalf("assistant RunRequest attachments = %+v, want the submitted image", runRequest.Attachments)
	}
	action := waitForAssistantAction(t, service, created.ID)
	decisionRequest := httptest.NewRequest(
		http.MethodPost,
		"http://"+host+"/api/v1/assistant/sessions/"+created.ID+"/actions/"+action.ID+"/decision",
		strings.NewReader(`{"decision":"approve"}`),
	)
	decisionRequest.Header.Set("Authorization", "Bearer "+token)
	decisionRequest.Header.Set("Origin", "http://"+host)
	decisionResponse := httptest.NewRecorder()
	handler.ServeHTTP(decisionResponse, decisionRequest)
	if decisionResponse.Code != http.StatusAccepted {
		t.Fatalf("POST assistant action decision = %d %s", decisionResponse.Code, decisionResponse.Body.String())
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := service.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	sent, err := service.Conversation(destination.ID)
	if err != nil || len(sent.Turns) != 1 || sent.Messages[0].Content != "HTTP-approved text" {
		t.Fatalf("destination after HTTP-approved action = turns:%d messages:%+v error:%v", len(sent.Turns), sent.Messages, err)
	}

	unauthorized := httptest.NewRequest(http.MethodGet, "http://"+host+"/api/v1/assistant/providers", nil)
	unauthorizedResponse := httptest.NewRecorder()
	handler.ServeHTTP(unauthorizedResponse, unauthorized)
	if unauthorizedResponse.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized assistant providers = %d, want 401", unauthorizedResponse.Code)
	}
}

func TestAssistantHTTPCancelsRunningTurn(t *testing.T) {
	service, adapter := newAssistantTestService(t, "propose_send_message", `{}`)
	const host = "127.0.0.1:4321"
	const token = "assistant-test-token"
	handler := NewHTTPHandler(service, token, host)
	session, err := service.CreateAssistantSession(context.Background(), "model-fast", "")
	if err != nil {
		t.Fatalf("CreateAssistantSession() error = %v", err)
	}
	destination, err := service.CreateConversation(context.Background(), "codex", "model-fast", "", "on-request", RoutingModeManual)
	if err != nil {
		t.Fatalf("CreateConversation() error = %v", err)
	}
	adapter.args, err = json.Marshal(map[string]string{
		"conversation_id": destination.ID,
		"content":         "Proposed text",
	})
	if err != nil {
		t.Fatalf("encode fixture arguments: %v", err)
	}

	messageRequest := httptest.NewRequest(
		http.MethodPost,
		"http://"+host+"/api/v1/assistant/sessions/"+session.ID+"/messages",
		strings.NewReader(`{"content":"Start a turn"}`),
	)
	messageRequest.Header.Set("Authorization", "Bearer "+token)
	messageRequest.Header.Set("Origin", "http://"+host)
	messageResponse := httptest.NewRecorder()
	handler.ServeHTTP(messageResponse, messageRequest)
	if messageResponse.Code != http.StatusAccepted {
		t.Fatalf("POST assistant message = %d %s", messageResponse.Code, messageResponse.Body.String())
	}
	var result struct {
		TurnID string `json:"turn_id"`
	}
	if err := json.Unmarshal(messageResponse.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode started assistant turn: %v", err)
	}
	waitForAssistantAction(t, service, session.ID)

	cancelRequest := httptest.NewRequest(
		http.MethodPost,
		"http://"+host+"/api/v1/assistant/sessions/"+session.ID+"/turns/"+result.TurnID+"/cancel",
		strings.NewReader(`{}`),
	)
	cancelRequest.Header.Set("Authorization", "Bearer "+token)
	cancelRequest.Header.Set("Origin", "http://"+host)
	cancelResponse := httptest.NewRecorder()
	handler.ServeHTTP(cancelResponse, cancelRequest)
	if cancelResponse.Code != http.StatusAccepted {
		t.Fatalf("POST assistant turn cancellation = %d %s", cancelResponse.Code, cancelResponse.Body.String())
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := service.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	conversation, err := service.Conversation(session.ID)
	if err != nil || len(conversation.Turns) != 1 || conversation.Turns[0].Status != "cancelled" {
		t.Fatalf("cancelled assistant conversation = %+v, error = %v", conversation, err)
	}
}
