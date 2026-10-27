package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/i18n"
	"github.com/lleitep3/aicockpit/internal/logging"
)

func TestNewUpdateCommand(t *testing.T) {
	log, err := logging.NewManager("/tmp/test-cockpit")
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	cfg := &config.Config{
		Version:          "0.1.0",
		Language:         "en-us",
		LogLevel:         "info",
		EnabledProviders: []string{"antigravity"},
	}
	translator := i18n.New("en-us")

	cmd := NewUpdateCommand(log, cfg, translator)

	if cmd == nil {
		t.Fatal("NewUpdateCommand() returned nil")
	}

	if cmd.Use != "update" {
		t.Errorf("NewUpdateCommand() Use = %v, want %v", cmd.Use, "update")
	}

	if cmd.Short == "" {
		t.Error("expected non-empty Short description")
	}

	if cmd.Long == "" {
		t.Error("expected non-empty Long description")
	}
}

func TestRunCommand_Success(t *testing.T) {
	err := runCommand("echo", "hello")
	if err != nil {
		t.Errorf("runCommand('echo', 'hello') error = %v", err)
	}
}

func TestRunCommand_Failure(t *testing.T) {
	err := runCommand("false")
	if err == nil {
		t.Error("runCommand('false') should fail")
	}
}

func TestRunCommand_NotFound(t *testing.T) {
	err := runCommand("nonexistent-binary-xyz-12345")
	if err == nil {
		t.Error("runCommand with nonexistent binary should fail")
	}
}

func TestNewUpdateCommand_Execute(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cockpitDir := filepath.Join(tmpDir, ".cockpit")
	if err := os.MkdirAll(cockpitDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	cfgYaml := `version: "0.1.0"
language: en-us
auto_update_check: true
`
	if err := os.WriteFile(filepath.Join(cockpitDir, "config.yaml"), []byte(cfgYaml), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	log, err := logging.NewManager(filepath.Join(tmpDir, "logs"))
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	cfg := &config.Config{
		Version:          "0.1.0",
		Language:         "en-us",
		AutoUpdateCheck:  true,
		EnabledProviders: []string{"antigravity"},
	}
	translator := i18n.New("en-us")

	cmd := NewUpdateCommand(log, cfg, translator)
	// Execute with stdin "n" to decline any prompt
	withStdinUpdate(t, "n", func() {
		// May error (network issues) — that's fine, just exercises the RunE lambda
		_ = cmd.Execute()
	})
}

// TestRunUpdate exercises the runUpdate function. It calls the network (GitHub API)
// to check for updates. The test provides stdin "n\n" so if a prompt appears, it declines.
func TestRunUpdate_DeclinesUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cockpitDir := filepath.Join(tmpDir, ".cockpit")
	if err := os.MkdirAll(cockpitDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	cfgYaml := `version: "0.1.0"
language: en-us
auto_update_check: true
`
	if err := os.WriteFile(filepath.Join(cockpitDir, "config.yaml"), []byte(cfgYaml), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	log, err := logging.NewManager(filepath.Join(tmpDir, "logs"))
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	cfg := &config.Config{
		Version:          "0.1.0",
		Language:         "en-us",
		LogLevel:         "info",
		AutoUpdateCheck:  true,
		EnabledProviders: []string{"antigravity"},
	}
	translator := i18n.New("en-us")

	// Provide "n" to decline the update prompt (if one appears)
	withStdinUpdate(t, "n", func() {
		// runUpdate may error (network issues or no update available) — both are fine
		_ = runUpdate(log, cfg, translator)
	})
}

// withStdinUpdate temporarily replaces os.Stdin with a pipe containing the given input.
func withStdinUpdate(t *testing.T, input string, fn func()) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	_, _ = w.WriteString(input + "\n")
	w.Close()

	origStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = origStdin }()

	fn()
}

// pipeWithInput creates a pipe *os.File with the given content for use with runUpdateWithDeps.
func pipeWithInput(t *testing.T, input string) *os.File {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.WriteString(input)
	w.Close()
	t.Cleanup(func() { r.Close() })
	return r
}

// ── runUpdateWithDeps mock-based tests ────────────────────────────────────

func TestRunUpdateWithDeps_CheckError(t *testing.T) {
	log, err := logging.NewManager(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{Version: "0.1.0", Language: "en-us"}
	tr := i18n.New("en-us")

	mock := &mockUpdateCheckerUpdate{err: fmt.Errorf("network down")}
	stdin := pipeWithInput(t, "")
	if err := runUpdateWithDeps(log, cfg, tr, mock, stdin); err == nil {
		t.Error("expected error when CheckForUpdates fails")
	}
}

func TestRunUpdateWithDeps_AlreadyUpToDate(t *testing.T) {
	log, err := logging.NewManager(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{Version: "0.1.0", Language: "en-us"}
	tr := i18n.New("en-us")

	mock := &mockUpdateCheckerUpdate{version: "", releaseURL: ""}
	stdin := pipeWithInput(t, "")
	if err := runUpdateWithDeps(log, cfg, tr, mock, stdin); err != nil {
		t.Errorf("expected nil error when up to date, got %v", err)
	}
}

func TestRunUpdateWithDeps_UserDeclines(t *testing.T) {
	log, err := logging.NewManager(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{Version: "0.1.0", Language: "en-us"}
	tr := i18n.New("en-us")

	mock := &mockUpdateCheckerUpdate{version: "9.9.9", releaseURL: "https://example.com"}
	stdin := pipeWithInput(t, "n\n")
	if err := runUpdateWithDeps(log, cfg, tr, mock, stdin); err != nil {
		t.Errorf("expected nil error when user declines, got %v", err)
	}
}

func TestRunUpdateWithDeps_UserAccepts_PreparationFails(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cockpitDir := filepath.Join(tmpDir, ".cockpit")
	os.MkdirAll(cockpitDir, 0o755)
	os.WriteFile(filepath.Join(cockpitDir, "config.yaml"), []byte("version: \"0.1.0\"\nlanguage: en-us\n"), 0o644)

	log, err := logging.NewManager(filepath.Join(tmpDir, "logs"))
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{Version: "0.1.0", Language: "en-us"}
	tr := i18n.New("en-us")

	mock := &mockUpdateCheckerUpdate{version: "9.9.9", releaseURL: "https://example.com"}
	original := performUpdateFunc
	performUpdateFunc = func(string) error { return fmt.Errorf("isolated build failed") }
	defer func() { performUpdateFunc = original }()
	// A preparation failure must propagate without changing the current directory.
	stdin := pipeWithInput(t, "y\n")

	// Change to a temp dir without .git
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	err = runUpdateWithDeps(log, cfg, tr, mock, stdin)
	if err == nil {
		t.Error("expected isolated build error")
	}
}

// mockUpdateCheckerUpdate is the same as mockUpdateChecker but local to this file.
type mockUpdateCheckerUpdate struct {
	version    string
	releaseURL string
	err        error
}

func (m *mockUpdateCheckerUpdate) CheckForUpdates() (string, string, error) {
	return m.version, m.releaseURL, m.err
}

// ── performUpdate tests ───────────────────────────────────────────────────

func TestRunUpdateWithDeps_UserAccepts_UpdateSucceeds_DeclinesSetup(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cockpitDir := filepath.Join(tmpDir, ".cockpit")
	os.MkdirAll(cockpitDir, 0o755)
	os.WriteFile(filepath.Join(cockpitDir, "config.yaml"), []byte("version: \"0.1.0\"\nlanguage: en-us\n"), 0o644)

	log, err := logging.NewManager(filepath.Join(tmpDir, "logs"))
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{Version: "0.1.0", Language: "en-us"}
	tr := i18n.New("en-us")

	// Mock performUpdateFunc to succeed
	origPerform := performUpdateFunc
	performUpdateFunc = func(v string) error { return nil }
	defer func() { performUpdateFunc = origPerform }()

	mock := &mockUpdateCheckerUpdate{version: "9.9.9", releaseURL: "https://example.com"}
	// User says "y" to update, then "n" to decline setup
	stdin := pipeWithInput(t, "y\nn\n")

	if err := runUpdateWithDeps(log, cfg, tr, mock, stdin); err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestRunUpdateWithDeps_UserAccepts_UpdateSucceeds_AcceptsSetup(t *testing.T) {
	originalCommand := runCommandFunc
	calledSetup := false
	runCommandFunc = func(name string, args ...string) error {
		executable, err := os.Executable()
		if err != nil {
			t.Fatal(err)
		}
		if name != executable || len(args) != 1 || args[0] != "setup" {
			t.Fatalf("wrong new executable setup: %s %v", name, args)
		}
		calledSetup = true
		return nil
	}
	defer func() { runCommandFunc = originalCommand }()
	defer func() {
		if !calledSetup {
			t.Error("new executable setup not called")
		}
	}()

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Chdir(tmpDir)

	cockpitDir := filepath.Join(tmpDir, ".cockpit")
	os.MkdirAll(cockpitDir, 0o755)
	os.WriteFile(filepath.Join(cockpitDir, "config.yaml"), []byte("version: \"0.1.0\"\nlanguage: en-us\n"), 0o644)

	log, err := logging.NewManager(filepath.Join(tmpDir, "logs"))
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{Version: "0.1.0", Language: "en-us"}
	tr := i18n.New("en-us")

	origPerform := performUpdateFunc
	performUpdateFunc = func(v string) error { return nil }
	defer func() { performUpdateFunc = origPerform }()

	mock := &mockUpdateCheckerUpdate{version: "9.9.9", releaseURL: "https://example.com"}
	// User says "y" to update, then "y" to setup
	// Also override os.Stdin for runSetup which reads from it directly
	stdin := pipeWithInput(t, "y\ny\n1\n\n")
	origStdin := os.Stdin
	os.Stdin = stdin
	defer func() { os.Stdin = origStdin }()

	if err := runUpdateWithDeps(log, cfg, tr, mock, stdin); err != nil {
		t.Fatal(err)
	}
}

func TestPerformUpdateRejectsInvalidRelease(t *testing.T) {
	if err := performUpdate("../not-a-release"); err == nil {
		t.Fatal("invalid release accepted")
	}
}
