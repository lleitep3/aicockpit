package ask

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/lleitep3/aicockpit/internal/agents"
	"github.com/lleitep3/aicockpit/internal/memory"
)

const (
	// KindCLI is a local claude or codex worker.
	KindCLI = "cli"
	// KindAPI is an HTTP provider worker.
	KindAPI = "api"

	recallLimit  = 5
	snippetLimit = 400
	taskLimit    = 500
	resultLimit  = 1200
)

var (
	// ErrNoBackend is returned when no worker can run the task.
	ErrNoBackend = errors.New("no agent backend available")
	// ErrBackendUnavailable is returned when the requested worker cannot run.
	ErrBackendUnavailable = errors.New("agent backend is not available")
	// ErrEmptyTask is returned when the task text is blank.
	ErrEmptyTask = errors.New("empty task")
	// ErrMemoryUnavailable is returned when a turn needs memory and none was provided.
	ErrMemoryUnavailable = errors.New("memory is not configured")
	errNilContext        = errors.New("nil context")
	errInvalidTimeout    = errors.New("invalid timeout")
)

// Request is one cockpit ask invocation.
type Request struct {
	// Task is the user task. It is required.
	Task string
	// Backend is an explicit worker: claude, codex, api, or api:<provider>.
	// Empty uses ConfigBackend and then the first available worker.
	Backend string
	// Model is an optional model override.
	Model string
	// WorkDir is the working directory. Empty uses the current directory.
	WorkDir string
	// Timeout limits the worker. Zero adds no deadline beyond ctx.
	Timeout time.Duration
	// NoMemory skips recall and retain.
	NoMemory bool
	// JSON prints one outcome object instead of streaming text.
	JSON bool
}

// Deps are the collaborators for one turn.
type Deps struct {
	// Memory is the local memory. It is required unless Request.NoMemory is set.
	Memory memory.Memory
	// Extra is an optional second memory, such as Hindsight.
	// Recall and retain errors from Extra are warnings.
	Extra memory.Memory
	// Catalog selects and opens a worker.
	Catalog Catalog
	// Out receives the answer.
	Out io.Writer
	// ErrOut receives warnings. Nil discards them.
	ErrOut io.Writer
	// ConfigBackend is agents.backend from configuration. Empty means auto.
	ConfigBackend string
}

// Outcome is the shared human and JSON result of a turn.
type Outcome struct {
	// Backend is the selected worker, such as claude or api:anthropic.
	Backend string `json:"backend"`
	// Model is the model reported by the worker or requested for the turn.
	Model string `json:"model,omitempty"`
	// Text is the final answer.
	Text string `json:"text"`
	// SessionID is the worker session id when one was reported.
	SessionID string `json:"session_id,omitempty"`
	// Recalled is the number of memory hits added to the prompt.
	Recalled int `json:"recalled"`
	// Retained is true when the local memory accepted the record.
	Retained bool `json:"retained"`
}

// Ref names one worker.
type Ref struct {
	// Kind is KindCLI or KindAPI.
	Kind string
	// ID is claude, codex, or the API provider id. Empty for a bare "api" ref
	// until Check fills it from configuration.
	ID string
	// Raw is the flag value, such as claude or api:openai.
	Raw string
}

// Candidate is one worker that might run a turn.
type Candidate struct {
	// Ref is the worker.
	Ref Ref
	// Available is true when the worker can be opened.
	Available bool
	// Reason explains why the worker cannot run. It contains no secrets.
	Reason string
}

// Catalog reports workers and opens one.
type Catalog interface {
	// Check reports whether ref can run.
	Check(ctx context.Context, ref Ref) (Candidate, error)
	// Auto returns workers in preference order.
	Auto(ctx context.Context) ([]Candidate, error)
	// Open returns a worker for ref.
	Open(ctx context.Context, ref Ref) (agents.Backend, error)
}

// Run executes one turn.
// Text is streamed to Deps.Out unless Request.JSON is set. JSON output is one
// object with the same fields as Outcome.
func Run(ctx context.Context, req Request, deps Deps) (*Outcome, error) {
	if ctx == nil {
		return nil, fmt.Errorf("ask: %w", errNilContext)
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("ask: %w", err)
	}
	task := strings.TrimSpace(req.Task)
	if task == "" {
		return nil, fmt.Errorf("ask: %w", ErrEmptyTask)
	}
	if req.Timeout < 0 {
		return nil, fmt.Errorf("ask: %w", errInvalidTimeout)
	}
	if deps.Catalog == nil {
		return nil, fmt.Errorf("ask: %w: no catalog", ErrNoBackend)
	}
	if !req.NoMemory && deps.Memory == nil {
		return nil, fmt.Errorf("ask: %w", ErrMemoryUnavailable)
	}

	ref, backend, err := resolve(ctx, req.Backend, deps.ConfigBackend, deps.Catalog)
	if err != nil {
		return nil, err
	}

	hits, err := recall(ctx, req, deps, task)
	if err != nil {
		return nil, err
	}
	prompt := buildPrompt(task, hits)
	out := writer(deps.Out)
	stream := &printer{out: out, json: req.JSON}
	result, runErr := backend.Run(ctx, agents.Task{
		Prompt:  prompt,
		WorkDir: req.WorkDir,
		Timeout: req.Timeout,
		Model:   req.Model,
		OnEvent: stream.onEvent,
	})
	outcome := newOutcome(ref, result, req.Model, len(hits))
	if stream.err != nil && runErr == nil {
		runErr = fmt.Errorf("write output: %w", stream.err)
	}
	if runErr != nil {
		if req.JSON {
			_ = writeJSON(out, outcome)
		}
		return outcome, fmt.Errorf("ask: %w", runErr)
	}
	stream.finish(outcome.Text)
	if stream.err != nil {
		return outcome, fmt.Errorf("ask: write output: %w", stream.err)
	}
	if !req.NoMemory {
		if err := retain(ctx, deps, task, outcome); err != nil {
			if req.JSON {
				_ = writeJSON(out, outcome)
			}
			return outcome, err
		}
		outcome.Retained = true
	}
	if req.JSON {
		if err := writeJSON(out, outcome); err != nil {
			return outcome, fmt.Errorf("ask: write output: %w", err)
		}
	}
	return outcome, nil
}

func recall(ctx context.Context, req Request, deps Deps, task string) ([]memory.Hit, error) {
	if req.NoMemory {
		return nil, nil
	}
	query := memory.Query{Text: task, Limit: recallLimit}
	found, err := deps.Memory.Recall(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("ask: recall: %w", err)
	}
	var hits []memory.Hit
	if found != nil {
		hits = found.Hits
	}
	if deps.Extra == nil {
		return mergeHits(hits, nil, recallLimit), nil
	}
	extra, err := deps.Extra.Recall(ctx, query)
	if err != nil {
		warn(deps.ErrOut, "hindsight recall skipped: "+err.Error())
		return mergeHits(hits, nil, recallLimit), nil
	}
	var extraHits []memory.Hit
	if extra != nil {
		extraHits = extra.Hits
	}
	return mergeHits(hits, extraHits, recallLimit), nil
}

func retain(ctx context.Context, deps Deps, task string, outcome *Outcome) error {
	item := record(task, outcome)
	if _, err := deps.Memory.Retain(ctx, item); err != nil {
		return fmt.Errorf("ask: retain: %w", err)
	}
	if deps.Extra == nil {
		return nil
	}
	if _, err := deps.Extra.Retain(ctx, item); err != nil {
		warn(deps.ErrOut, "hindsight retain skipped: "+err.Error())
	}
	return nil
}

func newOutcome(ref Ref, result *agents.Result, model string, recalled int) *Outcome {
	out := &Outcome{Backend: displayRef(ref), Model: strings.TrimSpace(model), Recalled: recalled}
	if result == nil {
		return out
	}
	out.Text = result.Text
	out.SessionID = result.SessionID
	if result.Model != "" {
		out.Model = result.Model
	}
	if result.Backend != "" && ref.Kind == KindCLI {
		out.Backend = result.Backend
	}
	return out
}

func record(task string, outcome *Outcome) memory.Item {
	result := strings.TrimSpace(outcome.Text)
	if result == "" {
		result = "(no output)"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Task: %s\nBackend: %s\n", truncate(oneLine(task), taskLimit), outcome.Backend)
	if outcome.Model != "" {
		fmt.Fprintf(&b, "Model: %s\n", oneLine(outcome.Model))
	}
	fmt.Fprintf(&b, "Result: %s", truncate(result, resultLimit))
	return memory.Item{
		Title:    truncate("ask: "+oneLine(task), 80),
		Content:  b.String(),
		Context:  "cockpit ask",
		Tags:     []string{"ask"},
		Metadata: map[string]string{"backend": outcome.Backend},
	}
}

func buildPrompt(task string, hits []memory.Hit) string {
	if len(hits) == 0 {
		return task
	}
	var b strings.Builder
	b.WriteString(task)
	b.WriteString("\n\nRelevant memory:\n")
	for _, hit := range hits {
		title := strings.TrimSpace(hit.Item.Title)
		if title == "" {
			title = strings.TrimSpace(hit.Item.ID)
		}
		if title == "" {
			title = "memory"
		}
		fmt.Fprintf(&b, "- %s: %s\n", oneLine(title), truncate(oneLine(hit.Item.Content), snippetLimit))
	}
	return strings.TrimRight(b.String(), "\n")
}

func mergeHits(primary, extra []memory.Hit, limit int) []memory.Hit {
	if limit <= 0 {
		return nil
	}
	out := make([]memory.Hit, 0, limit)
	seen := make(map[string]struct{}, limit)
	add := func(hits []memory.Hit) {
		for _, hit := range hits {
			if len(out) >= limit {
				return
			}
			key := hit.Item.ID + "\n" + hit.Item.Title + "\n" + hit.Item.Content
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, hit)
		}
	}
	add(primary)
	add(extra)
	return out
}

func resolve(ctx context.Context, flag, configured string, catalog Catalog) (Ref, agents.Backend, error) {
	flag = strings.TrimSpace(flag)
	configured = strings.TrimSpace(configured)
	if flag != "" {
		ref, err := ParseRef(flag)
		if err != nil {
			return Ref{}, nil, fmt.Errorf("ask: %w", err)
		}
		return openRequired(ctx, catalog, ref)
	}
	if configured != "" {
		ref, err := ParseRef(configured)
		if err != nil {
			return Ref{}, nil, fmt.Errorf("ask: agents.backend: %w", err)
		}
		return openRequired(ctx, catalog, ref)
	}
	candidates, err := catalog.Auto(ctx)
	if err != nil {
		return Ref{}, nil, fmt.Errorf("ask: %w", err)
	}
	var reasons []string
	for _, candidate := range candidates {
		if candidate.Available {
			backend, err := catalog.Open(ctx, candidate.Ref)
			if err != nil {
				return Ref{}, nil, fmt.Errorf("ask: backend %s: %w", displayRef(candidate.Ref), err)
			}
			return candidate.Ref, backend, nil
		}
		reason := strings.TrimSpace(candidate.Reason)
		if reason == "" {
			reason = "unavailable"
		}
		reasons = append(reasons, displayRef(candidate.Ref)+": "+reason)
	}
	if len(reasons) == 0 {
		return Ref{}, nil, fmt.Errorf("ask: %w: install claude or codex, or configure agents.api", ErrNoBackend)
	}
	return Ref{}, nil, fmt.Errorf("ask: %w: %s", ErrNoBackend, strings.Join(reasons, "; "))
}

func openRequired(ctx context.Context, catalog Catalog, ref Ref) (Ref, agents.Backend, error) {
	candidate, err := catalog.Check(ctx, ref)
	if err != nil {
		return Ref{}, nil, fmt.Errorf("ask: backend %s: %w", displayRef(ref), err)
	}
	if candidate.Ref.Raw == "" {
		candidate.Ref = ref
	}
	name := displayRef(candidate.Ref)
	if !candidate.Available {
		reason := strings.TrimSpace(candidate.Reason)
		if reason == "" {
			reason = "unavailable"
		}
		return candidate.Ref, nil, fmt.Errorf("ask: backend %s: %s: %w", name, reason, ErrBackendUnavailable)
	}
	backend, err := catalog.Open(ctx, candidate.Ref)
	if err != nil {
		return candidate.Ref, nil, fmt.Errorf("ask: backend %s: %w", name, err)
	}
	return candidate.Ref, backend, nil
}

// ParseRef parses a backend flag or config value.
func ParseRef(raw string) (Ref, error) {
	raw = strings.ToLower(strings.TrimSpace(raw))
	switch raw {
	case "claude", "codex":
		return Ref{Kind: KindCLI, ID: raw, Raw: raw}, nil
	case "api":
		return Ref{Kind: KindAPI, Raw: "api"}, nil
	}
	if rest, ok := strings.CutPrefix(raw, "api:"); ok {
		provider := strings.TrimSpace(rest)
		if provider == "" {
			return Ref{}, fmt.Errorf("backend %q: %w", raw, agents.ErrUnknownProvider)
		}
		resolved, err := (agents.APISettings{Provider: provider}).Resolve()
		if err != nil {
			return Ref{}, fmt.Errorf("backend %q: %w", raw, err)
		}
		return Ref{Kind: KindAPI, ID: resolved.Provider, Raw: "api:" + resolved.Provider}, nil
	}
	return Ref{}, fmt.Errorf("backend %q: %w", raw, agents.ErrUnknownBackend)
}

func displayRef(ref Ref) string {
	if ref.Raw != "" {
		return ref.Raw
	}
	if ref.ID != "" {
		return ref.ID
	}
	return "backend"
}

func warn(w io.Writer, message string) {
	fmt.Fprintf(writer(w), "warning: %s\n", message)
}

func writer(w io.Writer) io.Writer {
	if w == nil {
		return io.Discard
	}
	return w
}

func writeJSON(w io.Writer, outcome *Outcome) error {
	if outcome == nil {
		outcome = &Outcome{}
	}
	encoded, err := json.MarshalIndent(outcome, "", "  ")
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, string(encoded)); err != nil {
		return err
	}
	return nil
}

func oneLine(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

func truncate(text string, limit int) string {
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	if limit <= 3 {
		return string(runes[:limit])
	}
	return string(runes[:limit-3]) + "..."
}

type printer struct {
	out     io.Writer
	json    bool
	partial bool
	wrote   bool
	buf     strings.Builder
	err     error
}

func (p *printer) onEvent(ev agents.Event) {
	if p == nil || p.json {
		return
	}
	switch ev.Kind {
	case agents.KindText:
		if ev.Partial {
			p.partial = true
			p.emit(ev.Text, false)
			return
		}
		if p.partial {
			return
		}
		p.emit(ev.Text, true)
	case agents.KindResult:
		if ev.Failed || ev.Text == "" {
			return
		}
		if p.wrote {
			p.finishLine()
			return
		}
		p.emit(ev.Text, true)
	case agents.KindError:
		p.emit(ev.Text, true)
	}
}

func (p *printer) finish(text string) {
	if p == nil || p.json {
		return
	}
	if !p.wrote && strings.TrimSpace(text) != "" {
		p.emit(text, true)
		return
	}
	p.finishLine()
}

func (p *printer) emit(text string, block bool) {
	if text == "" {
		return
	}
	p.write(text)
	if block {
		p.finishLine()
	}
}

func (p *printer) finishLine() {
	if !p.wrote || strings.HasSuffix(p.buf.String(), "\n") {
		return
	}
	p.write("\n")
}

func (p *printer) write(text string) {
	if _, err := io.WriteString(writer(p.out), text); err != nil && p.err == nil {
		p.err = err
	}
	p.buf.WriteString(text)
	p.wrote = true
}
