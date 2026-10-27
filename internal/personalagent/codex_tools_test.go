package personalagent

import (
	"context"
	"encoding/json"
	"io"
	"testing"
)

func TestCodexAdapterRespondsToStructuredToolCall(t *testing.T) {
	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()
	client := &appServerClient{stdin: writer}
	arguments := json.RawMessage(`{"conversation_id":"conv-1"}`)
	message := rpcMessage{
		ID:     json.RawMessage(`17`),
		Method: "item/tool/call",
		Params: json.RawMessage(`{"tool":"read_conversation","arguments":{"conversation_id":"conv-1"}}`),
	}
	called := false
	adapter := &CodexAdapter{}
	done := make(chan error, 1)
	go func() {
		done <- adapter.handleServerRequest(
			context.Background(),
			client,
			message,
			nil,
			func(_ context.Context, name string, raw json.RawMessage) (string, error) {
				called = name == "read_conversation" && string(raw) == string(arguments)
				return `{"title":"Test"}`, nil
			},
			&RunResult{},
		)
	}()
	var response struct {
		ID     int `json:"id"`
		Result struct {
			Success      bool `json:"success"`
			ContentItems []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"contentItems"`
		} `json:"result"`
	}
	if err := json.NewDecoder(reader).Decode(&response); err != nil {
		t.Fatalf("decode tool response: %v", err)
	}
	if err := <-done; err != nil {
		t.Fatalf("handleServerRequest() error = %v", err)
	}
	if !called || response.ID != 17 || !response.Result.Success {
		t.Fatalf("tool callback/response = called:%v response:%+v", called, response)
	}
	if len(response.Result.ContentItems) != 1 || response.Result.ContentItems[0].Text != `{"title":"Test"}` {
		t.Fatalf("tool content result = %+v", response.Result.ContentItems)
	}
}

func TestCodexAdapterFailsClosedWhenToolCallbackUnavailable(t *testing.T) {
	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()
	client := &appServerClient{stdin: writer}
	message := rpcMessage{
		ID:     json.RawMessage(`18`),
		Method: "item/tool/call",
		Params: json.RawMessage(`{"tool":"read_conversation","arguments":{}}`),
	}
	done := make(chan error, 1)
	go func() {
		done <- (&CodexAdapter{}).handleServerRequest(context.Background(), client, message, nil, nil, &RunResult{})
	}()
	var response struct {
		Result struct {
			Success bool `json:"success"`
			Content []struct {
				Text string `json:"text"`
			} `json:"contentItems"`
		} `json:"result"`
	}
	if err := json.NewDecoder(reader).Decode(&response); err != nil {
		t.Fatalf("decode unavailable-tool response: %v", err)
	}
	if err := <-done; err != nil {
		t.Fatalf("handleServerRequest() error = %v", err)
	}
	if response.Result.Success || len(response.Result.Content) != 1 || response.Result.Content[0].Text == "" {
		t.Fatalf("unavailable tool response = %+v", response.Result)
	}
}
