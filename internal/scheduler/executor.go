package scheduler

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ExecutionResult contains the output and metadata of a command execution.
type ExecutionResult struct {
	Output   string
	ExitCode int
	Error    error
	Duration time.Duration
}

// Executor defines the interface for executing scheduled commands.
type Executor interface {
	Execute(ctx context.Context, command string) ExecutionResult
}

// ShellExecutor runs commands using the system shell.
type ShellExecutor struct {
	shell string
}

// NewShellExecutor creates a new shell-based executor.
func NewShellExecutor() *ShellExecutor {
	return &ShellExecutor{shell: "/bin/sh"}
}

// Execute runs the command string via the shell.
func (e *ShellExecutor) Execute(ctx context.Context, command string) ExecutionResult {
	start := time.Now()

	cmd := exec.CommandContext(ctx, e.shell, "-c", command)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	duration := time.Since(start)

	result := ExecutionResult{
		Output:   strings.TrimSpace(out.String()),
		Duration: duration,
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = -1
		}
		result.Error = err
	}

	return result
}

// LoggingExecutor wraps an executor and logs results.
type LoggingExecutor struct {
	inner  Executor
	logger func(format string, args ...interface{})
}

// NewLoggingExecutor creates a logging executor wrapper.
func NewLoggingExecutor(inner Executor, logger func(format string, args ...interface{})) *LoggingExecutor {
	if logger == nil {
		logger = func(format string, args ...interface{}) {
			fmt.Printf(format+"\n", args...)
		}
	}
	return &LoggingExecutor{inner: inner, logger: logger}
}

// Execute runs the command and logs the result.
func (e *LoggingExecutor) Execute(ctx context.Context, command string) ExecutionResult {
	result := e.inner.Execute(ctx, command)

	if result.Error != nil {
		e.logger("[scheduler] command failed after %s: %s (exit %d)", result.Duration, command, result.ExitCode)
		if result.Output != "" {
			e.logger("[scheduler] output: %s", result.Output)
		}
	} else {
		e.logger("[scheduler] command completed in %s: %s", result.Duration, command)
	}

	return result
}
