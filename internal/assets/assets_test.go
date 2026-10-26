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

func TestRestoreAssets_GitHubCopilotProviderConfiguration(t *testing.T) {
	tmpDir := t.TempDir()
	if err := RestoreAssets(tmpDir); err != nil {
		t.Fatalf("RestoreAssets failed: %v", err)
	}

	config, err := providers.LoadProvidersConfig(filepath.Join(tmpDir, "providers.yaml"))
	if err != nil {
		t.Fatalf("LoadProvidersConfig failed: %v", err)
	}
	copilot := config.GetProvider("github-copilot")
	if copilot == nil {
		t.Fatal("GitHub Copilot provider is missing from embedded providers.yaml")
	}
	if !copilot.Enabled || copilot.Binary != "copilot" || copilot.Workspace != "~/.copilot" {
		t.Errorf("GitHub Copilot provider identity = enabled %t, binary %q, workspace %q", copilot.Enabled, copilot.Binary, copilot.Workspace)
	}
	for _, feature := range []string{"rules", "skills", "workflows", "agents"} {
		if !copilot.SupportsFeature(feature) {
			t.Errorf("GitHub Copilot should support enabled feature %q", feature)
		}
	}
	if copilot.SupportsFeature("permissions") {
		t.Error("GitHub Copilot should not enable canonical permissions")
	}
	if got := copilot.Features["rules"].Path; got != "~/.copilot/copilot-instructions.md" {
		t.Errorf("GitHub Copilot rules path = %q", got)
	}
	if got := copilot.Features["workflows"].Path; got != "~/.copilot/skills" {
		t.Errorf("GitHub Copilot workflows path = %q", got)
	}
	if got := copilot.Features["mcp"].Path; got != "~/.copilot/mcp-config.json" {
		t.Errorf("GitHub Copilot mcp path = %q", got)
	}
}

func TestRestoreAssets_HindsightMCPProviders(t *testing.T) {
	tmpDir := t.TempDir()
	if err := RestoreAssets(tmpDir); err != nil {
		t.Fatalf("RestoreAssets failed: %v", err)
	}
	config, err := providers.LoadProvidersConfig(filepath.Join(tmpDir, "providers.yaml"))
	if err != nil {
		t.Fatalf("LoadProvidersConfig failed: %v", err)
	}
	checks := map[string]string{
		"claude": "claude",
		"gemini": "gemini",
		"codex":  "codex",
	}
	paths := map[string]string{
		"claude": ".mcp.json",
		"gemini": ".gemini/settings.json",
		"codex":  ".codex/config.toml",
	}
	for name, binary := range checks {
		provider := config.GetProvider(name)
		if provider == nil {
			t.Fatalf("provider %s is missing", name)
		}
		if provider.Binary != binary || !provider.SupportsFeature("mcp") {
			t.Errorf("%s identity = binary %q, mcp %t", name, provider.Binary, provider.SupportsFeature("mcp"))
		}
		if got := provider.Features["mcp"].Path; got != paths[name] {
			t.Errorf("%s mcp path = %q", name, got)
		}
	}
}
