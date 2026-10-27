package agents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	// KindStatus is a lifecycle event such as session start.
	KindStatus Kind = "status"
	// KindText is assistant text, including a token delta when Partial is set.
	KindText Kind = "text"
	// KindReasoning is model reasoning text.
	KindReasoning Kind = "reasoning"
	// KindTool is a tool or command the worker invoked.
	KindTool Kind = "tool"
	// KindResult is the terminal success event.
	KindResult Kind = "result"
	// KindError is a worker-reported failure.
	KindError Kind = "error"
)

var (
	// ErrEmptyPrompt is returned when the task prompt is blank.
	ErrEmptyPrompt = errors.New("empty prompt")
	// ErrInvalidPrompt is returned when the prompt cannot be passed as one argument.
	ErrInvalidPrompt = errors.New("invalid prompt")
	// ErrInvalidModel is returned when the model override is not a single token.
	ErrInvalidModel = errors.New("invalid model")
	// ErrInvalidWorkDir is returned when the working directory cannot be used.
	ErrInvalidWorkDir = errors.New("invalid working directory")
	// ErrInvalidTimeout is returned when the timeout is negative.
	ErrInvalidTimeout = errors.New("invalid timeout")
	// ErrNilContext is returned when Run is called with a nil context.
	ErrNilContext = errors.New("nil context")
	// ErrNotInstalled is returned when the worker binary is not on PATH.
	ErrNotInstalled = errors.New("agent cli not installed")
	// ErrUnknownBackend is returned when the backend id is not registered.
	ErrUnknownBackend = errors.New("unknown agent backend")
	// ErrRunFailed is returned when the worker exits without a successful result.
	ErrRunFailed = errors.New("agent run failed")
	// ErrBadOutput is returned when stdout is not the CLI's JSON event stream.
	ErrBadOutput = errors.New("invalid agent output")
)

// modelPattern matches a model id or alias such as "sonnet" or "gpt-5.4".
var modelPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:@+-]{0,127}$`)

// Kind is the normalized event kind shared by every adapter.
type Kind string

// Backend runs one task on an agent worker.
//
// Run streams events through Task.OnEvent as they arrive and returns the
// final result. A cancelled ctx stops the worker. Task.Timeout adds a
// deadline on top of ctx. Task.WorkDir is the working directory. Task.Model
// is optional.
type Backend interface {
	// ID is a stable backend name such as "claude" or "codex".
	ID() string
	// Run executes one task.
	Run(ctx context.Context, task Task) (*Result, error)
}

// Task is one worker invocation.
type Task struct {
	// Prompt is the task text. It is required and must not start with "-".
	Prompt string
	// WorkDir is the working directory. Empty uses the current directory.
	WorkDir string
	// Timeout limits the run. Zero means no extra deadline beyond ctx.
	Timeout time.Duration
	// Model is an optional model override. Empty keeps the CLI default.
	Model string
	// OnEvent receives each event in order, including the final one.
	// A nil func skips streaming. The callback must not retain and mutate
	// the event.
	OnEvent func(Event)
}

// Event is one streamed update from a worker.
type Event struct {
	// Kind is the normalized kind.
	Kind Kind
	// Type is the CLI-native event type.
	Type string
	// Subtype is a CLI-native subtype or item type.
	Subtype string
	// Text is the human-readable payload.
	Text string
	// SessionID is the CLI session or thread id when the event carries one.
	SessionID string
	// Model is the model name when the event carries one.
	Model string
	// Tool is the tool name for KindTool.
	Tool string
	// Partial is true for a token delta rather than a complete message.
	Partial bool
	// Final marks the terminal event of a successful or failed run.
	Final bool
	// Failed marks a worker-reported failure.
	Failed bool
	// Raw is a copy of the JSON line that produced the event.
	Raw json.RawMessage
}

// Result is the final outcome of a task.
// It is returned together with an error when the process started and then
// failed, so the caller can inspect the events already streamed.
type Result struct {
	// Backend is the worker id.
	Backend string
	// Text is the final answer. It comes from the terminal event.
	Text string
	// SessionID is the CLI session or thread id when one was reported.
	SessionID string
	// Model is the model reported by the CLI, which may be empty.
	Model string
	// Events are the events in arrival order.
	Events []Event
	// ToolCalls are model-requested tools in arrival order.
	// CLI runs leave this empty. API runs record the calls and do not execute them.
	ToolCalls []ToolCall
	// ExitCode is the process exit code, or -1 when the process did not exit normally.
	// API runs use 0 on success, the HTTP status on an error response, and -1
	// when the request did not complete.
	ExitCode int
	// Failed is true when the worker reported a failed terminal event.
	Failed bool
}

func (e Event) clone() Event {
	if len(e.Raw) > 0 {
		raw := make([]byte, len(e.Raw))
		copy(raw, e.Raw)
		e.Raw = raw
	}
	return e
}

func bindContext(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc, error) {
	if ctx == nil {
		return nil, nil, fmt.Errorf("agents run: %w", ErrNilContext)
	}
	if err := ctx.Err(); err != nil {
		return nil, nil, fmt.Errorf("agents run: %w", err)
	}
	if timeout < 0 {
		return nil, nil, fmt.Errorf("agents run: %w", ErrInvalidTimeout)
	}
	if timeout == 0 {
		// A child context lets a bad stdout stream cancel the process without
		// cancelling the caller.
		child, cancel := context.WithCancel(ctx)
		return child, cancel, nil
	}
	child, cancel := context.WithTimeout(ctx, timeout)
	return child, cancel, nil
}

func normalizeTask(task Task) (Task, error) {
	task.Prompt = strings.TrimSpace(task.Prompt)
	if task.Prompt == "" {
		return Task{}, fmt.Errorf("agents run: %w", ErrEmptyPrompt)
	}
	if strings.ContainsRune(task.Prompt, '\x00') || strings.HasPrefix(task.Prompt, "-") {
		return Task{}, fmt.Errorf("agents run: %w", ErrInvalidPrompt)
	}

	task.Model = strings.TrimSpace(task.Model)
	if task.Model != "" && !modelPattern.MatchString(task.Model) {
		return Task{}, fmt.Errorf("agents run: %w", ErrInvalidModel)
	}

	task.WorkDir = strings.TrimSpace(task.WorkDir)
	if task.WorkDir == "" {
		return task, nil
	}
	if strings.ContainsRune(task.WorkDir, '\x00') {
		return Task{}, fmt.Errorf("agents run: %w", ErrInvalidWorkDir)
	}
	abs, err := filepath.Abs(task.WorkDir)
	if err != nil {
		return Task{}, fmt.Errorf("agents run: working directory: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		return Task{}, fmt.Errorf("agents run: %w", ErrInvalidWorkDir)
	}
	task.WorkDir = abs
	return task, nil
}
