package tui

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lleitep3/aicockpit/internal/personalagent"
)

func TestFilterTUIConversations(t *testing.T) {
	summaries := []personalagent.ConversationSummary{
		{ID: "conversation-one", Title: "Go tooling", MessageCount: 2},
		{ID: "conversation-two", Title: "Other topic", ProjectID: "aicockpit", MessageCount: 1},
		{ID: "conversation-empty", Title: "Draft", MessageCount: 0},
		{ID: "conversation-assistant", Title: "Assistant", MessageCount: 1, AssistantMode: true},
		{ID: "conversation-archived", Title: "Archived", MessageCount: 1, Archived: true},
	}
	records := map[string]personalagent.Conversation{
		"conversation-one": {
			ID: "conversation-one",
			Messages: []personalagent.Message{
				{Role: "user", Content: "How do I format Go code?"},
			},
		},
		"conversation-two": {
			ID: "conversation-two",
			Messages: []personalagent.Message{
				{Role: "user", Content: "Release checklist"},
			},
		},
	}

	tests := []struct {
		name  string
		query string
		want  []string
	}{
		{name: "all visible chats", want: []string{"conversation-one", "conversation-two"}},
		{name: "title search is case insensitive", query: "GO TOOL", want: []string{"conversation-one"}},
		{name: "project search", query: "AICockpit", want: []string{"conversation-two"}},
		{name: "message search", query: "FORMAT GO", want: []string{"conversation-one"}},
		{name: "no matches", query: "not present", want: []string{}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := filterTUIConversations(summaries, records, test.query)
			if len(got) != len(test.want) {
				t.Fatalf("got %d conversations, want %d", len(got), len(test.want))
			}
			for index, id := range test.want {
				if got[index].ID != id {
					t.Errorf("conversation[%d] = %q, want %q", index, got[index].ID, id)
				}
			}
		})
	}
}

func TestTerminalTextHelpers(t *testing.T) {
	tests := []struct {
		name string
		run  func() string
		want string
	}{
		{
			name: "remove last unicode rune",
			run:  func() string { return removeLastRune("chat é") },
			want: "chat ",
		},
		{
			name: "strip terminal controls",
			run:  func() string { return sanitizeTerminalText("safe\x1b[2J\ntext\x00") },
			want: "safe [2J\ntext ",
		},
		{
			name: "truncate by terminal columns",
			run:  func() string { return truncateTerminalText("conversation title", 10) },
			want: "convers...",
		},
		{
			name: "preserve short text",
			run:  func() string { return truncateTerminalText("short", 10) },
			want: "short",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.run(); got != test.want {
				t.Errorf("got %q, want %q", got, test.want)
			}
		})
	}
}

func TestFilterTUIConversationsPreservesServiceOrder(t *testing.T) {
	newer := time.Now()
	summaries := []personalagent.ConversationSummary{
		{ID: "older", UpdatedAt: newer.Add(-time.Minute), MessageCount: 1},
		{ID: "newer", UpdatedAt: newer, MessageCount: 1},
	}
	got := filterTUIConversations(summaries, nil, "")
	if got[0].ID != "older" {
		t.Fatalf("filter reordered summaries: first ID = %q", got[0].ID)
	}
}

func TestRunTUIKeepsNewChatAsDraftUntilSend(t *testing.T) {
	ui, store := newTestTerminalUI(t, nil)
	ui.keys = keyChannel('n', 'q')

	if err := ui.run(); err != nil {
		t.Fatalf("run() error = %v", err)
	}
	conversations, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(conversations) != 0 {
		t.Fatalf("draft created %d persisted conversations, want none", len(conversations))
	}
}

func TestRunTUIOpensConversationAndReturnsToList(t *testing.T) {
	ui, _ := newTestTerminalUI(t, []personalagent.Message{
		{ID: "message-1", Role: "user", Content: "hello"},
	})
	ui.keys = keyChannel('\r', 'b', 'q')

	if err := ui.run(); err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if ui.mode != tuiList || ui.conversation != nil {
		t.Fatalf("state after returning = mode %v, conversation %v", ui.mode, ui.conversation)
	}
}

func TestRunTUISearchesConversationContent(t *testing.T) {
	ui, _ := newTestTerminalUI(t, []personalagent.Message{
		{ID: "message-1", Role: "user", Content: "Needle inside transcript"},
	})
	ui.keys = keyChannel('/', 'n', 'e', 'e', 'd', 'l', 'e', '\r', 'q')

	if err := ui.run(); err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if len(ui.conversations) != 1 || ui.conversations[0].ID != testConversationID {
		t.Fatalf("search results = %#v, want conversation %s", ui.conversations, testConversationID)
	}
}

func TestComposeEmptyMessageDoesNotCreateConversation(t *testing.T) {
	ui, store := newTestTerminalUI(t, nil)
	ui.mode = tuiCompose

	if err := ui.handleComposeKey('\r'); err != nil {
		t.Fatalf("handleComposeKey() error = %v", err)
	}
	conversations, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(conversations) != 0 {
		t.Fatalf("empty send created %d conversations, want none", len(conversations))
	}
}

func TestDrawSanitizesConversationText(t *testing.T) {
	ui, _ := newTestTerminalUI(t, []personalagent.Message{
		{ID: "message-1", Role: "user", Content: "safe\x1b[2J"},
	})
	conversation, err := ui.service.Conversation(testConversationID)
	if err != nil {
		t.Fatalf("Conversation() error = %v", err)
	}
	ui.conversation = &conversation
	ui.mode = tuiConversation

	if err := ui.draw(); err != nil {
		t.Fatalf("draw() error = %v", err)
	}
	rendered := ui.output.(*bytes.Buffer).String()
	if !strings.Contains(rendered, "You: safe [2J") {
		t.Fatalf("rendered text did not preserve safe content: %q", rendered)
	}
}

func TestApplyTurnEventsUpdatesStreamingState(t *testing.T) {
	ui, _ := newTestTerminalUI(t, nil)
	ui.conversation = &personalagent.Conversation{ID: testConversationID}

	events := []personalagent.TurnEvent{
		{Type: "delta", Data: map[string]any{"content": "partial"}},
		{Type: "status", Data: map[string]any{"status": "working"}},
		{Type: "error", Data: map[string]any{"message": "provider failed"}},
	}
	for _, event := range events {
		finished, err := ui.applyTurnEvent("turn-test", event)
		if err != nil || finished {
			t.Fatalf("applyTurnEvent(%q) = (%v, %v), want unfinished without error", event.Type, finished, err)
		}
	}
	if ui.streaming != "partial" || ui.status != "Turn failed: provider failed" {
		t.Fatalf("stream state = %q, status = %q", ui.streaming, ui.status)
	}
}

func TestApprovalWithoutRequestIDFailsClosed(t *testing.T) {
	ui, _ := newTestTerminalUI(t, nil)
	if err := ui.confirmApproval("turn-test", map[string]any{"method": "shell"}); err == nil {
		t.Fatal("confirmApproval() accepted request without ID")
	}
}

func TestRunTUIRejectsNonTerminalInput(t *testing.T) {
	input, err := os.CreateTemp(t.TempDir(), "stdin")
	if err != nil {
		t.Fatalf("CreateTemp() error = %v", err)
	}
	defer input.Close()

	err = RunTUI(context.Background(), nil, input, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "interactive terminal required") {
		t.Fatalf("RunTUI() error = %v, want actionable terminal error", err)
	}
}

func TestReadTerminalKeys(t *testing.T) {
	input, err := os.CreateTemp(t.TempDir(), "keys")
	if err != nil {
		t.Fatalf("CreateTemp() error = %v", err)
	}
	defer input.Close()
	if _, err := input.Write([]byte("jk")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if _, err := input.Seek(0, 0); err != nil {
		t.Fatalf("Seek() error = %v", err)
	}

	keys := readTerminalKeys(context.Background(), input)
	var got []byte
	for key := range keys {
		got = append(got, key)
	}
	if string(got) != "jk" {
		t.Fatalf("keys = %q, want jk", got)
	}
}

func TestTerminalUIStateTransitions(t *testing.T) {
	ui, _ := newTestTerminalUI(t, nil)
	ui.conversations = []personalagent.ConversationSummary{{ID: "one"}, {ID: "two"}}
	if done, err := ui.handleKey('j'); done || err != nil || ui.selected != 1 {
		t.Fatalf("move down = (%v, %v), selected %d", done, err, ui.selected)
	}
	if done, err := ui.handleKey('k'); done || err != nil || ui.selected != 0 {
		t.Fatalf("move up = (%v, %v), selected %d", done, err, ui.selected)
	}
	if done, err := ui.handleKey('/'); done || err != nil || ui.mode != tuiSearch {
		t.Fatalf("open search = (%v, %v), mode %v", done, err, ui.mode)
	}
	if done, err := ui.handleKey('x'); done || err != nil {
		t.Fatalf("type search = (%v, %v)", done, err)
	}
	if done, err := ui.handleKey(27); done || err != nil || ui.mode != tuiList || ui.composer != "" {
		t.Fatalf("cancel search = (%v, %v), mode %v", done, err, ui.mode)
	}
	if done, err := ui.handleKey('q'); !done || err != nil {
		t.Fatalf("quit = (%v, %v), want done", done, err)
	}
}

func TestComposeEditingAndCancellation(t *testing.T) {
	ui, _ := newTestTerminalUI(t, nil)
	ui.mode = tuiCompose
	ui.composer = "discard"
	for _, key := range []byte{21, 'a', 10, 'b', 127, 3} {
		if err := ui.handleComposeKey(key); err != nil {
			t.Fatalf("handleComposeKey(%d) error = %v", key, err)
		}
	}
	if ui.composer != "" || ui.mode != tuiConversation || ui.status != "Message cancelled." {
		t.Fatalf("compose state = text %q, mode %v, status %q", ui.composer, ui.mode, ui.status)
	}
}

func TestSendMessageFromDraftAndHandleProviderEvents(t *testing.T) {
	service := newFakeChatService()
	service.events = []personalagent.TurnEvent{
		{Type: "delta", Data: map[string]any{"content": "answer"}},
		{Type: "complete", Data: map[string]any{"status": "completed"}},
	}
	ui, _ := newTestTerminalUI(t, nil)
	ui.service = service
	ui.mode = tuiCompose
	ui.composer = "question"

	if err := ui.sendMessage(); err != nil {
		t.Fatalf("sendMessage() error = %v", err)
	}
	if service.createCalls != 1 || service.startCalls != 1 {
		t.Fatalf("create/start calls = %d/%d, want 1/1", service.createCalls, service.startCalls)
	}
	if ui.status != "Turn completed." || ui.conversation == nil || ui.streaming != "" {
		t.Fatalf("post-turn state = status %q, conversation %v, streaming %q", ui.status, ui.conversation, ui.streaming)
	}
	if len(ui.conversation.Messages) != 2 || ui.conversation.Messages[1].Content != "answer" {
		t.Fatalf("persisted messages = %#v", ui.conversation.Messages)
	}
}

func TestSendMessageReportsCreateAndTurnErrors(t *testing.T) {
	tests := []struct {
		name      string
		createErr error
		startErr  error
		want      string
	}{
		{name: "create failure", createErr: errors.New("catalog unavailable"), want: "create conversation: catalog unavailable"},
		{name: "turn failure", startErr: errors.New("quota unavailable"), want: "send message: quota unavailable"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := newFakeChatService()
			service.createErr = test.createErr
			service.startErr = test.startErr
			ui, _ := newTestTerminalUI(t, nil)
			ui.service = service
			ui.composer = "message"
			err := ui.sendMessage()
			if err == nil || err.Error() != test.want {
				t.Fatalf("sendMessage() error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestSendMessageDetectsClosedEventStream(t *testing.T) {
	service := newFakeChatService()
	ui, _ := newTestTerminalUI(t, nil)
	ui.service = service
	ui.conversation = &personalagent.Conversation{ID: "existing"}
	ui.composer = "message"

	err := ui.sendMessage()
	if err == nil || err.Error() != "turn event stream closed" {
		t.Fatalf("sendMessage() error = %v, want closed stream error", err)
	}
}

func TestApprovalDecisionIsExplicit(t *testing.T) {
	tests := []struct {
		name      string
		keys      []byte
		want      string
		wantError bool
	}{
		{name: "accept", keys: []byte{'y'}, want: "accept"},
		{name: "decline", keys: []byte{'x', 'n'}, want: "decline"},
		{name: "closed terminal", wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := newFakeChatService()
			ui, _ := newTestTerminalUI(t, nil)
			ui.service = service
			ui.conversation = &personalagent.Conversation{ID: "existing"}
			ui.keys = keyChannel(test.keys...)
			err := ui.confirmApproval("turn", map[string]any{
				"request_id": "approval-1",
				"method":     "command",
				"params":     map[string]any{"command": "go test"},
			})
			if test.wantError {
				if err == nil {
					t.Fatal("confirmApproval() error = nil, want closed input error")
				}
				return
			}
			if err != nil || service.decision != test.want {
				t.Fatalf("confirmApproval() = %v, decision %q; want %q", err, service.decision, test.want)
			}
		})
	}
}

func TestDrawShowsListSearchAndComposerStates(t *testing.T) {
	ui, _ := newTestTerminalUI(t, nil)
	ui.conversations = []personalagent.ConversationSummary{{ID: "one", Title: "Conversation"}}
	ui.mode = tuiList
	if err := ui.draw(); err != nil {
		t.Fatalf("draw list: %v", err)
	}
	ui.mode = tuiSearch
	ui.composer = "needle"
	if err := ui.draw(); err != nil {
		t.Fatalf("draw search: %v", err)
	}
	ui.mode = tuiCompose
	ui.status = "Type message"
	if err := ui.draw(); err != nil {
		t.Fatalf("draw composer: %v", err)
	}
	rendered := ui.output.(*bytes.Buffer).String()
	for _, expected := range []string{"Cockpit Agent", "Search: needle", "Type message"} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("rendered output missing %q", expected)
		}
	}
}

func TestWrapTerminalText(t *testing.T) {
	got := wrapTerminalText("one two\nthree", 4)
	want := []string{"one ", "two", "thre", "e"}
	if len(got) != len(want) {
		t.Fatalf("wrapTerminalText() = %#v, want %#v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Errorf("line %d = %q, want %q", index, got[index], want[index])
		}
	}
}

const testConversationID = "conv_11111111111111111111111111111111"

type fakeChatService struct {
	conversations map[string]personalagent.Conversation
	summaries     []personalagent.ConversationSummary
	events        []personalagent.TurnEvent
	createErr     error
	startErr      error
	decision      string
	createCalls   int
	startCalls    int
}

func newFakeChatService() *fakeChatService {
	return &fakeChatService{conversations: make(map[string]personalagent.Conversation)}
}

func (service *fakeChatService) ConversationSummaries() ([]personalagent.ConversationSummary, error) {
	return service.summaries, nil
}

func (service *fakeChatService) Conversations() ([]personalagent.Conversation, error) {
	conversations := make([]personalagent.Conversation, 0, len(service.conversations))
	for _, conversation := range service.conversations {
		conversations = append(conversations, conversation)
	}
	return conversations, nil
}

func (service *fakeChatService) Conversation(id string) (personalagent.Conversation, error) {
	conversation, exists := service.conversations[id]
	if !exists {
		return personalagent.Conversation{}, errors.New("conversation not found")
	}
	return conversation, nil
}

func (service *fakeChatService) CreateConversation(
	_ context.Context,
	_, _, _, _, _ string,
) (personalagent.Conversation, error) {
	service.createCalls++
	if service.createErr != nil {
		return personalagent.Conversation{}, service.createErr
	}
	conversation := personalagent.Conversation{
		ID:       "new-conversation",
		Title:    "New conversation",
		Messages: []personalagent.Message{},
	}
	service.conversations[conversation.ID] = conversation
	return conversation, nil
}

func (service *fakeChatService) StartTurn(
	conversationID, content string,
	_ []personalagent.Attachment,
) (string, error) {
	service.startCalls++
	if service.startErr != nil {
		return "", service.startErr
	}
	conversation := service.conversations[conversationID]
	conversation.Messages = append(conversation.Messages,
		personalagent.Message{Role: "user", Content: content},
		personalagent.Message{Role: "assistant", Content: ""},
	)
	service.conversations[conversationID] = conversation
	return "turn-test", nil
}

func (service *fakeChatService) Subscribe(
	_ string,
	_ uint64,
) ([]personalagent.TurnEvent, <-chan personalagent.TurnEvent, func()) {
	events := make(chan personalagent.TurnEvent, len(service.events))
	for _, event := range service.events {
		if event.Type == "delta" {
			if content, ok := event.Data["content"].(string); ok {
				conversation := service.conversations["new-conversation"]
				last := len(conversation.Messages) - 1
				if last >= 0 {
					conversation.Messages[last].Content += content
					service.conversations[conversation.ID] = conversation
				}
			}
		}
		events <- event
	}
	close(events)
	return nil, events, func() {}
}

func (service *fakeChatService) DecideApproval(_, _, _, decision string) error {
	service.decision = decision
	return nil
}

func (service *fakeChatService) CancelTurn(_, _ string) error {
	return nil
}

func newTestTerminalUI(
	t *testing.T,
	messages []personalagent.Message,
) (*terminalUI, *personalagent.Store) {
	t.Helper()
	root := t.TempDir()
	store, err := personalagent.NewStore(filepath.Join(root, "conversations"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	service, err := personalagent.NewService(store, personalagent.NewProviderRegistry(), root)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	if messages != nil {
		conversation := personalagent.Conversation{
			SchemaVersion: personalagent.CurrentSchemaVersion,
			ID:            testConversationID,
			Title:         "Terminal test",
			CreatedAt:     time.Now().UTC(),
			UpdatedAt:     time.Now().UTC(),
			Workspace:     root,
			Messages:      messages,
		}
		if err := store.Save(conversation); err != nil {
			t.Fatalf("Save() error = %v", err)
		}
	}
	input, err := os.CreateTemp(root, "terminal")
	if err != nil {
		t.Fatalf("CreateTemp() error = %v", err)
	}
	t.Cleanup(func() { _ = input.Close() })
	return &terminalUI{
		ctx:     context.Background(),
		service: service,
		input:   input,
		output:  &bytes.Buffer{},
	}, store
}

func keyChannel(keys ...byte) <-chan byte {
	channel := make(chan byte, len(keys))
	for _, key := range keys {
		channel <- key
	}
	close(channel)
	return channel
}
