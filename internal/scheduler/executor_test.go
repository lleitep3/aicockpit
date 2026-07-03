package scheduler

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestShellExecutorSuccess(t *testing.T) {
	exec := NewShellExecutor()
	ctx := context.Background()

	result := exec.Execute(ctx, "echo hello")
	if result.Error != nil {
		t.Fatalf("expected no error, got %v", result.Error)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if result.Output != "hello" {
		t.Errorf("expected output 'hello', got %q", result.Output)
	}
}

func TestShellExecutorFailure(t *testing.T) {
	exec := NewShellExecutor()
	ctx := context.Background()

	result := exec.Execute(ctx, "exit 42")
	if result.Error == nil {
		t.Fatal("expected error for failing command")
	}
	if result.ExitCode != 42 {
		t.Errorf("expected exit code 42, got %d", result.ExitCode)
	}
}

func TestShellExecutorTimeout(t *testing.T) {
	exec := NewShellExecutor()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	result := exec.Execute(ctx, "sleep 5")
	if result.Error == nil {
		t.Fatal("expected timeout error")
	}
	if result.ExitCode != -1 && result.ExitCode != 0 {
		t.Errorf("unexpected exit code %d", result.ExitCode)
	}
}

func TestShellExecutorCapturesStderr(t *testing.T) {
	exec := NewShellExecutor()
	ctx := context.Background()

	result := exec.Execute(ctx, "echo error >&2; exit 1")
	if !strings.Contains(result.Output, "error") {
		t.Errorf("expected output to contain stderr, got %q", result.Output)
	}
}

func TestLoggingExecutor(t *testing.T) {
	inner := NewShellExecutor()
	var logs []string
	logger := func(format string, args ...interface{}) {
		logs = append(logs, format)
	}

	logExec := NewLoggingExecutor(inner, logger)
	ctx := context.Background()

	result := logExec.Execute(ctx, "echo hello")
	if result.Error != nil {
		t.Fatalf("expected success, got %v", result.Error)
	}
	if len(logs) == 0 {
		t.Error("expected at least one log entry")
	}
}

func TestLoggingExecutorFailure(t *testing.T) {
	inner := NewShellExecutor()
	var logs []string
	logger := func(format string, args ...interface{}) {
		logs = append(logs, format)
	}

	logExec := NewLoggingExecutor(inner, logger)
	ctx := context.Background()

	result := logExec.Execute(ctx, "exit 1")
	if result.Error == nil {
		t.Fatal("expected failure")
	}
	if len(logs) == 0 {
		t.Error("expected log entries for failure")
	}
}

func TestLoggingExecutorDefaultLogger(t *testing.T) {
	inner := NewShellExecutor()
	logExec := NewLoggingExecutor(inner, nil)
	ctx := context.Background()

	result := logExec.Execute(ctx, "echo hello")
	if result.Error != nil {
		t.Fatalf("expected success, got %v", result.Error)
	}
}

func TestShellExecutorContextCancel(t *testing.T) {
	exec := NewShellExecutor()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := exec.Execute(ctx, "echo hello")
	if result.Error == nil {
		t.Fatal("expected error for cancelled context")
	}
}
