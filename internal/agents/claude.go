package agents

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// Claude drives the Claude Code CLI in non-interactive print mode.
//
// Flags follow the Claude Code headless guide
// (https://code.claude.com/docs/en/headless) and the CLI reference
// (https://code.claude.com/docs/en/cli-reference):
//
//	claude -p --output-format stream-json --verbose --include-partial-messages
//
// stream-json is newline-delimited JSON. --verbose and
// --include-partial-messages are the documented pair that emits token
// deltas. --model is omitted when the task does not set one. The main
// command has no working-directory flag, so the process directory is the
// task directory. CLAUDECODE and CLAUDE_CODE_ENTRYPOINT are removed from
// the inherited environment because a nested claude process refuses to
// start or waits for an IPC handshake when they are set.
type Claude struct{}

// ID is the backend name.
func (Claude) ID() string { return "claude" }

// Binary is the executable name on PATH.
func (Claude) Binary() string { return "claude" }

// DisplayName is the doctor label.
func (Claude) DisplayName() string { return "Claude CLI" }

// VersionArgs prints the CLI version.
func (Claude) VersionArgs() []string { return []string{"--version"} }

// DropEnv removes Claude Code nesting guards from the inherited environment.
func (Claude) DropEnv() []string {
	return []string{"CLAUDECODE", "CLAUDE_CODE_ENTRYPOINT"}
}

// Args builds the print-mode argument vector. The prompt is the last argument.
func (Claude) Args(task Task) ([]string, error) {
	args := []string{
		"-p",
		"--output-format", "stream-json",
		"--verbose",
		"--include-partial-messages",
	}
	if task.Model != "" {
		args = append(args, "--model", task.Model)
	}
	args = append(args, task.Prompt)
	return args, nil
}

// NewParser returns a parser for stream-json lines.
func (Claude) NewParser() Parser { return &claudeParser{} }

type claudeParser struct{}

type claudeLine struct {
	Type      string          `json:"type"`
	Subtype   string          `json:"subtype"`
	Result    json.RawMessage `json:"result"`
	IsError   bool            `json:"is_error"`
	SessionID string          `json:"session_id"`
	Model     string          `json:"model"`
	Message   json.RawMessage `json:"message"`
	Event     json.RawMessage `json:"event"`
}

type claudeMessage struct {
	Model   string          `json:"model"`
	Content json.RawMessage `json:"content"`
}

type claudeBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
	Name string `json:"name"`
}

type claudeDelta struct {
	Type  string `json:"type"`
	Delta struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"delta"`
}

func (*claudeParser) ParseLine(line []byte) ([]Event, error) {
	line = bytes.TrimSpace(line)
	if len(line) == 0 {
		return nil, nil
	}
	var msg claudeLine
	if err := json.Unmarshal(line, &msg); err != nil {
		return nil, fmt.Errorf("claude: %w", err)
	}
	base := Event{Type: msg.Type, Subtype: msg.Subtype, SessionID: msg.SessionID, Model: msg.Model, Raw: append([]byte(nil), line...)}
	switch msg.Type {
	case "result":
		ev := base
		ev.Kind = KindResult
		ev.Text = rawString(msg.Result)
		ev.Final = true
		if msg.IsError || strings.Contains(strings.ToLower(msg.Subtype), "error") {
			ev.Kind = KindError
			ev.Failed = true
		}
		return []Event{ev}, nil
	case "assistant":
		return claudeAssistant(base, msg.Message), nil
	case "stream_event":
		ev, ok := claudeStream(base, msg.Event)
		if !ok {
			return nil, nil
		}
		return []Event{ev}, nil
	case "system", "user":
		ev := base
		ev.Kind = KindStatus
		return []Event{ev}, nil
	default:
		ev := base
		ev.Kind = KindStatus
		return []Event{ev}, nil
	}
}

func claudeAssistant(base Event, raw json.RawMessage) []Event {
	var message claudeMessage
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &message)
	}
	if message.Model != "" {
		base.Model = message.Model
	}
	texts, tools := claudeContent(message.Content)
	if len(texts) == 0 && len(tools) == 0 {
		base.Kind = KindStatus
		return []Event{base}
	}
	events := make([]Event, 0, len(texts)+len(tools))
	for _, text := range texts {
		ev := base
		ev.Kind = KindText
		ev.Text = text
		events = append(events, ev)
	}
	for _, name := range tools {
		ev := base
		ev.Kind = KindTool
		ev.Tool = name
		ev.Text = name
		events = append(events, ev)
	}
	return events
}

func claudeContent(raw json.RawMessage) (texts, tools []string) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		if text != "" {
			return []string{text}, nil
		}
		return nil, nil
	}
	var blocks []claudeBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return []string{string(raw)}, nil
	}
	for _, block := range blocks {
		switch block.Type {
		case "tool_use":
			if block.Name != "" {
				tools = append(tools, block.Name)
			}
		case "text":
			if block.Text != "" {
				texts = append(texts, block.Text)
			}
		}
	}
	return texts, tools
}

func claudeStream(base Event, raw json.RawMessage) (Event, bool) {
	var delta claudeDelta
	if len(raw) == 0 || json.Unmarshal(raw, &delta) != nil || delta.Delta.Text == "" {
		return Event{}, false
	}
	base.Text = delta.Delta.Text
	base.Partial = true
	base.Subtype = delta.Delta.Type
	switch delta.Delta.Type {
	case "thinking_delta":
		base.Kind = KindReasoning
	default:
		base.Kind = KindText
	}
	return base, true
}

func rawString(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text
	}
	return string(raw)
}
