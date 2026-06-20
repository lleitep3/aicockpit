package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lleite/aicockpit/internal/version"
	"gopkg.in/yaml.v3"
)

// Config represents the AICockpit configuration.
type Config struct {
	Version    string   `yaml:"version"`
	Language   string   `yaml:"language"`
	LogLevel   string   `yaml:"log_level"`
	AIProvider string   `yaml:"ai_provider"`
	KB         KBConfig `yaml:"kb"`
}

// KBConfig represents the Knowledge Base configuration.
type KBConfig struct {
	Roots []string `yaml:"roots"`
}

var defaultConfig = Config{
	Version:    version.Version,
	Language:   "en-us",
	LogLevel:   "info",
	AIProvider: "claude",
	KB: KBConfig{
		Roots: []string{filepath.Join(GetCockpitDir(), "kb")},
	},
}

// GetCockpitDir returns the AICockpit home directory.
func GetCockpitDir() string {
	return filepath.Join(os.ExpandEnv("$HOME"), ".cockpit")
}

// GetConfigPath returns the path to the config file.
func GetConfigPath() string {
	return filepath.Join(GetCockpitDir(), "config.yaml")
}

// Load loads the configuration from disk or creates default if not exists.
func Load() (*Config, error) {
	configPath := GetConfigPath()

	// If config doesn't exist, create default
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return createDefault()
	}

	// Read existing config
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Set defaults for missing values
	if cfg.Version == "" {
		cfg.Version = defaultConfig.Version
	}
	if cfg.Language == "" {
		cfg.Language = defaultConfig.Language
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = defaultConfig.LogLevel
	}
	if cfg.AIProvider == "" {
		cfg.AIProvider = defaultConfig.AIProvider
	}

	return &cfg, nil
}

// createDefault creates the default configuration and directory structure.
func createDefault() (*Config, error) {
	cockpitDir := GetCockpitDir()

	// Create cockpit directory structure
	dirs := []string{
		cockpitDir,
		filepath.Join(cockpitDir, "logs"),
		filepath.Join(cockpitDir, "cache"),
		filepath.Join(cockpitDir, "packages"),
		filepath.Join(cockpitDir, "vault"),
		filepath.Join(cockpitDir, "agents"),
		filepath.Join(cockpitDir, "skills"),
		filepath.Join(cockpitDir, "rules"),
		filepath.Join(cockpitDir, "hooks"),
		filepath.Join(cockpitDir, "kb"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	cfg := &Config{
		Version:    defaultConfig.Version,
		Language:   defaultConfig.Language,
		LogLevel:   defaultConfig.LogLevel,
		AIProvider: defaultConfig.AIProvider,
	}

	// Save config
	if err := Save(cfg); err != nil {
		return nil, fmt.Errorf("failed to save config: %w", err)
	}

	return cfg, nil
}

// Save saves the configuration to disk.
func Save(cfg *Config) error {
	configPath := GetConfigPath()

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// Update updates specific configuration values.
func (c *Config) Update(updates map[string]interface{}) error {
	for key, value := range updates {
		switch key {
		case "language":
			if v, ok := value.(string); ok {
				c.Language = v
			}
		case "log_level":
			if v, ok := value.(string); ok {
				c.LogLevel = v
			}
		case "ai_provider":
			if v, ok := value.(string); ok {
				c.AIProvider = v
			}
		}
	}

	return Save(c)
}

// Save saves the configuration to disk.
func (c *Config) Save() error {
	return Save(c)
}
