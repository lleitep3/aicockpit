package providers

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)


func TestGoosePermissions(t *testing.T) {
	// Let's test the specific ensureGooseExtensionsEnabled function or Goose permissions logic
	compiler := NewGooseCompiler()

	// Create mock config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := "extensions:\n  developer: {enabled: false}\n"
	os.WriteFile(configPath, []byte(configContent), 0o644)

	provider := &Provider{
		Features: map[string]*FeatureConfig{
			"permissions": {Enabled: true, Path: configPath},
		},
	}

	// Assuming perms doesn't actually matter for goose compiler except for config parsing
	perms := &CanonicalPermissions{}

	files, err := compiler.CompilePermissions(perms, provider)
	if err != nil {
		t.Fatalf("CompilePermissions failed: %v", err)
	}

	// Goose adapter reads the actual file path and writes it, so the returned map might just have the modified content
	if content, ok := files[configPath]; ok {
		if !reflect.DeepEqual(strings.Contains(content, "enabled: true"), true) {
			t.Errorf("expected enabled: true, got %s", content)
		}
	}
}

func TestAntigravityPermissions(t *testing.T) {
	compiler := NewAntigravityCompiler()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	provider := &Provider{
		Features: map[string]*FeatureConfig{
			"permissions": {Enabled: true, Path: configPath},
		},
	}

	perms := &CanonicalPermissions{
		AllowedCommands: []string{"custom_cmd"},
		AllowedDirs:     []string{"/custom_dir"},
	}

	_, err := compiler.CompilePermissions(perms, provider)
	if err != nil {
		t.Fatalf("CompilePermissions failed: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read json: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "command(custom_cmd)") {
		t.Errorf("expected command(custom_cmd), got %s", content)
	}
	if !strings.Contains(content, "read_file(/custom_dir)") {
		t.Errorf("expected read_file(/custom_dir), got %s", content)
	}
}

func TestDevinPermissions(t *testing.T) {
	compiler := NewDevinCompiler()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	provider := &Provider{
		Features: map[string]*FeatureConfig{
			"permissions": {Enabled: true, Path: configPath},
		},
	}

	perms := &CanonicalPermissions{
		AllowedCommands: []string{"custom_cmd"},
		AllowedDirs:     []string{"/custom_dir"},
	}

	_, err := compiler.CompilePermissions(perms, provider)
	if err != nil {
		t.Fatalf("CompilePermissions failed: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read yaml: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "custom_cmd") {
		t.Errorf("expected custom_cmd, got %s", content)
	}
}
