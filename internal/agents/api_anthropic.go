package agents

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
)

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	Stream    bool               `json:"stream"`
	Messages  []anthropicMessage `json:"messages"`
	Tools     []anthropicTool    `json:"tools,omitempty"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"input_schema"`
}

type anthropicStream struct {
	Type         string               `json:"type"`
	Index        int                  `json:"index"`
	Message      anthropicMessageBody `json:"message"`
	ContentBlock anthropicBlock       `json:"content_block"`
	Delta        struct {
		Type        string `json:"type"`
		Text        string `json:"text"`
		Thinking    string `json:"thinking"`
		PartialJSON string `json:"partial_json"`
		StopReason  string `json:"stop_reason"`
	} `json:"delta"`
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

type anthropicMessageBody struct {
	ID         string           `json:"id"`
	Type       string           `json:"type"`
	Model      string           `json:"model"`
	StopReason string           `json:"stop_reason"`
	Content    []anthropicBlock `json:"content"`
	Error      struct {
		Message string `json:"message"`
	} `json:"error"`
}

type anthropicBlock struct {
	Type     string          `json:"type"`
	Text     string          `json:"text"`
	Thinking string          `json:"thinking"`
	ID       string          `json:"id"`
	Name     string          `json:"name"`
	Input    json.RawMessage `json:"input"`
}

type anthBlock struct {
	kind string
	id   string
	name string
	buf  strings.Builder
	done bool
}

type anthState struct {
	emit     func(Event)
	model    string
	session  string
	text     strings.Builder
	blocks   map[int]*anthBlock
	calls    []ToolCall
	finished bool
	failed   bool
	reason   string
}

func anthropicBody(model, prompt string, maxTokens int, tools []ToolSpec) ([]byte, error) {
	if err := validateTools(tools); err != nil {
		return nil, err
	}
	req := anthropicRequest{
		Model:     model,
		MaxTokens: maxTokens,
		Stream:    true,
		Messages:  []anthropicMessage{{Role: "user", Content: prompt}},
	}
	for _, tool := range tools {
		schema, err := schemaOrEmpty(tool.InputSchema)
		if err != nil {
			return nil, err
		}
		req.Tools = append(req.Tools, anthropicTool{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: schema,
		})
	}
	return json.Marshal(req)
}

func parseAnthropic(r io.Reader, fallback string, emit func(Event)) (apiSummary, error) {
	st := &anthState{emit: emit, model: fallback, blocks: map[int]*anthBlock{}}
	err := sniffBody(r, func(data []byte) error {
		return st.consumeJSON(data)
	}, func(stream io.Reader) error {
		return readSSE(stream, st.consumeSSE)
	})
	if err != nil {
		if errors.Is(err, io.EOF) {
			return st.summary(), fmt.Errorf("%w: empty response", ErrRunFailed)
		}
		return st.summary(), err
	}
	if !st.finished {
		st.finish()
	}
	return st.summary(), nil
}

func (s *anthState) summary() apiSummary {
	return apiSummary{
		text:    s.text.String(),
		model:   s.model,
		session: s.session,
		calls:   cloneCalls(s.calls),
		failed:  s.failed,
	}
}

func (s *anthState) consumeJSON(data []byte) error {
	var msg anthropicMessageBody
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("%w: %s", ErrBadOutput, clip(err.Error()))
	}
	if msg.Type == "error" || msg.Error.Message != "" {
		s.fail(msg.Error.Message)
		return nil
	}
	if msg.Model != "" {
		s.model = msg.Model
	}
	if msg.ID != "" {
		s.session = msg.ID
	}
	s.reason = msg.StopReason
	for _, block := range msg.Content {
		switch block.Type {
		case "text":
			if block.Text != "" {
				s.text.WriteString(block.Text)
				s.emit(s.base(KindText, block.Text, false))
			}
		case "thinking":
			if block.Thinking != "" {
				s.emit(s.base(KindReasoning, block.Thinking, false))
			}
		case "tool_use":
			call := ToolCall{ID: block.ID, Name: block.Name, Arguments: normalizeArgs(string(block.Input))}
			if call.Name == "" {
				continue
			}
			s.calls = append(s.calls, call)
			ev := s.base(KindTool, string(call.Arguments), false)
			ev.Tool = call.Name
			ev.Type = "tool_use"
			s.emit(ev)
		}
	}
	s.finish()
	return nil
}

func (s *anthState) consumeSSE(eventName, data string) error {
	if strings.TrimSpace(data) == "" {
		return nil
	}
	var msg anthropicStream
	if err := json.Unmarshal([]byte(data), &msg); err != nil {
		return fmt.Errorf("%w: %s", ErrBadOutput, clip(err.Error()))
	}
	kind := msg.Type
	if kind == "" {
		kind = eventName
	}
	switch kind {
	case "ping":
		return nil
	case "error":
		s.fail(msg.Error.Message)
		return errStreamEnd
	case "message_start":
		if msg.Message.Model != "" {
			s.model = msg.Message.Model
		}
		if msg.Message.ID != "" {
			s.session = msg.Message.ID
		}
		ev := s.base(KindStatus, "", false)
		ev.Type = "message_start"
		ev.Raw = rawMessage(data)
		s.emit(ev)
	case "content_block_start":
		s.startBlock(msg.Index, msg.ContentBlock)
	case "content_block_delta":
		s.delta(msg.Index, msg.Delta.Type, msg.Delta.Text, msg.Delta.Thinking, msg.Delta.PartialJSON, data)
	case "content_block_stop":
		s.stopBlock(msg.Index)
	case "message_delta":
		if msg.Delta.StopReason != "" {
			s.reason = msg.Delta.StopReason
		}
	case "message_stop":
		s.finish()
		return errStreamEnd
	default:
		ev := s.base(KindStatus, "", false)
		ev.Type = kind
		ev.Raw = rawMessage(data)
		s.emit(ev)
	}
	return nil
}

func (s *anthState) startBlock(index int, block anthropicBlock) {
	acc := &anthBlock{kind: block.Type, id: block.ID, name: block.Name}
	if len(block.Input) > 0 && string(bytesTrim(block.Input)) != "{}" && string(bytesTrim(block.Input)) != "null" {
		acc.buf.Write(block.Input)
	}
	s.blocks[index] = acc
	if block.Type == "text" && block.Text != "" {
		s.text.WriteString(block.Text)
		ev := s.base(KindText, block.Text, true)
		ev.Subtype = "text"
		s.emit(ev)
	}
}

func (s *anthState) delta(index int, kind, text, thinking, partial, raw string) {
	block := s.blocks[index]
	switch kind {
	case "text_delta":
		if text == "" {
			return
		}
		s.text.WriteString(text)
		ev := s.base(KindText, text, true)
		ev.Subtype = kind
		ev.Raw = rawMessage(raw)
		s.emit(ev)
	case "thinking_delta":
		if thinking == "" {
			return
		}
		ev := s.base(KindReasoning, thinking, true)
		ev.Subtype = kind
		s.emit(ev)
	case "input_json_delta":
		if block != nil {
			block.buf.WriteString(partial)
		}
	}
}

func (s *anthState) flushBlocks() {
	indexes := make([]int, 0, len(s.blocks))
	for index := range s.blocks {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)
	for _, index := range indexes {
		s.stopBlock(index)
	}
}

func (s *anthState) stopBlock(index int) {
	block := s.blocks[index]
	if block == nil || block.done || block.kind != "tool_use" {
		return
	}
	block.done = true
	if block.name == "" {
		return
	}
	call := ToolCall{ID: block.id, Name: block.name, Arguments: normalizeArgs(block.buf.String())}
	s.calls = append(s.calls, call)
	ev := s.base(KindTool, string(call.Arguments), false)
	ev.Tool = call.Name
	ev.Type = "tool_use"
	s.emit(ev)
}

func (s *anthState) finish() {
	if s.finished {
		return
	}
	s.finished = true
	s.flushBlocks()
	if stopFailed(s.reason) || (s.text.Len() == 0 && len(s.calls) == 0) {
		s.failed = true
	}
	text := s.text.String()
	if s.failed && text == "" {
		text = "empty response"
		if stopFailed(s.reason) && s.reason != "" {
			text = s.reason
		}
	}
	if s.failed {
		ev := s.base(KindError, text, false)
		ev.Final = true
		ev.Failed = true
		ev.Type = "error"
		s.emit(ev)
		return
	}
	ev := s.base(KindResult, text, false)
	ev.Final = true
	ev.Type = "result"
	s.emit(ev)
}

func (s *anthState) fail(message string) {
	if strings.TrimSpace(message) == "" {
		message = "provider reported a failure"
	}
	s.failed = true
	s.finished = true
	ev := s.base(KindError, message, false)
	ev.Final = true
	ev.Failed = true
	ev.Type = "error"
	s.emit(ev)
}

func (s *anthState) base(kind Kind, text string, partial bool) Event {
	return Event{
		Kind:      kind,
		Type:      "message",
		Text:      text,
		Model:     s.model,
		SessionID: s.session,
		Partial:   partial,
	}
}

func bytesTrim(raw json.RawMessage) []byte {
	return []byte(strings.TrimSpace(string(raw)))
}
