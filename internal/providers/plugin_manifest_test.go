package providers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAdapterManifest_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "provider-adapter.yaml")
	content := `name: testprovider
description: Test provider
version: "1.0.0"
artifacts:
  entrypoint:
    enabled: true
    path: "context.md"
    template: "{{ .ProjectContext }}"
  rules:
    enabled: false
    path: ""
    template: ""
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	manifest, err := LoadAdapterManifest(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if manifest.Name != "testprovider" {
		t.Errorf("expected name testprovider, got %s", manifest.Name)
	}
	if manifest.Description != "Test provider" {
		t.Errorf("expected description 'Test provider', got %s", manifest.Description)
	}
	if manifest.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", manifest.Version)
	}
	if len(manifest.Artifacts) != 2 {
		t.Errorf("expected 2 artifacts, got %d", len(manifest.Artifacts))
	}

	entrypoint, ok := manifest.Artifacts["entrypoint"]
	if !ok {
		t.Fatalf("expected entrypoint artifact")
	}
	if !entrypoint.Enabled {
		t.Errorf("expected entrypoint enabled")
	}
	if entrypoint.Path != "context.md" {
		t.Errorf("expected path context.md, got %s", entrypoint.Path)
	}
	if entrypoint.Template != "{{ .ProjectContext }}" {
		t.Errorf("unexpected template: %s", entrypoint.Template)
	}
}

func TestLoadAdapterManifest_MissingName(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "provider-adapter.yaml")
	content := `description: Missing name
version: "1.0.0"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	_, err := LoadAdapterManifest(path)
	if err == nil {
		t.Fatal("expected error for missing name")
	}
	if !strings.Contains(err.Error(), "missing required field: name") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestLoadAdapterManifest_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "provider-adapter.yaml")
	if err := os.WriteFile(path, []byte("not: valid: yaml: ["), 0o644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	_, err := LoadAdapterManifest(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
	if !strings.Contains(err.Error(), "failed to parse adapter manifest") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestLoadAdapterManifest_MissingFile(t *testing.T) {
	_, err := LoadAdapterManifest(filepath.Join(t.TempDir(), "missing.yaml"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !strings.Contains(err.Error(), "failed to read adapter manifest") {
		t.Errorf("unexpected error message: %v", err)
	}
}
