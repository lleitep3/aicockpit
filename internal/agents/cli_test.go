package agents

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeRunner struct {
	specs    []Spec
	stdout   string
	stderr   string
	code     int
	waitErr  error
	startErr error
	nilIO    bool
	block    bool
	partial  bool
	blocked  chan struct{}
}

func (f *fakeRunner) Start(ctx context.Context, spec Spec) (Proc, error) {
	f.specs = append(f.specs, spec)
	if f.startErr != nil {
		return nil, f.startErr
	}
	var stdout io.Reader
	switch {
	case f.nilIO:
		stdout = nil
	case f.block || f.partial:
		first := f.stdout
		if f.partial {
			first = "{"
		}
		stdout = &blockReader{ctx: ctx, first: []byte(first), blocked: f.blocked}
	default:
		stdout = strings.NewReader(f.stdout)
	}
	var stderr io.Reader
	if f.nilIO {
		stderr = nil
	} else {
		stderr = strings.NewReader(f.stderr)
	}
	return &fakeProc{stdout: stdout, stderr: stderr, code: f.code, waitErr: f.waitErr}, nil
}

type fakeProc struct {
	stdout  io.Reader
	stderr  io.Reader
	code    int
	waitErr error
}

func (p *fakeProc) Stdout() io.Reader { return p.stdout }

func (p *fakeProc) Stderr() io.Reader { return p.stderr }

func (p *fakeProc) Wait() (int, error) { return p.code, p.waitErr }

type blockReader struct {
	ctx     context.Context
	first   []byte
	off     int
	blocked chan struct{}
	once    sync.Once
}

func (b *blockReader) Read(p []byte) (int, error) {
	if b.off < len(b.first) {
		n := copy(p, b.first[b.off:])
		b.off += n
		return n, nil
	}
	if b.blocked != nil {
		b.once.Do(func() { close(b.blocked) })
	}
	<-b.ctx.Done()
	return 0, io.EOF
}

type badArgs struct{ Claude }

func (badArgs) Args(Task) ([]string, error) { return nil, errors.New("bad args") }

func TestNewCLIBackend(t *testing.T) {
	backend, err := NewCLIBackend(" claude ", &fakeRunner{})
	if err != nil || backend.ID() != "claude" {
		t.Fatalf("claude: %v %v", backend, err)
	}
	backend, err = NewCLIBackend("codex", &fakeRunner{})
	if err != nil || backend.ID() != "codex" {
		t.Fatalf("codex: %v %v", backend, err)
	}
	if _, err := NewCLIBackend("gemini", nil); !errors.Is(err, ErrUnknownBackend) {
		t.Fatalf("unknown: %v", err)
	}
	if got := Adapters(); len(got) != 2 || got[0].ID() != "claude" || got[1].ID() != "codex" {
		t.Fatalf("adapters = %v", got)
	}
	var nilBackend *CLIBackend
	if nilBackend.ID() != "" {
		t.Fatal("nil id")
	}
	if _, err := nilBackend.Run(context.Background(), Task{Prompt: "hi"}); !errors.Is(err, ErrUnknownBackend) {
		t.Fatalf("nil run: %v", err)
	}
	if _, err := (&CLIBackend{}).Run(context.Background(), Task{Prompt: "hi"}); !errors.Is(err, ErrUnknownBackend) {
		t.Fatalf("empty run: %v", err)
	}
}

func TestCLIBackendRunFixture(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "claude", "success.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	runner := &fakeRunner{stdout: string(raw)}
	backend, err := NewCLIBackend("claude", runner)
	if err != nil {
		t.Fatal(err)
	}
	var streamed []Event
	result, err := backend.Run(context.Background(), Task{
		Prompt: "Explain recursion",
		Model:  "sonnet",
		OnEvent: func(ev Event) {
			if len(ev.Raw) > 0 {
				ev.Raw[0] = 'X'
			}
			streamed = append(streamed, ev)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Text != "Hello from claude" || result.SessionID != "sess-1" || result.Model != "claude-sonnet-5" || result.Failed {
		t.Fatalf("result = %+v", result)
	}
	if len(streamed) == 0 || len(streamed) != len(result.Events) {
		t.Fatalf("streamed %d stored %d", len(streamed), len(result.Events))
	}
	if result.Events[0].Raw[0] != '{' {
		t.Fatal("stored raw was mutated by the callback")
	}
	args := runner.specs[0].Args
	want := []string{"-p", "--output-format", "stream-json", "--verbose", "--include-partial-messages", "--model", "sonnet", "Explain recursion"}
	if strings.Join(args, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("args = %#v", args)
	}
	if len(runner.specs[0].DropEnv) != 2 {
		t.Fatalf("drop = %#v", runner.specs[0].DropEnv)
	}
}

func TestCLIBackendRunFailures(t *testing.T) {
	success, err := os.ReadFile(filepath.Join("testdata", "codex", "official.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	failed, err := os.ReadFile(filepath.Join("testdata", "codex", "failed.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()

	tests := []struct {
		name     string
		id       string
		task     Task
		runner   *fakeRunner
		nilCtx   bool
		ctx      func() (context.Context, context.CancelFunc)
		wantErr  error
		wantText string
		noStart  bool
	}{
		{
			name:    "nil context",
			id:      "claude",
			task:    Task{Prompt: "hi"},
			runner:  &fakeRunner{},
			nilCtx:  true,
			wantErr: ErrNilContext,
			noStart: true,
		},
		{
			name:    "negative timeout",
			id:      "claude",
			task:    Task{Prompt: "hi", Timeout: -1},
			runner:  &fakeRunner{},
			wantErr: ErrInvalidTimeout,
			noStart: true,
		},
		{
			name:    "not installed",
			id:      "codex",
			task:    Task{Prompt: "hi", WorkDir: dir, Model: "gpt-5.4"},
			runner:  &fakeRunner{startErr: fmt.Errorf("%w: codex", ErrNotInstalled)},
			wantErr: ErrNotInstalled,
		},
		{
			name:    "bad stdout",
			id:      "claude",
			task:    Task{Prompt: "hi"},
			runner:  &fakeRunner{stdout: "not-json\n"},
			wantErr: ErrBadOutput,
		},
		{
			name:     "missing final",
			id:       "claude",
			task:     Task{Prompt: "hi"},
			runner:   &fakeRunner{stdout: "{\"type\":\"system\",\"subtype\":\"init\"}\n", stderr: "still starting"},
			wantErr:  ErrRunFailed,
			wantText: "",
		},
		{
			name:     "nonzero exit",
			id:       "codex",
			task:     Task{Prompt: "summarize the repo structure", WorkDir: dir},
			runner:   &fakeRunner{stdout: string(success), code: 2, stderr: "boom"},
			wantErr:  ErrRunFailed,
			wantText: "Repo contains docs, sdk, and examples directories.",
		},
		{
			name:     "exit without wait error",
			id:       "claude",
			task:     Task{Prompt: "hi"},
			runner:   &fakeRunner{stdout: "{\"type\":\"result\",\"subtype\":\"success\",\"result\":\"x\"}\n", code: 1},
			wantErr:  ErrRunFailed,
			wantText: "x",
		},
		{
			name:     "worker failure",
			id:       "codex",
			task:     Task{Prompt: "hi"},
			runner:   &fakeRunner{stdout: string(failed)},
			wantErr:  ErrRunFailed,
			wantText: "sandbox denied the command",
		},
		{
			name:     "wait error only",
			id:       "claude",
			task:     Task{Prompt: "hi"},
			runner:   &fakeRunner{stdout: "{\"type\":\"result\",\"subtype\":\"success\",\"result\":\"x\"}\n", waitErr: errors.New("killed")},
			wantErr:  ErrRunFailed,
			wantText: "x",
		},
		{
			name:    "empty streams",
			id:      "claude",
			task:    Task{Prompt: "hi"},
			runner:  &fakeRunner{nilIO: true},
			wantErr: ErrRunFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend, err := NewCLIBackend(tt.id, tt.runner)
			if err != nil {
				t.Fatal(err)
			}
			var ctx context.Context
			var cancel context.CancelFunc = func() {}
			if !tt.nilCtx {
				ctx = context.Background()
			}
			if tt.ctx != nil {
				ctx, cancel = tt.ctx()
			}
			defer cancel()
			result, err := backend.Run(ctx, tt.task)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if tt.noStart && len(tt.runner.specs) != 0 {
				t.Fatal("process started")
			}
			if errors.Is(tt.wantErr, ErrNotInstalled) || tt.noStart {
				if result != nil {
					t.Fatal("result should be nil before start")
				}
				return
			}
			if result == nil {
				t.Fatal("missing result")
			}
			if result.Text != tt.wantText {
				t.Fatalf("text = %q", result.Text)
			}
		})
	}
}

func TestCLIBackendCancelAndTimeout(t *testing.T) {
	t.Run("cancel", func(t *testing.T) {
		blocked := make(chan struct{})
		runner := &fakeRunner{
			block:   true,
			stdout:  "{\"type\":\"result\",\"subtype\":\"success\",\"result\":\"late\"}\n",
			blocked: blocked,
		}
		backend, err := NewCLIBackend("claude", runner)
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := make(chan error, 1)
		go func() {
			_, err := backend.Run(ctx, Task{Prompt: "hi"})
			done <- err
		}()
		select {
		case <-blocked:
		case <-time.After(2 * time.Second):
			t.Fatal("run did not block")
		}
		cancel()
		select {
		case err := <-done:
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("err = %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("cancel did not return")
		}
	})

	t.Run("timeout", func(t *testing.T) {
		runner := &fakeRunner{block: true, blocked: make(chan struct{})}
		backend, err := NewCLIBackend("codex", runner)
		if err != nil {
			t.Fatal(err)
		}
		start := time.Now()
		_, err = backend.Run(context.Background(), Task{Prompt: "hi", Timeout: 40 * time.Millisecond})
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("err = %v", err)
		}
		if time.Since(start) > 2*time.Second {
			t.Fatal("timeout took too long")
		}
	})

	t.Run("partial line prefers context", func(t *testing.T) {
		blocked := make(chan struct{})
		runner := &fakeRunner{partial: true, blocked: blocked}
		backend, err := NewCLIBackend("claude", runner)
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := make(chan error, 1)
		go func() {
			_, err := backend.Run(ctx, Task{Prompt: "hi"})
			done <- err
		}()
		select {
		case <-blocked:
		case <-time.After(2 * time.Second):
			t.Fatal("run did not block")
		}
		cancel()
		select {
		case err := <-done:
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("err = %v", err)
			}
			if errors.Is(err, ErrBadOutput) {
				t.Fatal("parse error replaced cancellation")
			}
		case <-time.After(2 * time.Second):
			t.Fatal("cancel did not return")
		}
	})
}

func TestCLIBackendEdges(t *testing.T) {
	t.Run("bad args", func(t *testing.T) {
		backend := &CLIBackend{adapter: badArgs{}, runner: &fakeRunner{}}
		_, err := backend.Run(context.Background(), Task{Prompt: "hi"})
		if err == nil || !strings.Contains(err.Error(), "bad args") {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("line too long", func(t *testing.T) {
		previous := maxStdoutLine
		maxStdoutLine = 16
		t.Cleanup(func() { maxStdoutLine = previous })
		runner := &fakeRunner{stdout: strings.Repeat("x", 64) + "\n"}
		backend, err := NewCLIBackend("claude", runner)
		if err != nil {
			t.Fatal(err)
		}
		_, err = backend.Run(context.Background(), Task{Prompt: "hi"})
		if !errors.Is(err, ErrBadOutput) {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("stderr clipped", func(t *testing.T) {
		runner := &fakeRunner{
			code:   1,
			stderr: strings.Repeat("a", stderrClip) + "TAIL",
		}
		backend, err := NewCLIBackend("claude", runner)
		if err != nil {
			t.Fatal(err)
		}
		_, err = backend.Run(context.Background(), Task{Prompt: "hi"})
		if !errors.Is(err, ErrRunFailed) || !strings.Contains(err.Error(), "aaa") || strings.Contains(err.Error(), "TAIL") {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("relative workdir", func(t *testing.T) {
		root := t.TempDir()
		if err := os.Mkdir(filepath.Join(root, "proj"), 0o755); err != nil {
			t.Fatal(err)
		}
		t.Chdir(root)
		runner := &fakeRunner{stdout: "{\"type\":\"thread.started\",\"thread_id\":\"t\"}\n{\"type\":\"turn.completed\"}\n"}
		backend, err := NewCLIBackend("codex", runner)
		if err != nil {
			t.Fatal(err)
		}
		result, err := backend.Run(context.Background(), Task{Prompt: "hello\nthere", WorkDir: "proj"})
		if err != nil {
			t.Fatal(err)
		}
		if result.SessionID != "t" {
			t.Fatalf("result = %+v", result)
		}
		spec := runner.specs[0]
		abs, err := filepath.Abs(filepath.Join(root, "proj"))
		if err != nil {
			t.Fatal(err)
		}
		if spec.Dir != abs || !strings.Contains(strings.Join(spec.Args, "\x00"), "--cd\x00"+abs) {
			t.Fatalf("spec = %#v", spec)
		}
		if spec.Args[len(spec.Args)-1] != "hello\nthere" {
			t.Fatalf("prompt arg = %q", spec.Args[len(spec.Args)-1])
		}
	})
}

func TestReadEventsEmpty(t *testing.T) {
	events, err := ReadEvents(Claude{}.NewParser(), nil)
	if err != nil || len(events) != 0 {
		t.Fatalf("events=%v err=%v", events, err)
	}
	_, err = ReadEvents(Claude{}.NewParser(), []byte("{"))
	if !errors.Is(err, ErrBadOutput) {
		t.Fatalf("err = %v", err)
	}
}
