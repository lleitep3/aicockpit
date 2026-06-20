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
