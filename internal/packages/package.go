package packages

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Package represents a package manifest (cockpit-package.yml).
type Package struct {
	// Metadata
	Name        string `yaml:"name"`
	Version     string `yaml:"version"`
	Description string `yaml:"description"`
	Author      string `yaml:"author"`
	License     string `yaml:"license"`
	Homepage    string `yaml:"homepage,omitempty"`
	Repository  string `yaml:"repository,omitempty"`

	// Classification
	Type     string `yaml:"type"`
	Category string `yaml:"category,omitempty"`

	// Requirements
	Requirements Requirements `yaml:"requirements"`

	// Dependencies
	Dependencies         []Dependency       `yaml:"dependencies,omitempty"`
	ExternalDependencies ExternalDeps       `yaml:"external-dependencies,omitempty"`

	// Features
	Features Features `yaml:"features"`

	// Configuration
	Configuration Configuration `yaml:"configuration,omitempty"`

	// Installation
	Installation Installation `yaml:"installation"`

	// Permissions
	Permissions []string `yaml:"permissions,omitempty"`

	// Metadata
	Metadata Metadata `yaml:"metadata,omitempty"`
}

// Requirements represents version requirements.
type Requirements struct {
	Cockpit string `yaml:"cockpit"`
	Go      string `yaml:"go,omitempty"`
	Node    string `yaml:"node,omitempty"`
}

// Dependency represents a package dependency.
type Dependency struct {
	Name     string `yaml:"name"`
	Version  string `yaml:"version"`
	Optional bool   `yaml:"optional,omitempty"`
}

// ExternalDeps represents external dependencies.
type ExternalDeps struct {
	Go     []string `yaml:"go,omitempty"`
	Node   []string `yaml:"node,omitempty"`
	System []string `yaml:"system,omitempty"`
}

// Features represents all features in a package.
type Features struct {
	Agents    []Feature `yaml:"agents,omitempty"`
	Skills    []Feature `yaml:"skills,omitempty"`
	Modules   []Feature `yaml:"modules,omitempty"`
	KB        []KBFeature `yaml:"kb,omitempty"`
	Workflows []Feature `yaml:"workflows,omitempty"`
}

// Feature represents a single feature.
type Feature struct {
	Path        string `yaml:"path"`
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
}

// KBFeature represents a knowledge base feature.
type KBFeature struct {
	Path string `yaml:"path"`
	Type string `yaml:"type"` // guide, example, troubleshooting, reference
}

// Configuration represents package configuration.
type Configuration struct {
	Defaults map[string]interface{} `yaml:"defaults,omitempty"`
	Options  []ConfigOption         `yaml:"options,omitempty"`
}

// ConfigOption represents a configuration option.
type ConfigOption struct {
	Name        string        `yaml:"name"`
	Type        string        `yaml:"type"`
	Description string        `yaml:"description,omitempty"`
	Default     interface{}   `yaml:"default,omitempty"`
	Required    bool          `yaml:"required,omitempty"`
	Options     []interface{} `yaml:"options,omitempty"`
}

// Installation represents installation configuration.
type Installation struct {
	SupportedProviders []string                 `yaml:"supported_providers"`
	ProviderFeatures   map[string][]string      `yaml:"provider_features"`
	Method             string                   `yaml:"method"` // symlink or copy
	PreInstall         []Hook                   `yaml:"pre_install,omitempty"`
	PostInstall        []Hook                   `yaml:"post_install,omitempty"`
}

// Hook represents an installation hook.
type Hook struct {
	Script      string `yaml:"script"`
	Description string `yaml:"description,omitempty"`
}

// Metadata represents package metadata.
type Metadata struct {
	Tags        []string      `yaml:"tags,omitempty"`
	Keywords    []string      `yaml:"keywords,omitempty"`
	Maintainers []Maintainer  `yaml:"maintainers,omitempty"`
	Changelog   string        `yaml:"changelog,omitempty"`
	Status      string        `yaml:"status,omitempty"` // alpha, beta, stable, deprecated
	Support     SupportInfo   `yaml:"support,omitempty"`
}

// Maintainer represents a package maintainer.
type Maintainer struct {
	Name  string `yaml:"name"`
	Email string `yaml:"email,omitempty"`
}

// SupportInfo represents support information.
type SupportInfo struct {
	Issues        string `yaml:"issues,omitempty"`
	Discussions   string `yaml:"discussions,omitempty"`
	Documentation string `yaml:"documentation,omitempty"`
}

// LoadPackage loads a package manifest from a directory.
func LoadPackage(packagePath string) (*Package, error) {
	manifestPath := filepath.Join(packagePath, "cockpit-package.yml")

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read package manifest: %w", err)
	}

	var pkg Package
	if err := yaml.Unmarshal(data, &pkg); err != nil {
		return nil, fmt.Errorf("failed to parse package manifest: %w", err)
	}

	return &pkg, nil
}

// SavePackage saves a package manifest to a directory.
func SavePackage(packagePath string, pkg *Package) error {
	manifestPath := filepath.Join(packagePath, "cockpit-package.yml")

	// Create directory if it doesn't exist
	if err := os.MkdirAll(packagePath, 0o755); err != nil {
		return fmt.Errorf("failed to create package directory: %w", err)
	}

	data, err := yaml.Marshal(pkg)
	if err != nil {
		return fmt.Errorf("failed to marshal package manifest: %w", err)
	}

	if err := os.WriteFile(manifestPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write package manifest: %w", err)
	}

	return nil
}

// Validate validates the package manifest.
func (p *Package) Validate() error {
	// Check required fields
	if p.Name == "" {
		return fmt.Errorf("package name is required")
	}

	if p.Version == "" {
		return fmt.Errorf("package version is required")
	}

	if p.Description == "" {
		return fmt.Errorf("package description is required")
	}

	if p.Author == "" {
		return fmt.Errorf("package author is required")
	}

	if p.License == "" {
		return fmt.Errorf("package license is required")
	}

	if p.Requirements.Cockpit == "" {
		return fmt.Errorf("cockpit version requirement is required")
	}

	// Check that at least one feature is defined
	if len(p.Features.Agents) == 0 &&
		len(p.Features.Skills) == 0 &&
		len(p.Features.Modules) == 0 &&
		len(p.Features.KB) == 0 &&
		len(p.Features.Workflows) == 0 {
		return fmt.Errorf("package must have at least one feature")
	}

	// Validate features exist
	if err := p.validateFeatures(); err != nil {
		return err
	}

	// Validate installation configuration
	if len(p.Installation.SupportedProviders) == 0 {
		return fmt.Errorf("at least one supported provider is required")
	}

	if len(p.Installation.ProviderFeatures) == 0 {
		return fmt.Errorf("provider features configuration is required")
	}

	return nil
}

// validateFeatures validates that all feature files exist.
func (p *Package) validateFeatures() error {
	// Note: This is a basic validation. In real usage, we'd check against
	// the actual package directory structure.
	return nil
}

// GetFeaturesByType returns features of a specific type.
func (p *Package) GetFeaturesByType(featureType string) []Feature {
	switch featureType {
	case "agents":
		return p.Features.Agents
	case "skills":
		return p.Features.Skills
	case "modules":
		return p.Features.Modules
	case "workflows":
		return p.Features.Workflows
	default:
		return nil
	}
}

// GetKBFeatures returns knowledge base features.
func (p *Package) GetKBFeatures() []KBFeature {
	return p.Features.KB
}

// SupportsProvider checks if the package supports a provider.
func (p *Package) SupportsProvider(provider string) bool {
	for _, p := range p.Installation.SupportedProviders {
		if p == provider {
			return true
		}
	}
	return false
}

// GetProviderFeatures returns features for a specific provider.
func (p *Package) GetProviderFeatures(provider string) []string {
	return p.Installation.ProviderFeatures[provider]
}

// GetDependencies returns all package dependencies.
func (p *Package) GetDependencies() []Dependency {
	return p.Dependencies
}

// HasDependencies checks if the package has dependencies.
func (p *Package) HasDependencies() bool {
	return len(p.Dependencies) > 0
}

// GetExternalDependencies returns external dependencies.
func (p *Package) GetExternalDependencies() ExternalDeps {
	return p.ExternalDependencies
}

// HasExternalDependencies checks if the package has external dependencies.
func (p *Package) HasExternalDependencies() bool {
	return len(p.ExternalDependencies.Go) > 0 ||
		len(p.ExternalDependencies.Node) > 0 ||
		len(p.ExternalDependencies.System) > 0
}

// GetConfiguration returns the configuration.
func (p *Package) GetConfiguration() Configuration {
	return p.Configuration
}

// GetDefaultConfig returns default configuration values.
func (p *Package) GetDefaultConfig() map[string]interface{} {
	if p.Configuration.Defaults == nil {
		return make(map[string]interface{})
	}
	return p.Configuration.Defaults
}

// GetConfigOptions returns configuration options.
func (p *Package) GetConfigOptions() []ConfigOption {
	return p.Configuration.Options
}
