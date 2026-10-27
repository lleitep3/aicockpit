package agents

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// Codex drives the Codex CLI in non-interactive exec mode.
//
// Flags follow the OpenAI non-interactive guide
// (https://developers.openai.com/codex/noninteractive) and the CLI reference
// (https://developers.openai.com/codex/cli):
//
//	codex exec --json --skip-git-repo-check [--model MODEL] [--cd DIR] PROMPT
//
// --json selects the JSONL event stream on stdout (thread.*, turn.*, item.*,
// error). --model and --cd are the documented global overrides and are
// omitted when unset. --skip-git-repo-check is set because a task may use a
// working directory that is not a Git repository; without it the CLI refuses
// to start. The sandbox flag is left unset so the CLI default (read-only)
// stays in effect. --cd receives the absolute task directory.
type Codex struct{}

// ID is the backend name.
func (Codex) ID() string { return "codex" }

// Binary is the executable name on PATH.
func (Codex) Binary() string { return "codex" }

// DisplayName is the doctor label.
func (Codex) DisplayName() string { return "Codex CLI" }

// VersionArgs prints the CLI version.
func (Codex) VersionArgs() []string { return []string{"--version"} }

// DropEnv removes nothing. Codex has no nesting guard to clear.
func (Codex) DropEnv() []string { return nil }

// Args builds the exec argument vector. The prompt is the last argument.
func (Codex) Args(task Task) ([]string, error) {
	args := []string{"exec", "--json", "--skip-git-repo-check"}
	if task.Model != "" {
		args = append(args, "--model", task.Model)
	}
	if task.WorkDir != "" {
		args = append(args, "--cd", task.WorkDir)
	}
	args = append(args, task.Prompt)
	return args, nil
}

// NewParser returns a parser for codex exec JSONL.
func (Codex) NewParser() Parser { return &codexParser{} }

type codexParser struct {
	sessionID string
	lastText  string
}

type codexHead struct {
	Type     string          `json:"type"`
	ThreadID string          `json:"thread_id"`
	Message  string          `json:"message"`
	Error    json.RawMessage `json:"error"`
	Item     json.RawMessage `json:"item"`
}

type codexItem struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Text    string `json:"text"`
	Command string `json:"command"`
	Status  string `json:"status"`
	Server  string `json:"server"`
	Tool    string `json:"tool"`
	Query   string `json:"query"`
	Message string `json:"message"`
}

type codexError struct {
	Message string `json:"message"`
}

func (p *codexParser) ParseLine(line []byte) ([]Event, error) {
	line = bytes.TrimSpace(line)
	if len(line) == 0 {
		return nil, nil
	}
	var head codexHead
	if err := json.Unmarshal(line, &head); err != nil {
		return nil, fmt.Errorf("codex: %w", err)
	}
	raw := append([]byte(nil), line...)
	switch head.Type {
	case "thread.started":
		if head.ThreadID != "" {
			p.sessionID = head.ThreadID
		}
		return []Event{{
			Kind: KindStatus, Type: head.Type, SessionID: p.sessionID, Raw: raw,
		}}, nil
	case "turn.started":
		return []Event{{
			Kind: KindStatus, Type: head.Type, SessionID: p.sessionID, Raw: raw,
		}}, nil
	case "turn.completed":
		return []Event{{
			Kind: KindResult, Type: head.Type, Text: p.lastText, SessionID: p.sessionID, Final: true, Raw: raw,
		}}, nil
	case "turn.failed":
		return []Event{{
			Kind: KindError, Type: head.Type, Text: codexErrorText(head.Error), SessionID: p.sessionID,
			Final: true, Failed: true, Raw: raw,
		}}, nil
	case "error":
		text := head.Message
		if text == "" {
			text = codexErrorText(head.Error)
		}
		return []Event{{
			Kind: KindError, Type: head.Type, Text: text, SessionID: p.sessionID,
			Final: true, Failed: true, Raw: raw,
		}}, nil
	case "item.started", "item.updated", "item.completed":
		ev := p.itemEvent(head.Type, head.Item, raw)
		return []Event{ev}, nil
	default:
		return []Event{{
			Kind: KindStatus, Type: head.Type, SessionID: p.sessionID, Raw: raw,
		}}, nil
	}
}

func (p *codexParser) itemEvent(eventType string, rawItem json.RawMessage, raw []byte) Event {
	ev := Event{Kind: KindStatus, Type: eventType, SessionID: p.sessionID, Raw: raw}
	var item codexItem
	if len(rawItem) == 0 || json.Unmarshal(rawItem, &item) != nil {
		return ev
	}
	ev.Subtype = item.Type
	switch item.Type {
	case "agent_message":
		ev.Kind = KindText
		ev.Text = item.Text
		if eventType == "item.completed" {
			p.lastText = item.Text
		}
	case "reasoning":
		ev.Kind = KindReasoning
		ev.Text = item.Text
	case "command_execution":
		ev.Kind = KindTool
		ev.Tool = "command"
		ev.Text = item.Command
	case "mcp_tool_call":
		ev.Kind = KindTool
		ev.Tool = item.Tool
		if item.Server != "" && item.Tool != "" {
			ev.Text = item.Server + "/" + item.Tool
		} else {
			ev.Text = item.Tool
		}
	case "collab_tool_call":
		ev.Kind = KindTool
		ev.Tool = item.Tool
		ev.Text = item.Tool
	case "web_search":
		ev.Kind = KindTool
		ev.Tool = "web_search"
		ev.Text = item.Query
	case "file_change":
		ev.Kind = KindTool
		ev.Tool = "file_change"
		ev.Text = item.Status
	case "error":
		ev.Kind = KindError
		ev.Text = item.Message
		ev.Failed = true
	default:
		ev.Text = item.Type
	}
	return ev
}

func codexErrorText(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text
	}
	var body codexError
	if err := json.Unmarshal(raw, &body); err == nil && body.Message != "" {
		return body.Message
	}
	return string(raw)
}
