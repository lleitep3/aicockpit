package agents

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

type openAIRequest struct {
	Model     string          `json:"model"`
	Stream    bool            `json:"stream"`
	Messages  []openAIMessage `json:"messages"`
	MaxTokens int             `json:"max_tokens,omitempty"`
	Tools     []openAITool    `json:"tools,omitempty"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAITool struct {
	Type     string         `json:"type"`
	Function openAIFunction `json:"function"`
}

type openAIFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

type openAIChunk struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Model   string         `json:"model"`
	Choices []openAIChoice `json:"choices"`
	Error   struct {
		Message string `json:"message"`
	} `json:"error"`
}

type openAIChoice struct {
	Index        int            `json:"index"`
	FinishReason string         `json:"finish_reason"`
	Delta        openAIDelta    `json:"delta"`
	Message      openAIResponse `json:"message"`
}

type openAIDelta struct {
	Content          json.RawMessage   `json:"content"`
	ReasoningContent json.RawMessage   `json:"reasoning_content"`
	ToolCalls        []openAIToolDelta `json:"tool_calls"`
}

type openAIResponse struct {
	Content          json.RawMessage   `json:"content"`
	ReasoningContent json.RawMessage   `json:"reasoning_content"`
	ToolCalls        []openAIToolFinal `json:"tool_calls"`
}

type openAIToolDelta struct {
	Index    int    `json:"index"`
	ID       string `json:"id"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type openAIToolFinal struct {
	ID       string `json:"id"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type oaiTool struct {
	id   string
	name string
	args strings.Builder
}

type oaiState struct {
	emit      func(Event)
	model     string
	session   string
	text      strings.Builder
	tools     map[int]*oaiTool
	toolOrder []int
	calls     []ToolCall
	finished  bool
	failed    bool
	reason    string
	toolsSent bool
}

func openAIBody(model, prompt string, maxTokens int, tools []ToolSpec) ([]byte, error) {
	if err := validateTools(tools); err != nil {
		return nil, err
	}
	req := openAIRequest{
		Model:     model,
		Stream:    true,
		Messages:  []openAIMessage{{Role: "user", Content: prompt}},
		MaxTokens: maxTokens,
	}
	for _, tool := range tools {
		schema, err := schemaOrEmpty(tool.InputSchema)
		if err != nil {
			return nil, err
		}
		req.Tools = append(req.Tools, openAITool{
			Type: "function",
			Function: openAIFunction{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  schema,
			},
		})
	}
	return json.Marshal(req)
}

func parseOpenAI(r io.Reader, fallback string, emit func(Event)) (apiSummary, error) {
	st := &oaiState{emit: emit, model: fallback}
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

func (s *oaiState) summary() apiSummary {
	return apiSummary{
		text:    s.text.String(),
		model:   s.model,
		session: s.session,
		calls:   cloneCalls(s.calls),
		failed:  s.failed,
	}
}

func (s *oaiState) consumeJSON(data []byte) error {
	var chunk openAIChunk
	if err := json.Unmarshal(data, &chunk); err != nil {
		return fmt.Errorf("%w: %s", ErrBadOutput, clip(err.Error()))
	}
	if chunk.Error.Message != "" {
		s.fail(chunk.Error.Message)
		return nil
	}
	s.noteMeta(chunk.ID, chunk.Model)
	if len(chunk.Choices) == 0 {
		s.finish()
		return nil
	}
	choice := chunk.Choices[0]
	s.reason = choice.FinishReason
	if text, ok := jsonText(choice.Message.Content); ok {
		s.text.WriteString(text)
		s.emit(s.base(KindText, text, false))
	}
	if text, ok := jsonText(choice.Message.ReasoningContent); ok {
		s.emit(s.base(KindReasoning, text, false))
	}
	for i, tool := range choice.Message.ToolCalls {
		delta := openAIToolDelta{Index: i, ID: tool.ID}
		delta.Function.Name = tool.Function.Name
		delta.Function.Arguments = tool.Function.Arguments
		s.addTool(delta)
	}
	if text, ok := jsonText(choice.Delta.Content); ok && s.text.Len() == 0 {
		s.text.WriteString(text)
		ev := s.base(KindText, text, false)
		s.emit(ev)
	}
	s.finish()
	return nil
}

func (s *oaiState) consumeSSE(eventName, data string) error {
	data = strings.TrimSpace(data)
	if data == "" {
		return nil
	}
	if data == "[DONE]" {
		s.finish()
		return errStreamEnd
	}
	var chunk openAIChunk
	if err := json.Unmarshal([]byte(data), &chunk); err != nil {
		return fmt.Errorf("%w: %s", ErrBadOutput, clip(err.Error()))
	}
	if chunk.Error.Message != "" || eventName == "error" {
		message := chunk.Error.Message
		if message == "" {
			message = "provider reported a failure"
		}
		s.fail(message)
		return errStreamEnd
	}
	s.noteMeta(chunk.ID, chunk.Model)
	for _, choice := range chunk.Choices {
		if choice.Index != 0 && len(chunk.Choices) > 1 {
			continue
		}
		if choice.FinishReason != "" {
			s.reason = choice.FinishReason
		}
		if text, ok := jsonText(choice.Delta.Content); ok {
			s.text.WriteString(text)
			ev := s.base(KindText, text, true)
			ev.Type = "chat.completion.chunk"
			ev.Raw = rawMessage(data)
			s.emit(ev)
		}
		if text, ok := jsonText(choice.Delta.ReasoningContent); ok {
			ev := s.base(KindReasoning, text, true)
			ev.Subtype = "reasoning"
			s.emit(ev)
		}
		for _, tool := range choice.Delta.ToolCalls {
			s.addTool(tool)
		}
	}
	return nil
}

func (s *oaiState) noteMeta(id, model string) {
	if id != "" {
		s.session = id
	}
	if model != "" {
		s.model = model
	}
}

func (s *oaiState) addTool(delta openAIToolDelta) {
	if s.tools == nil {
		s.tools = map[int]*oaiTool{}
	}
	acc, ok := s.tools[delta.Index]
	if !ok {
		acc = &oaiTool{}
		s.tools[delta.Index] = acc
		s.toolOrder = append(s.toolOrder, delta.Index)
	}
	if delta.ID != "" {
		acc.id = delta.ID
	}
	if delta.Function.Name != "" {
		acc.name += delta.Function.Name
	}
	if delta.Function.Arguments != "" {
		acc.args.WriteString(delta.Function.Arguments)
	}
}

func (s *oaiState) finish() {
	if s.finished {
		return
	}
	s.finished = true
	if stopFailed(s.reason) {
		s.failed = true
	}
	if !s.toolsSent {
		s.toolsSent = true
		for _, index := range s.toolOrder {
			acc := s.tools[index]
			if acc == nil || acc.name == "" {
				continue
			}
			call := ToolCall{ID: acc.id, Name: acc.name, Arguments: normalizeArgs(acc.args.String())}
			s.calls = append(s.calls, call)
			ev := s.base(KindTool, string(call.Arguments), false)
			ev.Tool = call.Name
			ev.Type = "tool_calls"
			s.emit(ev)
		}
	}
	if !s.failed && s.text.Len() == 0 && len(s.calls) == 0 {
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

func (s *oaiState) fail(message string) {
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

func (s *oaiState) base(kind Kind, text string, partial bool) Event {
	return Event{
		Kind:      kind,
		Type:      "chat.completion.chunk",
		Text:      text,
		Model:     s.model,
		SessionID: s.session,
		Partial:   partial,
	}
}
