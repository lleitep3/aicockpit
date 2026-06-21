package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lleite/aicockpit/internal/packages"
	"github.com/lleite/aicockpit/internal/version"
	"gopkg.in/yaml.v3"
)

// Config represents the AICockpit configuration.
type Config struct {
	Version           string                    `yaml:"version"`
	Language          string                    `yaml:"language"`
	LogLevel          string                    `yaml:"log_level"`
	AIProvider        string                    `yaml:"ai_provider"`
	AIProviders       ProvidersConfig           `yaml:"ai_providers"`
	KB                KBConfig                  `yaml:"kb"`
	PackageRegistries []packages.RegistryConfig `yaml:"package_registries"`
}

// ProvidersConfig represents configuration for multiple AI providers.
type ProvidersConfig struct {
	Enabled       []string        `yaml:"enabled"`
	Devin         *ProviderConfig `yaml:"devin"`
	Goose         *ProviderConfig `yaml:"goose"`
	ClaudeCode    *ProviderConfig `yaml:"claude_code"`
	GitHubCopilot *ProviderConfig `yaml:"github_copilot"`
}

// ProviderConfig represents configuration for a single AI provider.
type ProviderConfig struct {
	Enabled bool     `yaml:"enabled"`
	Path    string   `yaml:"path"`
	KB      KBConfig `yaml:"kb"`
}

// KBConfig represents the Knowledge Base configuration.
type KBConfig struct {
	Roots []string `yaml:"roots"`
}

var defaultConfig = Config{
	Version:    version.Version,
	Language:   "en-us",
	LogLevel:   "info",
	AIProvider: "antigravity",
	AIProviders: ProvidersConfig{
		Enabled: []string{"antigravity", "devin", "goose"},
	},
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
	if len(cfg.AIProviders.Enabled) == 0 {
		cfg.AIProviders.Enabled = defaultConfig.AIProviders.Enabled
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
		Version:     defaultConfig.Version,
		Language:    defaultConfig.Language,
		LogLevel:    defaultConfig.LogLevel,
		AIProvider:  defaultConfig.AIProvider,
		AIProviders: defaultConfig.AIProviders,
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

// EnableProvider enables an AI provider.
func (c *Config) EnableProvider(provider string) error {
	if c.AIProviders.Enabled == nil {
		c.AIProviders.Enabled = []string{}
	}

	// Check if already enabled
	for _, p := range c.AIProviders.Enabled {
		if p == provider {
			return nil // Already enabled
		}
	}

	c.AIProviders.Enabled = append(c.AIProviders.Enabled, provider)
	return Save(c)
}

// DisableProvider disables an AI provider.
func (c *Config) DisableProvider(provider string) error {
	if c.AIProviders.Enabled == nil {
		return nil
	}

	var newEnabled []string
	for _, p := range c.AIProviders.Enabled {
		if p != provider {
			newEnabled = append(newEnabled, p)
		}
	}

	c.AIProviders.Enabled = newEnabled
	return Save(c)
}

// IsProviderEnabled checks if a provider is enabled.
func (c *Config) IsProviderEnabled(provider string) bool {
	if c.AIProviders.Enabled == nil {
		return false
	}

	for _, p := range c.AIProviders.Enabled {
		if p == provider {
			return true
		}
	}

	return false
}

// GetEnabledProviders returns all enabled providers.
func (c *Config) GetEnabledProviders() []string {
	if c.AIProviders.Enabled == nil {
		return []string{}
	}
	return c.AIProviders.Enabled
}

// SetProviderPath sets the path for a provider.
func (c *Config) SetProviderPath(provider, path string) error {
	if c.AIProviders.Devin == nil && provider == "devin" {
		c.AIProviders.Devin = &ProviderConfig{}
	}
	if c.AIProviders.Goose == nil && provider == "goose" {
		c.AIProviders.Goose = &ProviderConfig{}
	}
	if c.AIProviders.ClaudeCode == nil && provider == "claude-code" {
		c.AIProviders.ClaudeCode = &ProviderConfig{}
	}
	if c.AIProviders.GitHubCopilot == nil && provider == "github-copilot" {
		c.AIProviders.GitHubCopilot = &ProviderConfig{}
	}

	switch provider {
	case "devin":
		if c.AIProviders.Devin != nil {
			c.AIProviders.Devin.Path = path
		}
	case "goose":
		if c.AIProviders.Goose != nil {
			c.AIProviders.Goose.Path = path
		}
	case "claude-code":
		if c.AIProviders.ClaudeCode != nil {
			c.AIProviders.ClaudeCode.Path = path
		}
	case "github-copilot":
		if c.AIProviders.GitHubCopilot != nil {
			c.AIProviders.GitHubCopilot.Path = path
		}
	default:
		return fmt.Errorf("unknown provider: %s", provider)
	}

	return Save(c)
}

// GetProviderPath gets the path for a provider.
func (c *Config) GetProviderPath(provider string) string {
	switch provider {
	case "devin":
		if c.AIProviders.Devin != nil {
			return c.AIProviders.Devin.Path
		}
		return filepath.Join(os.Getenv("HOME"), ".cockpit")
	case "goose":
		if c.AIProviders.Goose != nil {
			return c.AIProviders.Goose.Path
		}
		return filepath.Join(os.Getenv("HOME"), ".goose")
	case "claude-code":
		if c.AIProviders.ClaudeCode != nil {
			return c.AIProviders.ClaudeCode.Path
		}
		return filepath.Join(os.Getenv("HOME"), ".claude-code")
	case "github-copilot":
		if c.AIProviders.GitHubCopilot != nil {
			return c.AIProviders.GitHubCopilot.Path
		}
		return filepath.Join(os.Getenv("HOME"), ".github-copilot")
	default:
		return ""
	}
}
