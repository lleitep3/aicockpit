package agents

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClaudeArgs(t *testing.T) {
	adapter := Claude{}
	if adapter.ID() != "claude" || adapter.Binary() != "claude" || adapter.DisplayName() != "Claude CLI" {
		t.Fatalf("identity = %s %s %s", adapter.ID(), adapter.Binary(), adapter.DisplayName())
	}
	if got := adapter.VersionArgs(); len(got) != 1 || got[0] != "--version" {
		t.Fatalf("version args = %#v", got)
	}
	drop := adapter.DropEnv()
	if len(drop) != 2 || drop[0] != "CLAUDECODE" || drop[1] != "CLAUDE_CODE_ENTRYPOINT" {
		t.Fatalf("drop = %#v", drop)
	}

	tests := []struct {
		name string
		task Task
		want []string
	}{
		{
			name: "default",
			task: Task{Prompt: "Explain recursion"},
			want: []string{"-p", "--output-format", "stream-json", "--verbose", "--include-partial-messages", "Explain recursion"},
		},
		{
			name: "model",
			task: Task{Prompt: "hi", Model: "sonnet"},
			want: []string{"-p", "--output-format", "stream-json", "--verbose", "--include-partial-messages", "--model", "sonnet", "hi"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := adapter.Args(tt.task)
			if err != nil {
				t.Fatal(err)
			}
			if stringsJoin(got) != stringsJoin(tt.want) {
				t.Fatalf("args = %#v", got)
			}
		})
	}
}

func TestClaudeFixtures(t *testing.T) {
	tests := []struct {
		file    string
		text    string
		failed  bool
		session string
		model   string
		partial int
		tools   int
	}{
		{file: "success.jsonl", text: "Hello from claude", session: "sess-1", model: "claude-sonnet-5", partial: 2, tools: 1},
		{file: "error.jsonl", text: "Not authenticated", failed: true, session: "sess-err"},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			events := parseFixture(t, Claude{}.NewParser(), filepath.Join("testdata", "claude", tt.file))
			final := lastFinal(t, events)
			if final.Text != tt.text || final.Failed != tt.failed || final.SessionID != tt.session {
				t.Fatalf("final = %+v", final)
			}
			if tt.failed && final.Kind != KindError {
				t.Fatalf("kind = %s", final.Kind)
			}
			if !tt.failed && final.Kind != KindResult {
				t.Fatalf("kind = %s", final.Kind)
			}
			var partial, tools int
			var sawReasoning, sawUser, sawRetry bool
			var model string
			for _, ev := range events {
				if ev.Partial {
					partial++
				}
				if ev.Kind == KindTool {
					tools++
					if ev.Tool != "Read" {
						t.Fatalf("tool = %+v", ev)
					}
				}
				if ev.Kind == KindReasoning && ev.Text == "hmm" {
					sawReasoning = true
				}
				if ev.Type == "user" {
					sawUser = true
				}
				if ev.Subtype == "api_retry" {
					sawRetry = true
				}
				if ev.Model != "" {
					model = ev.Model
				}
				if len(ev.Raw) == 0 {
					t.Fatal("missing raw")
				}
			}
			if partial != tt.partial || tools != tt.tools {
				t.Fatalf("partial=%d tools=%d", partial, tools)
			}
			if tt.model != "" && model != tt.model {
				t.Fatalf("model = %s", model)
			}
			if tt.file == "success.jsonl" && (!sawReasoning || !sawUser) {
				t.Fatalf("events = %+v", events)
			}
			if tt.file == "error.jsonl" && !sawRetry {
				t.Fatalf("events = %+v", events)
			}
		})
	}
}

func TestClaudeParseEdges(t *testing.T) {
	parser := Claude{}.NewParser()
	tests := []struct {
		name    string
		line    string
		want    int
		kind    Kind
		text    string
		partial bool
		fail    bool
	}{
		{name: "blank", line: "  \n", want: 0},
		{name: "bad json", line: "{", fail: true},
		{name: "result object", line: `{"type":"result","subtype":"success","result":{"ok":true},"session_id":"s"}`, want: 1, kind: KindResult, text: `{"ok":true}`},
		{name: "is_error only", line: `{"type":"result","is_error":true,"result":"nope"}`, want: 1, kind: KindError, text: "nope"},
		{name: "empty assistant", line: `{"type":"assistant","message":{"content":[]}}`, want: 1, kind: KindStatus},
		{name: "unknown", line: `{"type":"rate_limit_event"}`, want: 1, kind: KindStatus},
		{name: "empty delta", line: `{"type":"stream_event","event":{"type":"content_block_stop"}}`, want: 0},
		{name: "text delta", line: `{"type":"stream_event","event":{"delta":{"type":"text_delta","text":"Hi"}}}`, want: 1, kind: KindText, text: "Hi", partial: true},
		{name: "tool only", line: `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Bash"}]}}`, want: 1, kind: KindTool, text: "Bash"},
		{name: "content fallback", line: `{"type":"assistant","message":{"content":1}}`, want: 1, kind: KindText, text: "1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events, err := parser.ParseLine([]byte(tt.line))
			if tt.fail {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(events) != tt.want {
				t.Fatalf("len = %d, events = %+v", len(events), events)
			}
			if tt.want == 0 {
				return
			}
			if events[0].Kind != tt.kind || events[0].Text != tt.text || events[0].Partial != tt.partial {
				t.Fatalf("event = %+v", events[0])
			}
		})
	}
}

func parseFixture(t *testing.T, parser Parser, path string) []Event {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	events, err := ReadEvents(parser, raw)
	if err != nil {
		t.Fatal(err)
	}
	return events
}

func lastFinal(t *testing.T, events []Event) Event {
	t.Helper()
	var final Event
	var saw bool
	for _, ev := range events {
		if ev.Final {
			final = ev
			saw = true
		}
	}
	if !saw {
		t.Fatalf("no final event in %+v", events)
	}
	return final
}

func stringsJoin(parts []string) string {
	return string(joinParts(parts))
}
