package resilience

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeFakeGit(t *testing.T, script string) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "fake-git")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"+script), 0o755); err != nil {
		t.Fatalf("write fake git: %v", err)
	}
	return bin
}

func TestGitRunner_Success(t *testing.T) {
	bin := writeFakeGit(t, "echo hello")
	t.Setenv("COCKPIT_TEST_GIT", bin)

	gr := DefaultGitRunner()
	out, err := gr.Run(context.Background(), "", "status")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != "hello\n" {
		t.Fatalf("unexpected output: %q", string(out))
	}
}

func TestGitRunner_RetriesThenSucceeds(t *testing.T) {
	counterFile := filepath.Join(t.TempDir(), "counter")
	script := `count=0
if [ -f "$COUNTER_FILE" ]; then count=$(cat "$COUNTER_FILE"); fi
count=$((count + 1))
echo "$count" > "$COUNTER_FILE"
if [ "$count" -lt 3 ]; then echo "transient failure" >&2; exit 1; fi
echo "ok"
`
	bin := writeFakeGit(t, script)
	t.Setenv("COCKPIT_TEST_GIT", bin)
	t.Setenv("COUNTER_FILE", counterFile)

	cfg := Config{MaxAttempts: 5, InitialDelay: 1 * time.Millisecond}
	gr := NewGitRunner(cfg)
	out, err := gr.Run(context.Background(), "", "fetch")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != "ok\n" {
		t.Fatalf("unexpected output: %q", string(out))
	}
}

func TestGitRunner_MaxAttemptsExceeded(t *testing.T) {
	bin := writeFakeGit(t, "echo fail >&2; exit 1")
	t.Setenv("COCKPIT_TEST_GIT", bin)

	cfg := Config{MaxAttempts: 2, InitialDelay: 1 * time.Millisecond}
	gr := NewGitRunner(cfg)
	_, err := gr.Run(context.Background(), "", "clone")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGitRunner_ContextCancelled(t *testing.T) {
	bin := writeFakeGit(t, "sleep 10")
	t.Setenv("COCKPIT_TEST_GIT", bin)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	gr := DefaultGitRunner()
	_, err := gr.Run(ctx, "", "fetch")
	if err == nil {
		t.Fatal("expected error")
	}
}
