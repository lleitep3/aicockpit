package assets

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lleitep3/aicockpit/internal/providers"
)

func TestRestoreAssets(t *testing.T) {
	tmpDir := t.TempDir()

	err := RestoreAssets(tmpDir)
	if err != nil {
		t.Fatalf("RestoreAssets failed: %v", err)
	}

	// Verify expected files were created
	expectedFiles := []string{
		"config.yaml",
		"providers.yaml",
	}

	for _, file := range expectedFiles {
		path := filepath.Join(tmpDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file %s was not created", file)
		}
	}
}

func TestRestoreAssets_CodexProviderConfiguration(t *testing.T) {
	tmpDir := t.TempDir()
	if err := RestoreAssets(tmpDir); err != nil {
		t.Fatalf("RestoreAssets failed: %v", err)
	}

	config, err := providers.LoadProvidersConfig(filepath.Join(tmpDir, "providers.yaml"))
	if err != nil {
		t.Fatalf("LoadProvidersConfig failed: %v", err)
	}
	codex := config.GetProvider("codex")
	if codex == nil {
		t.Fatal("Codex provider is missing from embedded providers.yaml")
	}
	if codex.Binary != "codex" || codex.Workspace != "~/.codex" {
		t.Errorf("Codex provider identity = binary %q, workspace %q", codex.Binary, codex.Workspace)
	}
	for _, feature := range []string{"rules", "skills", "workflows", "permissions", "agents"} {
		if !codex.SupportsFeature(feature) {
			t.Errorf("Codex should support enabled feature %q", feature)
		}
	}
}
