package codexconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureSandboxWritableRootCreatesConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "codex", "config.toml")
	root := filepath.Join(t.TempDir(), "cockpit", "logs")

	changed, err := EnsureSandboxWritableRoot(configPath, root)
	if err != nil {
		t.Fatalf("EnsureSandboxWritableRoot() error = %v", err)
	}
	if !changed {
		t.Fatal("EnsureSandboxWritableRoot() changed = false, want true")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	contents := string(data)
	for _, expected := range []string{
		`sandbox_mode = "workspace-write"`,
		"[sandbox_workspace_write]",
		`writable_roots = ["` + root + `"]`,
	} {
		if !strings.Contains(contents, expected) {
			t.Errorf("config missing %q:\n%s", expected, contents)
		}
	}
}

func TestEnsureSandboxWritableRootIsIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")
	root := filepath.Join(tmpDir, "logs")

	if _, err := EnsureSandboxWritableRoot(configPath, root); err != nil {
		t.Fatalf("first EnsureSandboxWritableRoot() error = %v", err)
	}
	changed, err := EnsureSandboxWritableRoot(configPath, root)
	if err != nil {
		t.Fatalf("second EnsureSandboxWritableRoot() error = %v", err)
	}
	if changed {
		t.Fatal("second EnsureSandboxWritableRoot() changed = true, want false")
	}

	data, _ := os.ReadFile(configPath)
	if got := strings.Count(string(data), "[sandbox_workspace_write]"); got != 1 {
		t.Fatalf("sandbox table count = %d, want 1", got)
	}
	if got := strings.Count(string(data), `"`+root+`"`); got != 1 {
		t.Fatalf("writable root count = %d, want 1", got)
	}
}

func TestEnsureSandboxWritableRootPreservesExistingConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")
	root := filepath.Join(tmpDir, "cockpit", "logs")
	initial := `model = "gpt-6-astra"
sandbox_mode = "read-only"

[sandbox_workspace_write]
writable_roots = ["/tmp/existing"]

[history]
cleanup = true
`
	if err := os.WriteFile(configPath, []byte(initial), 0o600); err != nil {
		t.Fatal(err)
	}

	changed, err := EnsureSandboxWritableRoot(configPath, root)
	if err != nil {
		t.Fatalf("EnsureSandboxWritableRoot() error = %v", err)
	}
	if !changed {
		t.Fatal("EnsureSandboxWritableRoot() changed = false, want true")
	}

	data, _ := os.ReadFile(configPath)
	contents := string(data)
	for _, expected := range []string{
		`model = "gpt-6-astra"`,
		`sandbox_mode = "workspace-write"`,
		`"/tmp/existing"`,
		`"` + root + `"`,
		"[history]",
		"cleanup = true",
	} {
		if !strings.Contains(contents, expected) {
			t.Errorf("config lost or missed %q:\n%s", expected, contents)
		}
	}

	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("config permissions = %o, want 600", got)
	}
}

func TestEnsureSandboxWritableRootExtendsMultilineArray(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")
	root := filepath.Join(tmpDir, "logs")
	initial := `[sandbox_workspace_write]
writable_roots = [
    "/tmp/existing",
]
`
	if err := os.WriteFile(configPath, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := EnsureSandboxWritableRoot(configPath, root); err != nil {
		t.Fatalf("EnsureSandboxWritableRoot() error = %v", err)
	}

	data, _ := os.ReadFile(configPath)
	contents := string(data)
	if !strings.Contains(contents, `"/tmp/existing",`) || !strings.Contains(contents, `"`+root+`",`) {
		t.Errorf("multiline array was not extended:\n%s", contents)
	}
}
