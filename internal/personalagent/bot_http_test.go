package personalagent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type botHTTPFixture struct {
	sendCalls int
	bot       map[string]any
}

func (f *botHTTPFixture) Save(context.Context, any) error { return nil }
func (f *botHTTPFixture) Load(context.Context, string) (any, error) {
	return f.bot, nil
}
func (f *botHTTPFixture) List(context.Context) ([]any, error) {
	return []any{f.bot}, nil
}
func (f *botHTTPFixture) Delete(context.Context, string) error { return nil }
func (f *botHTTPFixture) CreateBot(_ context.Context, profile any) (any, error) {
	f.bot = profile.(map[string]any)
	return f.bot, nil
}
func (f *botHTTPFixture) UpdateBot(_ context.Context, _ string, profile any) (any, error) {
	f.bot = profile.(map[string]any)
	return f.bot, nil
}
func (f *botHTTPFixture) SendMessage(context.Context, string, string, []Attachment) (string, error) {
	f.sendCalls++
	return "turn-bot-test", nil
}
func (f *botHTTPFixture) ListObservers(context.Context, string) ([]any, error) {
	return []any{}, nil
}
func (f *botHTTPFixture) CreateObserver(context.Context, string, any, any) (any, error) {
	return map[string]any{"id": "observer-test"}, nil
}
func (f *botHTTPFixture) UpdateObserver(context.Context, string, string, any) (any, error) {
	return map[string]any{"id": "observer-test"}, nil
}
func (f *botHTTPFixture) DeleteObserver(context.Context, string, string) error { return nil }

func TestBotHTTPRoutesUseLocalSessionAndManager(t *testing.T) {
	service, store := newRoutingTestService(t)
	const host = "127.0.0.1:4321"
	bot := &botHTTPFixture{bot: map[string]any{"id": "bot-test", "name": "Test"}}
	service.SetBotStore(bot)
	conversationID, err := NewID("conv_")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(Conversation{
		ID:            conversationID,
		BotID:         "bot-test",
		AssistantMode: true,
		Title:         "Test",
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
		Workspace:     t.TempDir(),
		SchemaVersion: CurrentSchemaVersion,
		Messages:      []Message{{ID: "message-test", Role: "assistant", Content: "ready"}},
		Turns:         []Turn{},
	}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	handler := NewHTTPHandler(service, "token", host)

	request := httptest.NewRequest(http.MethodGet, "http://"+host+"/api/v1/bots", nil)
	request.Header.Set("Authorization", "Bearer token")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"bot-test"`) {
		t.Fatalf("GET bots = %d %s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodPost, "http://"+host+"/api/v1/bots/bot-test/messages", strings.NewReader(`{"content":"hello"}`))
	request.Header.Set("Authorization", "Bearer token")
	request.Header.Set("Origin", "http://"+host)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted || !strings.Contains(response.Body.String(), `"turn_id":"turn-bot-test"`) || bot.sendCalls != 1 {
		t.Fatalf("POST bot messages = %d %s, send calls = %d", response.Code, response.Body.String(), bot.sendCalls)
	}

	request = httptest.NewRequest(http.MethodGet, "http://"+host+"/api/v1/bots/bot-test/messages", nil)
	request.Header.Set("Authorization", "Bearer token")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	var messages map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &messages); err != nil {
		t.Fatalf("decode messages response: %v", err)
	}
	if response.Code != http.StatusOK || messages["bot_id"] != "bot-test" {
		t.Fatalf("GET bot messages = %d %s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "http://"+host+"/api/v1/bots", nil)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized GET bots status = %d, want 401", response.Code)
	}
}

func TestBotConversationSummariesStayOutOfPersonalAgentList(t *testing.T) {
	_, store := newRoutingTestService(t)
	chatID, err := NewID("conv_")
	if err != nil {
		t.Fatal(err)
	}
	botConversationID, err := NewID("conv_")
	if err != nil {
		t.Fatal(err)
	}
	for _, conversation := range []Conversation{
		{
			ID: chatID, Title: "regular chat", CreatedAt: time.Now(), UpdatedAt: time.Now(), SchemaVersion: CurrentSchemaVersion,
			Messages: []Message{{ID: "message-chat", Role: "user", Content: "hello"}},
		},
		{
			ID: botConversationID, BotID: "bot", AssistantMode: true, Title: "bot chat",
			CreatedAt: time.Now(), UpdatedAt: time.Now(), SchemaVersion: CurrentSchemaVersion,
			Messages: []Message{{ID: "message-bot", Role: "user", Content: "hello"}},
		},
	} {
		if err := store.Save(conversation); err != nil {
			t.Fatalf("Save(%s) error = %v", conversation.ID, err)
		}
	}
	summaries, err := store.ListConversationSummaries()
	if err != nil {
		t.Fatalf("ListConversationSummaries() error = %v", err)
	}
	if len(summaries) != 1 || summaries[0].ID != chatID {
		t.Fatalf("summaries = %+v, want only regular chat", summaries)
	}
}
