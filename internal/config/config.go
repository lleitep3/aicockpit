package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/lleitep3/aicockpit/internal/packages"
	"github.com/lleitep3/aicockpit/internal/version"
	"gopkg.in/yaml.v3"
)

// Config represents the AICockpit configuration.
type Config struct {
	Version           string                    `yaml:"version"`
	Language          string                    `yaml:"language"`
	LogLevel          string                    `yaml:"log_level"`
	EnabledProviders  []string                  `yaml:"enabled_providers"`
	KB                KBConfig                  `yaml:"kb"`
	PackageRegistries []packages.RegistryConfig `yaml:"package_registries"`
	LastUpdateCheck   string                    `yaml:"last_update_check"`
	AutoUpdateCheck   bool                      `yaml:"auto_update_check"`
}

// KBConfig represents the Knowledge Base configuration.
type KBConfig struct {
	Roots []string `yaml:"roots"`
}

var defaultConfig = Config{
	Version:          version.Version,
	Language:         "en-us",
	LogLevel:         "info",
	AutoUpdateCheck:  true,
	EnabledProviders: []string{"antigravity", "devin", "goose"},
	KB: KBConfig{
		Roots: []string{filepath.Join(GetCockpitDir(), "kb")},
	},
	PackageRegistries: []packages.RegistryConfig{
		{
			Name:     "official",
			URL:      "https://github.com/lleitep3/cockpit-registry",
			Branch:   "main",
			Enabled:  true,
			Priority: 1,
		},
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

	// Try to parse with a legacy-aware wrapper to migrate old fields
	var raw legacyConfig
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	cfg := raw.toConfig()

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
	if len(cfg.EnabledProviders) == 0 {
		cfg.EnabledProviders = defaultConfig.EnabledProviders
	}
	if !cfg.AutoUpdateCheck {
		cfg.AutoUpdateCheck = defaultConfig.AutoUpdateCheck
	}

	return &cfg, nil
}

// legacyConfig is used only during Load to transparently migrate old config fields.
type legacyConfig struct {
	Version           string                    `yaml:"version"`
	Language          string                    `yaml:"language"`
	LogLevel          string                    `yaml:"log_level"`
	EnabledProviders  []string                  `yaml:"enabled_providers"`
	KB                KBConfig                  `yaml:"kb"`
	PackageRegistries []packages.RegistryConfig `yaml:"package_registries"`
	LastUpdateCheck   string                    `yaml:"last_update_check"`
	AutoUpdateCheck   bool                      `yaml:"auto_update_check"`
	// Legacy fields — read-only, never written back
	AIProvider  string `yaml:"ai_provider"`
	AIProviders struct {
		Enabled []string `yaml:"enabled"`
	} `yaml:"ai_providers"`
}

func (l *legacyConfig) toConfig() Config {
	cfg := Config{
		Version:           l.Version,
		Language:          l.Language,
		LogLevel:          l.LogLevel,
		EnabledProviders:  l.EnabledProviders,
		KB:                l.KB,
		PackageRegistries: l.PackageRegistries,
		LastUpdateCheck:   l.LastUpdateCheck,
		AutoUpdateCheck:   l.AutoUpdateCheck,
	}

	// Migrate from ai_providers.enabled → enabled_providers
	if len(cfg.EnabledProviders) == 0 && len(l.AIProviders.Enabled) > 0 {
		cfg.EnabledProviders = l.AIProviders.Enabled
	}

	// Migrate from legacy singular ai_provider → enabled_providers
	if len(cfg.EnabledProviders) == 0 && l.AIProvider != "" {
		cfg.EnabledProviders = []string{l.AIProvider}
	}

	return cfg
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
		Version:          defaultConfig.Version,
		Language:         defaultConfig.Language,
		LogLevel:         defaultConfig.LogLevel,
		EnabledProviders: defaultConfig.EnabledProviders,
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
		}
	}

	return Save(c)
}

// Save saves the configuration to disk.
func (c *Config) Save() error {
	return Save(c)
}

// GetEnabledProviders returns the list of enabled provider names.
// Provider capabilities and workspace paths live in providers.yaml, not here.
func (c *Config) GetEnabledProviders() []string {
	if c.EnabledProviders == nil {
		return []string{}
	}
	return c.EnabledProviders
}

// SetLastUpdateCheck sets the last update check timestamp.
func (c *Config) SetLastUpdateCheck(timestamp string) error {
	c.LastUpdateCheck = timestamp
	return Save(c)
}

// ShouldCheckUpdate returns true if an update check should be performed based on the last check time.
func (c *Config) ShouldCheckUpdate() bool {
	if !c.AutoUpdateCheck {
		return false
	}

	if c.LastUpdateCheck == "" {
		return true
	}

	lastCheck, err := time.Parse(time.RFC3339, c.LastUpdateCheck)
	if err != nil {
		return true // Invalid timestamp, check again
	}

	// Check if 24 hours have passed
	return time.Since(lastCheck) > 24*time.Hour
}
