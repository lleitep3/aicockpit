package agents

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestOSRunnerGuards(t *testing.T) {
	var runner OSRunner
	if _, err := runner.Start(nil, Spec{Binary: "claude"}); !errors.Is(err, ErrNilContext) { //nolint:staticcheck // nil context is the condition under test
		t.Fatalf("nil ctx: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := runner.Start(ctx, Spec{Binary: "claude"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled: %v", err)
	}
	if _, err := runner.Start(context.Background(), Spec{}); !errors.Is(err, ErrUnknownBackend) {
		t.Fatalf("empty binary: %v", err)
	}
	t.Setenv("PATH", t.TempDir())
	if runner.Installed("claude") {
		t.Fatal("empty PATH reported claude")
	}
	if runner.Installed("") {
		t.Fatal("empty name")
	}
	_, err := runner.Start(context.Background(), Spec{Binary: "claude", Args: []string{"--version"}})
	if !errors.Is(err, ErrNotInstalled) {
		t.Fatalf("missing: %v", err)
	}
}

func TestOSRunnerClaudeStub(t *testing.T) {
	bin := t.TempDir()
	script := "#!/bin/sh\n" +
		"if [ \"$1\" = \"--version\" ]; then echo '9.9.9-test (stub)'; exit 0; fi\n" +
		"if [ -n \"$CLAUDECODE\" ] || [ -n \"$CLAUDE_CODE_ENTRYPOINT\" ]; then echo LEAKED; exit 3; fi\n" +
		"if [ \"$COCKPIT_AGENT_KEEP\" != \"yes\" ]; then echo DROPPED_KEEP; exit 4; fi\n" +
		"echo '{\"type\":\"system\",\"subtype\":\"init\",\"session_id\":\"s\",\"model\":\"sonnet\"}'\n" +
		"echo '{\"type\":\"result\",\"subtype\":\"success\",\"is_error\":false,\"result\":\"from-stub\",\"session_id\":\"s\"}'\n"
	writeExec(t, filepath.Join(bin, "claude"), script)
	t.Setenv("PATH", bin)
	t.Setenv("CLAUDECODE", "nested")
	t.Setenv("CLAUDE_CODE_ENTRYPOINT", "ipc")
	t.Setenv("COCKPIT_AGENT_KEEP", "yes")

	backend, err := NewCLIBackend("claude", nil)
	if err != nil {
		t.Fatal(err)
	}
	result, err := backend.Run(context.Background(), Task{Prompt: "hi", Model: "sonnet"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Text != "from-stub" || result.SessionID != "s" {
		t.Fatalf("result = %+v", result)
	}

	statuses := Detect(context.Background(), nil)
	var claude Status
	for _, status := range statuses {
		if status.ID == "claude" {
			claude = status
		}
	}
	if !claude.Installed || claude.Version != "9.9.9-test (stub)" {
		t.Fatalf("status = %+v", claude)
	}
	view := claude.Doctor()
	if view.Fatal || view.Status != "ok" || !strings.Contains(view.Message, "9.9.9-test") {
		t.Fatalf("doctor = %+v", view)
	}
}

func TestOSRunnerCodexStub(t *testing.T) {
	bin := t.TempDir()
	script := "#!/bin/sh\n" +
		"if [ \"$1\" = \"--version\" ]; then echo 'codex-cli 9.9.9-test' >&2; exit 0; fi\n" +
		"if [ \"$1\" != \"exec\" ] || [ \"$2\" != \"--json\" ]; then echo \"bad $*\" >&2; exit 2; fi\n" +
		"echo '{\"type\":\"thread.started\",\"thread_id\":\"t\"}'\n" +
		"echo '{\"type\":\"item.completed\",\"item\":{\"id\":\"i\",\"type\":\"agent_message\",\"text\":\"from-codex\"}}'\n" +
		"echo '{\"type\":\"turn.completed\",\"usage\":{\"input_tokens\":1,\"output_tokens\":1}}'\n"
	writeExec(t, filepath.Join(bin, "codex"), script)
	t.Setenv("PATH", bin)

	backend := &CLIBackend{adapter: Codex{}}
	dir := t.TempDir()
	result, err := backend.Run(context.Background(), Task{Prompt: "hi", WorkDir: dir, Model: "gpt-5.4"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Text != "from-codex" || result.SessionID != "t" || result.ExitCode != 0 {
		t.Fatalf("result = %+v", result)
	}

	found := Detect(context.Background(), OSRunner{})
	var codex Status
	for _, status := range found {
		if status.ID == "codex" {
			codex = status
		}
	}
	if !codex.Installed || codex.Version != "codex-cli 9.9.9-test" {
		t.Fatalf("status = %+v", codex)
	}
}

func TestOSRunnerCancelAndFailure(t *testing.T) {
	bin := t.TempDir()
	writeExec(t, filepath.Join(bin, "claude"), "#!/bin/sh\nexec sleep 30\n")
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	backend, err := NewCLIBackend("claude", OSRunner{})
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	_, err = backend.Run(context.Background(), Task{Prompt: "hi", Timeout: 200 * time.Millisecond})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v", err)
	}
	if time.Since(start) > 5*time.Second {
		t.Fatal("sleep was not cancelled")
	}

	codex, err := NewCLIBackend("codex", OSRunner{})
	if err != nil {
		t.Fatal(err)
	}
	writeExec(t, filepath.Join(bin, "codex"), "#!/bin/sh\necho '{'\nexec sleep 30\n")
	start = time.Now()
	_, err = codex.Run(context.Background(), Task{Prompt: "hi"})
	if !errors.Is(err, ErrBadOutput) {
		t.Fatalf("err = %v", err)
	}
	if time.Since(start) > 5*time.Second {
		t.Fatal("bad stdout did not stop the process")
	}

	writeExec(t, filepath.Join(bin, "codex"), "#!/bin/sh\necho 'stderr-fail' >&2\nexit 7\n")
	result, err := codex.Run(context.Background(), Task{Prompt: "hi"})
	if !errors.Is(err, ErrRunFailed) || !strings.Contains(err.Error(), "stderr-fail") {
		t.Fatalf("err = %v", err)
	}
	if result == nil || result.ExitCode != 7 {
		t.Fatalf("result = %+v", result)
	}
}

func TestWithoutEnv(t *testing.T) {
	got := withoutEnv([]string{"A=1", "NOVALUE", "B=2"}, []string{"A"})
	if strings.Join(got, ",") != "B=2" {
		t.Fatalf("env = %#v", got)
	}
	if got := clip(strings.Repeat("z", stderrClip+10)); len(got) != stderrClip {
		t.Fatalf("clip len = %d", len(got))
	}
	var limited limitedWriter
	limited.max = 4
	if _, err := limited.Write([]byte("abcdef")); err != nil || limited.String() != "abcd" {
		t.Fatalf("limited = %q err=%v", limited.String(), err)
	}
	limited.max = 0
	if _, err := limited.Write([]byte("xyz")); err != nil {
		t.Fatal(err)
	}
}

func writeExec(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
}
