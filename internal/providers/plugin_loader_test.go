package providers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeManifest(t *testing.T, dir, name, content string) {
	t.Helper()
	providerDir := filepath.Join(dir, name)
	if err := os.MkdirAll(providerDir, 0o755); err != nil {
		t.Fatalf("failed to create provider dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(providerDir, "provider-adapter.yaml"), []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}
}

func validManifest(name string) string {
	return `name: ` + name + `
description: Test provider
version: "1.0.0"
artifacts:
  rules:
    enabled: true
    path: "rules.md"
    template: "{{ range . }}{{ .Name }}\n{{ .Content }}\n{{ end }}"
`
}

func TestPluginLoader_Discover_NonExistentDir(t *testing.T) {
	loader := NewPluginLoader(filepath.Join(t.TempDir(), "does-not-exist"))
	manifests, err := loader.Discover()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if manifests != nil {
		t.Errorf("expected nil manifests for nonexistent dir, got %v", manifests)
	}
}

func TestPluginLoader_Discover_ReadDirError(t *testing.T) {
	dir := t.TempDir()
	// A file at the pluginsDir path makes os.ReadDir fail with a non-NotExist error.
	filePath := filepath.Join(dir, "notadir")
	if err := os.WriteFile(filePath, []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	loader := NewPluginLoader(filePath)
	_, err := loader.Discover()
	if err == nil {
		t.Fatal("expected error when pluginsDir is not a directory")
	}
}

func TestPluginLoader_Discover_EmptyDir(t *testing.T) {
	loader := NewPluginLoader(t.TempDir())
	manifests, err := loader.Discover()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(manifests) != 0 {
		t.Errorf("expected empty manifests, got %d", len(manifests))
	}
}

func TestPluginLoader_Discover_SkipsNonDirectoryEntries(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "provider-adapter.yaml"), []byte("name: invalid"), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	loader := NewPluginLoader(dir)
	manifests, err := loader.Discover()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(manifests) != 0 {
		t.Errorf("expected no manifests, got %d", len(manifests))
	}
}

func TestPluginLoader_Discover_SkipsDirWithoutManifest(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "noplugin"), 0o755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	loader := NewPluginLoader(dir)
	manifests, err := loader.Discover()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(manifests) != 0 {
		t.Errorf("expected no manifests, got %d", len(manifests))
	}
}

func TestPluginLoader_Discover_LoadsValidManifest(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, "myplugin", validManifest("myplugin"))

	loader := NewPluginLoader(dir)
	manifests, err := loader.Discover()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(manifests) != 1 {
		t.Fatalf("expected 1 manifest, got %d", len(manifests))
	}
	if manifests[0].Name != "myplugin" {
		t.Errorf("expected name myplugin, got %s", manifests[0].Name)
	}
}

func TestPluginLoader_Discover_SkipsInvalidManifest(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, "validplugin", validManifest("validplugin"))
	writeManifest(t, dir, "invalidplugin", "not: valid: yaml: [")

	loader := NewPluginLoader(dir)
	manifests, err := loader.Discover()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(manifests) != 1 {
		t.Fatalf("expected 1 manifest, got %d", len(manifests))
	}
	if manifests[0].Name != "validplugin" {
		t.Errorf("expected validplugin, got %s", manifests[0].Name)
	}
}

func TestPluginLoader_DiscoverAndRegister_RegistersCompiler(t *testing.T) {
	dir := t.TempDir()
	pluginsDir := filepath.Join(dir, "providers")
	writeManifest(t, pluginsDir, "myplugin", validManifest("myplugin"))

	config := &ProvidersConfig{
		Providers: map[string]*Provider{
			"myplugin": {
				Name:      "myplugin",
				Enabled:   true,
				Workspace: ".",
				Features: map[string]*FeatureConfig{
					"rules": {Enabled: true, Path: "rules.md"},
				},
			},
		},
	}

	cockpitDir := t.TempDir()
	rulesDir := filepath.Join(cockpitDir, "rules")
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		t.Fatalf("failed to create rules dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rulesDir, "rule.md"), []byte("rule content"), 0o644); err != nil {
		t.Fatalf("failed to write rule file: %v", err)
	}

	pm := NewProviderManagerWithPlugins(config, dir)
	projectDir := t.TempDir()

	err := pm.Deploy("myplugin", cockpitDir, projectDir)
	if err != nil {
		t.Fatalf("deploy failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(projectDir, "rules.md"))
	if err != nil {
		t.Fatalf("expected compiled rules file: %v", err)
	}
	if !strings.Contains(string(content), "rule.md") || !strings.Contains(string(content), "rule content") {
		t.Errorf("unexpected rules content: %s", string(content))
	}
}

func TestPluginLoader_DiscoverAndRegister_DiscoverError(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "notadir")
	if err := os.WriteFile(filePath, []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	pm := NewProviderManager(&ProvidersConfig{})
	loader := NewPluginLoader(filePath)
	_, err := loader.DiscoverAndRegister(pm)
	if err == nil {
		t.Fatal("expected error from DiscoverAndRegister")
	}
}

func TestPluginLoader_DiscoverAndRegister_MultiplePlugins(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, "plugin1", validManifest("plugin1"))
	writeManifest(t, dir, "plugin2", validManifest("plugin2"))

	pm := NewProviderManager(&ProvidersConfig{})
	loader := NewPluginLoader(dir)
	registered, err := loader.DiscoverAndRegister(pm)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(registered) != 2 {
		t.Fatalf("expected 2 registered plugins, got %d", len(registered))
	}

	names := make(map[string]bool)
	for _, name := range registered {
		names[name] = true
	}
	if !names["plugin1"] || !names["plugin2"] {
		t.Errorf("expected plugin1 and plugin2 registered, got %v", registered)
	}
}
