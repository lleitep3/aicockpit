package agents

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// maxAPIBody bounds a provider response so a stream cannot grow without limit.
var maxAPIBody = 8 << 20

const errorBodyLimit = 4096

// minRedactLen avoids wiping ordinary words when a test or provider uses a very short token.
const minRedactLen = 8

type apiSummary struct {
	text    string
	model   string
	session string
	calls   []ToolCall
	failed  bool
}

type maxReader struct {
	r     io.Reader
	n     int
	limit int
}

func (m *maxReader) Read(p []byte) (int, error) {
	if m.limit <= 0 {
		m.limit = 8 << 20
	}
	if m.n > m.limit {
		return 0, errBodyLimit
	}
	remain := m.limit - m.n + 1
	if len(p) > remain {
		p = p[:remain]
	}
	n, err := m.r.Read(p)
	m.n += n
	if m.n > m.limit {
		return n, errBodyLimit
	}
	return n, err
}

func endpointURL(provider, base string) (string, error) {
	base = strings.TrimSpace(base)
	if base == "" || strings.ContainsAny(base, "\r\n\x00 ") {
		return "", fmt.Errorf("agents api: %w", ErrInvalidBaseURL)
	}
	u, err := url.Parse(base)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.Fragment != "" {
		return "", fmt.Errorf("agents api: %w", ErrInvalidBaseURL)
	}
	path := strings.TrimRight(u.Path, "/")
	switch provider {
	case ProviderAnthropic:
		switch {
		case strings.HasSuffix(path, "/v1/messages"):
		case strings.HasSuffix(path, "/v1"):
			path += "/messages"
		default:
			path += "/v1/messages"
		}
	default:
		switch {
		case strings.HasSuffix(path, "/chat/completions"):
		case strings.HasSuffix(path, "/v1"):
			path += "/chat/completions"
		default:
			path += "/v1/chat/completions"
		}
	}
	u.Path = path
	return u.String(), nil
}

func newProviderRequest(ctx context.Context, cfg ResolvedAPI, endpoint, token, model, prompt string, tools []ToolSpec) (*http.Request, error) {
	if strings.TrimSpace(token) == "" {
		return nil, ErrTokenMissing
	}
	var (
		body []byte
		err  error
	)
	switch cfg.Provider {
	case ProviderAnthropic:
		body, err = anthropicBody(model, prompt, cfg.MaxTokens, tools)
	default:
		body, err = openAIBody(model, prompt, cfg.MaxTokens, tools)
	}
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream, application/json")
	// The token is sent only as a header. It is not placed in the URL or body.
	if cfg.Provider == ProviderAnthropic {
		req.Header.Set("x-api-key", token)
		req.Header.Set("anthropic-version", anthropicVersion)
	} else {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req, nil
}

func parseProvider(provider string, body io.Reader, model string, emit func(Event)) (apiSummary, error) {
	if provider == ProviderAnthropic {
		return parseAnthropic(body, model, emit)
	}
	return parseOpenAI(body, model, emit)
}

func readErrorMessage(body io.Reader, secret string) string {
	data, _ := io.ReadAll(&maxReader{r: body, limit: errorBodyLimit})
	msg := apiErrorMessage(data)
	msg = redact(msg, secret)
	if strings.TrimSpace(msg) == "" {
		return "empty error response"
	}
	return clip(msg)
}

func apiErrorMessage(body []byte) string {
	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		return ""
	}
	var wrap struct {
		Type    string `json:"type"`
		Message string `json:"message"`
		Error   struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &wrap) == nil {
		if wrap.Error.Message != "" {
			return wrap.Error.Message
		}
		if wrap.Message != "" {
			return wrap.Message
		}
	}
	return string(body)
}

func redact(text, secret string) string {
	if text == "" || len(secret) < minRedactLen {
		return text
	}
	text = strings.ReplaceAll(text, secret, "[redacted]")
	if enc := url.QueryEscape(secret); enc != secret {
		text = strings.ReplaceAll(text, enc, "[redacted]")
	}
	return text
}

func scrubEvent(ev Event, secret string) Event {
	ev.Type = redact(ev.Type, secret)
	ev.Subtype = redact(ev.Subtype, secret)
	ev.Text = redact(ev.Text, secret)
	ev.SessionID = redact(ev.SessionID, secret)
	ev.Model = redact(ev.Model, secret)
	ev.Tool = redact(ev.Tool, secret)
	if len(ev.Raw) > 0 {
		ev.Raw = json.RawMessage(redact(string(ev.Raw), secret))
	}
	return ev
}

func scrubCalls(calls []ToolCall, secret string) []ToolCall {
	out := cloneCalls(calls)
	for i := range out {
		out[i].ID = redact(out[i].ID, secret)
		out[i].Name = redact(out[i].Name, secret)
		if len(out[i].Arguments) > 0 {
			out[i].Arguments = json.RawMessage(redact(string(out[i].Arguments), secret))
		}
	}
	return out
}

func readSSE(r io.Reader, fn func(event, data string) error) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), maxStdoutLine)
	var event string
	var data strings.Builder
	flush := func() error {
		if data.Len() == 0 && event == "" {
			return nil
		}
		err := fn(event, data.String())
		event = ""
		data.Reset()
		return err
	}
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")
		if line == "" {
			if err := flush(); err != nil {
				if errors.Is(err, errStreamEnd) {
					return nil
				}
				return err
			}
			continue
		}
		if strings.HasPrefix(line, ":") {
			continue
		}
		field, value, _ := strings.Cut(line, ":")
		value = strings.TrimPrefix(value, " ")
		switch field {
		case "data":
			if data.Len() > 0 {
				data.WriteByte('\n')
			}
			data.WriteString(value)
		case "event":
			event = strings.TrimSpace(value)
		}
	}
	if err := scanner.Err(); err != nil {
		if errors.Is(err, errBodyLimit) {
			return err
		}
		return fmt.Errorf("%w: %s", ErrBadOutput, clip(err.Error()))
	}
	if err := flush(); err != nil && !errors.Is(err, errStreamEnd) {
		return err
	}
	return nil
}

func sniffBody(r io.Reader, onJSON func([]byte) error, onSSE func(io.Reader) error) error {
	br := bufio.NewReader(r)
	var first byte
	for {
		b, err := br.ReadByte()
		if err != nil {
			if err == io.EOF {
				return io.EOF
			}
			return err
		}
		if b == ' ' || b == '\n' || b == '\r' || b == '\t' {
			continue
		}
		first = b
		break
	}
	if err := br.UnreadByte(); err != nil {
		return err
	}
	if first == '{' {
		data, err := io.ReadAll(br)
		if err != nil {
			return err
		}
		return onJSON(data)
	}
	return onSSE(br)
}

func jsonText(raw json.RawMessage) (string, bool) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return "", false
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text, text != ""
	}
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &parts); err != nil {
		return "", false
	}
	var b strings.Builder
	for _, part := range parts {
		if part.Text == "" {
			continue
		}
		if part.Type != "" && part.Type != "text" {
			continue
		}
		b.WriteString(part.Text)
	}
	if b.Len() == 0 {
		return "", false
	}
	return b.String(), true
}

func stopFailed(reason string) bool {
	switch strings.ToLower(strings.TrimSpace(reason)) {
	case "", "stop", "end_turn", "tool_use", "tool_calls", "length", "max_tokens":
		return false
	default:
		return true
	}
}

func rawMessage(data string) json.RawMessage {
	data = strings.TrimSpace(data)
	if data == "" || !json.Valid([]byte(data)) {
		return nil
	}
	return json.RawMessage(data)
}

func validateTools(tools []ToolSpec) error {
	for _, tool := range tools {
		if !toolNamePattern.MatchString(tool.Name) {
			return errInvalidTool
		}
		if _, err := schemaOrEmpty(tool.InputSchema); err != nil {
			return err
		}
	}
	return nil
}
