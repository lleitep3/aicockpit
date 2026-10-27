package ask

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/lleitep3/aicockpit/internal/agents"
	"github.com/lleitep3/aicockpit/internal/memory"
)

type scriptBackend struct {
	id      string
	events  []agents.Event
	text    string
	model   string
	session string
	err     error
	task    agents.Task
}

func (s *scriptBackend) ID() string { return s.id }

func (s *scriptBackend) Run(ctx context.Context, task agents.Task) (*agents.Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.task = task
	for _, ev := range s.events {
		if task.OnEvent != nil {
			task.OnEvent(ev)
		}
	}
	result := &agents.Result{Backend: s.id, Text: s.text, Model: s.model, SessionID: s.session}
	if s.err != nil {
		return result, s.err
	}
	result.ExitCode = 0
	return result, nil
}

type fakeCatalog struct {
	auto    []Candidate
	autoErr error
	check   func(Ref) (Candidate, error)
	open    func(Ref) (agents.Backend, error)
	opened  []Ref
}

func (f *fakeCatalog) Check(ctx context.Context, ref Ref) (Candidate, error) {
	if err := ctx.Err(); err != nil {
		return Candidate{}, err
	}
	if f.check != nil {
		return f.check(ref)
	}
	for _, candidate := range f.auto {
		if candidate.Ref.Raw == ref.Raw || (ref.Kind == KindAPI && ref.ID == "" && candidate.Ref.Kind == KindAPI) {
			return candidate, nil
		}
	}
	return Candidate{Ref: ref, Reason: "not installed"}, nil
}

func (f *fakeCatalog) Auto(ctx context.Context) ([]Candidate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if f.autoErr != nil {
		return nil, f.autoErr
	}
	return f.auto, nil
}

func (f *fakeCatalog) Open(ctx context.Context, ref Ref) (agents.Backend, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f.opened = append(f.opened, ref)
	if f.open != nil {
		return f.open(ref)
	}
	return nil, errors.New("not opened")
}

func claudeCandidate(available bool) Candidate {
	candidate := Candidate{Ref: Ref{Kind: KindCLI, ID: "claude", Raw: "claude"}}
	if available {
		candidate.Available = true
		return candidate
	}
	candidate.Reason = "claude is not installed"
	return candidate
}

func TestParseRef(t *testing.T) {
	tests := []struct {
		raw     string
		want    Ref
		wantErr error
	}{
		{raw: " claude ", want: Ref{Kind: KindCLI, ID: "claude", Raw: "claude"}},
		{raw: "CODEX", want: Ref{Kind: KindCLI, ID: "codex", Raw: "codex"}},
		{raw: "api", want: Ref{Kind: KindAPI, Raw: "api"}},
		{raw: "api:OpenAI", want: Ref{Kind: KindAPI, ID: "openai", Raw: "api:openai"}},
		{raw: "api:open-router", want: Ref{Kind: KindAPI, ID: "openrouter", Raw: "api:openrouter"}},
		{raw: "api:", wantErr: agents.ErrUnknownProvider},
		{raw: "api:nope", wantErr: agents.ErrUnknownProvider},
		{raw: "gemini", wantErr: agents.ErrUnknownBackend},
		{raw: "", wantErr: agents.ErrUnknownBackend},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			got, err := ParseRef(tt.raw)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("error = %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("ref = %+v", got)
			}
		})
	}
}

func TestRunStreamsRecallAndRetain(t *testing.T) {
	ctx := context.Background()
	local := memory.NewFake()
	if _, err := local.Retain(ctx, memory.Item{Title: "Aqueduct", Content: "the aqueduct is open"}); err != nil {
		t.Fatal(err)
	}
	extra := memory.NewFake()
	if _, err := extra.Retain(ctx, memory.Item{Title: "West", Content: "the aqueduct west gate"}); err != nil {
		t.Fatal(err)
	}
	worker := &scriptBackend{
		id:      "claude",
		model:   "sonnet",
		session: "sess-1",
		text:    "Hello",
		events: []agents.Event{
			{Kind: agents.KindText, Text: "Hel", Partial: true},
			{Kind: agents.KindText, Text: "lo", Partial: true},
			{Kind: agents.KindText, Text: "Hello"},
			{Kind: agents.KindResult, Text: "Hello", Final: true},
		},
	}
	catalog := &fakeCatalog{
		auto: []Candidate{claudeCandidate(true)},
		open: func(Ref) (agents.Backend, error) { return worker, nil },
	}
	var out, errOut bytes.Buffer
	outcome, err := Run(ctx, Request{
		Task:    "check the aqueduct",
		Model:   "haiku",
		WorkDir: "/work",
		Timeout: time.Second,
	}, Deps{
		Memory:  local,
		Extra:   extra,
		Catalog: catalog,
		Out:     &out,
		ErrOut:  &errOut,
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.String() != "Hello\n" {
		t.Fatalf("output = %q", out.String())
	}
	if errOut.Len() != 0 {
		t.Fatalf("warnings = %q", errOut.String())
	}
	if outcome.Backend != "claude" || outcome.Model != "sonnet" || outcome.Text != "Hello" || outcome.SessionID != "sess-1" || !outcome.Retained || outcome.Recalled != 2 {
		t.Fatalf("outcome = %+v", outcome)
	}
	if worker.task.Model != "haiku" || worker.task.WorkDir != "/work" || worker.task.Timeout != time.Second {
		t.Fatalf("task = %+v", worker.task)
	}
	if !strings.Contains(worker.task.Prompt, "check the aqueduct") || !strings.Contains(worker.task.Prompt, "Aqueduct") || !strings.Contains(worker.task.Prompt, "West") {
		t.Fatalf("prompt = %q", worker.task.Prompt)
	}
	if len(local.RetainCalls) < 2 {
		t.Fatalf("retains = %d", len(local.RetainCalls))
	}
	record := local.RetainCalls[len(local.RetainCalls)-1]
	if !strings.Contains(record.Content, "Task: check the aqueduct") || !strings.Contains(record.Content, "Result: Hello") || !strings.Contains(record.Content, "Backend: claude") {
		t.Fatalf("record = %q", record.Content)
	}
	if len(extra.RetainCalls) != 2 {
		t.Fatalf("extra retains = %d", len(extra.RetainCalls))
	}
	if len(local.RecallCalls) != 1 || local.RecallCalls[0].Text != "check the aqueduct" {
		t.Fatalf("recall = %+v", local.RecallCalls)
	}
}

func TestRunTable(t *testing.T) {
	worker := func(text string, err error) *scriptBackend {
		events := []agents.Event{{Kind: agents.KindResult, Text: text, Final: true}}
		if err != nil {
			events = []agents.Event{{Kind: agents.KindError, Text: text}}
		}
		return &scriptBackend{id: "codex", text: text, err: err, events: events}
	}
	ready := func(ref Ref) (Candidate, error) {
		return Candidate{Ref: ref, Available: true}, nil
	}
	tests := []struct {
		name    string
		req     Request
		setup   func() (Deps, *scriptBackend, *memory.Fake)
		wantErr error
		wantOut string
		wantB   string
	}{
		{
			name: "flag overrides config",
			req:  Request{Task: "ship it", Backend: "codex"},
			setup: func() (Deps, *scriptBackend, *memory.Fake) {
				mem := memory.NewFake()
				got := worker("done", nil)
				return Deps{
					Memory:        mem,
					ConfigBackend: "claude",
					Catalog: &fakeCatalog{
						check: ready,
						open: func(ref Ref) (agents.Backend, error) {
							if ref.Raw != "codex" {
								t.Fatalf("opened %s", ref.Raw)
							}
							return got, nil
						},
					},
				}, got, mem
			},
			wantOut: "done\n",
			wantB:   "codex",
		},
		{
			name: "config backend",
			req:  Request{Task: "ship it"},
			setup: func() (Deps, *scriptBackend, *memory.Fake) {
				mem := memory.NewFake()
				got := worker("from-config", nil)
				return Deps{
					Memory:        mem,
					ConfigBackend: "api:openai",
					Catalog: &fakeCatalog{
						check: ready,
						open:  func(Ref) (agents.Backend, error) { return got, nil },
					},
				}, got, mem
			},
			wantOut: "from-config\n",
			wantB:   "api:openai",
		},
		{
			name: "no memory flag",
			req:  Request{Task: "ship it", NoMemory: true},
			setup: func() (Deps, *scriptBackend, *memory.Fake) {
				got := worker("bare", nil)
				return Deps{
					Catalog: &fakeCatalog{
						auto: []Candidate{claudeCandidate(true)},
						open: func(Ref) (agents.Backend, error) { return got, nil },
					},
				}, got, nil
			},
			wantOut: "bare\n",
		},
		{
			name: "json",
			req:  Request{Task: "ship it", JSON: true, NoMemory: true},
			setup: func() (Deps, *scriptBackend, *memory.Fake) {
				got := &scriptBackend{id: "claude", text: "json-answer", model: "sonnet", events: []agents.Event{
					{Kind: agents.KindText, Text: "nope", Partial: true},
					{Kind: agents.KindResult, Text: "json-answer", Final: true},
				}}
				return Deps{
					Catalog: &fakeCatalog{
						auto: []Candidate{claudeCandidate(true)},
						open: func(Ref) (agents.Backend, error) { return got, nil },
					},
				}, got, nil
			},
			wantOut: `"text": "json-answer"`,
			wantB:   "claude",
		},
		{
			name:    "empty task",
			req:     Request{Task: "  "},
			setup:   func() (Deps, *scriptBackend, *memory.Fake) { return Deps{}, nil, nil },
			wantErr: ErrEmptyTask,
		},
		{
			name:    "negative timeout",
			req:     Request{Task: "ship it", Timeout: -time.Second, NoMemory: true},
			setup:   func() (Deps, *scriptBackend, *memory.Fake) { return Deps{}, nil, nil },
			wantErr: errInvalidTimeout,
		},
		{
			name:    "nil catalog",
			req:     Request{Task: "ship it", NoMemory: true},
			setup:   func() (Deps, *scriptBackend, *memory.Fake) { return Deps{}, nil, nil },
			wantErr: ErrNoBackend,
		},
		{
			name: "memory required",
			req:  Request{Task: "ship it"},
			setup: func() (Deps, *scriptBackend, *memory.Fake) {
				return Deps{Catalog: &fakeCatalog{auto: []Candidate{claudeCandidate(true)}}}, nil, nil
			},
			wantErr: ErrMemoryUnavailable,
		},
		{
			name: "unknown flag",
			req:  Request{Task: "ship it", Backend: "gemini", NoMemory: true},
			setup: func() (Deps, *scriptBackend, *memory.Fake) {
				return Deps{Catalog: &fakeCatalog{}}, nil, nil
			},
			wantErr: agents.ErrUnknownBackend,
		},
		{
			name: "bad config backend",
			req:  Request{Task: "ship it", NoMemory: true},
			setup: func() (Deps, *scriptBackend, *memory.Fake) {
				return Deps{Catalog: &fakeCatalog{}, ConfigBackend: "nope"}, nil, nil
			},
			wantErr: agents.ErrUnknownBackend,
		},
		{
			name: "explicit unavailable",
			req:  Request{Task: "ship it", Backend: "claude", NoMemory: true},
			setup: func() (Deps, *scriptBackend, *memory.Fake) {
				return Deps{Catalog: &fakeCatalog{}}, nil, nil
			},
			wantErr: ErrBackendUnavailable,
		},
		{
			name: "none available",
			req:  Request{Task: "ship it", NoMemory: true},
			setup: func() (Deps, *scriptBackend, *memory.Fake) {
				return Deps{Catalog: &fakeCatalog{auto: []Candidate{
					claudeCandidate(false),
					{Ref: Ref{Kind: KindCLI, ID: "codex", Raw: "codex"}},
				}}}, nil, nil
			},
			wantErr: ErrNoBackend,
		},
		{
			name: "empty auto",
			req:  Request{Task: "ship it", NoMemory: true},
			setup: func() (Deps, *scriptBackend, *memory.Fake) {
				return Deps{Catalog: &fakeCatalog{}}, nil, nil
			},
			wantErr: ErrNoBackend,
		},
		{
			name: "recall error",
			req:  Request{Task: "ship it"},
			setup: func() (Deps, *scriptBackend, *memory.Fake) {
				mem := memory.NewFake()
				mem.RecallErr = errors.New("index down")
				return Deps{Memory: mem, Catalog: &fakeCatalog{auto: []Candidate{claudeCandidate(true)}, open: func(Ref) (agents.Backend, error) {
					return worker("x", nil), nil
				}}}, nil, mem
			},
			wantErr: errors.New("index down"),
		},
		{
			name: "run error",
			req:  Request{Task: "ship it", NoMemory: true},
			setup: func() (Deps, *scriptBackend, *memory.Fake) {
				got := worker("boom", errors.New("worker failed"))
				return Deps{Catalog: &fakeCatalog{
					auto: []Candidate{claudeCandidate(true)},
					open: func(Ref) (agents.Backend, error) { return got, nil },
				}}, got, nil
			},
			wantErr: errors.New("worker failed"),
			wantOut: "boom\n",
		},
		{
			name: "retain error",
			req:  Request{Task: "ship it"},
			setup: func() (Deps, *scriptBackend, *memory.Fake) {
				mem := memory.NewFake()
				mem.RetainErr = errors.New("disk full")
				got := worker("kept", nil)
				return Deps{Memory: mem, Catalog: &fakeCatalog{
					auto: []Candidate{claudeCandidate(true)},
					open: func(Ref) (agents.Backend, error) { return got, nil },
				}}, got, mem
			},
			wantErr: errors.New("disk full"),
			wantOut: "kept\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps, _, _ := tt.setup()
			var out bytes.Buffer
			deps.Out = &out
			outcome, err := Run(context.Background(), tt.req, deps)
			if tt.wantErr != nil {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr.Error()) {
					t.Fatalf("error = %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if tt.wantOut != "" && !strings.Contains(out.String(), tt.wantOut) {
				t.Fatalf("output = %q", out.String())
			}
			if tt.wantB != "" && (outcome == nil || outcome.Backend != tt.wantB) {
				t.Fatalf("outcome = %+v", outcome)
			}
		})
	}
}

func TestRunEdges(t *testing.T) {
	if _, err := Run(nil, Request{Task: "x", NoMemory: true}, Deps{}); !errors.Is(err, errNilContext) { //nolint:staticcheck // SA1012: nil context is the case under test.
		t.Fatalf("nil ctx = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Run(ctx, Request{Task: "x", NoMemory: true}, Deps{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled = %v", err)
	}

	mem := memory.NewFake()
	extra := memory.NewFake()
	extra.RecallErr = errors.New("hindsight down")
	extra.RetainErr = errors.New("hindsight retain down")
	worker := &scriptBackend{id: "claude", text: "ok", events: []agents.Event{{Kind: agents.KindError, Text: "notice"}, {Kind: agents.KindResult, Text: "ok", Final: true}}}
	var errOut bytes.Buffer
	_, err := Run(context.Background(), Request{Task: "only local"}, Deps{
		Memory: mem,
		Extra:  extra,
		ErrOut: &errOut,
		Catalog: &fakeCatalog{
			auto: []Candidate{claudeCandidate(true)},
			open: func(Ref) (agents.Backend, error) { return worker, nil },
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(errOut.String(), "hindsight recall skipped") || !strings.Contains(errOut.String(), "hindsight retain skipped") {
		t.Fatalf("warnings = %q", errOut.String())
	}

	checkCatalog := &fakeCatalog{check: func(Ref) (Candidate, error) {
		return Candidate{}, errors.New("probe failed")
	}}
	_, err = Run(context.Background(), Request{Task: "x", Backend: "claude", NoMemory: true}, Deps{Catalog: checkCatalog})
	if err == nil || !strings.Contains(err.Error(), "probe failed") {
		t.Fatalf("check = %v", err)
	}
	openCatalog := &fakeCatalog{
		auto: []Candidate{claudeCandidate(true)},
		open: func(Ref) (agents.Backend, error) { return nil, errors.New("open failed") },
	}
	_, err = Run(context.Background(), Request{Task: "x", NoMemory: true}, Deps{Catalog: openCatalog})
	if err == nil || !strings.Contains(err.Error(), "open failed") {
		t.Fatalf("open = %v", err)
	}
	autoCatalog := &fakeCatalog{autoErr: errors.New("auto failed")}
	_, err = Run(context.Background(), Request{Task: "x", NoMemory: true}, Deps{Catalog: autoCatalog})
	if err == nil || !strings.Contains(err.Error(), "auto failed") {
		t.Fatalf("auto = %v", err)
	}

	var sink bytes.Buffer
	fail := &errWriter{}
	got := &scriptBackend{id: "claude", text: "hidden", events: []agents.Event{{Kind: agents.KindText, Text: "hidden", Partial: true}}}
	_, err = Run(context.Background(), Request{Task: "x", NoMemory: true}, Deps{
		Out: fail,
		Catalog: &fakeCatalog{
			auto: []Candidate{claudeCandidate(true)},
			open: func(Ref) (agents.Backend, error) { return got, nil },
		},
	})
	if err == nil || !strings.Contains(err.Error(), "disk full") {
		t.Fatalf("write = %v", err)
	}
	_, err = Run(context.Background(), Request{Task: "x", JSON: true, NoMemory: true}, Deps{
		Out: fail,
		Catalog: &fakeCatalog{
			auto: []Candidate{claudeCandidate(true)},
			open: func(Ref) (agents.Backend, error) {
				return &scriptBackend{id: "claude", text: "j", events: []agents.Event{{Kind: agents.KindResult, Text: "j", Final: true}}}, nil
			},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "disk full") {
		t.Fatalf("json write = %v", err)
	}

	plain := &scriptBackend{id: "claude", text: "late"}
	outcome, err := Run(context.Background(), Request{Task: "x", NoMemory: true}, Deps{
		Out: &sink,
		Catalog: &fakeCatalog{
			auto: []Candidate{claudeCandidate(true)},
			open: func(Ref) (agents.Backend, error) { return plain, nil },
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sink.String(), "late") || outcome.Text != "late" {
		t.Fatalf("sink = %q outcome = %+v", sink.String(), outcome)
	}

	emptyMem := memory.NewFake()
	empty := &scriptBackend{id: "claude"}
	outcome, err = Run(context.Background(), Request{Task: "x"}, Deps{
		Memory: emptyMem,
		Catalog: &fakeCatalog{
			auto: []Candidate{claudeCandidate(true)},
			open: func(Ref) (agents.Backend, error) { return empty, nil },
		},
	})
	if err != nil || outcome == nil || !outcome.Retained {
		t.Fatalf("empty retain = %v %+v", err, outcome)
	}
	if len(emptyMem.RetainCalls) != 1 || !strings.Contains(emptyMem.RetainCalls[0].Content, "(no output)") {
		t.Fatalf("record = %+v", emptyMem.RetainCalls)
	}
}

type errWriter struct{}

func (errWriter) Write([]byte) (int, error) {
	return 0, errors.New("disk full")
}

func TestPromptAndMerge(t *testing.T) {
	if buildPrompt("task", nil) != "task" {
		t.Fatal("empty hits")
	}
	prompt := buildPrompt("task", []memory.Hit{
		{Item: memory.Item{Content: "body"}},
		{Item: memory.Item{ID: "id-1", Content: strings.Repeat("a", snippetLimit+10)}},
	})
	if !strings.Contains(prompt, "- memory: body") || !strings.Contains(prompt, "id-1") || !strings.Contains(prompt, "...") {
		t.Fatalf("prompt = %q", prompt)
	}
	hits := mergeHits([]memory.Hit{
		{Item: memory.Item{ID: "a", Content: "one"}},
		{Item: memory.Item{ID: "a", Content: "one"}},
	}, []memory.Hit{
		{Item: memory.Item{ID: "b", Content: "two"}},
		{Item: memory.Item{ID: "c", Content: "three"}},
	}, 2)
	if len(hits) != 2 || hits[0].Item.ID != "a" || hits[1].Item.ID != "b" {
		t.Fatalf("hits = %+v", hits)
	}
	if mergeHits(nil, nil, 0) != nil {
		t.Fatal("limit")
	}
	if truncate("abcdef", 3) != "abc" || truncate("ab", 10) != "ab" || truncate("abcdef", 5) != "ab..." {
		t.Fatal("truncate")
	}
	if oneLine("a\n\tb  c") != "a b c" {
		t.Fatal("one line")
	}
	if displayRef(Ref{}) != "backend" || displayRef(Ref{ID: "claude"}) != "claude" {
		t.Fatal("display")
	}
	var buf bytes.Buffer
	if err := writeJSON(&buf, nil); err != nil || !strings.Contains(buf.String(), `"text"`) {
		t.Fatalf("json = %q %v", buf.String(), err)
	}
	var discarded printer
	discarded.onEvent(agents.Event{Kind: agents.KindText, Text: "x"})
	discarded.finish("x")
	warn(nil, "quiet")
	long := record(strings.Repeat("task ", 200), &Outcome{Backend: "claude", Model: "sonnet", Text: strings.Repeat("result ", 400)})
	if !strings.Contains(long.Content, "...") || !strings.Contains(long.Content, "Model: sonnet") {
		t.Fatalf("record = %q", long.Content)
	}
	if !strings.Contains(record("task", &Outcome{Backend: "codex"}).Content, "(no output)") {
		t.Fatal("empty result")
	}
}

func TestPrinterSkipsTools(t *testing.T) {
	var buf bytes.Buffer
	p := &printer{out: &buf}
	p.onEvent(agents.Event{Kind: agents.KindTool, Text: "bash"})
	p.onEvent(agents.Event{Kind: agents.KindReasoning, Text: "thinking"})
	p.onEvent(agents.Event{Kind: agents.KindText, Text: "answer"})
	p.onEvent(agents.Event{Kind: agents.KindResult, Text: "answer", Final: true})
	p.finish("answer")
	if buf.String() != "answer\n" {
		t.Fatalf("output = %q", buf.String())
	}
	var failed bytes.Buffer
	bad := &printer{out: &failed}
	bad.onEvent(agents.Event{Kind: agents.KindResult, Text: "nope", Failed: true})
	bad.onEvent(agents.Event{Kind: agents.KindError, Text: ""})
	if failed.Len() != 0 {
		t.Fatalf("failed output = %q", failed.String())
	}
}
