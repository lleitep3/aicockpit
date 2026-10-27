package agents

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

// stderrClip bounds stderr copied into errors.
const stderrClip = 300

// Runner starts a worker process. Tests inject an implementation.
// Start must return when ctx is cancelled.
type Runner interface {
	Start(ctx context.Context, spec Spec) (Proc, error)
}

// InstallChecker reports whether a binary is on PATH.
// OSRunner implements it. Runners that do not are probed by starting
// the version command.
type InstallChecker interface {
	Installed(binary string) bool
}

// Spec is one process start. This package never adds environment variables.
// DropEnv names are removed from the inherited environment so a nested CLI
// can start; values are not logged.
type Spec struct {
	// Binary is the executable name or path.
	Binary string
	// Args is the argument vector without the binary.
	Args []string
	// Dir is the working directory. Empty inherits the current directory.
	Dir string
	// DropEnv lists inherited variable names to remove.
	DropEnv []string
}

// Proc is a running process. Callers read both streams to completion
// before Wait.
type Proc interface {
	Stdout() io.Reader
	Stderr() io.Reader
	Wait() (int, error)
}

// OSRunner starts executables with os/exec.
// A nil Env list is not used: the child inherits the parent environment,
// minus Spec.DropEnv. No variables are added.
type OSRunner struct{}

var (
	_ Runner         = OSRunner{}
	_ InstallChecker = OSRunner{}
)

// Installed reports whether binary is found on PATH.
func (OSRunner) Installed(binary string) bool {
	if binary == "" {
		return false
	}
	_, err := exec.LookPath(binary)
	return err == nil
}

// Start launches spec. Cancellation of ctx kills the process.
func (OSRunner) Start(ctx context.Context, spec Spec) (Proc, error) {
	if ctx == nil {
		return nil, fmt.Errorf("agents start: %w", ErrNilContext)
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("agents start: %w", err)
	}
	if strings.TrimSpace(spec.Binary) == "" {
		return nil, fmt.Errorf("agents start: %w", ErrUnknownBackend)
	}
	path, err := exec.LookPath(spec.Binary)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrNotInstalled, spec.Binary)
	}
	cmd := exec.CommandContext(ctx, path, spec.Args...)
	cmd.Dir = spec.Dir
	cmd.WaitDelay = 200 * time.Millisecond
	if len(spec.DropEnv) > 0 {
		cmd.Env = withoutEnv(os.Environ(), spec.DropEnv)
	}
	stdin, err := os.Open(os.DevNull)
	if err != nil {
		return nil, fmt.Errorf("agents start: stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, fmt.Errorf("agents start: stdout: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, fmt.Errorf("agents start: stderr: %w", err)
	}
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return nil, fmt.Errorf("agents start: %w", err)
	}
	return &osProc{cmd: cmd, stdout: stdout, stderr: stderr, stdin: stdin}, nil
}

type osProc struct {
	cmd    *exec.Cmd
	stdout io.Reader
	stderr io.Reader
	stdin  io.Closer
}

func (p *osProc) Stdout() io.Reader { return p.stdout }

func (p *osProc) Stderr() io.Reader { return p.stderr }

func (p *osProc) Wait() (int, error) {
	err := p.cmd.Wait()
	if p.stdin != nil {
		_ = p.stdin.Close()
	}
	if err == nil {
		return 0, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode(), err
	}
	return -1, err
}

func withoutEnv(environ, drop []string) []string {
	skip := make(map[string]struct{}, len(drop))
	for _, key := range drop {
		skip[key] = struct{}{}
	}
	out := make([]string, 0, len(environ))
	for _, entry := range environ {
		key, _, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		if _, found := skip[key]; found {
			continue
		}
		out = append(out, entry)
	}
	return out
}

type limitedWriter struct {
	buf bytes.Buffer
	max int
}

func (w *limitedWriter) Write(p []byte) (int, error) {
	limit := w.max
	if limit <= 0 {
		limit = stderrClip
	}
	if remain := limit - w.buf.Len(); remain > 0 {
		if len(p) > remain {
			_, _ = w.buf.Write(p[:remain])
		} else {
			_, _ = w.buf.Write(p)
		}
	}
	return len(p), nil
}

func (w *limitedWriter) String() string { return w.buf.String() }

func clip(text string) string {
	text = strings.TrimSpace(text)
	if len(text) > stderrClip {
		return text[:stderrClip]
	}
	return text
}
