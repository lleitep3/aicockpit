package agents

import (
	"path/filepath"
	"testing"
)

func TestCodexArgs(t *testing.T) {
	adapter := Codex{}
	if adapter.ID() != "codex" || adapter.Binary() != "codex" || adapter.DisplayName() != "Codex CLI" {
		t.Fatalf("identity = %s %s %s", adapter.ID(), adapter.Binary(), adapter.DisplayName())
	}
	if got := adapter.VersionArgs(); len(got) != 1 || got[0] != "--version" {
		t.Fatalf("version args = %#v", got)
	}
	if adapter.DropEnv() != nil {
		t.Fatalf("drop = %#v", adapter.DropEnv())
	}

	dir := t.TempDir()
	tests := []struct {
		name string
		task Task
		want []string
	}{
		{
			name: "default",
			task: Task{Prompt: "summarize the repo structure"},
			want: []string{"exec", "--json", "--skip-git-repo-check", "summarize the repo structure"},
		},
		{
			name: "model and directory",
			task: Task{Prompt: "hi", Model: "gpt-5.4", WorkDir: dir},
			want: []string{"exec", "--json", "--skip-git-repo-check", "--model", "gpt-5.4", "--cd", dir, "hi"},
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

func TestCodexFixtures(t *testing.T) {
	tests := []struct {
		file    string
		text    string
		failed  bool
		session string
		tools   int
	}{
		{
			file:    "official.jsonl",
			text:    "Repo contains docs, sdk, and examples directories.",
			session: "0199a213-81c0-7800-8aa1-bbab2a035a53",
			tools:   1,
		},
		{file: "failed.jsonl", text: "sandbox denied the command", failed: true, session: "thread-fail"},
		{file: "error.jsonl", text: "authentication failed", failed: true, session: "thread-err"},
		{file: "items.jsonl", text: "Repo contains docs.", session: "thread-items", tools: 4},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			events := parseFixture(t, Codex{}.NewParser(), filepath.Join("testdata", "codex", tt.file))
			final := lastFinal(t, events)
			if final.Text != tt.text || final.Failed != tt.failed || final.SessionID != tt.session {
				t.Fatalf("final = %+v", final)
			}
			var tools int
			var sawReasoning, sawItemError, sawTodo, sawPartial bool
			for _, ev := range events {
				if ev.Kind == KindTool {
					tools++
				}
				if ev.Kind == KindReasoning {
					sawReasoning = true
				}
				if ev.Kind == KindError && !ev.Final {
					sawItemError = true
				}
				if ev.Subtype == "todo_list" {
					sawTodo = true
				}
				if ev.Kind == KindText && ev.Text == "partial" && !ev.Final {
					sawPartial = true
				}
			}
			if tools != tt.tools {
				t.Fatalf("tools = %d, events = %+v", tools, events)
			}
			if tt.file == "official.jsonl" {
				if events[2].Tool != "command" || events[2].Text != "bash -lc ls" {
					t.Fatalf("command event = %+v", events[2])
				}
			}
			if tt.file == "items.jsonl" && (!sawReasoning || !sawItemError || !sawTodo || !sawPartial) {
				t.Fatalf("events = %+v", events)
			}
			if tt.file == "items.jsonl" {
				var mcp Event
				for _, ev := range events {
					if ev.Subtype == "mcp_tool_call" {
						mcp = ev
					}
				}
				if mcp.Tool != "search" || mcp.Text != "kb/search" {
					t.Fatalf("mcp = %+v", mcp)
				}
			}
		})
	}
}

func TestCodexParseEdges(t *testing.T) {
	tests := []struct {
		name string
		line string
		kind Kind
		text string
		fail bool
		skip bool
	}{
		{name: "blank", line: " ", skip: true},
		{name: "bad json", line: "{", fail: true},
		{name: "error string", line: `{"type":"turn.failed","error":"plain"}`, kind: KindError, text: "plain"},
		{name: "error raw", line: `{"type":"turn.failed","error":{"other":1}}`, kind: KindError, text: `{"other":1}`},
		{name: "error field fallback", line: `{"type":"error","error":{"message":"from-field"}}`, kind: KindError, text: "from-field"},
		{name: "mcp tool only", line: `{"type":"item.completed","item":{"id":"i","type":"mcp_tool_call","tool":"search"}}`, kind: KindTool, text: "search"},
		{name: "missing item", line: `{"type":"item.completed"}`, kind: KindStatus},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := Codex{}.NewParser()
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
			if tt.skip {
				if len(events) != 0 {
					t.Fatalf("events = %+v", events)
				}
				return
			}
			if len(events) != 1 || events[0].Kind != tt.kind || events[0].Text != tt.text {
				t.Fatalf("event = %+v", events[0])
			}
		})
	}
}
