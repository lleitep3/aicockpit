package providers

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ProvidersConfig represents the providers configuration from YAML.
type ProvidersConfig struct {
	Version     string                `yaml:"version"`
	Description string                `yaml:"description"`
	Providers   map[string]*Provider  `yaml:"providers"`
	Features    map[string]*Feature   `yaml:"features"`
}

// Provider represents a single AI provider configuration.
type Provider struct {
	Enabled     bool                  `yaml:"enabled"`
	Name        string                `yaml:"name"`
	Description string                `yaml:"description"`
	Workspace   string                `yaml:"workspace"`
	Version     string                `yaml:"version"`
	Features    map[string]*FeatureConfig `yaml:"features"`
}

// FeatureConfig represents a feature configuration for a provider.
type FeatureConfig struct {
	Enabled     bool   `yaml:"enabled"`
	Path        string `yaml:"path"`
	Description string `yaml:"description"`
}

// Feature represents a feature definition.
type Feature struct {
	Description string `yaml:"description"`
	Example     string `yaml:"example"`
}

// LoadProvidersConfig loads the providers configuration from YAML file.
func LoadProvidersConfig(configPath string) (*ProvidersConfig, error) {
	// Expand home directory
	expandedPath := os.ExpandEnv(configPath)
	if expandedPath[0] == '~' {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		expandedPath = filepath.Join(homeDir, expandedPath[1:])
	}

	// Read file
	data, err := os.ReadFile(expandedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read providers config: %w", err)
	}

	// Parse YAML
	var config ProvidersConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse providers config: %w", err)
	}

	return &config, nil
}

// SaveProvidersConfig saves the providers configuration to YAML file.
func SaveProvidersConfig(configPath string, config *ProvidersConfig) error {
	// Expand home directory
	expandedPath := os.ExpandEnv(configPath)
	if expandedPath[0] == '~' {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		expandedPath = filepath.Join(homeDir, expandedPath[1:])
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(expandedPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal providers config: %w", err)
	}

	// Write file
	if err := os.WriteFile(expandedPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write providers config: %w", err)
	}

	return nil
}

// GetProvider returns a provider by name.
func (c *ProvidersConfig) GetProvider(name string) *Provider {
	if c.Providers == nil {
		return nil
	}
	return c.Providers[name]
}

// GetEnabledProviders returns all enabled providers.
func (c *ProvidersConfig) GetEnabledProviders() []*Provider {
	var enabled []*Provider
	if c.Providers == nil {
		return enabled
	}

	for _, provider := range c.Providers {
		if provider.Enabled {
			enabled = append(enabled, provider)
		}
	}

	return enabled
}

// GetProviderNames returns all provider names.
func (c *ProvidersConfig) GetProviderNames() []string {
	var names []string
	if c.Providers == nil {
		return names
	}

	for name := range c.Providers {
		names = append(names, name)
	}

	return names
}

// EnableProvider enables a provider.
func (c *ProvidersConfig) EnableProvider(name string) error {
	provider := c.GetProvider(name)
	if provider == nil {
		return fmt.Errorf("provider not found: %s", name)
	}

	provider.Enabled = true
	return nil
}

// DisableProvider disables a provider.
func (c *ProvidersConfig) DisableProvider(name string) error {
	provider := c.GetProvider(name)
	if provider == nil {
		return fmt.Errorf("provider not found: %s", name)
	}

	provider.Enabled = false
	return nil
}

// IsProviderEnabled checks if a provider is enabled.
func (c *ProvidersConfig) IsProviderEnabled(name string) bool {
	provider := c.GetProvider(name)
	if provider == nil {
		return false
	}
	return provider.Enabled
}

// GetSupportedFeatures returns all features supported by a provider.
func (p *Provider) GetSupportedFeatures() []string {
	var features []string
	if p.Features == nil {
		return features
	}

	for name, feature := range p.Features {
		if feature.Enabled {
			features = append(features, name)
		}
	}

	return features
}

// SupportsFeature checks if a provider supports a feature.
func (p *Provider) SupportsFeature(feature string) bool {
	if p.Features == nil {
		return false
	}

	featureConfig, exists := p.Features[feature]
	if !exists {
		return false
	}

	return featureConfig.Enabled
}

// GetFeaturePath returns the path for a feature in a provider.
func (p *Provider) GetFeaturePath(feature string) string {
	if p.Features == nil {
		return ""
	}

	featureConfig, exists := p.Features[feature]
	if !exists {
		return ""
	}

	// Expand workspace path
	expandedWorkspace := os.ExpandEnv(p.Workspace)
	if expandedWorkspace[0] == '~' {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		expandedWorkspace = filepath.Join(homeDir, expandedWorkspace[1:])
	}

	return filepath.Join(expandedWorkspace, featureConfig.Path)
}

// GetWorkspacePath returns the expanded workspace path for a provider.
func (p *Provider) GetWorkspacePath() string {
	expandedWorkspace := os.ExpandEnv(p.Workspace)
	if expandedWorkspace[0] == '~' {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return p.Workspace
		}
		expandedWorkspace = filepath.Join(homeDir, expandedWorkspace[1:])
	}
	return expandedWorkspace
}

// AddProvider adds a new provider to the configuration.
func (c *ProvidersConfig) AddProvider(name string, provider *Provider) error {
	if c.Providers == nil {
		c.Providers = make(map[string]*Provider)
	}

	if _, exists := c.Providers[name]; exists {
		return fmt.Errorf("provider already exists: %s", name)
	}

	c.Providers[name] = provider
	return nil
}

// RemoveProvider removes a provider from the configuration.
func (c *ProvidersConfig) RemoveProvider(name string) error {
	if c.Providers == nil {
		return fmt.Errorf("no providers configured")
	}

	if _, exists := c.Providers[name]; !exists {
		return fmt.Errorf("provider not found: %s", name)
	}

	delete(c.Providers, name)
	return nil
}

// ValidateProvider validates a provider configuration.
func (p *Provider) Validate() error {
	if p.Name == "" {
		return fmt.Errorf("provider name is required")
	}

	if p.Workspace == "" {
		return fmt.Errorf("provider workspace is required")
	}

	if p.Features == nil || len(p.Features) == 0 {
		return fmt.Errorf("provider must have at least one feature")
	}

	return nil
}

// ValidateConfig validates the entire providers configuration.
func (c *ProvidersConfig) ValidateConfig() error {
	if c.Version == "" {
		return fmt.Errorf("config version is required")
	}

	if c.Providers == nil || len(c.Providers) == 0 {
		return fmt.Errorf("at least one provider must be configured")
	}

	for name, provider := range c.Providers {
		if err := provider.Validate(); err != nil {
			return fmt.Errorf("provider %s validation failed: %w", name, err)
		}
	}

	return nil
}
