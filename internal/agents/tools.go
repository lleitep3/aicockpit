package agents

import (
	"bytes"
	"encoding/json"
	"strings"
)

// ToolSpec is a tool definition reserved for a later tool loop.
// APIBackend can serialize it and does not send or execute tools from Run.
type ToolSpec struct {
	// Name is the tool name.
	Name string
	// Description is optional human text.
	Description string
	// InputSchema is JSON Schema for the arguments. Empty becomes an object schema.
	InputSchema json.RawMessage
}

// ToolCall is one model-requested tool invocation.
// Arguments is a JSON value. The API backend records the call and does not execute it.
type ToolCall struct {
	// ID is the provider call id when one was reported.
	ID string
	// Name is the tool name.
	Name string
	// Arguments is the raw JSON argument value.
	Arguments json.RawMessage
}

func cloneCalls(calls []ToolCall) []ToolCall {
	if len(calls) == 0 {
		return nil
	}
	out := make([]ToolCall, len(calls))
	for i, call := range calls {
		out[i] = call
		if len(call.Arguments) > 0 {
			out[i].Arguments = append(json.RawMessage(nil), call.Arguments...)
		}
	}
	return out
}

func normalizeArgs(raw string) json.RawMessage {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if json.Valid([]byte(raw)) {
		return json.RawMessage(raw)
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	return encoded
}

func schemaOrEmpty(raw json.RawMessage) (json.RawMessage, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return json.RawMessage(`{}`), nil
	}
	if !json.Valid(raw) {
		return nil, errInvalidTool
	}
	return append(json.RawMessage(nil), raw...), nil
}
