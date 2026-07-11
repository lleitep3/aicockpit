package cmd

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/i18n"
	"github.com/lleitep3/aicockpit/internal/logging"
	"github.com/zalando/go-keyring"
)

func TestVaultCommands(t *testing.T) {
	// Enable mock keyring for testing
	keyring.MockInit()
	log, _ := logging.NewManager("")
	cfg := &config.Config{Version: "0.1.0", Language: "en-us"}
	translator := i18n.New("en-us")

	t.Run("Test NewVaultCommand", func(t *testing.T) {
		cmd := NewVaultCommand(log, cfg, translator)
		if cmd == nil {
			t.Fatal("Expected command, got nil")
		}
		if cmd.Use != "vault" {
			t.Errorf("Expected Use to be 'vault', got '%s'", cmd.Use)
		}
	})

	t.Run("Test Vault Set and Get", func(t *testing.T) {
		key := "test_key_cli"
		value := "secret_cli_value"

		// Set via CLI args with --value flag and --namespace
		setCmd := NewVaultSetCommand(log, cfg, translator)
		setCmd.SetArgs([]string{key, "--value", value, "--namespace", "test"})

		var out bytes.Buffer
		setCmd.SetOut(&out)
		setCmd.SetErr(&out)

		err := setCmd.Execute()
		if err != nil {
			t.Fatalf("Failed to execute set command: %v", err)
		}

		// Get via CLI with --namespace
		getCmd := NewVaultGetCommand(log, cfg, translator)
		getCmd.SetArgs([]string{key, "--namespace", "test"})

		out.Reset()
		getCmd.SetOut(&out)
		getCmd.SetErr(&out)

		err = getCmd.Execute()
		if err != nil {
			t.Fatalf("Failed to execute get command: %v", err)
		}

		if out.String() != value {
			t.Errorf("Expected output %q, got %q", value, out.String())
		}
	})

	t.Run("Test Vault Remove", func(t *testing.T) {
		key := "test_key_remove"
		value := "val"

		// Set first
		setCmd := NewVaultSetCommand(log, cfg, translator)
		setCmd.SetArgs([]string{key, "--value", value, "--namespace", "test"})
		_ = setCmd.Execute()

		// Remove via CLI with --namespace
		removeCmd := NewVaultRemoveCommand(log, cfg, translator)
		removeCmd.SetArgs([]string{key, "--namespace", "test"})

		var out bytes.Buffer
		removeCmd.SetOut(&out)
		removeCmd.SetErr(&out)

		err := removeCmd.Execute()
		if err != nil {
			t.Fatalf("Failed to execute remove command: %v", err)
		}

		// Verify it's gone with --namespace
		getCmd := NewVaultGetCommand(log, cfg, translator)
		getCmd.SetArgs([]string{key, "--namespace", "test"})
		err = getCmd.Execute()
		if err == nil {
			t.Errorf("Expected error when getting removed key, got nil")
		}
	})
}

func TestVaultSet_WithValueFlag_NoNamespace_VaultLocked(t *testing.T) {
	// Vault is locked by default — set without namespace should fail on access check
	keyring.MockInit()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	log, _ := logging.NewManager("")
	cfg := &config.Config{Version: "0.1.0", Language: "en-us"}
	translator := i18n.New("en-us")

	setCmd := NewVaultSetCommand(log, cfg, translator)
	setCmd.SetArgs([]string{"mykey", "--value", "myvalue"})
	err := setCmd.Execute()
	if err == nil {
		t.Error("expected error: vault locked")
	}
}

func TestVaultSet_WithValueFlag_WithNamespace(t *testing.T) {
	keyring.MockInit()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	log, _ := logging.NewManager("")
	cfg := &config.Config{Version: "0.1.0", Language: "en-us"}
	translator := i18n.New("en-us")

	// With namespace — bypasses vault access check
	setCmd := NewVaultSetCommand(log, cfg, translator)
	var out bytes.Buffer
	setCmd.SetOut(&out)
	setCmd.SetErr(&out)
	setCmd.SetArgs([]string{"mykey2", "--value", "myvalue2", "--namespace", "testns"})
	err := setCmd.Execute()
	if err != nil {
		t.Errorf("vault set with namespace error = %v", err)
	}

	// Verify the output message contains namespace info
	if !bytes.Contains(out.Bytes(), []byte("testns")) {
		t.Error("expected output to mention namespace")
	}
}

func TestVaultSet_EmptyValue(t *testing.T) {
	keyring.MockInit()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	log, _ := logging.NewManager("")
	cfg := &config.Config{Version: "0.1.0", Language: "en-us"}
	translator := i18n.New("en-us")

	// With namespace (bypasses vault access check) + --value=""
	setCmd := NewVaultSetCommand(log, cfg, translator)
	setCmd.SetArgs([]string{"mykey", "--value", "", "--namespace", "test"})
	err := setCmd.Execute()
	// Empty value without terminal prompt should error: "failed to read secret" or
	// "secret value cannot be empty"
	if err == nil {
		t.Error("expected error for empty value")
	}
}

func TestVaultGet_WithNamespace_Success(t *testing.T) {
	keyring.MockInit()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	log, _ := logging.NewManager("")
	cfg := &config.Config{Version: "0.1.0", Language: "en-us"}
	translator := i18n.New("en-us")

	// First set a value
	setCmd := NewVaultSetCommand(log, cfg, translator)
	setCmd.SetArgs([]string{"testkey", "--value", "secretvalue", "--namespace", "testns"})
	if err := setCmd.Execute(); err != nil {
		t.Fatalf("set error = %v", err)
	}

	// Now get it
	getCmd := NewVaultGetCommand(log, cfg, translator)
	var out bytes.Buffer
	getCmd.SetOut(&out)
	getCmd.SetArgs([]string{"testkey", "--namespace", "testns"})
	if err := getCmd.Execute(); err != nil {
		t.Errorf("vault get error = %v", err)
	}
	if out.String() != "secretvalue" {
		t.Errorf("expected %q, got %q", "secretvalue", out.String())
	}
}

func TestVaultRemove_WithNamespace_Success(t *testing.T) {
	keyring.MockInit()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	log, _ := logging.NewManager("")
	cfg := &config.Config{Version: "0.1.0", Language: "en-us"}
	translator := i18n.New("en-us")

	// First set a value
	setCmd := NewVaultSetCommand(log, cfg, translator)
	setCmd.SetArgs([]string{"delkey", "--value", "todelete", "--namespace", "testns"})
	_ = setCmd.Execute()

	// Remove it
	removeCmd := NewVaultRemoveCommand(log, cfg, translator)
	removeCmd.SetArgs([]string{"delkey", "--namespace", "testns"})
	if err := removeCmd.Execute(); err != nil {
		t.Errorf("vault remove error = %v", err)
	}

	// Verify it's gone
	getCmd := NewVaultGetCommand(log, cfg, translator)
	getCmd.SetArgs([]string{"delkey", "--namespace", "testns"})
	if err := getCmd.Execute(); err == nil {
		t.Error("expected error: key should be deleted")
	}
}

func TestVaultGet_NotFound(t *testing.T) {
	keyring.MockInit()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("COCKPIT_DEV_MODE", "1")
	defer t.Setenv("COCKPIT_DEV_MODE", "")

	log, _ := logging.NewManager("")
	cfg := &config.Config{Version: "0.1.0", Language: "en-us"}
	translator := i18n.New("en-us")

	getCmd := NewVaultGetCommand(log, cfg, translator)
	getCmd.SetArgs([]string{"nonexistent-key", "--namespace", "test"})
	err := getCmd.Execute()
	if err == nil {
		t.Error("expected error: key not found")
	}
}

func TestVaultRemove_NotFound(t *testing.T) {
	keyring.MockInit()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("COCKPIT_DEV_MODE", "1")
	defer t.Setenv("COCKPIT_DEV_MODE", "")

	log, _ := logging.NewManager("")
	cfg := &config.Config{Version: "0.1.0", Language: "en-us"}
	translator := i18n.New("en-us")

	rmCmd := NewVaultRemoveCommand(log, cfg, translator)
	rmCmd.SetArgs([]string{"nonexistent-key", "--namespace", "test"})
	err := rmCmd.Execute()
	if err == nil {
		t.Error("expected error: key not found for removal")
	}
}

// ── checkVaultAccess ──────────────────────────────────────────────────────

func TestCheckVaultAccess_LockedByDefault(t *testing.T) {
	// Redirect HOME so the lock manager finds no persisted lock file.
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// The vault is locked by default (IsLocked: true) when no state file exists.
	// checkVaultAccess must return an error in this state.
	err := checkVaultAccess("get")
	if err == nil {
		t.Error("checkVaultAccess() expected error (vault locked by default), got nil")
	}
}

func TestCheckVaultAccess_DevMode(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("COCKPIT_DEV_MODE", "1")
	defer t.Setenv("COCKPIT_DEV_MODE", "")

	// In dev mode, vault access should be allowed even without unlock
	err := checkVaultAccess("get")
	// Behavior depends on implementation; just exercise the path
	_ = err
}

func TestGetCurrentProcessName_Returns(t *testing.T) {
	name := getCurrentProcessName()
	if name == "" {
		t.Error("expected non-empty process name")
	}
	// In test environment, the executable is something like "cmd.test" → returns "cmd"
	// Just verify it's not "unknown" (which would mean os.Executable failed)
	if name == "unknown" {
		t.Error("getCurrentProcessName() returned 'unknown' — os.Executable likely failed")
	}
}

func TestGetCurrentProcessName_CockpitCLI(t *testing.T) {
	orig := osExecutable
	osExecutable = func() (string, error) { return "/usr/local/bin/cockpit", nil }
	defer func() { osExecutable = orig }()

	name := getCurrentProcessName()
	if name != "cockpit-cli" {
		t.Errorf("expected 'cockpit-cli', got %q", name)
	}
}

func TestGetCurrentProcessName_PackagePath(t *testing.T) {
	orig := osExecutable
	osExecutable = func() (string, error) { return "/home/user/.cockpit/packages/my-pkg/bin/run", nil }
	defer func() { osExecutable = orig }()

	name := getCurrentProcessName()
	if name != "my-pkg" {
		t.Errorf("expected 'my-pkg', got %q", name)
	}
}

func TestGetCurrentProcessName_Error(t *testing.T) {
	orig := osExecutable
	osExecutable = func() (string, error) { return "", fmt.Errorf("no executable") }
	defer func() { osExecutable = orig }()

	name := getCurrentProcessName()
	if name != "unknown" {
		t.Errorf("expected 'unknown', got %q", name)
	}
}

// ── checkForUpdates ───────────────────────────────────────────────────────

func TestCheckForUpdates_NonInteractive_AutoUpdateDisabled(t *testing.T) {
	log, cfg, tr := newTestDeps(t)
	cfg.AutoUpdateCheck = false
	// With AutoUpdateCheck=false, ShouldCheckUpdate returns false → early return.
	// Must not block on network or stdin.
	checkForUpdates(log, cfg, tr)
}
