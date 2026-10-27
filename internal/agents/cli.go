package agents

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
)

// maxStdoutLine is the largest JSON line accepted from a worker.
var maxStdoutLine = 1 << 20

// Parser translates one stdout record from a CLI.
type Parser interface {
	ParseLine(line []byte) ([]Event, error)
}

// Adapter is one local agent CLI.
type Adapter interface {
	ID() string
	Binary() string
	DisplayName() string
	VersionArgs() []string
	DropEnv() []string
	Args(task Task) ([]string, error)
	NewParser() Parser
}

// CLIBackend runs a local agent CLI as a subprocess.
type CLIBackend struct {
	adapter Adapter
	runner  Runner
}

var _ Backend = (*CLIBackend)(nil)

// NewCLIBackend returns a CLI backend for "claude" or "codex".
// A nil runner uses OSRunner.
func NewCLIBackend(id string, runner Runner) (*CLIBackend, error) {
	adapter, err := AdapterByID(id)
	if err != nil {
		return nil, err
	}
	if runner == nil {
		runner = OSRunner{}
	}
	return &CLIBackend{adapter: adapter, runner: runner}, nil
}

// AdapterByID returns the adapter for id.
func AdapterByID(id string) (Adapter, error) {
	switch strings.TrimSpace(id) {
	case "claude":
		return Claude{}, nil
	case "codex":
		return Codex{}, nil
	default:
		return nil, fmt.Errorf("agents backend %q: %w", id, ErrUnknownBackend)
	}
}

// Adapters returns the CLIs this package can launch, in doctor order.
func Adapters() []Adapter {
	return []Adapter{Claude{}, Codex{}}
}

// ID returns the adapter id.
func (b *CLIBackend) ID() string {
	if b == nil || b.adapter == nil {
		return ""
	}
	return b.adapter.ID()
}

// Run starts the CLI and reads its event stream until it exits.
func (b *CLIBackend) Run(ctx context.Context, task Task) (*Result, error) {
	if b == nil || b.adapter == nil {
		return nil, fmt.Errorf("agents run: %w", ErrUnknownBackend)
	}
	runCtx, cancel, err := bindContext(ctx, task.Timeout)
	if err != nil {
		return nil, err
	}
	defer cancel()

	task, err = normalizeTask(task)
	if err != nil {
		return nil, err
	}
	args, err := b.adapter.Args(task)
	if err != nil {
		return nil, fmt.Errorf("agents run %s: %w", b.adapter.ID(), err)
	}
	runner := b.runner
	if runner == nil {
		runner = OSRunner{}
	}
	proc, err := runner.Start(runCtx, Spec{
		Binary:  b.adapter.Binary(),
		Args:    args,
		Dir:     task.WorkDir,
		DropEnv: b.adapter.DropEnv(),
	})
	if err != nil {
		return nil, fmt.Errorf("agents run %s: %w", b.adapter.ID(), err)
	}
	return collect(runCtx, cancel, b.adapter, task, proc)
}

func collect(ctx context.Context, cancel context.CancelFunc, adapter Adapter, task Task, proc Proc) (*Result, error) {
	id := adapter.ID()
	result := &Result{Backend: id}
	parser := adapter.NewParser()

	stderrReader := proc.Stderr()
	if stderrReader == nil {
		stderrReader = bytes.NewReader(nil)
	}
	stdout := proc.Stdout()
	if stdout == nil {
		stdout = bytes.NewReader(nil)
	}

	var stderr limitedWriter
	stderr.max = stderrClip
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _ = io.Copy(&stderr, stderrReader)
	}()

	scanErr := scanStdout(parser, stdout, func(ev Event) {
		if task.OnEvent != nil {
			task.OnEvent(ev.clone())
		}
		applyEvent(result, ev)
	})
	// Record why the run context ended before cancelling it ourselves.
	// A parse error cancels only the child context so the process cannot
	// block on a full stdout pipe. The caller's cancellation and deadline
	// are reported instead of the parse error.
	stopped := ctx.Err()
	if scanErr != nil && stopped == nil {
		cancel()
	}
	wg.Wait()
	code, waitErr := proc.Wait()
	result.ExitCode = code

	if stopped != nil {
		return result, fmt.Errorf("agents run %s: %w", id, stopped)
	}
	if scanErr != nil {
		return result, fmt.Errorf("agents run %s: %w", id, scanErr)
	}
	if result.Failed {
		return result, fmt.Errorf("agents run %s: %s: %w", id, clip(result.Text), ErrRunFailed)
	}
	if waitErr != nil || code != 0 {
		detail := clip(stderr.String())
		if detail == "" && waitErr != nil {
			detail = clip(waitErr.Error())
		}
		if detail == "" {
			detail = fmt.Sprintf("exit %d", code)
		}
		return result, fmt.Errorf("agents run %s: %s: %w", id, detail, ErrRunFailed)
	}
	if !hasFinal(result.Events) {
		detail := clip(stderr.String())
		if detail == "" {
			detail = "missing final result"
		}
		return result, fmt.Errorf("agents run %s: %s: %w", id, detail, ErrRunFailed)
	}
	return result, nil
}

func scanStdout(parser Parser, stdout io.Reader, emit func(Event)) error {
	scanner := bufio.NewScanner(stdout)
	bufSize := 64 * 1024
	if maxStdoutLine < bufSize {
		bufSize = maxStdoutLine
	}
	scanner.Buffer(make([]byte, 0, bufSize), maxStdoutLine)
	for scanner.Scan() {
		events, err := parser.ParseLine(scanner.Bytes())
		if err != nil {
			return fmt.Errorf("%w: %s", ErrBadOutput, clip(err.Error()))
		}
		for _, ev := range events {
			emit(ev)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("%w: %s", ErrBadOutput, clip(err.Error()))
	}
	return nil
}

func applyEvent(result *Result, ev Event) {
	result.Events = append(result.Events, ev.clone())
	if ev.SessionID != "" {
		result.SessionID = ev.SessionID
	}
	if ev.Model != "" {
		result.Model = ev.Model
	}
	if ev.Final {
		result.Text = ev.Text
		result.Failed = ev.Failed
	}
}

func hasFinal(events []Event) bool {
	for _, ev := range events {
		if ev.Final {
			return true
		}
	}
	return false
}

// ReadEvents parses a full stdout buffer with parser. Blank lines are ignored.
func ReadEvents(parser Parser, stdout []byte) ([]Event, error) {
	var events []Event
	err := scanStdout(parser, bytes.NewReader(stdout), func(ev Event) {
		events = append(events, ev)
	})
	if err != nil {
		return events, err
	}
	return events, nil
}
