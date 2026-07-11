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

func TestPerformUpdate_NotInGitRepo(t *testing.T) {
	tmpDir := t.TempDir()
	// Ensure we're not in a git repo
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(tmpDir)

	err := performUpdate("1.0.0")
	if err == nil {
		t.Error("performUpdate() should fail when not in git repo")
	}
}

func TestPerformUpdate_InGitRepo_FetchFails(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a fake .git dir so performUpdate passes the first check
	if err := os.MkdirAll(filepath.Join(tmpDir, ".git"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(tmpDir)

	// git fetch will fail because it's not a real repo
	err := performUpdate("1.0.0")
	if err == nil {
		t.Error("performUpdate() should fail when git fetch fails")
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

func TestRunUpdateWithDeps_UserAccepts_NoGitDir(t *testing.T) {
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
	// User says yes but performUpdate will fail (no .git dir in cwd)
	stdin := pipeWithInput(t, "y\n")

	// Change to a temp dir without .git
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	err = runUpdateWithDeps(log, cfg, tr, mock, stdin)
	if err == nil {
		t.Error("expected error when no .git dir")
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

	// runSetup may fail (providers) — that's fine, we just exercise the branches
	_ = runUpdateWithDeps(log, cfg, tr, mock, stdin)
}

func TestPerformUpdate_NoGitDir(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	if err := performUpdate("1.0.0"); err == nil {
		t.Error("expected error when .git doesn't exist")
	}
}

func TestPerformUpdate_FetchFails(t *testing.T) {
	tmpDir := t.TempDir()
	// Create .git dir to pass the first check
	os.MkdirAll(filepath.Join(tmpDir, ".git"), 0o755)

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	// Use real runCommand — git fetch will fail since it's not a real repo
	origCmd := runCommandFunc
	runCommandFunc = runCommandDefault
	defer func() { runCommandFunc = origCmd }()

	err := performUpdate("1.0.0")
	if err == nil {
		t.Error("expected error when git fetch fails")
	}
}

func TestPerformUpdate_AllCommandsSucceed(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, ".git"), 0o755)

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	// Mock all commands to succeed
	origCmd := runCommandFunc
	runCommandFunc = func(name string, args ...string) error { return nil }
	defer func() { runCommandFunc = origCmd }()

	err := performUpdate("1.0.0")
	if err != nil {
		t.Errorf("performUpdate with mocked commands error = %v", err)
	}
}

func TestPerformUpdate_CheckoutFails(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, ".git"), 0o755)

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	callCount := 0
	origCmd := runCommandFunc
	runCommandFunc = func(name string, args ...string) error {
		callCount++
		if callCount == 2 { // checkout (second call)
			return fmt.Errorf("checkout failed")
		}
		return nil
	}
	defer func() { runCommandFunc = origCmd }()

	err := performUpdate("1.0.0")
	if err == nil {
		t.Error("expected error when checkout fails")
	}
}

func TestPerformUpdate_PullFails(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, ".git"), 0o755)

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	callCount := 0
	origCmd := runCommandFunc
	runCommandFunc = func(name string, args ...string) error {
		callCount++
		if callCount == 3 { // pull (third call)
			return fmt.Errorf("pull failed")
		}
		return nil
	}
	defer func() { runCommandFunc = origCmd }()

	err := performUpdate("1.0.0")
	if err == nil {
		t.Error("expected error when pull fails")
	}
}

func TestPerformUpdate_BuildFails(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, ".git"), 0o755)

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	callCount := 0
	origCmd := runCommandFunc
	runCommandFunc = func(name string, args ...string) error {
		callCount++
		if callCount == 4 { // make build (fourth call)
			return fmt.Errorf("build failed")
		}
		return nil
	}
	defer func() { runCommandFunc = origCmd }()

	err := performUpdate("1.0.0")
	if err == nil {
		t.Error("expected error when build fails")
	}
}

func TestPerformUpdate_InstallFails(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, ".git"), 0o755)

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	callCount := 0
	origCmd := runCommandFunc
	runCommandFunc = func(name string, args ...string) error {
		callCount++
		if callCount == 5 { // make install-local (fifth call)
			return fmt.Errorf("install failed")
		}
		return nil
	}
	defer func() { runCommandFunc = origCmd }()

	err := performUpdate("1.0.0")
	if err == nil {
		t.Error("expected error when install fails")
	}
}
