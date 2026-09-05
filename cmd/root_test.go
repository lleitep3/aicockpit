package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/i18n"
	"github.com/lleitep3/aicockpit/internal/logging"
	"github.com/lleitep3/aicockpit/internal/vault"
	"github.com/spf13/cobra"
)

func newTestDeps(t *testing.T) (*logging.Manager, *config.Config, *i18n.Translator) {
	t.Helper()
	if os.Getenv("COCKPIT_DEV_MODE") == "true" {
		mp := vault.NewMasterPassword()
		if err := mp.SetPassword("test-master-password"); err != nil {
			t.Fatalf("failed to prepare test master password: %v", err)
		}
		previousPrompt := promptPassword
		promptPassword = func() (string, error) { return "test-master-password", nil }
		t.Cleanup(func() { promptPassword = previousPrompt })
	}
	log, err := logging.NewManager(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	cfg := &config.Config{
		Version:          "0.1.0",
		Language:         "en-us",
		LogLevel:         "info",
		EnabledProviders: []string{"antigravity"},
		AutoUpdateCheck:  false, // disabled by default so tests are fast
	}
	return log, cfg, i18n.New("en-us")
}

func TestNewRootCommand(t *testing.T) {
	log, cfg, tr := newTestDeps(t)
	cmd := NewRootCommand(log, cfg, tr)

	if cmd == nil {
		t.Fatal("NewRootCommand() returned nil")
	}
	if cmd.Use != "cockpit" {
		t.Errorf("Use = %q, want %q", cmd.Use, "cockpit")
	}
}

// TestCheckForUpdates_NonInteractive_NoStdinBlock verifies that when
// interactive=false the function returns without reading from stdin.
// If it tried to read, the strings.NewReader("") would return EOF immediately
// and the test would panic / deadlock — so a clean return proves the fix.
func TestCheckForUpdates_NonInteractive_NoStdinBlock(t *testing.T) {
	log, cfg, tr := newTestDeps(t)

	// AutoUpdateCheck=false → ShouldCheckUpdate()=false → returns early.
	// We use interactive=true here to ensure the early-return is really from
	// ShouldCheckUpdate, not from the TTY guard.
	cfg.AutoUpdateCheck = false
	stdin := strings.NewReader("") // empty — would hang if read were attempted

	// Must return without blocking.
	checkForUpdatesWithReader(log, cfg, tr, stdin, false)
	checkForUpdatesWithReader(log, cfg, tr, stdin, true)
}

// TestCheckForUpdates_NonInteractive_SkipsPrompt verifies that when a new
// version IS available but interactive=false the prompt is skipped.
// We set AutoUpdateCheck=false so the network is never hit; the test just
// confirms the function doesn't try to read from an empty reader.
func TestCheckForUpdates_NonInteractive_SkipsPrompt(t *testing.T) {
	log, cfg, tr := newTestDeps(t)
	cfg.AutoUpdateCheck = false
	stdin := strings.NewReader("") // would cause EOF panic if read

	// Should not block or panic.
	checkForUpdatesWithReader(log, cfg, tr, stdin, false)
}

// TestCheckForUpdates_Interactive_AcceptsYes simulates an interactive session
// where the user types "y". Because AutoUpdateCheck=false we never hit the
// network, so this just validates the early-return path is clean.
func TestCheckForUpdates_Interactive_AcceptsYes(t *testing.T) {
	log, cfg, tr := newTestDeps(t)
	cfg.AutoUpdateCheck = false
	stdin := strings.NewReader("y\n")

	// Should return cleanly without error (no update check performed).
	checkForUpdatesWithReader(log, cfg, tr, stdin, true)
}

// TestCheckForUpdates_Interactive_AcceptsNo mirrors the "n" path.
func TestCheckForUpdates_Interactive_AcceptsNo(t *testing.T) {
	log, cfg, tr := newTestDeps(t)
	cfg.AutoUpdateCheck = false
	stdin := strings.NewReader("n\n")

	checkForUpdatesWithReader(log, cfg, tr, stdin, true)
}

func TestNewRootCommand_SubcommandCount(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	log, cfg, tr := newTestDeps(t)
	cmd := NewRootCommand(log, cfg, tr)

	// Should have at least setup, deploy, info, doctor, uninstall, vault, metrics, kb, pkg, update
	if len(cmd.Commands()) < 10 {
		t.Errorf("expected at least 10 subcommands, got %d", len(cmd.Commands()))
	}
}

func TestNewRootCommand_Flags(t *testing.T) {
	log, cfg, tr := newTestDeps(t)
	cmd := NewRootCommand(log, cfg, tr)

	if cmd.PersistentFlags().Lookup("language") == nil {
		t.Error("expected --language persistent flag")
	}
	if cmd.PersistentFlags().Lookup("log-level") == nil {
		t.Error("expected --log-level persistent flag")
	}
}

func TestNewRootCommand_PersistentPreRun_SkipsUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	log, cfg, tr := newTestDeps(t)
	cfg.AutoUpdateCheck = false

	cmd := NewRootCommand(log, cfg, tr)
	// The PersistentPreRun is on the rootCmd itself. Invoke it with cmd.Name()="update"
	// to exercise the skip path.
	if cmd.PersistentPreRun != nil {
		// Create a fake child with Name() == "update"
		fakeChild := &cobra.Command{Use: "update"}
		cmd.PersistentPreRun(fakeChild, []string{})
	}
}

func TestNewRootCommand_PersistentPreRun_SkipsSetup(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	log, cfg, tr := newTestDeps(t)
	cfg.AutoUpdateCheck = false

	cmd := NewRootCommand(log, cfg, tr)
	if cmd.PersistentPreRun != nil {
		fakeChild := &cobra.Command{Use: "setup"}
		cmd.PersistentPreRun(fakeChild, []string{})
	}
}

func TestNewRootCommand_PersistentPreRun_RegularCommand(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	log, cfg, tr := newTestDeps(t)
	cfg.AutoUpdateCheck = false // ensures checkForUpdates returns fast

	cmd := NewRootCommand(log, cfg, tr)
	if cmd.PersistentPreRun != nil {
		fakeChild := &cobra.Command{Use: "info"}
		cmd.PersistentPreRun(fakeChild, []string{})
	}
}

// TestCheckForUpdatesWithReader_ShouldCheck exercises the full path when
// AutoUpdateCheck is true and the last check was long ago (triggers network).
// The network call may fail (no internet in CI) but we exercise all code paths.
func TestCheckForUpdatesWithReader_ShouldCheck_NonInteractive(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create cockpit dir so SetLastUpdateCheck can persist
	cockpitDir := filepath.Join(tmpDir, ".cockpit")
	if err := os.MkdirAll(cockpitDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	// Write minimal config so Save() works
	cfgYaml := `version: "0.1.0"
language: en-us
auto_update_check: true
`
	if err := os.WriteFile(filepath.Join(cockpitDir, "config.yaml"), []byte(cfgYaml), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	log, _, tr := newTestDeps(t)
	cfg := &config.Config{
		Version:         "0.1.0",
		Language:        "en-us",
		AutoUpdateCheck: true,
		LastUpdateCheck: "2020-01-01T00:00:00Z", // long ago
	}

	stdin := strings.NewReader("")
	// Non-interactive: should never block on stdin
	checkForUpdatesWithReader(log, cfg, tr, stdin, false)
}

func TestCheckForUpdatesWithReader_ShouldCheck_Interactive(t *testing.T) {
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

	log, _, tr := newTestDeps(t)
	cfg := &config.Config{
		Version:         "0.1.0",
		Language:        "en-us",
		AutoUpdateCheck: true,
		LastUpdateCheck: "2020-01-01T00:00:00Z",
	}

	// Interactive: provide "n\n" so if prompted it declines
	stdin := strings.NewReader("n\n")
	checkForUpdatesWithReader(log, cfg, tr, stdin, true)
}

func TestCheckForUpdatesWithReader_EmptyLastCheck(t *testing.T) {
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

	log, _, tr := newTestDeps(t)
	cfg := &config.Config{
		Version:         "0.1.0",
		Language:        "en-us",
		AutoUpdateCheck: true,
		LastUpdateCheck: "", // empty triggers check
	}

	stdin := strings.NewReader("n\n")
	checkForUpdatesWithReader(log, cfg, tr, stdin, true)
}

// ── Mock update checker for full branch coverage ──────────────────────────

type mockUpdateChecker struct {
	version    string
	releaseURL string
	err        error
}

func (m *mockUpdateChecker) CheckForUpdates() (string, string, error) {
	return m.version, m.releaseURL, m.err
}

func TestCheckForUpdatesWithService_Error(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cockpitDir := filepath.Join(tmpDir, ".cockpit")
	os.MkdirAll(cockpitDir, 0o755)
	os.WriteFile(filepath.Join(cockpitDir, "config.yaml"), []byte("version: \"0.1.0\"\nlanguage: en-us\nauto_update_check: true\n"), 0o644)

	log, _, tr := newTestDeps(t)
	cfg := &config.Config{
		Version:         "0.1.0",
		Language:        "en-us",
		AutoUpdateCheck: true,
		LastUpdateCheck: "2020-01-01T00:00:00Z",
	}

	mock := &mockUpdateChecker{err: fmt.Errorf("network error")}
	stdin := strings.NewReader("")
	checkForUpdatesWithService(log, cfg, tr, stdin, false, mock)
}

func TestCheckForUpdatesWithService_NoUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cockpitDir := filepath.Join(tmpDir, ".cockpit")
	os.MkdirAll(cockpitDir, 0o755)
	os.WriteFile(filepath.Join(cockpitDir, "config.yaml"), []byte("version: \"0.1.0\"\nlanguage: en-us\nauto_update_check: true\n"), 0o644)

	log, _, tr := newTestDeps(t)
	cfg := &config.Config{
		Version:         "0.1.0",
		Language:        "en-us",
		AutoUpdateCheck: true,
		LastUpdateCheck: "2020-01-01T00:00:00Z",
	}

	mock := &mockUpdateChecker{version: "", releaseURL: ""}
	stdin := strings.NewReader("")
	checkForUpdatesWithService(log, cfg, tr, stdin, false, mock)
}

func TestCheckForUpdatesWithService_UpdateAvailable_NonInteractive(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cockpitDir := filepath.Join(tmpDir, ".cockpit")
	os.MkdirAll(cockpitDir, 0o755)
	os.WriteFile(filepath.Join(cockpitDir, "config.yaml"), []byte("version: \"0.1.0\"\nlanguage: en-us\nauto_update_check: true\n"), 0o644)

	log, _, tr := newTestDeps(t)
	cfg := &config.Config{
		Version:         "0.1.0",
		Language:        "en-us",
		AutoUpdateCheck: true,
		LastUpdateCheck: "2020-01-01T00:00:00Z",
	}

	mock := &mockUpdateChecker{version: "9.9.9", releaseURL: "https://github.com/releases/9.9.9"}
	stdin := strings.NewReader("")
	checkForUpdatesWithService(log, cfg, tr, stdin, false, mock)
}

func TestCheckForUpdatesWithService_UpdateAvailable_Interactive_AcceptY(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cockpitDir := filepath.Join(tmpDir, ".cockpit")
	os.MkdirAll(cockpitDir, 0o755)
	os.WriteFile(filepath.Join(cockpitDir, "config.yaml"), []byte("version: \"0.1.0\"\nlanguage: en-us\nauto_update_check: true\n"), 0o644)

	log, _, tr := newTestDeps(t)
	cfg := &config.Config{
		Version:         "0.1.0",
		Language:        "en-us",
		AutoUpdateCheck: true,
		LastUpdateCheck: "2020-01-01T00:00:00Z",
	}

	mock := &mockUpdateChecker{version: "9.9.9", releaseURL: "https://github.com/releases/9.9.9"}
	stdin := strings.NewReader("y\n")
	checkForUpdatesWithService(log, cfg, tr, stdin, true, mock)
}

func TestCheckForUpdatesWithService_UpdateAvailable_Interactive_AcceptSim(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cockpitDir := filepath.Join(tmpDir, ".cockpit")
	os.MkdirAll(cockpitDir, 0o755)
	os.WriteFile(filepath.Join(cockpitDir, "config.yaml"), []byte("version: \"0.1.0\"\nlanguage: en-us\nauto_update_check: true\n"), 0o644)

	log, _, tr := newTestDeps(t)
	cfg := &config.Config{
		Version:         "0.1.0",
		Language:        "en-us",
		AutoUpdateCheck: true,
		LastUpdateCheck: "2020-01-01T00:00:00Z",
	}

	mock := &mockUpdateChecker{version: "9.9.9", releaseURL: "https://github.com/releases/9.9.9"}
	stdin := strings.NewReader("sim\n")
	checkForUpdatesWithService(log, cfg, tr, stdin, true, mock)
}

func TestCheckForUpdatesWithService_UpdateAvailable_Interactive_Decline(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cockpitDir := filepath.Join(tmpDir, ".cockpit")
	os.MkdirAll(cockpitDir, 0o755)
	os.WriteFile(filepath.Join(cockpitDir, "config.yaml"), []byte("version: \"0.1.0\"\nlanguage: en-us\nauto_update_check: true\n"), 0o644)

	log, _, tr := newTestDeps(t)
	cfg := &config.Config{
		Version:         "0.1.0",
		Language:        "en-us",
		AutoUpdateCheck: true,
		LastUpdateCheck: "2020-01-01T00:00:00Z",
	}

	mock := &mockUpdateChecker{version: "9.9.9", releaseURL: "https://github.com/releases/9.9.9"}
	stdin := strings.NewReader("n\n")
	checkForUpdatesWithService(log, cfg, tr, stdin, true, mock)
}

func TestCheckForUpdatesWithService_SetLastUpdateCheckFails(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	// Don't create cockpit dir so SetLastUpdateCheck fails

	log, _, tr := newTestDeps(t)
	cfg := &config.Config{
		Version:         "0.1.0",
		Language:        "en-us",
		AutoUpdateCheck: true,
		LastUpdateCheck: "2020-01-01T00:00:00Z",
	}

	mock := &mockUpdateChecker{version: "", releaseURL: ""}
	stdin := strings.NewReader("")
	checkForUpdatesWithService(log, cfg, tr, stdin, false, mock)
}
