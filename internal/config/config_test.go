package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
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
	if len(cfg.EnabledProviders) == 0 {
		t.Error("EnabledProviders should not be empty after Load")
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

	cockpitDir := GetCockpitDir()
	if err := os.MkdirAll(cockpitDir, 0o755); err != nil {
		t.Fatalf("Failed to create cockpit directory: %v", err)
	}

	cfg := &Config{
		Version:          "0.1.0",
		Language:         "en-us",
		LogLevel:         "info",
		EnabledProviders: []string{"devin"},
	}

	err := Save(cfg)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

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

	cockpitDir := GetCockpitDir()
	if err := os.MkdirAll(cockpitDir, 0o755); err != nil {
		t.Fatalf("Failed to create cockpit directory: %v", err)
	}

	cfg := &Config{
		Version:          "0.1.0",
		Language:         "en-us",
		LogLevel:         "info",
		EnabledProviders: []string{"devin"},
	}

	updates := map[string]interface{}{
		"language":  "pt-br",
		"log_level": "debug",
	}

	if err := cfg.Update(updates); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if cfg.Language != "pt-br" {
		t.Errorf("Expected language pt-br, got %s", cfg.Language)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("Expected log_level debug, got %s", cfg.LogLevel)
	}
}

func TestGetEnabledProviders(t *testing.T) {
	tests := []struct {
		name     string
		cfg      Config
		expected []string
	}{
		{
			name:     "nil slice returns empty",
			cfg:      Config{EnabledProviders: nil},
			expected: []string{},
		},
		{
			name:     "returns configured providers",
			cfg:      Config{EnabledProviders: []string{"devin", "goose"}},
			expected: []string{"devin", "goose"},
		},
		{
			name:     "single provider",
			cfg:      Config{EnabledProviders: []string{"antigravity"}},
			expected: []string{"antigravity"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.GetEnabledProviders()
			if len(got) != len(tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, got)
				return
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("Expected %v, got %v", tt.expected, got)
				}
			}
		})
	}
}

func TestLegacyMigration_AIProvidersEnabled(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	cockpitDir := GetCockpitDir()
	if err := os.MkdirAll(cockpitDir, 0o755); err != nil {
		t.Fatalf("Failed to create cockpit directory: %v", err)
	}

	// Write a legacy config using old ai_providers.enabled format
	legacyYAML := `version: "0.3.0"
language: en-us
log_level: info
ai_provider: devin
ai_providers:
  enabled:
    - devin
    - goose
`
	if err := os.WriteFile(GetConfigPath(), []byte(legacyYAML), 0o644); err != nil {
		t.Fatalf("Failed to write legacy config: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Should have migrated ai_providers.enabled → EnabledProviders
	if len(cfg.EnabledProviders) != 2 {
		t.Errorf("Expected 2 providers after migration, got %d: %v", len(cfg.EnabledProviders), cfg.EnabledProviders)
	}
}

func TestLegacyMigration_SingularAIProvider(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	cockpitDir := GetCockpitDir()
	if err := os.MkdirAll(cockpitDir, 0o755); err != nil {
		t.Fatalf("Failed to create cockpit directory: %v", err)
	}

	// Write a config with only the old singular ai_provider field
	legacyYAML := `version: "0.2.0"
language: en-us
log_level: info
ai_provider: goose
`
	if err := os.WriteFile(GetConfigPath(), []byte(legacyYAML), 0o644); err != nil {
		t.Fatalf("Failed to write legacy config: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Should have migrated ai_provider → EnabledProviders
	if len(cfg.EnabledProviders) != 1 || cfg.EnabledProviders[0] != "goose" {
		t.Errorf("Expected [goose] after migration, got %v", cfg.EnabledProviders)
	}
}

func TestNewConfigFormat(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	cockpitDir := GetCockpitDir()
	if err := os.MkdirAll(cockpitDir, 0o755); err != nil {
		t.Fatalf("Failed to create cockpit directory: %v", err)
	}

	// Write a config in the new format
	newYAML := `version: "0.5.0"
language: pt-br
log_level: debug
enabled_providers:
  - devin
  - antigravity
  - goose
`
	if err := os.WriteFile(GetConfigPath(), []byte(newYAML), 0o644); err != nil {
		t.Fatalf("Failed to write new config: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(cfg.EnabledProviders) != 3 {
		t.Errorf("Expected 3 providers, got %d: %v", len(cfg.EnabledProviders), cfg.EnabledProviders)
	}
	if cfg.Language != "pt-br" {
		t.Errorf("Expected pt-br, got %s", cfg.Language)
	}
}

func TestLoadPreservesHindsightSettings(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	if err := os.MkdirAll(GetCockpitDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	raw := []byte("version: \"0.4.51\"\nlanguage: en-us\nmemory:\n  hindsight:\n    enabled: true\n    llm_provider: groq\n    api_port: 8888\n    ui_port: 9999\n")
	if err := os.WriteFile(GetConfigPath(), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Memory.Hindsight.Enabled || cfg.Memory.Hindsight.LLMProvider != "groq" || cfg.Memory.Hindsight.APIPort != 8888 {
		t.Fatalf("memory = %+v", cfg.Memory.Hindsight)
	}
	if strings.Contains(string(raw), "sk-") {
		t.Fatal("fixture contained a secret")
	}
	if err := Save(cfg); err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(GetConfigPath())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(saved), "enabled: true") || strings.Contains(string(saved), "api_key") {
		t.Fatalf("saved = %s", saved)
	}
}

func TestLoadPreservesHindsightBanks(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)
	if err := os.MkdirAll(GetCockpitDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	raw := []byte("version: \"0.4.52\"\nlanguage: en-us\nmemory:\n  hindsight:\n    enabled: true\n    api_url: http://127.0.0.1:8888\n    api_token_name: api_token\n    banks:\n      project: crew\n      user: ada\n")
	if err := os.WriteFile(GetConfigPath(), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	got := cfg.Memory.Hindsight
	if got.APIURL != "http://127.0.0.1:8888" || got.APITokenName != "api_token" || got.Banks.Project != "crew" || got.Banks.User != "ada" {
		t.Fatalf("hindsight = %+v", got)
	}
	if err := Save(cfg); err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(GetConfigPath())
	if err != nil {
		t.Fatal(err)
	}
	text := string(saved)
	if !strings.Contains(text, "project: crew") || !strings.Contains(text, "api_token_name: api_token") || strings.Contains(text, "super-secret") {
		t.Fatalf("saved = %s", text)
	}
}

func TestLoadPreservesAgentAPISettings(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)
	if err := os.MkdirAll(GetCockpitDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	const secret = "sk-live-secret-value"
	raw := []byte("version: \"0.4.55\"\nlanguage: en-us\nagents:\n  backend: api:openrouter\n  api:\n    provider: openrouter\n    model: openai/gpt-test\n    base_url: https://openrouter.ai/api\n    vault_namespace: agents\n    vault_key: openrouter_api_key\n    max_tokens: 128\n")
	if strings.Contains(string(raw), secret) {
		t.Fatal("fixture contained a secret")
	}
	if err := os.WriteFile(GetConfigPath(), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Agents.Backend != "api:openrouter" {
		t.Fatalf("backend = %q", cfg.Agents.Backend)
	}
	got := cfg.Agents.API
	if got.Provider != "openrouter" || got.Model != "openai/gpt-test" || got.BaseURL != "https://openrouter.ai/api" || got.VaultNamespace != "agents" || got.VaultKey != "openrouter_api_key" || got.MaxTokens != 128 {
		t.Fatalf("api = %+v", got)
	}
	if err := Save(cfg); err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(GetConfigPath())
	if err != nil {
		t.Fatal(err)
	}
	text := string(saved)
	if !strings.Contains(text, "vault_key: openrouter_api_key") || strings.Contains(text, secret) {
		t.Fatalf("saved = %s", text)
	}
}

func TestShouldCheckUpdate(t *testing.T) {
	tests := []struct {
		name            string
		autoUpdate      bool
		lastCheck       string
		expectShouldRun bool
	}{
		{"disabled auto update", false, "", false},
		{"no last check", true, "", true},
		{"invalid timestamp", true, "not-a-date", true},
		{"recent check", true, "2099-01-01T00:00:00Z", false},
		{"old check", true, "2000-01-01T00:00:00Z", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				AutoUpdateCheck: tt.autoUpdate,
				LastUpdateCheck: tt.lastCheck,
			}
			if got := cfg.ShouldCheckUpdate(); got != tt.expectShouldRun {
				t.Errorf("ShouldCheckUpdate() = %v, want %v", got, tt.expectShouldRun)
			}
		})
	}
}
