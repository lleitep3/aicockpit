package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	// Set temporary home directory for testing
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Version == "" {
		t.Error("Version should not be empty")
	}

	if cfg.Language == "" {
		t.Error("Language should not be empty")
	}

	if cfg.AIProvider == "" {
		t.Error("AIProvider should not be empty")
	}
}

func TestGetCockpitDir(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	dir := GetCockpitDir()
	expected := filepath.Join(tmpDir, ".cockpit")

	if dir != expected {
		t.Errorf("Expected %s, got %s", expected, dir)
	}
}

func TestGetConfigPath(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	path := GetConfigPath()
	expected := filepath.Join(tmpDir, ".cockpit", "config.yaml")

	if path != expected {
		t.Errorf("Expected %s, got %s", expected, path)
	}
}

func TestSave(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create the .cockpit directory first
	cockpitDir := GetCockpitDir()
	if err := os.MkdirAll(cockpitDir, 0o755); err != nil {
		t.Fatalf("Failed to create cockpit directory: %v", err)
	}

	cfg := &Config{
		Version:    "0.1.0",
		Language:   "en-us",
		LogLevel:   "info",
		AIProvider: "claude",
	}

	err := Save(cfg)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file was created
	configPath := GetConfigPath()
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}
}

func TestUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create the .cockpit directory first
	cockpitDir := GetCockpitDir()
	if err := os.MkdirAll(cockpitDir, 0o755); err != nil {
		t.Fatalf("Failed to create cockpit directory: %v", err)
	}

	cfg := &Config{
		Version:    "0.1.0",
		Language:   "en-us",
		LogLevel:   "info",
		AIProvider: "claude",
	}

	updates := map[string]interface{}{
		"language":    "pt-br",
		"log_level":   "debug",
		"ai_provider": "openai",
	}

	err := cfg.Update(updates)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if cfg.Language != "pt-br" {
		t.Errorf("Expected language pt-br, got %s", cfg.Language)
	}

	if cfg.LogLevel != "debug" {
		t.Errorf("Expected log_level debug, got %s", cfg.LogLevel)
	}

	if cfg.AIProvider != "openai" {
		t.Errorf("Expected ai_provider openai, got %s", cfg.AIProvider)
	}
}

func TestEnableProvider(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create the .cockpit directory first
	cockpitDir := GetCockpitDir()
	if err := os.MkdirAll(cockpitDir, 0o755); err != nil {
		t.Fatalf("Failed to create cockpit directory: %v", err)
	}

	cfg := &Config{
		Version:    "0.2.0",
		Language:   "en-us",
		LogLevel:   "info",
		AIProvider: "devin",
	}

	// Enable providers
	if err := cfg.EnableProvider("devin"); err != nil {
		t.Fatalf("EnableProvider failed: %v", err)
	}

	if err := cfg.EnableProvider("goose"); err != nil {
		t.Fatalf("EnableProvider failed: %v", err)
	}

	if err := cfg.EnableProvider("claude-code"); err != nil {
		t.Fatalf("EnableProvider failed: %v", err)
	}

	// Verify providers are enabled
	if !cfg.IsProviderEnabled("devin") {
		t.Error("devin should be enabled")
	}

	if !cfg.IsProviderEnabled("goose") {
		t.Error("goose should be enabled")
	}

	if !cfg.IsProviderEnabled("claude-code") {
		t.Error("claude-code should be enabled")
	}

	// Verify enabled providers list
	enabled := cfg.GetEnabledProviders()
	if len(enabled) != 3 {
		t.Errorf("Expected 3 enabled providers, got %d", len(enabled))
	}
}

func TestDisableProvider(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create the .cockpit directory first
	cockpitDir := GetCockpitDir()
	if err := os.MkdirAll(cockpitDir, 0o755); err != nil {
		t.Fatalf("Failed to create cockpit directory: %v", err)
	}

	cfg := &Config{
		Version:    "0.2.0",
		Language:   "en-us",
		LogLevel:   "info",
		AIProvider: "devin",
	}

	// Enable providers
	cfg.EnableProvider("devin")
	cfg.EnableProvider("goose")
	cfg.EnableProvider("claude-code")

	// Disable one provider
	if err := cfg.DisableProvider("goose"); err != nil {
		t.Fatalf("DisableProvider failed: %v", err)
	}

	// Verify provider is disabled
	if cfg.IsProviderEnabled("goose") {
		t.Error("goose should be disabled")
	}

	// Verify other providers are still enabled
	if !cfg.IsProviderEnabled("devin") {
		t.Error("devin should still be enabled")
	}

	if !cfg.IsProviderEnabled("claude-code") {
		t.Error("claude-code should still be enabled")
	}

	// Verify enabled providers list
	enabled := cfg.GetEnabledProviders()
	if len(enabled) != 2 {
		t.Errorf("Expected 2 enabled providers, got %d", len(enabled))
	}
}

func TestSetProviderPath(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create the .cockpit directory first
	cockpitDir := GetCockpitDir()
	if err := os.MkdirAll(cockpitDir, 0o755); err != nil {
		t.Fatalf("Failed to create cockpit directory: %v", err)
	}

	cfg := &Config{
		Version:    "0.2.0",
		Language:   "en-us",
		LogLevel:   "info",
		AIProvider: "devin",
	}

	// Set provider paths
	customPath := filepath.Join(tmpDir, "custom-devin")
	if err := cfg.SetProviderPath("devin", customPath); err != nil {
		t.Fatalf("SetProviderPath failed: %v", err)
	}

	// Verify path is set
	path := cfg.GetProviderPath("devin")
	if path != customPath {
		t.Errorf("Expected path %s, got %s", customPath, path)
	}
}

func TestGetProviderPathDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	cfg := &Config{
		Version:    "0.2.0",
		Language:   "en-us",
		LogLevel:   "info",
		AIProvider: "devin",
	}

	// Test default paths
	tests := []struct {
		provider string
		expected string
	}{
		{"devin", filepath.Join(tmpDir, ".cockpit")},
		{"goose", filepath.Join(tmpDir, ".goose")},
		{"claude-code", filepath.Join(tmpDir, ".claude-code")},
		{"github-copilot", filepath.Join(tmpDir, ".github-copilot")},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			path := cfg.GetProviderPath(tt.provider)
			if path != tt.expected {
				t.Errorf("Expected path %s, got %s", tt.expected, path)
			}
		})
	}
}

func TestMultiProviderConfiguration(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create the .cockpit directory first
	cockpitDir := GetCockpitDir()
	if err := os.MkdirAll(cockpitDir, 0o755); err != nil {
		t.Fatalf("Failed to create cockpit directory: %v", err)
	}

	cfg := &Config{
		Version:    "0.2.0",
		Language:   "en-us",
		LogLevel:   "info",
		AIProvider: "devin",
	}

	// Enable all providers
	providers := []string{"devin", "goose", "claude-code", "github-copilot"}
	for _, provider := range providers {
		if err := cfg.EnableProvider(provider); err != nil {
			t.Fatalf("EnableProvider failed for %s: %v", provider, err)
		}
	}

	// Verify all providers are enabled
	enabled := cfg.GetEnabledProviders()
	if len(enabled) != 4 {
		t.Errorf("Expected 4 enabled providers, got %d", len(enabled))
	}

	// Verify each provider is enabled
	for _, provider := range providers {
		if !cfg.IsProviderEnabled(provider) {
			t.Errorf("%s should be enabled", provider)
		}
	}

	// Set custom paths for all providers
	for _, provider := range providers {
		customPath := filepath.Join(tmpDir, "custom-"+provider)
		if err := cfg.SetProviderPath(provider, customPath); err != nil {
			t.Fatalf("SetProviderPath failed for %s: %v", provider, err)
		}
	}

	// Verify all paths are set correctly
	for _, provider := range providers {
		expectedPath := filepath.Join(tmpDir, "custom-"+provider)
		path := cfg.GetProviderPath(provider)
		if path != expectedPath {
			t.Errorf("Expected path %s for %s, got %s", expectedPath, provider, path)
		}
	}
}
